// Package strcat joins two strings so that building a long string one piece at a
// time costs time proportional to its length instead of its length squared.
//
// Go strings are immutable, so `s = s + piece` copies all of s every time. Join
// keeps the result of a long join in a buffer with spare room and remembers it:
// when the next join's left side is exactly that result, the piece is written
// into the spare room and the new string shares the buffer. This is safe because
// a string's own bytes are never written again — only the bytes past its end are.
package strcat

import (
	"sync"
	"unsafe"
)

const (
	// minLen is where sharing starts to pay off; shorter joins are a plain copy.
	minLen = 256
	slots  = 4
)

type buffer struct {
	data *byte // start of the allocation
	used int   // bytes of it that some string already covers
	size int   // bytes allocated
}

var (
	mu      sync.Mutex
	recent  [slots]buffer
	nextOne int
)

// Join returns a + b.
func Join(a, b string) string {
	total := len(a) + len(b)
	if total < minLen || len(a) == 0 || len(b) == 0 {
		return a + b
	}

	mu.Lock()
	defer mu.Unlock()

	start := unsafe.StringData(a)
	for k := range recent {
		buf := &recent[k]
		if buf.data == start && buf.used == len(a) && buf.size >= total {
			copy(unsafe.Slice(buf.data, buf.size)[len(a):total], b)
			buf.used = total
			return unsafe.String(buf.data, total)
		}
	}

	size := total + total/2 + 64
	data := make([]byte, size)
	copy(data, a)
	copy(data[len(a):], b)
	recent[nextOne] = buffer{data: &data[0], used: total, size: size}
	nextOne = (nextOne + 1) % slots
	return unsafe.String(&data[0], total)
}
