package tests

// Runtime spec 5.4: strings compare with < > <= >= in codepoint order. The
// engines used to answer OperandTypeMismatch; a string and a number still do.

import (
	"reflect"
	"testing"
)

func TestStringsCompareInCodepointOrder(t *testing.T) {
	c := dualRun{
		hari: `("a" < "b")을 출력하자
("b" <= "a")을 출력하자
("가" > "나")을 출력하자
("abc" >= "abc")을 출력하자
("Z" < "a")을 출력하자
("😀" > "ｱ")을 출력하자
("ab" < "abc")을 출력하자
'x'를 "사과"로 정하자
만약 ('x'가 "바나나"보다 크다) 라면:
    "크다"를 출력하자
's'를 "a"로 정하자
('s' < "aaa") 동안 반복하자:
    's'에 "a"를 더하자
's'를 출력하자
`,
		kanade: `(「a」 < 「b」)を出力しよう
(「b」 <= 「a」)を出力しよう
(「あ」 > 「い」)を出力しよう
(「abc」 >= 「abc」)を出力しよう
(「Z」 < 「a」)を出力しよう
(「😀」 > 「ｱ」)を出力しよう
(「ab」 < 「abc」)を出力しよう
`,
	}
	want := [4][]string{
		{"참", "거짓", "거짓", "참", "참", "참", "참", "크다", "aaa"},
		{"참", "거짓", "거짓", "참", "참", "참", "참", "크다", "aaa"},
		{"真", "偽", "偽", "真", "真", "真", "真"},
		{"真", "偽", "偽", "真", "真", "真", "真"},
	}
	outs, errs := runAllFour(t, c)
	for i, name := range []string{"hari tree", "hari bytecode", "kanade tree", "kanade bytecode"} {
		if errs[i] != nil || !reflect.DeepEqual(outs[i], want[i]) {
			t.Errorf("%s: got %v (%v), want %v", name, outs[i], errs[i], want[i])
		}
	}
}

func TestStringAndNumberStillDoNotCompare(t *testing.T) {
	for engine, r := range engines(t, `("1" < 2)을 출력하자`+"\n") {
		if r.err == nil {
			t.Errorf("%s: no error", engine)
			continue
		}
		requireErrorContains(t, r.err, "cannot combine")
	}
}
