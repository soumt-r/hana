package tests

// [CSV] (RFC 4180): 파싱 makes a list of rows of texts, 문자열화 writes them back.
// Cells are never turned into numbers; quotes are added only where needed.

import (
	"strings"
	"testing"

	"github.com/soumt-r/hana/errs"
)

func TestCSVParse(t *testing.T) {
	cases := []struct{ call, want string }{
		{`<파싱>("a,b")`, "[[a, b]]"},
		{`<파싱>("a,b\nc,d")`, "[[a, b], [c, d]]"},
		{`<파싱>("a,b\nc,d\n")`, "[[a, b], [c, d]]"},      // a final line break starts no row
		{`<파싱>("a" + <글자로>(13) + "\nb")`, "[[a], [b]]"}, // CRLF; Hari text has no  escape
		{`<파싱>("\"x,y\",z")`, "[[x,y, z]]"},
		{`<파싱>("\"a\"\"b\"")`, `[[a"b]]`},
		{`<파싱>("\"a\nb\",c")`, "[[a\nb, c]]"}, // a line break inside quotes belongs to the cell
		{`<파싱>("a,,c")`, "[[a, , c]]"},
		{`<파싱>("a,")`, "[[a, ]]"},
		{`<파싱>("")`, "[]"},
		{`<파싱>("a;b", ";")`, "[[a, b]]"},
		{`<파싱>("a\tb", "\t")`, "[[a, b]]"},
		{`<파싱>("a\n\nb")`, "[[a], [], [b]]"}, // a blank line is a row with one empty cell
		{`<파싱>("a\"b,c")`, `[[a"b, c]]`},     // a quote inside a cell that did not start with one is a character
		{`<파싱>("\"\"")`, "[[]]"},
		{`<파싱>("1,2")`, "[[1, 2]]"}, // cells stay text
		{`<문자열화>([["a", 1, 2.5]])`, "a,1,2.5"},
		{`<문자열화>([["a,b", "c\"d"]])`, `"a,b","c""d"`},
		{`<문자열화>([["x"], ["y"]])`, "x\ny"},
		{`<문자열화>([[""]])`, `""`},
		{`<문자열화>([["a", 비어있음]])`, "a,"},
		{`<문자열화>([["a b", "c"]], ";")`, "a b;c"},
		{`<문자열화>([["a;b", "c"]], ";")`, `"a;b";c`},
		{`<문자열화>([["줄\n바꿈"]])`, "\"줄\n바꿈\""},
		{`<문자열화>([])`, ""},
	}
	var code strings.Builder
	code.WriteString("[CSV]에서 <파싱>과 <문자열화>를 가져오자\n")
	var want []string
	for _, c := range cases {
		code.WriteString(c.call + "를 출력하자\n")
		want = append(want, c.want)
	}
	for engine, r := range engines(t, code.String()) {
		if r.err != nil {
			t.Errorf("%s: %v", engine, r.err)
			continue
		}
		for i, w := range want {
			if i >= len(r.out) || r.out[i] != w {
				got := ""
				if i < len(r.out) {
					got = r.out[i]
				}
				t.Errorf("%s %s: got %q, want %q", engine, cases[i].call, got, w)
			}
		}
	}
}

func TestCSVRoundTrip(t *testing.T) {
	roundTrip := `
[CSV]에서 <파싱>과 <문자열화>를 가져오자
'행들'을 [목록]인 [["이름", "메모"], ["하나", "쉼표, 그리고 \"따옴표\""], ["둘", "줄\n바꿈"], ["", "빈칸"], ["끝"]]로 정하자
'글'을 [문자열]인 <문자열화>('행들')로 정하자
'다시'를 [목록]인 <파싱>('글')로 정하자
<문자열화>('다시')를 출력하자
'글'을 출력하자
`
	for engine, r := range engines(t, roundTrip) {
		if r.err != nil || len(r.out) != 2 || r.out[0] != r.out[1] {
			t.Errorf("%s: writing what was read gave a different text: %v (%v)", engine, r.out, r.err)
		}
	}
}

func TestCSVErrors(t *testing.T) {
	cases := []struct{ call, want string }{
		{`<파싱>("\"abc")`, "CSV 형식이 올바르지 않아서 읽을 수 없어요."},
		{`<파싱>("\"a\"b")`, "CSV 형식이 올바르지 않아서 읽을 수 없어요."},
		{`<파싱>("a", ",,")`, "구분자는 글자 하나여야 하고"},
		{`<파싱>("a", "")`, "구분자는 글자 하나여야 하고"},
		{`<파싱>("a", "\"")`, "구분자는 글자 하나여야 하고"},
		{`<파싱>("a", 1)`, "TypeError"},
		{`<파싱>(1)`, "TypeError"},
		{`<문자열화>([[참]])`, "CSV로 바꿀 수 없는 값이에요."},
		{`<문자열화>(["a"])`, "CSV로 바꿀 수 없는 값이에요."},
		{`<문자열화>([[]])`, "CSV로 바꿀 수 없는 값이에요."},
		{`<문자열화>([["a"]], "ab")`, "구분자는 글자 하나여야 하고"},
	}
	for _, c := range cases {
		code := "[CSV]에서 <파싱>과 <문자열화>를 가져오자\n" + c.call + "를 출력하자\n"
		for engine, r := range engines(t, code) {
			if r.err == nil || !strings.Contains(errs.Localize(errs.Korean, r.err), c.want) {
				t.Errorf("%s %s: want an error containing %q, got %v", engine, c.call, c.want, r.err)
			}
		}
	}
}
