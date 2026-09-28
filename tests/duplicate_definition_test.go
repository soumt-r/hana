package tests

import (
	"strings"
	"testing"

	harilexer "github.com/soumt-r/hana/lexer/hari"
	kanadelexer "github.com/soumt-r/hana/lexer/kanade"
	hariparser "github.com/soumt-r/hana/parser/hari"
	kanadeparser "github.com/soumt-r/hana/parser/kanade"
)

func diagnostics(code string, kanade bool) []string {
	var ps *hariparser.Parser
	if kanade {
		ps = kanadeparser.New(kanadelexer.New(code))
	} else {
		ps = hariparser.New(harilexer.New(code))
	}
	ps.ParseProgram()
	var out []string
	for _, d := range ps.Diagnostics() {
		e := d.Err()
		out = append(out, string(e.Code)+" "+e.Error())
	}
	return out
}

// A function (or method) made twice in one place, and a class's second
// constructor, are syntax errors: Hari has no overloading. Overriding a
// parent's method, a class's own method beside its objects' of the same
// name, and the same name in different functions are not duplicates.
func TestDuplicateDefinitions(t *testing.T) {
	cases := []struct {
		name, code string
		kanade     bool
		want       []string
	}{
		{"top-level function", "<인사>를 만들자 ():\n    1을 돌려주자\n<인사>를 만들자 ('x'):\n    'x'를 돌려주자\n", false,
			[]string{"DuplicateFunction", "3", "<인사>"}},
		{"method", "[상자]를 설계하자:\n    <값>을 만들자 ():\n        1을 돌려주자\n    <값>을 만들자 ():\n        2를 돌려주자\n", false,
			[]string{"DuplicateFunction", "4", "<값>"}},
		{"constructor", "[점]을 설계하자:\n    처음 만들어질 때 () 다음과 같이 하자:\n        1을 출력하자\n    처음 만들어질 때 ('x') 다음과 같이 하자:\n        2를 출력하자\n", false,
			[]string{"DuplicateConstructor", "4"}},
		{"nested function", "<바깥>을 만들자 ():\n    <안>을 만들자 ():\n        1을 돌려주자\n    <안>을 만들자 ():\n        2를 돌려주자\n", false,
			[]string{"DuplicateFunction", "4", "<안>"}},
		{"a declaration that looks like an assignment", "<세배>를 만들자 ('x'):\n    ('x' * 3)를 돌려주자\n'보관'을 <세배>로 정하자\n", false,
			[]string{"DuplicateFunction", "3", "<세배>"}},
		{"kanade", "【箱】を設計しよう:\n    〈値〉を作ろう():\n        1を返そう\n    〈値〉を作ろう():\n        2を返そう\n", true,
			[]string{"DuplicateFunction", "4", "<値>"}},
	}
	for _, c := range cases {
		got := diagnostics(c.code, c.kanade)
		if len(got) != 1 {
			t.Errorf("%s: %q, want one diagnostic", c.name, got)
			continue
		}
		for _, w := range c.want {
			if !strings.Contains(got[0], w) {
				t.Errorf("%s: %q, want %q in it", c.name, got[0], w)
			}
		}
	}

	allowed := []string{
		// overriding in a child class
		"[동물]을 설계하자:\n    <소리>를 만들자 ():\n        \"...\"를 돌려주자\n[개]는 [동물]을 바탕으로 하고 설계하자:\n    <소리>를 만들자 ():\n        \"멍\"을 돌려주자\n",
		// a class's own method and its objects' method of the same name
		"[공장]을 설계하자:\n    <만들기>를 만들자 ():\n        1을 돌려주자\n    '우리'의 <만들기>를 만들자 ():\n        2를 돌려주자\n",
		// the same name in two functions' bodies
		"<가>를 만들자 ():\n    <도움>을 만들자 ():\n        1을 돌려주자\n<나>를 만들자 ():\n    <도움>을 만들자 ():\n        2를 돌려주자\n",
	}
	for _, code := range allowed {
		if got := diagnostics(code, false); len(got) != 0 {
			t.Errorf("%q: %q, want none", code, got)
		}
	}
}
