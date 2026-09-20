// The native half of the http_server package: Go's net/http behind hana's native
// library ABI v1 (see vm/native_plugin.go). Build it for the current platform with
// build.sh or build.ps1 in this folder; it needs a C toolchain (cgo).
//
// It depends on nothing but the Go standard library, so it can be built and
// shipped on its own.
package main

/*
#include <stdlib.h>

static char* call_host(void* f, char* id, char* args) {
	return ((char* (*)(char*, char*))f)(id, args);
}
*/
import "C"

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"
)

var (
	mu     sync.Mutex
	host   unsafe.Pointer
	routes = map[string]string{} // "METHOD path" -> hana callback id
	server *http.Server
)

//export HanaAlloc
func HanaAlloc(n C.size_t) *C.char { return (*C.char)(C.malloc(n)) }

//export HanaFree
func HanaFree(p *C.char) { C.free(unsafe.Pointer(p)) }

//export HanaInit
func HanaInit(h unsafe.Pointer) {
	mu.Lock()
	host = h
	mu.Unlock()
}

func reply(ok interface{}) *C.char {
	text, _ := json.Marshal(map[string]interface{}{"ok": ok})
	return C.CString(string(text))
}

func fail(format string, args ...interface{}) *C.char {
	text, _ := json.Marshal(map[string]interface{}{"error": fmt.Sprintf(format, args...)})
	return C.CString(string(text))
}

// arguments decodes the JSON array hana sends.
func arguments(raw *C.char) ([]json.RawMessage, error) {
	var args []json.RawMessage
	if err := json.Unmarshal([]byte(C.GoString(raw)), &args); err != nil {
		return nil, err
	}
	return args, nil
}

// Route registers a handler: [method, path, {"$fn": id}].
//
//export Route
func Route(raw *C.char) *C.char {
	args, err := arguments(raw)
	if err != nil || len(args) != 3 {
		return fail("Route needs (method, path, handler)")
	}
	var method, path string
	var fn struct {
		ID string `json:"$fn"`
	}
	if json.Unmarshal(args[0], &method) != nil || json.Unmarshal(args[1], &path) != nil || json.Unmarshal(args[2], &fn) != nil || fn.ID == "" {
		return fail("Route needs a method, a path and a function")
	}
	mu.Lock()
	routes[strings.ToUpper(method)+" "+path] = fn.ID
	mu.Unlock()
	return reply(nil)
}

// Listen serves until Shutdown: [port] where port is a number (8080) or an
// address text ("127.0.0.1:8080").
//
//export Listen
func Listen(raw *C.char) *C.char {
	args, err := arguments(raw)
	if err != nil || len(args) != 1 {
		return fail("Listen needs (port)")
	}
	addr := ""
	var n float64
	var s string
	switch {
	case json.Unmarshal(args[0], &n) == nil:
		addr = ":" + strconv.Itoa(int(n))
	case json.Unmarshal(args[0], &s) == nil:
		addr = s
		if !strings.Contains(s, ":") {
			addr = ":" + s
		}
	default:
		return fail("the port must be a number or an address")
	}

	srv := &http.Server{Addr: addr, Handler: http.HandlerFunc(serve)}
	mu.Lock()
	if server != nil {
		mu.Unlock()
		return fail("the server is already running")
	}
	server = srv
	mu.Unlock()

	err = srv.ListenAndServe()
	mu.Lock()
	server = nil
	mu.Unlock()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fail("%v", err)
	}
	return reply(nil)
}

// Shutdown stops the server; Listen returns once it has. It may be called from
// a handler, so it does not wait for the handlers to finish.
//
//export Shutdown
func Shutdown(raw *C.char) *C.char {
	mu.Lock()
	srv := server
	mu.Unlock()
	if srv != nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if srv.Shutdown(ctx) != nil {
				srv.Close()
			}
		}()
	}
	return reply(nil)
}

// serve turns a request into a hana call and the answer into a response.
func serve(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	id, ok := routes[r.Method+" "+r.URL.Path]
	h := host
	mu.Unlock()
	if !ok || h == nil {
		http.NotFound(w, r)
		return
	}

	body, _ := io.ReadAll(io.LimitReader(r.Body, 10<<20))
	query := map[string]interface{}{}
	for k, v := range r.URL.Query() {
		query[k] = v[0]
	}
	headers := map[string]interface{}{}
	for k, v := range r.Header {
		headers[strings.ToLower(k)] = v[0]
	}
	request := map[string]interface{}{
		"method": r.Method, "path": r.URL.Path, "query": query, "headers": headers,
		"body": string(body), "remote": r.RemoteAddr,
	}
	payload, _ := json.Marshal([]interface{}{request})

	cid, cargs := C.CString(id), C.CString(string(payload))
	ret := C.call_host(h, cid, cargs)
	C.free(unsafe.Pointer(cid))
	C.free(unsafe.Pointer(cargs))
	if ret == nil {
		http.Error(w, "the handler returned nothing", http.StatusInternalServerError)
		return
	}
	text := C.GoString(ret)
	C.free(unsafe.Pointer(ret))

	var envelope struct {
		OK    json.RawMessage `json:"ok"`
		Error *string         `json:"error"`
	}
	if err := json.Unmarshal([]byte(text), &envelope); err != nil {
		http.Error(w, "bad answer from the handler", http.StatusInternalServerError)
		return
	}
	if envelope.Error != nil {
		http.Error(w, *envelope.Error, http.StatusInternalServerError)
		return
	}
	writeResponse(w, envelope.OK)
}

// writeResponse sends what a handler returned: null (an empty 200), a string
// (text/plain), or {"status", "body", "type", "headers"}.
func writeResponse(w http.ResponseWriter, ok json.RawMessage) {
	var text string
	if len(ok) == 0 || string(ok) == "null" {
		return
	}
	if json.Unmarshal(ok, &text) == nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		io.WriteString(w, text)
		return
	}
	var resp struct {
		Status  *float64               `json:"status"`
		Body    string                 `json:"body"`
		Type    string                 `json:"type"`
		Headers map[string]interface{} `json:"headers"`
	}
	if err := json.Unmarshal(ok, &resp); err != nil {
		http.Error(w, "the handler must return a string or a response", http.StatusInternalServerError)
		return
	}
	for k, v := range resp.Headers {
		w.Header().Set(k, fmt.Sprint(v))
	}
	if resp.Type != "" {
		w.Header().Set("Content-Type", resp.Type)
	} else if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	}
	status := http.StatusOK
	if resp.Status != nil {
		status = int(*resp.Status)
		if status < 100 || status > 999 {
			http.Error(w, fmt.Sprintf("%d is not an HTTP status", status), http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(status)
	io.WriteString(w, resp.Body)
}

func main() {}
