// A tiny native library that follows hana's ABI v1 (see vm/native_plugin.go);
// tests/native_plugin_test.go builds it with `go build -buildmode=c-shared`.
package main

/*
#include <stdlib.h>
#include <string.h>

static char* call_host(void* f, char* id, char* args) {
	return ((char* (*)(char*, char*))f)(id, args);
}
*/
import "C"

import (
	"sync"
	"time"
	"unsafe"
)

var host unsafe.Pointer

//export HanaAlloc
func HanaAlloc(n C.size_t) *C.char { return (*C.char)(C.malloc(n)) }

//export HanaFree
func HanaFree(p *C.char) { C.free(unsafe.Pointer(p)) }

//export HanaInit
func HanaInit(h unsafe.Pointer) { host = h }

// Echo returns its arguments: {"ok": [args...]}.
//
//export Echo
func Echo(args *C.char) *C.char { return C.CString(`{"ok":` + C.GoString(args) + `}`) }

// Fail always fails.
//
//export Fail
func Fail(args *C.char) *C.char { return C.CString(`{"error":"boom"}`) }

// Nothing returns NULL, which hana reads as 비어있음.
//
//export Nothing
func Nothing(args *C.char) *C.char { return nil }

// Garbage returns text that is not JSON.
//
//export Garbage
func Garbage(args *C.char) *C.char { return C.CString(`not json`) }

// CallHost is called as (fn, value): it calls fn(value) back in hana and
// returns hana's envelope as its own.
//
//export CallHost
func CallHost(args *C.char) *C.char {
	return callHost(C.GoString(args), false)
}

// CallHostFromThread does the same from another goroutine (another OS thread),
// the way an HTTP server calls its handlers.
//
//export CallHostFromThread
func CallHostFromThread(args *C.char) *C.char {
	return callHost(C.GoString(args), true)
}

func callHost(argsJSON string, inThread bool) *C.char {
	// argsJSON is [{"$fn":"cb_1"}, value]; pull the id and value out by hand
	// so the library needs no JSON dependency.
	const marker = `{"$fn":"`
	start := indexOf(argsJSON, marker) + len(marker)
	end := start
	for argsJSON[end] != '"' {
		end++
	}
	id := argsJSON[start:end]
	rest := argsJSON[end+3:]  // skip the closing quote, brace and the comma
	rest = rest[:len(rest)-1] // drop the closing bracket
	call := func() *C.char {
		cid, cargs := C.CString(id), C.CString("["+rest+"]")
		defer C.free(unsafe.Pointer(cid))
		defer C.free(unsafe.Pointer(cargs))
		return C.call_host(host, cid, cargs)
	}
	if !inThread {
		return call()
	}
	done := make(chan *C.char)
	go func() { done <- call() }()
	return <-done
}

// CallLater is called as (fn, n): it returns at once and, from another goroutine,
// calls fn() back n times. Join waits for those calls to finish.
//
//export CallLater
func CallLater(args *C.char) *C.char {
	text := C.GoString(args)
	start := indexOf(text, `{"$fn":"`) + len(`{"$fn":"`)
	end := start
	for text[end] != '"' {
		end++
	}
	id := text[start:end]
	n := 0
	for _, ch := range text[end+3:] {
		if ch >= '0' && ch <= '9' {
			n = n*10 + int(ch-'0')
		}
	}
	later.Add(1)
	go func() {
		defer later.Done()
		// wait until the calling program has left this native call and is running
		time.Sleep(5 * time.Millisecond)
		for i := 0; i < n; i++ {
			cid, cargs := C.CString(id), C.CString("[]")
			ret := C.call_host(host, cid, cargs)
			C.free(unsafe.Pointer(cid))
			C.free(unsafe.Pointer(cargs))
			C.free(unsafe.Pointer(ret))
		}
	}()
	return C.CString(`{"ok":null}`)
}

// Join waits for every CallLater goroutine.
//
//export Join
func Join(args *C.char) *C.char {
	later.Wait()
	return C.CString(`{"ok":null}`)
}

var later sync.WaitGroup

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func main() {}
