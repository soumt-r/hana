package tests

// 그리고/또는 (logical AND/OR) were previously unimplemented anywhere in the
// pipeline: not tokenized, not parsed, not evaluated. spec 2.4 requires
// parens around each operand when combining comparisons this way.

import "testing"

func TestLogicalAndOr(t *testing.T) {
	interp, err := runHari(t, `
'나이'를 [숫자]인 25로 정하자
'돈'을 [숫자]인 2000으로 정하자
'and결과'를 [논리]인 거짓으로 정하자
만약 ('나이'가 20 이상이다) 그리고 ('돈'이 1000보다 크다) 라면:
    'and결과'를 참으로 정하자

'날씨'를 [문자열]인 "눈"으로 정하자
'or결과'를 [논리]인 거짓으로 정하자
만약 ('날씨'와 "비"가 같다) 또는 ('날씨'와 "눈"이 같다) 라면:
    'or결과'를 참으로 정하자
`)
	if err != nil {
		t.Fatalf("unexpected runtime error: %v", err)
	}
	if got, _ := interp.GlobalEnv.Get("and결과"); got != true {
		t.Errorf("and결과 = %v, want true", got)
	}
	if got, _ := interp.GlobalEnv.Get("or결과"); got != true {
		t.Errorf("or결과 = %v, want true", got)
	}
}

// Runtime spec 5.2: 그리고 must not evaluate its right operand once the left
// is false, and 또는 must not evaluate its right operand once the left is
// true — a right-hand function call with a side effect must not run.
func TestLogicalShortCircuit(t *testing.T) {
	interp, err := runHari(t, `
'호출횟수'를 [숫자]인 0으로 정하자
<부작용>을 만들자 ():
    '호출횟수'를 ('호출횟수' + 1)로 정하자
    참을 돌려주자

만약 (거짓) 그리고 (<부작용>()) 라면:
    "실행 안 됨"을 출력하자

만약 (참) 또는 (<부작용>()) 라면:
    "실행은 됨"을 출력하자
`)
	if err != nil {
		t.Fatalf("unexpected runtime error: %v", err)
	}
	if got, _ := interp.GlobalEnv.Get("호출횟수"); got != 0.0 {
		t.Errorf("호출횟수 = %v, want 0 (short-circuit should never call <부작용>)", got)
	}
}
