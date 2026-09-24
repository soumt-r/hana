package tests

import "testing"

// <숫자로>: numbers pass through, valid numeric strings parse, and anything
// else (including a partially-numeric string like "123abc", which the old
// fmt.Sscanf-based implementation silently accepted) raises ConversionError
// instead of silently returning 0.
func TestBuiltinToNumber(t *testing.T) {
	interp, err := runHari(t, `
'문자열에서'를 <숫자로>("42")로 정하자
'숫자통과'를 <숫자로>(3.5)로 정하자
`)
	if err != nil {
		t.Fatalf("unexpected runtime error: %v", err)
	}
	if got, _ := interp.GlobalEnv.Get("문자열에서"); got != 42.0 {
		t.Errorf("문자열에서 = %v, want 42", got)
	}
	if got, _ := interp.GlobalEnv.Get("숫자통과"); got != 3.5 {
		t.Errorf("숫자통과 = %v, want 3.5", got)
	}

	_, err = runHari(t, `'결과'를 <숫자로>("가나다")로 정하자`)
	requireErrorContains(t, err, "ConversionError")

	_, err = runHari(t, `'결과'를 <숫자로>("123abc")로 정하자`)
	requireErrorContains(t, err, "ConversionError")
}

// <코드로>: exactly one rune of input is required (Runtime spec 5.5); empty
// or multi-character strings must error instead of silently returning 0.
func TestBuiltinToCode(t *testing.T) {
	interp, err := runHari(t, `'결과'를 <코드로>("A")로 정하자`)
	if err != nil {
		t.Fatalf("unexpected runtime error: %v", err)
	}
	if got, _ := interp.GlobalEnv.Get("결과"); got != 65.0 {
		t.Errorf("결과 = %v, want 65", got)
	}

	_, err = runHari(t, `'결과'를 <코드로>("AB")로 정하자`)
	requireErrorContains(t, err, "ConversionError")

	_, err = runHari(t, `'결과'를 <코드로>("")로 정하자`)
	requireErrorContains(t, err, "ConversionError")
}

// <글자로>: a non-numeric argument must error instead of silently returning "".
func TestBuiltinToText(t *testing.T) {
	interp, err := runHari(t, `'결과'를 <글자로>(65)로 정하자`)
	if err != nil {
		t.Fatalf("unexpected runtime error: %v", err)
	}
	if got, _ := interp.GlobalEnv.Get("결과"); got != "A" {
		t.Errorf("결과 = %v, want \"A\"", got)
	}

	_, err = runHari(t, `'결과'를 <글자로>("A")로 정하자`)
	requireErrorContains(t, err, "ConversionError")
}
