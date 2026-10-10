package tests

// Runtime spec 5.1: a dictionary has <포함확인>(키), and 마다 반복하자 walks its
// keys — numbers from small to large, then strings in codepoint order, then
// 거짓 and 참 (value.SortedKeys) — as they were when the loop began.

import (
	"reflect"
	"testing"

	"github.com/soumt-r/hana/value"
)

func TestDictionaryContainsAndKeys(t *testing.T) {
	c := dualRun{
		hari: `'d'를 [(문자열, 숫자)사전]인 {"b": 1, "a": 2, "가": 3}로 정하자
('d'의 <포함확인>("a"))을 출력하자
('d'의 <포함확인>("z"))을 출력하자
('d'의 <포함확인>({}))을 출력하자
'd'의 '키'마다 반복하자:
    틀"{'키'}={'d'의 '키'}"를 출력하자
'섞임'을 {10: "열", 9: "아홉", "x": 1, 참: 2, 거짓: 3}로 정하자
'섞임'의 '키'마다 반복하자:
    '키'를 출력하자
'빈'을 {}로 정하자
'빈'의 '키'마다 반복하자:
    "안 나와요"를 출력하자
'단어들'을 ["a", "b", "a"]로 정하자
'세기'를 {}로 정하자
'단어들'의 '단어'마다 반복하자:
    만약 ('세기'의 <포함확인>('단어')) 라면:
        '세기'의 '단어'에 1을 더하자
    그렇지 않다면:
        '세기'의 '단어'를 1로 정하자
'세기'를 출력하자
'd'의 '키'마다 반복하자:
    'd'의 "새"를 0으로 정하자
('d'의 <포함확인>("새"))을 출력하자
`,
		kanade: `『d』を【(文字列, 数字)辞書】の{「b」: 1, 「a」: 2, 「あ」: 3}にしよう
(『d』の〈含むか確認〉(「a」))を出力しよう
(『d』の〈含むか確認〉(「z」))を出力しよう
『d』の『キー』ごとに繰り返そう:
    枠「{『キー』}={『d』の『キー』}」を出力しよう
『混ざり』を{10: 「十」, 9: 「九」, 「x」: 1, 真: 2, 偽: 3}にしよう
『混ざり』の『キー』ごとに繰り返そう:
    『キー』を出力しよう
`,
	}
	want := [4][]string{
		{"참", "거짓", "거짓", "a=2", "b=1", "가=3", "9", "10", "x", "거짓", "참", "{a: 2, b: 1}", "참"},
		{"참", "거짓", "거짓", "a=2", "b=1", "가=3", "9", "10", "x", "거짓", "참", "{a: 2, b: 1}", "참"},
		{"真", "偽", "a=2", "b=1", "あ=3", "9", "10", "x", "偽", "真"},
		{"真", "偽", "a=2", "b=1", "あ=3", "9", "10", "x", "偽", "真"},
	}
	outs, errs := runAllFour(t, c)
	for i, name := range []string{"hari tree", "hari bytecode", "kanade tree", "kanade bytecode"} {
		if errs[i] != nil || !reflect.DeepEqual(outs[i], want[i]) {
			t.Errorf("%s: got %v (%v), want %v", name, outs[i], errs[i], want[i])
		}
	}
}

func TestDictionaryMethodErrors(t *testing.T) {
	for _, c := range []struct{ code, want string }{
		{"'d'를 {}로 정하자\n('d'의 <없음>())을 출력하자\n", "MethodNotFound"},
		{"'d'를 {}로 정하자\n('d'의 <포함확인>())을 출력하자\n", "ArgumentError"},
	} {
		for engine, r := range engines(t, c.code) {
			if r.err == nil {
				t.Errorf("%s %q: no error", engine, c.code)
				continue
			}
			requireErrorContains(t, r.err, c.want)
		}
	}
}

func TestSortedKeys(t *testing.T) {
	m := map[interface{}]interface{}{"b": 1, 2.5: 1, true: 1, "a": 1, -1.0: 1, false: 1, "😀": 1, "ｱ": 1}
	want := []interface{}{-1.0, 2.5, "a", "b", "ｱ", "😀", false, true}
	if got := value.SortedKeys(m); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
