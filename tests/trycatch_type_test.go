package tests

// Regression tests for TryStatement's catch-type matching (Runtime spec
// 4.2). The original implementation matched a typed handler with
// strings.Contains(err.Error(), h.Type.Name) plus a hardcoded
// `|| strings.Contains(errStr, "DivideByZeroError") || strings.Contains(errStr, "MathError")`
// fallback — meaning *any* typed handler would also catch a divide-by-zero
// or generic math error regardless of its declared type, and matching in
// general was a string-sniff rather than an actual class check.

import "testing"

// A handler typed for an unrelated user-defined class must not catch an
// engine-raised error (which has no Hari class at all) — it must fall
// through to the untyped handler instead.
func TestTypedCatchDoesNotMatchUnrelatedEngineError(t *testing.T) {
	interp, err := runHari(t, `
[네트워크오류]를 설계하자:
    '메시지'를 [문자열]인 ""로 정하자
    처음 만들어질 때 ('메시지') 다음과 같이 하자:
        '나'의 '메시지'를 '메시지'로 정하자

'결과문자'를 [문자열]인 ""로 정하자

일단 해보자:
    '결과'를 10 / 0으로 정하자
[네트워크오류]가 발생했다면 ('에러'):
    '결과문자'를 "wrong-handler"로 정하자
오류가 발생했다면 ('에러'):
    '결과문자'를 "generic-handler"로 정하자
`)
	if err != nil {
		t.Fatalf("unexpected runtime error: %v", err)
	}
	if got, _ := interp.GlobalEnv.Get("결과문자"); got != "generic-handler" {
		t.Errorf("결과문자 = %v, want \"generic-handler\" (DivideByZeroError must not match an unrelated typed handler)", got)
	}
}

// A typed handler must match a thrown value that's an instance of a
// subclass of its declared type (upcasting), and a handler for an
// unrelated type declared earlier must be skipped correctly.
func TestTypedCatchMatchesViaUpcasting(t *testing.T) {
	interp, err := runHari(t, `
[네트워크오류]를 설계하자:
    '메시지'를 [문자열]인 ""로 정하자
    처음 만들어질 때 ('메시지') 다음과 같이 하자:
        '나'의 '메시지'를 '메시지'로 정하자

[네트워크오류]를 바탕으로 [연결시간초과오류]를 설계하자:
    처음 만들어질 때 ('메시지') 다음과 같이 하자:
        부모의 <처음 만들어질 때>('메시지')을 실행하자

'결과문자'를 [문자열]인 ""로 정하자

일단 해보자:
    새로운 [연결시간초과오류]("연결 시간 초과")를 발생시키자
[수학오류]가 발생했다면 ('에러'):
    '결과문자'를 "wrong-handler"로 정하자
[네트워크오류]가 발생했다면 ('에러'):
    '결과문자'를 '에러'의 '메시지'로 정하자
오류가 발생했다면 ('에러'):
    '결과문자'를 "wrong-fallback"으로 정하자
`)
	if err != nil {
		t.Fatalf("unexpected runtime error: %v", err)
	}
	if got, _ := interp.GlobalEnv.Get("결과문자"); got != "연결 시간 초과" {
		t.Errorf("결과문자 = %v, want \"연결 시간 초과\" (parent-type handler should match a subclass instance)", got)
	}
}
