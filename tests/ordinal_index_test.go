package tests

// '목록'의 'k'번째 is one operand. The member chain used to stop before 번째, so
// whatever followed it was read as something else: `- 5` became a dropped
// SOV right-hand side (a wrong number, no error), `+ 1` and comparisons were
// syntax errors, and `'격자'의 2번째의 1번째` lost its second index.

import (
	"reflect"
	"testing"

	lexer "github.com/soumt-r/hana/lexer/hari"
	parser "github.com/soumt-r/hana/parser/hari"
)

func TestOrdinalIndexIsOneOperand(t *testing.T) {
	c := dualRun{
		hari: `'a'를 [(숫자)목록]인 [10, 20, 30]로 정하자
'k'를 [숫자]인 2로 정하자
('a'의 'k'번째 - 5)을 출력하자
'y'를 [숫자]인 'a'의 3번째 - 'a'의 1번째로 정하자
'y'를 출력하자
('a'의 2번째 + 1)을 출력하자
('a'의 'k'번째 * 3)을 출력하자
'x'를 [숫자]인 'a'의 'k'번째 + 1로 정하자
'x'를 출력하자
('a'의 1번째 < 'a'의 2번째)을 출력하자
('a'의 'k'번째와 20이 같다)을 출력하자
'격자'를 [목록]인 [[1, 2], [3, 4]]로 정하자
'격자'의 2번째의 1번째를 출력하자
'줄'을 [목록]인 []로 정하자
'줄' 뒤에 '격자'의 2번째의 2번째를 추가하자
'줄'을 출력하자
('a'의 3번째 값 - 1)을 출력하자
`,
		kanade: `『a』を【(数字)リスト】の【10, 20, 30】にしよう
『k』を【数字】の2にしよう
(『a』の『k』番目 - 5)を出力しよう
『y』を【数字】の『a』の3番目 - 『a』の1番目にしよう
『y』を出力しよう
(『a』の2番目 + 1)を出力しよう
(『a』の『k』番目 * 3)を出力しよう
『x』を【数字】の『a』の『k』番目 + 1にしよう
『x』を出力しよう
(『a』の1番目 < 『a』の2番目)を出力しよう
(『a』の『k』番目と20が同じだ)を出力しよう
『格子』を【リスト】の【【1, 2】, 【3, 4】】にしよう
『格子』の2番目の1番目を出力しよう
『行』を【リスト】の【】にしよう
『行』の後ろに『格子』の2番目の2番目を追加しよう
『行』を出力しよう
(『a』の3番目の値 - 1)を出力しよう
`,
	}
	want := map[bool][]string{
		false: {"15", "20", "21", "60", "21", "참", "참", "3", "[4]", "29"},
		true:  {"15", "20", "21", "60", "21", "真", "真", "3", "[4]", "29"},
	}
	outs, errs := runAllFour(t, c)
	for i, name := range []string{"hari tree", "hari bytecode", "kanade tree", "kanade bytecode"} {
		if errs[i] != nil {
			t.Errorf("%s: %v", name, errs[i])
			continue
		}
		if w := want[i >= 2]; !reflect.DeepEqual(outs[i], w) {
			t.Errorf("%s: got %v, want %v", name, outs[i], w)
		}
	}
}

// A comparison's right side that no comparison follows is a syntax error at
// its first token, not silently dropped.
func TestComparisonWithoutOperatorIsASyntaxError(t *testing.T) {
	p := parser.New(lexer.New("'x'를 3으로 정하자\n('x' 5)을 출력하자\n"))
	p.ParseProgram()
	diags := p.Diagnostics()
	if len(diags) != 1 || diags[0].Literal != "5" || diags[0].Line != 2 || diags[0].Col != 5 {
		t.Fatalf("got %+v, want one diagnostic at 2:5 (0-based column) for '5'", diags)
	}
}
