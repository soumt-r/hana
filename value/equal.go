package value

import "reflect"

// Equal is hana's == on two values: numbers, strings, booleans and 비어있음 by
// value; lists, dictionaries, objects and functions by identity (the same one).
// Go's own == on interface values would panic for dictionaries (Go maps cannot
// be compared), so both engines and 따라 나누자 compare through here.
func Equal(a, b interface{}) bool {
	am, aIsMap := a.(map[interface{}]interface{})
	bm, bIsMap := b.(map[interface{}]interface{})
	if aIsMap || bIsMap {
		return aIsMap && bIsMap && reflect.ValueOf(am).Pointer() == reflect.ValueOf(bm).Pointer()
	}
	return a == b
}
