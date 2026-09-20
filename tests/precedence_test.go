package tests

// Arithmetic follows the usual order: * / % bind tighter than + -, and operators
// of one level go left to right. Before, everything was read left to right, so
// 2 + 3 * 4 came out as 20.

import (
	"strings"
	"testing"
)

func TestArithmeticPrecedence(t *testing.T) {
	cases := []struct{ expr, want string }{
		{"2 + 3 * 4", "14"},
		{"2 * 3 + 4", "10"},
		{"10 - 4 / 2", "8"},
		{"2 * 3 + 4 * 5", "26"},
		{"100 / 10 / 5", "2"}, // same level: left to right
		{"10 - 3 - 2", "5"},   // (10 - 3) - 2, not 10 - (3 - 2)
		{"7 % 4 + 1", "4"},
		{"1 + 7 % 4", "4"},
		{"2 + 3 * 4 - 5", "9"},
		{"(2 + 3) * 4", "20"},
		{"2 * (3 + 4)", "14"},
		{"20 / 2 * 5", "50"},    // (20 / 2) * 5
		{"1 + 2 * 3 % 4", "3"},  // 1 + ((2 * 3) % 4)
		{"10 - 2 * 3 + 1", "5"}, // (10 - 6) + 1
	}
	var b strings.Builder
	var want []string
	for _, c := range cases {
		b.WriteString("(" + c.expr + ")를 출력하자\n")
		want = append(want, c.want)
	}
	for engine, r := range engines(t, b.String()) {
		if r.err != nil {
			t.Errorf("%s: %v", engine, r.err)
			continue
		}
		for i, w := range want {
			if i >= len(r.out) || r.out[i] != w {
				t.Errorf("%s %q: got %v, want %s", engine, cases[i].expr, r.out, w)
				break
			}
		}
	}
}

func TestPrecedenceWithVariablesAndComparisons(t *testing.T) {
	code := `
'a'를 [숫자]인 2로 정하자
'b'를 [숫자]인 3으로 정하자
'c'를 [숫자]인 4로 정하자
('a' + 'b' * 'c')를 출력하자
('a' * 'b' + 'c')를 출력하자
만약 ('a' + 'b' * 'c' == 14) 라면:
    "참"을 출력하자
만약 ('a' + 'b' * 'c' > 'a' * 'b' + 'c') 라면:
    "큼"을 출력하자
`
	for engine, r := range engines(t, code) {
		if r.err != nil || strings.Join(r.out, "|") != "14|10|참|큼" {
			t.Errorf("%s: got %v (%v)", engine, r.out, r.err)
		}
	}
}

func TestPrecedenceInKanade(t *testing.T) {
	code := "(2 + 3 * 4)を出力しよう\n(10 - 4 / 2)を出力しよう\n(100 / 10 / 5)を出力しよう\n"
	tw, err1 := runKanade(t, code)
	bc, err2 := runKanadeBytecode(t, code)
	for label, run := range map[string]struct {
		out []string
		err error
	}{"tree-walker": {tw.Output, err1}, "bytecode": {bc.Output, err2}} {
		if run.err != nil || strings.Join(run.out, "|") != "14|8|2" {
			t.Errorf("kanade %s: got %v (%v)", label, run.out, run.err)
		}
	}
}
