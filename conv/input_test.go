package conv

import (
	"github.com/soumt-r/hana/typecheck"
	"strings"
	"testing"
)

var haja = typecheck.Names{String: "문자열", Number: "숫자", Boolean: "논리"}

func TestParseInputNumbers(t *testing.T) {
	ok := map[string]float64{"20": 20, " 20 ": 20, "-3": -3, "+4": 4, "3.5": 3.5, ".5": 0.5, "5.": 5, "007": 7}
	for in, want := range ok {
		got, err := ParseInput("숫자", haja, in, "참", "거짓")
		if err != nil || got != want {
			t.Errorf("%q -> %v, %v; want %v", in, got, err, want)
		}
	}
	// JS/strconv leniencies must all be rejected so every engine agrees.
	for _, bad := range []string{"", "  ", "abc", "0x10", "1e3", "inf", "NaN", "1,000", "--1", "1 2", "스물"} {
		if _, err := ParseInput("숫자", haja, bad, "참", "거짓"); err == nil {
			t.Errorf("%q should not parse as a number", bad)
		}
	}
}

func TestParseInputBooleansStringsAndUnknownTypes(t *testing.T) {
	if v, err := ParseInput("논리", haja, " 참 ", "참", "거짓"); err != nil || v != true {
		t.Errorf("true: %v %v", v, err)
	}
	if v, err := ParseInput("논리", haja, "거짓", "참", "거짓"); err != nil || v != false {
		t.Errorf("false: %v %v", v, err)
	}
	if _, err := ParseInput("논리", haja, "네", "참", "거짓"); err == nil {
		t.Error("an unknown word is not a boolean")
	}
	if v, _ := ParseInput("", haja, "  그대로  ", "참", "거짓"); v != "  그대로  " {
		t.Errorf("the default type keeps the text as-is, got %q", v)
	}
	if v, _ := ParseInput("문자열", haja, "x", "참", "거짓"); v != "x" {
		t.Errorf("explicit string: %v", v)
	}
	if _, err := ParseInput("목록", haja, "x", "참", "거짓"); err == nil {
		t.Error("a type input cannot read must be an error")
	}
}

func TestLineReaderStripsLineEndingsAndNeverBlocksAtEOF(t *testing.T) {
	next := NewLineReader(strings.NewReader("첫째\r\n둘째\n마지막"))
	got := []string{next(), next(), next(), next(), next()}
	want := []string{"첫째", "둘째", "마지막", "", ""}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d = %q, want %q (all: %q)", i, got[i], want[i], got)
		}
	}
}
