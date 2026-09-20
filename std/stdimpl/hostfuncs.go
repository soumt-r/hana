package stdimpl

import (
	"sort"

	"github.com/soumt-r/hana/errs"
	"github.com/soumt-r/hana/std"
	"github.com/soumt-r/hana/value"
)

// Caller runs a function the program passed in (`<이름>`) with the given
// arguments. Each engine supplies its own: the functions of the two Go engines
// are different values, so the shared code only ever calls them through this.
type Caller func(fn interface{}, args []interface{}) (interface{}, error)

// HostFunc is a native function that takes functions of the program as
// arguments, like 변환하기(목록, <함수>).
type HostFunc func(call Caller, args []interface{}) (interface{}, error)

// HostImpls maps the function IDs that need a Caller to their implementation;
// every other ID in std.Modules is in Impls.
var HostImpls = map[string]HostFunc{
	std.ListMap:    listMap,
	std.ListFilter: listFilter,
	std.ListReduce: listReduce,
	std.ListFind:   listFind,
	std.ListAny:    listAny,
	std.ListAll:    listAll,
	std.ListSortBy: listSortBy,
}

// callArgs reads (list, function, ...) for a function that takes n arguments.
func callArgs(args []interface{}, n int) ([]interface{}, interface{}, error) {
	if err := exactly(args, n); err != nil {
		return nil, nil, err
	}
	list, err := listArg(args, 0)
	if err != nil {
		return nil, nil, err
	}
	return list, args[1], nil
}

// condition asks fn about one item; the answer has to be true or false.
func condition(call Caller, fn, item interface{}) (bool, error) {
	v, err := call(fn, []interface{}{item})
	if err != nil {
		return false, err
	}
	b, ok := v.(bool)
	if !ok {
		return false, errs.New(errs.CallbackNotBoolean)
	}
	return b, nil
}

func listMap(call Caller, args []interface{}) (interface{}, error) {
	list, fn, err := callArgs(args, 2)
	if err != nil {
		return nil, err
	}
	out := make([]interface{}, len(list))
	for i, item := range list {
		if out[i], err = call(fn, []interface{}{item}); err != nil {
			return nil, err
		}
	}
	return value.NewList(out), nil
}

func listFilter(call Caller, args []interface{}) (interface{}, error) {
	list, fn, err := callArgs(args, 2)
	if err != nil {
		return nil, err
	}
	out := []interface{}{}
	for _, item := range list {
		keep, err := condition(call, fn, item)
		if err != nil {
			return nil, err
		}
		if keep {
			out = append(out, item)
		}
	}
	return value.NewList(out), nil
}

// listReduce folds the list into one value: fn(누적값, 항목), starting from the
// third argument.
func listReduce(call Caller, args []interface{}) (interface{}, error) {
	list, fn, err := callArgs(args, 3)
	if err != nil {
		return nil, err
	}
	acc := args[2]
	for _, item := range list {
		if acc, err = call(fn, []interface{}{acc, item}); err != nil {
			return nil, err
		}
	}
	return acc, nil
}

// listFind returns the first item fn accepts, or 비어있음.
func listFind(call Caller, args []interface{}) (interface{}, error) {
	list, fn, err := callArgs(args, 2)
	if err != nil {
		return nil, err
	}
	for _, item := range list {
		found, err := condition(call, fn, item)
		if err != nil {
			return nil, err
		}
		if found {
			return item, nil
		}
	}
	return nil, nil
}

func listAny(call Caller, args []interface{}) (interface{}, error) {
	list, fn, err := callArgs(args, 2)
	if err != nil {
		return nil, err
	}
	for _, item := range list {
		found, err := condition(call, fn, item)
		if err != nil {
			return nil, err
		}
		if found {
			return true, nil
		}
	}
	return false, nil
}

func listAll(call Caller, args []interface{}) (interface{}, error) {
	list, fn, err := callArgs(args, 2)
	if err != nil {
		return nil, err
	}
	for _, item := range list {
		ok, err := condition(call, fn, item)
		if err != nil {
			return nil, err
		}
		if !ok {
			return false, nil
		}
	}
	return true, nil
}

// listSortBy returns the items ordered by the value fn gives each of them: all
// numbers or all strings, ascending; items with equal keys keep their order.
func listSortBy(call Caller, args []interface{}) (interface{}, error) {
	list, fn, err := callArgs(args, 2)
	if err != nil {
		return nil, err
	}
	keys := make([]interface{}, len(list))
	for i, item := range list {
		if keys[i], err = call(fn, []interface{}{item}); err != nil {
			return nil, err
		}
	}
	less := func(a, b interface{}) bool { return false }
	if len(keys) > 0 {
		switch keys[0].(type) {
		case float64:
			for _, k := range keys {
				if _, ok := k.(float64); !ok {
					return nil, errs.New(errs.ListNotSortable)
				}
			}
			less = func(a, b interface{}) bool { return a.(float64) < b.(float64) }
		case string:
			for _, k := range keys {
				if _, ok := k.(string); !ok {
					return nil, errs.New(errs.ListNotSortable)
				}
			}
			less = func(a, b interface{}) bool { return a.(string) < b.(string) }
		default:
			return nil, errs.New(errs.ListNotSortable)
		}
	}
	order := make([]int, len(list))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(x, y int) bool { return less(keys[order[x]], keys[order[y]]) })
	out := make([]interface{}, len(list))
	for i, at := range order {
		out[i] = list[at]
	}
	return value.NewList(out), nil
}
