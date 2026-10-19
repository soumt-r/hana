package tests

// Three small fixes found while rewriting the tutorials:
//   - a minus in front of anything but a number literal (-'x', -(1 + 2)) used to
//     parse into the text "알수없음: -"; it now means 0 - operand.
//   - string escapes are read left to right once, so `\\n` is a backslash and an
//     n, and Kanade's \「 \」 print the bracket (they kept the backslash before).
//   - a Kanade variable name may start with a digit (『1位』), like Hari's '1등'.

import (
	"strings"
	"testing"

	"github.com/soumt-r/hana/ast"
)

func TestUnaryMinus(t *testing.T) {
	code := `'x'를 3으로 정하자
(-'x')를 출력하자
'y'를 -'x'로 정하자
'y'를 출력하자
(-(1 + 2))를 출력하자
(-'x' * 2)를 출력하자
(10 - -'x')를 출력하자
'목'을 [1, 2, 3]으로 정하자
(-'목'의 '길이')를 출력하자
(-'목'의 2번째 + 1)를 출력하자
만약 (-'x'가 0보다 작다) 라면:
    "음수"를 출력하자
틀"{-'x'}"를 출력하자
(-3)을 출력하자
`
	want := "-3|-3|-3|-6|13|-3|-1|음수|-3|-3"
	for engine, r := range engines(t, code) {
		if r.err != nil {
			t.Errorf("%s: %v", engine, r.err)
			continue
		}
		if got := strings.Join(r.out, "|"); got != want {
			t.Errorf("%s: got %q, want %q", engine, got, want)
		}
	}
}

func TestUnescape(t *testing.T) {
	cases := []struct{ in, want string }{
		{`a\nb`, "a\nb"},
		{`a\\nb`, `a\nb`},
		{`c:\\temp`, `c:\temp`},
		{`x\ty`, "x\ty"},
		{`그가 \"안녕\"`, `그가 "안녕"`},
		{`彼は\「やあ\」`, "彼は「やあ」"},
		{`\q`, `\q`},
		{`끝\`, `끝\`},
		{`끝\\`, `끝\`},
	}
	for _, c := range cases {
		if got := ast.Unescape(c.in); got != c.want {
			t.Errorf("Unescape(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestEscapesInBothEngines(t *testing.T) {
	code := `"a\\nb"를 출력하자
"c:\\temp"를 출력하자
"그가 \"안녕\""를 출력하자
`
	want := `a\nb|c:\temp|그가 "안녕"`
	for engine, r := range engines(t, code) {
		if r.err != nil {
			t.Errorf("%s: %v", engine, r.err)
			continue
		}
		if got := strings.Join(r.out, "|"); got != want {
			t.Errorf("%s: got %q, want %q", engine, got, want)
		}
	}
}

func TestKanadeBracketEscapesAndDigitNames(t *testing.T) {
	code := `「彼は\「やあ\」と言った」を出力しよう
『名前』を「花」にしよう
枠「{『名前』}は\「こんにちは\」と言った」を出力しよう
『1位』を(-『名前』の『長さ』)にしよう
『1位』を出力しよう
`
	want := "彼は「やあ」と言った|花は「こんにちは」と言った|-1"
	interp, err := runKanade(t, code)
	if err != nil {
		t.Fatalf("tree-walker: %v", err)
	}
	if got := strings.Join(interp.Output, "|"); got != want {
		t.Errorf("tree-walker: got %q, want %q", got, want)
	}
	vm, err := runKanadeBytecode(t, code)
	if err != nil {
		t.Fatalf("bytecode: %v", err)
	}
	if got := strings.Join(vm.Output, "|"); got != want {
		t.Errorf("bytecode: got %q, want %q", got, want)
	}
}
