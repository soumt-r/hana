package tests

import (
	"reflect"
	"testing"
)

// Comparing dictionaries with == used to crash hana (Go cannot compare maps).
// A dictionary is equal only to itself, like a list or an object, on both
// engines and in 따라 나누자.
func TestDictionariesCompareByIdentity(t *testing.T) {
	c := dualRun{
		name: "dict ==",
		hari: `'가'를 {"a": 1}로 정하자
'나'를 {"a": 1}로 정하자
'다'를 '가'로 정하자
('가' == '나')를 출력하자
('가' == '다')를 출력하자
('가' != '나')를 출력하자
('가' == 1)을 출력하자
('가' == 비어있음)을 출력하자
'가'에 따라 나누자:
    '나' 인 경우:
        "나"를 출력하자
    '다' 인 경우:
        "다"를 출력하자
`,
		kanade: `『甲』を{「a」: 1}にしよう
『乙』を{「a」: 1}にしよう
『丙』を『甲』にしよう
(『甲』 == 『乙』)を出力しよう
(『甲』 == 『丙』)を出力しよう
(『甲』 != 『乙』)を出力しよう
(『甲』 == 1)を出力しよう
(『甲』 == 空っぽ)を出力しよう
`,
	}
	outs, errs := runAllFour(t, c)
	wantKo := []string{"거짓", "참", "참", "거짓", "거짓", "다"}
	for i, name := range []string{"hari tree", "hari bytecode", "kanade tree", "kanade bytecode"} {
		if errs[i] != nil {
			t.Fatalf("%s: %v", name, errs[i])
		}
		if i < 2 && !reflect.DeepEqual(outs[i], wantKo) {
			t.Errorf("%s: got %q, want %q", name, outs[i], wantKo)
		}
	}
	if !reflect.DeepEqual(outs[2], outs[3]) || len(outs[2]) != 5 {
		t.Errorf("kanade engines differ or ran short: %q vs %q", outs[2], outs[3])
	}
}
