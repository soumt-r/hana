package stdimpl

import (
	"math"
	"sort"

	"github.com/soumt-r/hana/errs"
	"github.com/soumt-r/hana/value"
)

// clean turns -0 into 0 so every engine prints the same thing.
func clean(f float64) float64 {
	if f == 0 {
		return 0
	}
	return f
}

func finite(f float64) (interface{}, error) {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return nil, errs.New(errs.MathDomain)
	}
	return clean(f), nil
}

func numbersArg(args []interface{}, i int) ([]float64, error) {
	l, ok := args[i].(*value.List)
	if !ok {
		return nil, errs.New(errs.NativeArgList, i+1)
	}
	out := make([]float64, len(l.Items))
	for n, v := range l.Items {
		f, ok := v.(float64)
		if !ok {
			return nil, errs.New(errs.NativeListNumbers, i+1)
		}
		out[n] = f
	}
	return out, nil
}

// one is a function of a single number that may leave its domain.
func one(f func(float64) float64) Func {
	return func(args []interface{}) (interface{}, error) {
		if err := exactly(args, 1); err != nil {
			return nil, err
		}
		x, err := numberArg(args, 0)
		if err != nil {
			return nil, err
		}
		return finite(f(x))
	}
}

func mathSqrt(args []interface{}) (interface{}, error) { return one(math.Sqrt)(args) }
func mathAbs(args []interface{}) (interface{}, error)  { return one(math.Abs)(args) }
func mathSin(args []interface{}) (interface{}, error)  { return one(math.Sin)(args) }
func mathCos(args []interface{}) (interface{}, error)  { return one(math.Cos)(args) }
func mathTan(args []interface{}) (interface{}, error)  { return one(math.Tan)(args) }

func mathPow(args []interface{}) (interface{}, error) {
	if err := exactly(args, 2); err != nil {
		return nil, err
	}
	base, err := numberArg(args, 0)
	if err != nil {
		return nil, err
	}
	exp, err := numberArg(args, 1)
	if err != nil {
		return nil, err
	}
	return finite(math.Pow(base, exp))
}

// mathRound rounds half away from zero, to a whole number or to 0..15 digits.
func mathRound(args []interface{}) (interface{}, error) {
	if err := between(args, 1, 2); err != nil {
		return nil, err
	}
	x, err := numberArg(args, 0)
	if err != nil {
		return nil, err
	}
	digits := int64(0)
	if len(args) == 2 {
		if digits, err = integerArg(args, 1); err != nil {
			return nil, err
		}
		if digits < 0 || digits > 15 {
			return nil, errs.New(errs.NativeArgInteger, 2)
		}
	}
	scale := 1.0
	for d := int64(0); d < digits; d++ {
		scale *= 10
	}
	return finite(math.Round(x*scale) / scale)
}

// mathLog is the natural logarithm, or the logarithm in a base.
func mathLog(args []interface{}) (interface{}, error) {
	if err := between(args, 1, 2); err != nil {
		return nil, err
	}
	x, err := numberArg(args, 0)
	if err != nil {
		return nil, err
	}
	if x <= 0 {
		return nil, errs.New(errs.MathDomain)
	}
	if len(args) == 1 {
		return finite(math.Log(x))
	}
	base, err := numberArg(args, 1)
	if err != nil {
		return nil, err
	}
	if base <= 0 || base == 1 {
		return nil, errs.New(errs.MathDomain)
	}
	return finite(math.Log(x) / math.Log(base))
}

func mathPi(args []interface{}) (interface{}, error) {
	if err := exactly(args, 0); err != nil {
		return nil, err
	}
	return math.Pi, nil
}

func mathGcd(args []interface{}) (interface{}, error) {
	if err := exactly(args, 2); err != nil {
		return nil, err
	}
	a, err := integerArg(args, 0)
	if err != nil {
		return nil, err
	}
	b, err := integerArg(args, 1)
	if err != nil {
		return nil, err
	}
	if a < 0 {
		a = -a
	}
	if b < 0 {
		b = -b
	}
	for b != 0 {
		a, b = b, a%b
	}
	return float64(a), nil
}

// mathFactorial is n! for whole n from 0 up to 170 (the last one a number holds).
func mathFactorial(args []interface{}) (interface{}, error) {
	if err := exactly(args, 1); err != nil {
		return nil, err
	}
	n, err := integerArg(args, 0)
	if err != nil {
		return nil, err
	}
	if n < 0 || n > 170 {
		return nil, errs.New(errs.MathDomain)
	}
	result := 1.0
	for i := int64(2); i <= n; i++ {
		result *= float64(i)
	}
	return result, nil
}

// ---- stats: every function takes one list of numbers

func statsSum(args []interface{}) (interface{}, error) {
	if err := exactly(args, 1); err != nil {
		return nil, err
	}
	xs, err := numbersArg(args, 0)
	if err != nil {
		return nil, err
	}
	total := 0.0
	for _, x := range xs {
		total += x
	}
	return finite(total)
}

func statsMin(args []interface{}) (interface{}, error) {
	return extreme(args, func(a, b float64) bool { return a < b })
}

func statsMax(args []interface{}) (interface{}, error) {
	return extreme(args, func(a, b float64) bool { return a > b })
}

func extreme(args []interface{}, better func(a, b float64) bool) (interface{}, error) {
	if err := exactly(args, 1); err != nil {
		return nil, err
	}
	xs, err := numbersArg(args, 0)
	if err != nil {
		return nil, err
	}
	if len(xs) == 0 {
		return nil, errs.New(errs.StatsNotEnough)
	}
	best := xs[0]
	for _, x := range xs[1:] {
		if better(x, best) {
			best = x
		}
	}
	return clean(best), nil
}

func mean(xs []float64) float64 {
	total := 0.0
	for _, x := range xs {
		total += x
	}
	return total / float64(len(xs))
}

func statsMean(args []interface{}) (interface{}, error) {
	if err := exactly(args, 1); err != nil {
		return nil, err
	}
	xs, err := numbersArg(args, 0)
	if err != nil {
		return nil, err
	}
	if len(xs) == 0 {
		return nil, errs.New(errs.StatsNotEnough)
	}
	return finite(mean(xs))
}

func statsMedian(args []interface{}) (interface{}, error) {
	if err := exactly(args, 1); err != nil {
		return nil, err
	}
	xs, err := numbersArg(args, 0)
	if err != nil {
		return nil, err
	}
	if len(xs) == 0 {
		return nil, errs.New(errs.StatsNotEnough)
	}
	sort.Float64s(xs)
	mid := len(xs) / 2
	if len(xs)%2 == 1 {
		return clean(xs[mid]), nil
	}
	return finite((xs[mid-1] + xs[mid]) / 2)
}

// statsStdev is the sample standard deviation (divides by n-1), like Python's
// statistics.stdev; it needs at least two values.
func statsStdev(args []interface{}) (interface{}, error) {
	if err := exactly(args, 1); err != nil {
		return nil, err
	}
	xs, err := numbersArg(args, 0)
	if err != nil {
		return nil, err
	}
	if len(xs) < 2 {
		return nil, errs.New(errs.StatsNotEnough)
	}
	m := mean(xs)
	sum := 0.0
	for _, x := range xs {
		d := x - m
		sum += d * d
	}
	return finite(math.Sqrt(sum / float64(len(xs)-1)))
}
