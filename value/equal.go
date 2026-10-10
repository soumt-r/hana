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

// CompareStrings is <, <=, >, >= on two strings (Runtime spec 5.4): codepoint
// order, which Go's byte order on UTF-8 already is. ok is false for any other
// operator.
func CompareStrings(op, a, b string) (res bool, ok bool) {
	switch op {
	case "<":
		return a < b, true
	case "<=":
		return a <= b, true
	case ">":
		return a > b, true
	case ">=":
		return a >= b, true
	}
	return false, false
}
