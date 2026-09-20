package tests

// Regression tests for the string builtin pseudo-methods (자르기/바꾸기/
// 분리하기/포함확인) on the tree-walking interpreter. 바꾸기/분리하기/
// 포함확인 used to type-assert their args directly (e.g. args[0].(string))
// with no length/type check first, which panicked — crashing the whole
// process, uncatchable by 일단 해보자 — on a bad call. Found while adding
// these to the bytecode VM and fixed here to match: a clean
// ArgumentError/TypeError instead. tests/bytecode_test.go covers the same
// methods on the bytecode VM.

import (
	"strings"
	"testing"
)

func TestStringPseudoMethods(t *testing.T) {
	interp, err := runHaja(t, `
'문장'을 [문자열]인 "안녕하세요 세계"로 정하자
틀"{'문장'의 <자르기>(1, 2)}"를 출력하자
틀"{'문장'의 <바꾸기>("세계", "하자")}"를 출력하자
틀"{'문장'의 <분리하기>(" ")}"를 출력하자
틀"{'문장'의 <포함확인>("세계")}"를 출력하자
틀"{'문장'의 <포함확인>("없음")}"를 출력하자
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"안녕", "안녕하세요 하자", "[안녕하세요, 세계]", "참", "거짓"}
	if strings.Join(interp.Output, ",") != strings.Join(want, ",") {
		t.Errorf("Output = %v, want %v", interp.Output, want)
	}
}

// This used to be a Go runtime panic (args[0].(string) on a missing arg),
// crashing the process instead of raising a catchable Haja error.
func TestStringPseudoMethodMissingArgDoesNotPanic(t *testing.T) {
	interp, err := runHaja(t, `
'문장'을 [문자열]인 "test"로 정하자
일단 해보자:
    '문장'의 <포함확인>()을 실행하자
오류가 발생했다면 ('에러'):
    틀"잡힘: {'에러'의 '메시지'}"를 출력하자
"안 죽음"을 출력하자
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"잡힘: ArgumentError: 인자가 1개 필요해요.", "안 죽음"}
	if strings.Join(interp.Output, ",") != strings.Join(want, ",") {
		t.Errorf("Output = %v, want %v", interp.Output, want)
	}
}

// This used to be a Go runtime panic (Go slicing requires low <= high),
// since the original code never clamped start past end.
func TestStringSliceStartAfterEndDoesNotPanic(t *testing.T) {
	interp, err := runHaja(t, `
'문장'을 [문자열]인 "안녕하세요"로 정하자
'결과'를 [문자열]인 '문장'의 <자르기>(5, 2)로 정하자
"안 죽음"을 출력하자
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, _ := interp.GlobalEnv.Get("결과"); got != "" {
		t.Errorf("결과 = %v, want \"\"", got)
	}
	if len(interp.Output) != 1 || interp.Output[0] != "안 죽음" {
		t.Errorf("Output = %v, want [\"안 죽음\"]", interp.Output)
	}
}
