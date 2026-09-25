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
	// module is the frame that holds the top-level variables of the module this call's
	// function came from: where names are looked up after the call's own, instead of the
	// program's globals. nil for the program's own code.
	module *frame
}

// varSlot is one variable: its value, the [타입] it was declared with (Runtime
// spec 2.2; "" = none) and whether it was declared 고정하자/固定しよう.
//
// A loop or handler variable is declared anew even when the frame already has the
// name (Runtime spec 1.1: it lives in the loop's own scope and hides the outer one):
// the outer slot is then hidden, and the new one remembers it in shadows (its
// position + 1) so that dropping the new one with its scope brings the outer back.
//
// The fields are ordered so the struct packs into 48 bytes (56 in declaration-comment
// order): every call writes one per parameter and clears them all on return.
type varSlot struct {
	sym     symbol.Symbol
	shadows int32
	val     interface{}
	typ     string
	isConst bool
	hidden  bool
	marker  bool // a PUSH_SCOPE marker (see openScope)
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
	// from the end: the latest declaration of a name hides the earlier ones
	for i := len(f.slots) - 1; i >= 0; i-- {
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

// bindParam binds a parameter in a call's frame. Parameters are bound first, into a
// fresh frame, and their names differ, so while the frame is small the slot is just
// appended — no search for an existing one (put's find) on every call.
func (f *frame) bindParam(sym symbol.Symbol, val interface{}, typ string) {
	if f.index != nil || len(f.slots) >= frameIndexThreshold {
		f.put(sym, val).typ = typ
		return
	}
	f.slots = append(f.slots, varSlot{sym: sym, val: val, typ: typ})
}

// declare binds sym to val in a new slot even when the frame already has sym, hiding
// the earlier slot until closeScope drops the new one: how a loop or handler variable
// gets its own binding in the loop's scope.
func (f *frame) declare(sym symbol.Symbol, val interface{}) {
	outer := f.find(sym)
	f.slots = append(f.slots, varSlot{sym: sym, val: val})
	n := len(f.slots)
	if outer >= 0 {
		f.slots[outer].hidden = true
		f.slots[n-1].shadows = int32(outer) + 1
	}
	if f.index != nil {
		f.setIndex(sym, n-1)
	} else if n > frameIndexThreshold {
		for i := range f.slots {
			f.setIndex(f.slots[i].sym, i) // in order, so the latest slot of a name wins
		}
	}
}

// openScope records, in the hidden variable sym, the number of slots below it: what
// closeScope cuts the frame back to. A marker left behind by an aborted iteration
// is reused, so the position stays the same.
func (f *frame) openScope(sym symbol.Symbol) {
	i := f.find(sym)
	if i < 0 {
		f.put(sym, float64(len(f.slots))).marker = true
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
	f.dropFrom(at)
}

// unwindScopes drops the scopes still open above slot from: an error caught by a
// 일단 해보자 that started when the frame had from slots left the loops and handlers it
// jumped out of without their POP_SCOPE. Variables the try block declared outside any
// such scope stay, since the try block is not a scope of its own (Runtime spec 1.1).
func (f *frame) unwindScopes(from int) {
	for k := from; k < len(f.slots); k++ {
		if f.slots[k].marker {
			f.dropFrom(k)
			return
		}
	}
}

// dropFrom drops the slots from at on, bringing back what they hid.
func (f *frame) dropFrom(at int) {
	// Latest first, so a name dropped twice ends up pointing at the slot below them all.
	for k := len(f.slots) - 1; k >= at; k-- {
		s := &f.slots[k]
		if s.shadows > 0 {
			f.slots[s.shadows-1].hidden = false
		}
		if f.index != nil {
			f.index[s.sym] = s.shadows
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
