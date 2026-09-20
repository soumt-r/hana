//go:build windows

package cmd

import (
	"syscall"
	"unsafe"
)

// userLocaleName is the user's locale on Windows ("ko-KR", "ja-JP"), where the
// LANG-style variables are usually not set.
func userLocaleName() string {
	proc := syscall.NewLazyDLL("kernel32.dll").NewProc("GetUserDefaultLocaleName")
	var buf [85]uint16 // LOCALE_NAME_MAX_LENGTH
	n, _, _ := proc.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if n == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf[:])
}
