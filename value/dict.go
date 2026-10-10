package value

import "sort"

// SortedKeys is the order `'사전'의 '키'마다 반복하자` walks a dictionary's keys
// in (Runtime spec 5.1): numbers from small to large, then strings in codepoint
// order, then 거짓 and 참. Go maps have no order, so both engines and the
// browser engines sort the same way; keys of any other kind (a list or an
// object used as a key) come last in no set order.
func SortedKeys(m map[interface{}]interface{}) []interface{} {
	keys := make([]interface{}, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.SliceStable(keys, func(a, b int) bool { return keyLess(keys[a], keys[b]) })
	return keys
}

func keyRank(k interface{}) int {
	switch k.(type) {
	case float64:
		return 0
	case string:
		return 1
	case bool:
		return 2
	}
	return 3
}

func keyLess(a, b interface{}) bool {
	ra, rb := keyRank(a), keyRank(b)
	if ra != rb {
		return ra < rb
	}
	switch x := a.(type) {
	case float64:
		return x < b.(float64)
	case string:
		return x < b.(string)
	case bool:
		return !x && b.(bool)
	}
	return false
}

// DictHas is a dictionary's <포함확인>(키). A dictionary is never a key (Go
// cannot hash maps), so asking for one is just false.
func DictHas(m map[interface{}]interface{}, key interface{}) bool {
	if _, isDict := key.(map[interface{}]interface{}); isDict {
		return false
	}
	_, has := m[key]
	return has
}
