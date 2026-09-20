package stdimpl

import (
	"math/rand"

	"github.com/soumt-r/hana/errs"
)

// randomFloat is a number in [0, 1).
func randomFloat(args []interface{}) (interface{}, error) {
	if err := exactly(args, 0); err != nil {
		return nil, err
	}
	return rand.Float64(), nil
}

// randomInt is a whole number between min and max, both included.
func randomInt(args []interface{}) (interface{}, error) {
	if err := exactly(args, 2); err != nil {
		return nil, err
	}
	min, err := integerArg(args, 0)
	if err != nil {
		return nil, err
	}
	max, err := integerArg(args, 1)
	if err != nil {
		return nil, err
	}
	if min > max {
		return nil, errs.New(errs.RandomRange, min, max)
	}
	return float64(min + rand.Int63n(max-min+1)), nil
}

func randomChoice(args []interface{}) (interface{}, error) {
	if err := exactly(args, 1); err != nil {
		return nil, err
	}
	list, err := listArg(args, 0)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, errs.New(errs.RandomEmpty)
	}
	return list[rand.Intn(len(list))], nil
}

// randomShuffle returns a shuffled copy; lists are values, so the argument is untouched.
func randomShuffle(args []interface{}) (interface{}, error) {
	if err := exactly(args, 1); err != nil {
		return nil, err
	}
	list, err := listArg(args, 0)
	if err != nil {
		return nil, err
	}
	out := make([]interface{}, len(list))
	copy(out, list)
	rand.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out, nil
}
