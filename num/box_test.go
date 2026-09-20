package num

import (
	"math"
	"testing"
)

func TestBoxIsTheSameValueAsAConversion(t *testing.T) {
	for _, f := range []float64{0, 1, -1, 2.5, -1024, -1025, 65535, 65536, 1e18, math.MaxFloat64, math.Inf(1), 3.0000001, 12345} {
		got, ok := Box(f).(float64)
		if !ok || got != f {
			t.Errorf("Box(%v) = %v (%v)", f, got, ok)
		}
	}
	if got := Box(math.NaN()).(float64); !math.IsNaN(got) {
		t.Errorf("Box(NaN) = %v", got)
	}
}

func TestBoxOfAWholeNumberDoesNotAllocate(t *testing.T) {
	var sink interface{}
	n := 0.0
	allocs := testing.AllocsPerRun(1000, func() {
		n = float64(int(n+1) % 1000)
		sink = Box(n)
	})
	if allocs != 0 {
		t.Errorf("Box allocated %v times per call", allocs)
	}
	_ = sink
}

func TestBoxedNumbersKeepTheirValuesWhenOthersAreBoxed(t *testing.T) {
	a := Box(7)
	b := Box(7)
	c := Box(8)
	if a != b || a == c || a.(float64) != 7 || c.(float64) != 8 {
		t.Errorf("a=%v b=%v c=%v", a, b, c)
	}
}
