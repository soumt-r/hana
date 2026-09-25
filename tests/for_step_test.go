package tests

// The back edge of a counting loop is one FOR_STEP in the bytecode VM (add, compare,
// jump; see bytecode/peephole.go's fuseLoopSteps). It must count, stop and fail exactly
// as the unfused instructions and the tree-walker do, on its fast path (numbers) and on
// its way back to the plain instructions (a constant counter, a type that refuses).

import (
	"strings"
	"testing"

	"github.com/soumt-r/hana/bytecode"
	"github.com/soumt-r/hana/errs"
	lexer "github.com/soumt-r/hana/lexer/hari"
	parser "github.com/soumt-r/hana/parser/hari"
)

func TestForStep(t *testing.T) {
	cases := []struct {
		name, code, out, err string
		fused                bool // the program's main chunk must have a FOR_STEP
	}{
		{"a range to a variable", `
'n'을 5로 정하자
1부터 'n'까지 반복하자 ('수'):
    '수'를 이어출력하자
`, "12345", "", true},
		{"a range down to a variable", `
'n'을 -2로 정하자
2부터 'n'까지 반복하자 ('수'):
    '수'를 이어출력하자
    " "를 이어출력하자
`, "2 1 0 -1 -2 ", "", true},
		{"a range to a fraction", `
'n'을 3.5로 정하자
'시작'을 0.25로 정하자
'시작'부터 'n'까지 반복하자 ('수'):
    '수'를 이어출력하자
    " "를 이어출력하자
`, "0.25 1.25 2.25 3.25 ", "", true},
		{"a range that runs once", `
'n'을 7로 정하자
'n'부터 'n'까지 반복하자 ('수'):
    '수'를 이어출력하자
`, "7", "", true},
		{"a range whose end is not a number", `
'n'을 "다섯"으로 정하자
1부터 'n'까지 반복하자 ('수'):
    '수'를 이어출력하자
`, "", "TypeError", true},
		{"a range whose body declares variables", `
'n'을 4로 정하자
'합'을 [숫자]인 0으로 정하자
1부터 'n'까지 반복하자 ('수'):
    '제곱'을 [숫자]인 '수' * '수'로 정하자
    '합'에 '제곱'을 더하자
'합'을 출력하자
`, "30", "", true},
		{"a range whose body assigns the loop variable", `
'n'을 3으로 정하자
1부터 'n'까지 반복하자 ('수'):
    '수'를 이어출력하자
    '수'에 10을 더하자
`, "123", "", true},
		{"a while loop with a variable step", `
'i'를 [숫자]인 0으로 정하자
'걸음'을 [숫자]인 3으로 정하자
('i' <= 10) 인 동안 반복하자:
    'i'를 이어출력하자
    " "를 이어출력하자
    'i'에 '걸음'을 더하자
`, "0 3 6 9 ", "", true},
		{"a while loop that adds the counter to itself", `
'i'를 [숫자]인 1로 정하자
('i' < 20) 인 동안 반복하자:
    'i'를 이어출력하자
    " "를 이어출력하자
    'i'에 'i'를 더하자
`, "1 2 4 8 16 ", "", false},
		{"counting up", `
1부터 5까지 반복하자 ('수'):
    '수'를 이어출력하자
`, "12345", "", true},
		{"counting down", `
5부터 1까지 반복하자 ('수'):
    '수'를 이어출력하자
`, "54321", "", true},
		{"a single pass", `
3부터 3까지 반복하자 ('수'):
    '수'를 이어출력하자
`, "3", "", true},
		{"a fraction start", `
0.5부터 3까지 반복하자 ('수'):
    '수'를 이어출력하자
    " "를 이어출력하자
`, "0.5 1.5 2.5 ", "", true},
		{"numbers past the boxed table", `
1048574부터 1048577까지 반복하자 ('수'):
    '수'를 이어출력하자
    " "를 이어출력하자
`, "1048574 1048575 1048576 1048577 ", "", true},
		{"nested loops", `
'합'을 [숫자]인 0으로 정하자
1부터 30까지 반복하자 ('가'):
    1부터 40까지 반복하자 ('나'):
        '합'에 ('가' * '나')를 더하자
'합'을 출력하자
`, "381300", "", true},
		{"break", `
1부터 10까지 반복하자 ('수'):
    만약 ('수' == 4) 라면:
        반복을 끝내자
    '수'를 이어출력하자
`, "123", "", true},
		{"a body that assigns the loop variable counts on its own", `
1부터 4까지 반복하자 ('수'):
    '수'를 이어출력하자
    '수'에 10을 더하자
`, "1234", "", true},
		{"a while loop to a variable end", `
'i'를 [숫자]인 0으로 정하자
'끝'을 [숫자]인 4로 정하자
('i' < '끝') 인 동안 반복하자:
    'i'를 이어출력하자
    'i'에 1을 더하자
`, "0123", "", true},
		{"the end changing inside the loop", `
'i'를 [숫자]인 0으로 정하자
'끝'을 [숫자]인 3으로 정하자
('i' < '끝') 인 동안 반복하자:
    만약 ('i' == 0) 라면:
        '끝'에 2를 더하자
    'i'를 이어출력하자
    'i'에 1을 더하자
`, "01234", "", true},
		{"a counter that is not a number", `
'i'를 [아무거나]인 0으로 정하자
('i' < 3) 인 동안 반복하자:
    'i'를 이어출력하자
    'i'에 1을 더하자
    만약 ('i' == 2) 라면:
        'i'를 "둘"로 정하자
`, "01", "TypeError", false},
		{"a constant counter", `
'i'를 0으로 고정하자
('i' < 3) 인 동안 반복하자:
    'i'를 이어출력하자
    'i'에 1을 더하자
`, "0", "상수", true},
		{"in a function", `
[숫자]를 돌려주는 <합계>를 만들자 ([숫자]인 'n'):
    '합'을 [숫자]인 0으로 정하자
    1부터 'n'까지 반복하자 ('수'):
        '합'에 '수'를 더하자
    '합'을 돌려주자
'i'를 [숫자]인 0으로 정하자
('i' < 3) 인 동안 반복하자:
    <합계>(10)을 이어출력하자
    'i'에 1을 더하자
`, "555555", "", true},
	}
	for _, c := range cases {
		if got := hasForStep(t, c.code); got != c.fused {
			t.Errorf("%s: FOR_STEP in main = %v, want %v", c.name, got, c.fused)
		}
		for engine, r := range engines(t, c.code) {
			out := strings.Join(r.out, "")
			gotErr := ""
			if r.err != nil {
				gotErr = r.err.Error() + " / " + errs.Localize(errs.Korean, r.err)
			}
			if out != c.out || (c.err == "") != (r.err == nil) || (c.err != "" && !strings.Contains(gotErr, c.err)) {
				t.Errorf("%s: %s: out %q err %q, want out %q err containing %q", c.name, engine, out, gotErr, c.out, c.err)
			}
		}
	}
}

func hasForStep(t *testing.T, code string) bool {
	t.Helper()
	p := parser.New(lexer.New(code))
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse error: %v", p.Errors())
	}
	for _, in := range bytecode.NewCompiler().Compile(prog).Main.Instructions {
		if in.Op == bytecode.FOR_STEP {
			return true
		}
	}
	return false
}
