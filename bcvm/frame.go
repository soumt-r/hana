package bcvm

import (
	"github.com/soumt-r/hana/symbol"
)

// frame holds one function/method call's (or the top level's) local
// variables, plus — inside an instance or static method body — the bound
// 나 (this may be nil, e.g. inside a static method) and the 우리 class name.
type frame struct {
	slots     []varSlot
	index     []int32 // Symbol -> slot position + 1 (0: none); only kept once the frame is big
	this      *Object
	selfClass string
}

// varSlot is one variable: its value, the [타입] it was declared with (Runtime
// spec 2.2; "" = none) and whether it was declared 고정하자/固定しよう.
type varSlot struct {
	sym     symbol.Symbol
	val     interface{}
	typ     string
	isConst bool
}

// Most frames hold a handful of variables, where scanning a slice of small
// integers is fastest and needs no index; the top level can hold many, so past
// this size the frame keeps a Symbol-indexed table as well.
const frameIndexThreshold = 12

func newFrame() *frame {
	return &frame{}
}

// Call frames are recycled: nothing keeps a frame after its call returns, and
// a call otherwise allocates the frame and its variable slice every time.
func (vm *VM) takeFrame() *frame {
	if n := len(vm.framePool); n > 0 {
		f := vm.framePool[n-1]
		vm.framePool = vm.framePool[:n-1]
		return f
	}
	return &frame{slots: make([]varSlot, 0, 8)}
}

func (vm *VM) giveFrame(f *frame) {
	if cap(f.slots) > 64 || len(vm.framePool) >= 64 {
		return
	}
	clear(f.slots)
	*f = frame{slots: f.slots[:0]}
	vm.framePool = append(vm.framePool, f)
}

func (f *frame) find(sym symbol.Symbol) int {
	if f.index != nil {
		if int(sym) < len(f.index) {
			return int(f.index[sym]) - 1
		}
		return -1
	}
	for i := range f.slots {
		if f.slots[i].sym == sym {
			return i
		}
	}
	return -1
}

func (f *frame) get(sym symbol.Symbol) (interface{}, bool) {
	if i := f.find(sym); i >= 0 {
		return f.slots[i].val, true
	}
	return nil, false
}

// put binds sym to val, declaring it if the frame does not have it yet, and
// returns the slot (valid until the next put).
func (f *frame) put(sym symbol.Symbol, val interface{}) *varSlot {
	if i := f.find(sym); i >= 0 {
		f.slots[i].val = val
		return &f.slots[i]
	}
	f.slots = append(f.slots, varSlot{sym: sym, val: val})
	n := len(f.slots)
	if f.index != nil {
		f.setIndex(sym, n-1)
	} else if n > frameIndexThreshold {
		for i := range f.slots {
			f.setIndex(f.slots[i].sym, i)
		}
	}
	return &f.slots[n-1]
}

// openScope records, in the hidden variable sym, the number of slots below it: what
// closeScope cuts the frame back to. A marker left behind by an aborted iteration
// is reused, so the position stays the same.
func (f *frame) openScope(sym symbol.Symbol) {
	i := f.find(sym)
	if i < 0 {
		f.put(sym, float64(len(f.slots)))
		return
	}
	f.slots[i].val = float64(i)
}

// closeScope drops the marker and every variable declared after it.
func (f *frame) closeScope(sym symbol.Symbol) {
	i := f.find(sym)
	if i < 0 {
		return
	}
	at := int(f.slots[i].val.(float64))
	if at > len(f.slots) {
		return
	}
	if f.index != nil {
		for _, s := range f.slots[at:] {
			f.index[s.sym] = 0
		}
	}
	clear(f.slots[at:])
	f.slots = f.slots[:at]
}

func (f *frame) setIndex(sym symbol.Symbol, slot int) {
	if int(sym) >= len(f.index) {
		grown := make([]int32, symbol.Count())
		copy(grown, f.index)
		f.index = grown
	}
	f.index[sym] = int32(slot) + 1
}
