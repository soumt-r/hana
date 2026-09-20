//go:build linux || darwin

package native

import "github.com/ebitengine/purego"

type unixLibrary struct{ handle uintptr }

// openRawLibrary uses purego, which loads shared libraries without cgo, so hana
// still builds with CGO_ENABLED=0 for every OS.
func openRawLibrary(path string) (rawLibrary, error) {
	h, err := purego.Dlopen(path, purego.RTLD_NOW|purego.RTLD_LOCAL)
	if err != nil {
		return nil, err
	}
	return &unixLibrary{handle: h}, nil
}

func (l *unixLibrary) symbol(name string) (uintptr, error) {
	return purego.Dlsym(l.handle, name)
}

func (l *unixLibrary) call(fn uintptr, args ...uintptr) uintptr {
	r1, _, _ := purego.SyscallN(fn, args...)
	return r1
}

// newHostCallback makes a C-callable function pointer out of fn.
func newHostCallback(fn func(a, b uintptr) uintptr) uintptr {
	return purego.NewCallback(fn)
}
