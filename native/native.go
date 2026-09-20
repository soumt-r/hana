// Package native loads native libraries (packages/<모듈>/native/*.dll|.so|.dylib)
// for either engine: the tree-walker (vm) and the bytecode VM (bcvm) both use it.
// It is the way a third-party package reaches the operating system; hana's own
// native modules (예: [수학], [파일]) are compiled into the binary (see std) and
// never go through here.
//
// The binary interface (ABI v1) is small and language-neutral, so a library can
// be written in any language that can export C functions (Go with
// -buildmode=c-shared, C, Rust, ...):
//
//	char* HanaAlloc(size_t n)        // malloc-like; hana writes results here
//	void  HanaFree(char* p)          // frees what HanaAlloc or a function returned
//	void  HanaInit(hostcall)         // optional: hana's callback, see below
//	char* <Function>(char* argsJSON) // one exported function per native function
//
// Arguments arrive as one JSON array. A function returns a JSON envelope,
// {"ok": value} or {"error": "message"} (NULL means {"ok": null}); hana copies
// it and calls HanaFree. A hana function passed as an argument is sent as
// {"$fn": "cb_1"}; the library gives that id back to the callback
//
//	char* hostcall(char* fnID, char* argsJSON)
//
// which runs the hana function and returns its envelope in memory obtained
// from HanaAlloc (the library frees it with HanaFree). Callbacks may come from
// any thread of the library.
//
// Callbacks and the running program never execute hana code at the same time:
// the host (an engine) holds its execution lock while the program runs and gives
// it up only while the program is waiting inside a native call (Host.BeginNative),
// so a callback from another thread simply waits for its turn. Passing the same
// function to native code again reuses its id instead of registering another.
package native

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"unsafe"

	"github.com/soumt-r/hana/errs"
	"github.com/soumt-r/hana/pkg"
	"github.com/soumt-r/hana/std/stdimpl"
	"github.com/soumt-r/hana/value"
)

// Host is the engine side of a native call.
type Host interface {
	// Call runs a function value (a declared function, a bound method, ...) with
	// evaluated arguments, waiting for the program's execution lock first.
	Call(fn interface{}, args []interface{}) (interface{}, error)
	// Identify reports whether v is a function value that can be sent to native
	// code, and a comparable key that is the same each time the same function is
	// passed (so it is registered once).
	Identify(v interface{}) (key interface{}, ok bool)
	// BeginNative is called when the program enters a native function. It lets
	// callbacks run while the program waits, and returns the function that takes
	// the execution lock back when the native call returns.
	BeginNative() (end func())
}

// ExecLock is what an engine holds while it runs hana code: an engine acquires
// it for the whole run (and for each callback), and BeginNative gives it up for
// the length of a native call. Only the goroutine that holds it may release it.
type ExecLock struct {
	mu   sync.Mutex
	held bool // read only by the goroutine that holds mu
}

// Acquire takes the lock, waiting for whoever runs hana code now.
func (l *ExecLock) Acquire() { l.mu.Lock(); l.held = true }

// Release gives it up.
func (l *ExecLock) Release() { l.held = false; l.mu.Unlock() }

// Begin is the whole of Host.BeginNative for an engine: release the lock if the
// calling program holds it, and return what takes it back. Outside a run (nothing
// holds the lock) it does nothing.
func (l *ExecLock) Begin() func() {
	if !l.held {
		return func() {}
	}
	l.Release()
	return l.Acquire
}

// Func calls one exported function; host is the engine making the call.
type Func func(host Host, args []interface{}) (interface{}, error)

// rawLibrary is the operating system's part: find a symbol and call it with
// integer/pointer arguments (see raw_*.go).
type rawLibrary interface {
	symbol(name string) (uintptr, error)
	call(fn uintptr, args ...uintptr) uintptr
}

// Library is a loaded library that follows the ABI above.
type Library struct {
	raw         rawLibrary
	path        string
	alloc, free uintptr
	host        uintptr // the callback pointer handed to HanaInit
}

var (
	openMu   sync.Mutex
	openLibs = map[string]*Library{}

	callbackMu sync.Mutex
	callbacks  = map[string]registered{}
	callbackID = map[callbackKey]string{}
	callbackN  int
)

type registered struct {
	host Host
	fn   interface{}
}

type callbackKey struct {
	host Host
	key  interface{}
}

var errUnsupportedOS = errors.New("native libraries are not supported on this OS")

// Open loads path once; opening it again returns the same library (its HanaInit
// has already run).
func Open(path string) (*Library, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	openMu.Lock()
	defer openMu.Unlock()
	if lib, ok := openLibs[abs]; ok {
		return lib, nil
	}
	raw, err := openRawLibrary(abs)
	if err != nil {
		if errors.Is(err, errUnsupportedOS) {
			return nil, errs.New(errs.ImportPluginUnsupported, runtime.GOOS)
		}
		return nil, errs.New(errs.ImportNativeLoadFailed, path)
	}
	lib := &Library{raw: raw, path: path}
	if lib.alloc, err = raw.symbol("HanaAlloc"); err != nil {
		return nil, errs.New(errs.ImportNativeLoadFailed, path)
	}
	if lib.free, err = raw.symbol("HanaFree"); err != nil {
		return nil, errs.New(errs.ImportNativeLoadFailed, path)
	}
	if init, err := raw.symbol("HanaInit"); err == nil {
		lib.host = newHostCallback(lib.handleHostCall)
		raw.call(init, lib.host)
	}
	openLibs[abs] = lib
	return lib, nil
}

// LibraryPath is the native library file to load for a package: the one its
// manifest declares for this platform, or the legacy <모듈>.<확장자> next to it.
func LibraryPath(pkgDir, module string, manifest *pkg.Manifest) (string, error) {
	if manifest.HasNative() {
		file, ok := manifest.NativeFor(pkg.Platform())
		if !ok {
			return "", errs.New(errs.ImportNativeMissing, module, pkg.Platform())
		}
		path := filepath.Join(pkgDir, filepath.FromSlash(file.File))
		if _, err := os.Stat(path); err != nil {
			return "", errs.New(errs.ImportNativeMissing, module, pkg.Platform())
		}
		return path, nil
	}
	path := filepath.Join(pkgDir, module+pkg.LibraryExt(runtime.GOOS))
	if _, err := os.Stat(path); err != nil {
		return "", errs.New(errs.ImportDLLNotFound, module)
	}
	return path, nil
}

// Bundled maps a package name to the library file a packed program carries for
// it (see the pack package). OpenModule uses it before looking for package
// folders, so a packed program needs no packages/ folder at all.
var Bundled map[string]string

// OpenModule loads the native library of the package called module: the file a
// packed program carries for it, else the one in its package folder.
func OpenModule(module string) (*Library, error) {
	if path, ok := Bundled[module]; ok {
		if _, err := os.Stat(path); err != nil {
			return nil, errs.New(errs.ImportDLLNotFound, module)
		}
		return Open(path)
	}
	pkgDir := pkg.Dir(module)
	manifest, err := pkg.Load(pkgDir)
	if err != nil {
		return nil, errs.New(errs.ImportManifestInvalid, module, err.Error())
	}
	return OpenPackage(pkgDir, module, manifest)
}

// OpenPackage loads the native library of packages/<module> (its manifest is
// read here, so the working folder decides which package it is).
func OpenPackage(pkgDir, module string, manifest *pkg.Manifest) (*Library, error) {
	path, err := LibraryPath(pkgDir, module, manifest)
	if err != nil {
		return nil, err
	}
	return Open(path)
}

// Lookup finds an exported function by name.
func (l *Library) Lookup(name string) (Func, bool) {
	fn, err := l.raw.symbol(name)
	if err != nil {
		return nil, false
	}
	return func(host Host, args []interface{}) (interface{}, error) {
		return l.callFunction(fn, name, host, args)
	}, true
}

func (l *Library) callFunction(fn uintptr, name string, host Host, args []interface{}) (interface{}, error) {
	payload, err := stdimpl.ToJSON(wrap(host, value.NewList(args)))
	if err != nil {
		return nil, err
	}
	buf := append([]byte(payload), 0)
	end := host.BeginNative()
	ret := l.raw.call(fn, uintptr(unsafe.Pointer(&buf[0])))
	end()
	runtime.KeepAlive(buf)
	if ret == 0 {
		return nil, nil
	}
	text := readCString(ret)
	l.raw.call(l.free, ret)
	return decodeEnvelope(name, text)
}

// decodeEnvelope turns {"ok": v} / {"error": "m"} into a value or an error.
func decodeEnvelope(function, text string) (interface{}, error) {
	v, err := stdimpl.FromJSON(text)
	if err != nil {
		return nil, errs.New(errs.NativeCallFailed, function, "the library did not return valid JSON")
	}
	env, ok := v.(map[interface{}]interface{})
	if !ok {
		return nil, errs.New(errs.NativeCallFailed, function, "the library did not return an envelope")
	}
	if msg, failed := env["error"]; failed {
		return nil, errs.New(errs.NativeCallFailed, function, fmt.Sprint(msg))
	}
	return env["ok"], nil
}

// wrap replaces function values with {"$fn": id} so they survive JSON.
func wrap(host Host, v interface{}) interface{} {
	if key, ok := host.Identify(v); ok {
		return map[interface{}]interface{}{"$fn": register(host, key, v)}
	}
	switch x := v.(type) {
	case *value.List:
		out := make([]interface{}, len(x.Items))
		for i, el := range x.Items {
			out[i] = wrap(host, el)
		}
		return value.NewList(out)
	case map[interface{}]interface{}:
		out := make(map[interface{}]interface{}, len(x))
		for k, el := range x {
			out[k] = wrap(host, el)
		}
		return out
	}
	return v
}

// register gives fn an id, reusing the one it already has.
func register(host Host, key, fn interface{}) string {
	callbackMu.Lock()
	defer callbackMu.Unlock()
	k := callbackKey{host: host, key: key}
	if id, ok := callbackID[k]; ok {
		return id
	}
	callbackN++
	id := fmt.Sprintf("cb_%d", callbackN)
	callbacks[id] = registered{host: host, fn: fn}
	callbackID[k] = id
	return id
}

// handleHostCall is the callback the library invokes: run a hana function and
// hand back its envelope in library-allocated memory.
func (l *Library) handleHostCall(idPtr, argsPtr uintptr) (result uintptr) {
	defer func() {
		if r := recover(); r != nil {
			result = l.allocString(envelopeError(fmt.Sprint(r)))
		}
	}()
	id, argsJSON := readCString(idPtr), readCString(argsPtr)

	callbackMu.Lock()
	cb, ok := callbacks[id]
	callbackMu.Unlock()
	if !ok {
		return l.allocString(envelopeError("unknown callback " + id))
	}
	parsed, err := stdimpl.FromJSON(argsJSON)
	if err != nil {
		return l.allocString(envelopeError(errs.Message(errs.English, err)))
	}
	var args []interface{}
	if list, ok := parsed.(*value.List); ok {
		args = list.Items
	}

	ret, err := cb.host.Call(cb.fn, args)
	if err != nil {
		return l.allocString(envelopeError(errs.Message(errs.English, err)))
	}
	text, err := stdimpl.ToJSON(map[interface{}]interface{}{"ok": ret})
	if err != nil {
		return l.allocString(envelopeError(errs.Message(errs.English, err)))
	}
	return l.allocString(text)
}

func envelopeError(message string) string {
	text, _ := stdimpl.ToJSON(map[interface{}]interface{}{"error": message})
	return text
}

// allocString copies s into memory from the library's HanaAlloc.
func (l *Library) allocString(s string) uintptr {
	ptr := l.raw.call(l.alloc, uintptr(len(s)+1))
	if ptr == 0 {
		return 0
	}
	dst := unsafe.Slice((*byte)(pointerFrom(ptr)), len(s)+1)
	copy(dst, s)
	dst[len(s)] = 0
	return ptr
}

// pointerFrom turns an address in memory the Go runtime does not own (the
// library's C heap) into a pointer. Going through a pointer-sized variable
// keeps `go vet` from mistaking it for a Go pointer arithmetic bug.
func pointerFrom(addr uintptr) unsafe.Pointer {
	return *(*unsafe.Pointer)(unsafe.Pointer(&addr))
}

// readCString reads a NUL-terminated C string.
func readCString(addr uintptr) string {
	if addr == 0 {
		return ""
	}
	base := pointerFrom(addr)
	n := 0
	for *(*byte)(unsafe.Add(base, n)) != 0 {
		n++
	}
	return string(unsafe.Slice((*byte)(base), n))
}
