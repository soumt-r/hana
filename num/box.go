// Package num boxes numbers cheaply.
//
// Every number a program computes is a float64 held in an interface{} (the value
// type both engines share), and Go allocates eight bytes on the heap each time a
// float64 becomes an interface{}: the arithmetic of a loop spent a quarter of its
// time in the allocator. Box hands out the interface of a whole number in a small
// range from a table made once, with no allocation, and only falls back to the
// allocation for the rest. The table is read-only: nothing changes a number in place.
package num

import "unsafe"

const (
	low  = -1024
	high = 1 << 20
)

// table[i] holds the number low+i once Box has been asked for it: the table starts as
// zeros, which the operating system gives out without touching memory, and an entry is
// filled the first time its number is boxed.
var table [high - low]float64

// eface is what an interface{} looks like inside.
type eface struct {
	typ  unsafe.Pointer
	data unsafe.Pointer
}

var float64Type unsafe.Pointer

func init() {
	var sample interface{} = float64(1.5)
	float64Type = (*eface)(unsafe.Pointer(&sample)).typ
}

// Box is f as an interface{}: the same value as interface{}(f), without allocating for
// a whole number from -1024 up to 1048575. (-0 comes out as 0, which is how the
// engines print and compare it anyway.)
func Box(f float64) interface{} {
	if f >= low && f < high {
		if i := int(f); float64(i) == f {
			slot := &table[i-low]
			if *slot != f {
				*slot = f // 0 needs no filling, and -0 is stored as 0
			}
			var v interface{}
			e := (*eface)(unsafe.Pointer(&v))
			e.typ = float64Type
			e.data = unsafe.Pointer(slot)
			return v
		}
	}
	return f
}
