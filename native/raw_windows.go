//go:build windows

package native

import (
	"syscall"
)

type winLibrary struct{ handle syscall.Handle }

func openRawLibrary(path string) (rawLibrary, error) {
	h, err := syscall.LoadLibrary(path)
	if err != nil {
		return nil, err
	}
	return &winLibrary{handle: h}, nil
}

func (l *winLibrary) symbol(name string) (uintptr, error) {
	return syscall.GetProcAddress(l.handle, name)
}

func (l *winLibrary) call(fn uintptr, args ...uintptr) uintptr {
	r1, _, _ := syscall.SyscallN(fn, args...)
	return r1
}

// newHostCallback makes a C-callable function pointer out of fn.
func newHostCallback(fn func(a, b uintptr) uintptr) uintptr {
	return syscall.NewCallback(fn)
}
