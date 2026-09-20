//go:build !windows && !linux && !darwin

package native

// No way to load a shared library on this OS: importing a native package is a
// clear runtime error instead of a build failure.
func openRawLibrary(path string) (rawLibrary, error) { return nil, errUnsupportedOS }

func newHostCallback(fn func(a, b uintptr) uintptr) uintptr { return 0 }
