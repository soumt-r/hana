package tests

import (
	"reflect"
	"testing"
)

// Printing a whole dictionary used to fall through to Go's %v ("map[국어:90
// 수학:80]"). It now shows {키: 값, ...}, sorted by key so the output is
// deterministic (Go maps have no insertion order); the docs sites'
// TypeScript engines format it identically.
func TestDictionaryDisplay(t *testing.T) {
	c := dualRun{
		hari: `'점수'를 [(문자열, 숫자)사전]인 {"수학": 80, "국어": 90}으로 정하자
'점수'를 출력하자
'목록'을 [(아무거나)목록]인 [{"a": 참}, 1]로 정하자
'목록'을 출력하자
'빈'을 [(문자열, 숫자)사전]인 {}로 정하자
'빈'을 출력하자
`,
		kanade: `『点数』を【(文字列,数字)辞書】の{「数学」:80,「国語」:90}にしよう
『点数』を出力しよう
『リスト』を【(何でも)リスト】の【{「a」:真},1】にしよう
『リスト』を出力しよう
『空』を【(文字列,数字)辞書】の{}にしよう
『空』を出力しよう
`,
	}
	outs, errs := runAllFour(t, c)
	for i, e := range errs {
		if e != nil {
			t.Fatalf("engine %d: %v", i, e)
		}
	}
	wantKo := []string{"{국어: 90, 수학: 80}", "[{a: 참}, 1]", "{}"}
	wantJa := []string{"{国語: 90, 数学: 80}", "[{a: 真}, 1]", "{}"}
	for i, want := range [][]string{wantKo, wantKo, wantJa, wantJa} {
		if !reflect.DeepEqual(outs[i], want) {
			t.Errorf("engine %d printed %q, want %q", i, outs[i], want)
		}
	}
}
