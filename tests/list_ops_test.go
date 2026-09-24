package tests

import "testing"

// ListPopExpression ('목록' 앞에서/뒤에서 꺼낸 값) must both return the popped
// value and mutate the original list, for both front and back. Regression
// test: this syntax previously wasn't recognized as an expression at all —
// the parser silently misparsed "뒤에서 꺼낸 값" as stray tokens and the
// enclosing VariableDeclaration ended up aliasing the target name to the
// entire original list instead of the popped element.
func TestListPopExpression(t *testing.T) {
	interp, err := runHari(t, `
'과일'을 [(문자열)목록]인 ["사과", "포도"]로 정하자
'과일' 앞에 "바나나"를 추가하자
'과일' 뒤에 "수박"을 추가하자

'과일' 앞에서 꺼내자
'꺼낸과일'을 [문자열]인 '과일' 뒤에서 꺼낸 값으로 정하자
'남은개수'를 '과일'의 '길이'로 정하자
'첫번째'를 '과일'의 1번째로 정하자
`)
	if err != nil {
		t.Fatalf("unexpected runtime error: %v", err)
	}
	if got, _ := interp.GlobalEnv.Get("꺼낸과일"); got != "수박" {
		t.Errorf("꺼낸과일 = %v, want \"수박\"", got)
	}
	if got, _ := interp.GlobalEnv.Get("남은개수"); got != 2.0 {
		t.Errorf("남은개수 = %v, want 2", got)
	}
	if got, _ := interp.GlobalEnv.Get("첫번째"); got != "사과" {
		t.Errorf("첫번째 = %v, want \"사과\"", got)
	}
}

// Runtime spec 5.1: popping from an empty list must raise
// IndexOutOfBoundsError, for both the statement and expression forms.
func TestListPopEmptyListErrors(t *testing.T) {
	_, err := runHari(t, `
'빈목록'을 [목록]인 []로 정하자
'빈목록' 뒤에서 꺼내자
`)
	requireErrorContains(t, err, "IndexOutOfBoundsError")

	_, err = runHari(t, `
'빈목록'을 [목록]인 []로 정하자
'값'을 '빈목록' 뒤에서 꺼낸 값으로 정하자
`)
	requireErrorContains(t, err, "IndexOutOfBoundsError")
}
