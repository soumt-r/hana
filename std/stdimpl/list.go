package stdimpl

import (
	"math"
	"sort"

	"github.com/soumt-r/hana/errs"
)

// listSort returns a sorted copy: all numbers (ascending) or all strings
// (by code point).
func listSort(args []interface{}) (interface{}, error) {
	if err := exactly(args, 1); err != nil {
		return nil, err
	}
	list, err := listArg(args, 0)
	if err != nil {
		return nil, err
	}
	out := make([]interface{}, len(list))
	copy(out, list)
	if len(out) == 0 {
		return out, nil
	}
	switch out[0].(type) {
	case float64:
		nums := make([]float64, len(out))
		for i, v := range out {
			f, ok := v.(float64)
			if !ok {
				return nil, errs.New(errs.ListNotSortable)
			}
			nums[i] = f
		}
		sort.Float64s(nums)
		for i, f := range nums {
			out[i] = clean(f)
		}
	case string:
		strs := make([]string, len(out))
		for i, v := range out {
			s, ok := v.(string)
			if !ok {
				return nil, errs.New(errs.ListNotSortable)
			}
			strs[i] = s
		}
		sort.Strings(strs)
		for i, s := range strs {
			out[i] = s
		}
	default:
		return nil, errs.New(errs.ListNotSortable)
	}
	return out, nil
}

func listReverse(args []interface{}) (interface{}, error) {
	if err := exactly(args, 1); err != nil {
		return nil, err
	}
	list, err := listArg(args, 0)
	if err != nil {
		return nil, err
	}
	out := make([]interface{}, len(list))
	for i, v := range list {
		out[len(list)-1-i] = v
	}
	return out, nil
}

// listUnique keeps the first of each equal value, in order; only numbers,
// strings, booleans and 비어있음 can be compared.
func listUnique(args []interface{}) (interface{}, error) {
	if err := exactly(args, 1); err != nil {
		return nil, err
	}
	list, err := listArg(args, 0)
	if err != nil {
		return nil, err
	}
	seen := map[interface{}]bool{}
	out := []interface{}{}
	for _, v := range list {
		switch v.(type) {
		case nil, bool, float64, string:
		default:
			return nil, errs.New(errs.ListValueUnsupported)
		}
		if seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out, nil
}

// listRange is the whole numbers from start to end, both included. The step
// defaults to 1 (or -1 when counting down); a step pointing away from end
// gives an empty list.
func listRange(args []interface{}) (interface{}, error) {
	if err := between(args, 2, 3); err != nil {
		return nil, err
	}
	start, err := integerArg(args, 0)
	if err != nil {
		return nil, err
	}
	end, err := integerArg(args, 1)
	if err != nil {
		return nil, err
	}
	step := int64(1)
	if start > end {
		step = -1
	}
	if len(args) == 3 {
		if step, err = integerArg(args, 2); err != nil {
			return nil, err
		}
		if step == 0 {
			return nil, errs.New(errs.RangeStepZero)
		}
	}
	out := []interface{}{}
	if (step > 0 && start > end) || (step < 0 && start < end) {
		return out, nil
	}
	count := (end-start)/step + 1
	if count > maxResultRunes {
		return nil, errs.New(errs.ResultTooLarge)
	}
	for n := int64(0); n < count; n++ {
		out = append(out, float64(start+n*step))
	}
	return out, nil
}

// listFlatten opens one level of nested lists.
func listFlatten(args []interface{}) (interface{}, error) {
	if err := exactly(args, 1); err != nil {
		return nil, err
	}
	list, err := listArg(args, 0)
	if err != nil {
		return nil, err
	}
	out := []interface{}{}
	for _, v := range list {
		if inner, ok := v.([]interface{}); ok {
			out = append(out, inner...)
		} else {
			out = append(out, v)
		}
	}
	return out, nil
}

// listChunk cuts a list into lists of n (the last may be shorter).
func listChunk(args []interface{}) (interface{}, error) {
	if err := exactly(args, 2); err != nil {
		return nil, err
	}
	list, err := listArg(args, 0)
	if err != nil {
		return nil, err
	}
	n, err := integerArg(args, 1)
	if err != nil {
		return nil, err
	}
	if n < 1 {
		return nil, errs.New(errs.NativeArgInteger, 2)
	}
	out := []interface{}{}
	for i := int64(0); i < int64(len(list)); i += n {
		end := int64(math.Min(float64(i+n), float64(len(list))))
		piece := make([]interface{}, end-i)
		copy(piece, list[i:end])
		out = append(out, piece)
	}
	return out, nil
}

// listZip pairs up two lists element by element, up to the shorter one.
func listZip(args []interface{}) (interface{}, error) {
	if err := exactly(args, 2); err != nil {
		return nil, err
	}
	a, err := listArg(args, 0)
	if err != nil {
		return nil, err
	}
	b, err := listArg(args, 1)
	if err != nil {
		return nil, err
	}
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	out := make([]interface{}, n)
	for i := 0; i < n; i++ {
		out[i] = []interface{}{a[i], b[i]}
	}
	return out, nil
}
