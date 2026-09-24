package tests

// No implicit casting and no arithmetic on 비어있음 (Runtime spec 2.2 and 2.4),
// on tree-walker/bytecode x hari/kanade. Equality is exempt from the null rule.

import (
	"strings"
	"testing"
)

func TestNoImplicitCastingInOperators(t *testing.T) {
	runTypeCases(t, []typeCase{
		{
			name:    "number + string",
			hari:    "(10 + \"안녕\")를 출력하자\n",
			kanade:  "(10 + 「こんにちは」)を出力しよう\n",
			wantErr: "cannot combine",
		},
		{
			name:    "string + number",
			hari:    "(\"a\" + 1)를 출력하자\n",
			kanade:  "(「あ」 + 1)を出力しよう\n",
			wantErr: "cannot combine",
		},
		{
			name:    "string - string",
			hari:    "(\"a\" - \"b\")를 출력하자\n",
			kanade:  "(「あ」 - 「い」)を出力しよう\n",
			wantErr: "cannot combine",
		},
		{
			name:    "string > number",
			hari:    "(\"a\" > 1)을 출력하자\n",
			kanade:  "(「あ」 > 1)を出力しよう\n",
			wantErr: "cannot combine",
		},
		{
			name:   "string + string still joins",
			hari:   "(\"가\" + \"나\")를 출력하자\n",
			kanade: "(「あ」 + 「い」)を出力しよう\n",
			want:   "가나", wantKanade: "あい",
		},
		{
			name:   "numbers still calculate",
			hari:   "(2 * 3 + 1)을 출력하자\n",
			kanade: "(2 * 3 + 1)を出力しよう\n",
			want:   "7",
		},
		{
			name:    "compound assignment on a string",
			hari:    "'a'를 \"x\"로 정하자\n'a'에 3을 더하자\n",
			kanade:  "『a』を「x」にしよう\n『a』に3を足そう\n",
			wantErr: "cannot combine",
		},
	})
}

func TestNullOperandsAreErrorsExceptInEquality(t *testing.T) {
	runTypeCases(t, []typeCase{
		{
			name:    "number + null",
			hari:    "(1 + 비어있음)을 출력하자\n",
			kanade:  "(1 + 空っぽ)を出力しよう\n",
			wantErr: "null value",
		},
		{
			name:    "null > number",
			hari:    "(비어있음 > 1)을 출력하자\n",
			kanade:  "(空っぽ > 1)を出力しよう\n",
			wantErr: "null value",
		},
		{
			name:    "an unset variable in arithmetic",
			hari:    "'a'를 [숫자]인 비어있음으로 정하자\n('a' * 2)를 출력하자\n",
			kanade:  "『a』を【数字】の空っぽにしよう\n(『a』 * 2)を出力しよう\n",
			wantErr: "null value",
		},
		{
			name:       "null == null is fine",
			hari:       "(비어있음 == 비어있음)을 출력하자\n",
			kanade:     "(空っぽ == 空っぽ)を出力しよう\n",
			want:       "참",
			wantKanade: "真",
		},
		{
			name:       "null != number is fine",
			hari:       "(비어있음 != 1)을 출력하자\n",
			kanade:     "(空っぽ != 1)を出力しよう\n",
			want:       "참",
			wantKanade: "真",
		},
	})
}

// An operator the engine does not know (a typo like `'켜짐' 이다 참`) is an
// error, not a silent 비어있음/false. The bytecode compiler already refuses it
// at compile time, so this checks the tree-walker.
func TestUnknownOperatorIsAnError(t *testing.T) {
	_, err := runHari(t, "'켜짐'을 참으로 정하자\n만약 ('켜짐' 이다 참) 라면:\n    \"참\"을 출력하자\n그렇지 않다면:\n    \"거짓\"을 출력하자\n")
	if err == nil || !strings.Contains(err.Error(), "not supported here") {
		t.Errorf("tree-walker: want an unsupported-operator error, got %v", err)
	}
}
