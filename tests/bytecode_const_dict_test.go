package tests

import (
	"reflect"
	"testing"
)

// The bytecode VM used to ignore 고정하자 (a "constant" could be reassigned
// silently) and the compiler rejected dict literals outright. These tests run
// each program on the tree-walker and the bytecode VM, in both languages, and
// require the same outcome — the tree-walker is the reference.

type dualRun struct {
	name   string
	hari   string
	kanade string
}

func runAllFour(t *testing.T, c dualRun) (outs [4][]string, errs [4]error) {
	t.Helper()
	i0, e0 := runHari(t, c.hari)
	b0, e1 := runBytecode(t, c.hari)
	i1, e2 := runKanade(t, c.kanade)
	b1, e3 := runKanadeBytecode(t, c.kanade)
	outs[0], outs[1], outs[2], outs[3] = i0.Output, b0.Output, i1.Output, b1.Output
	errs[0], errs[1], errs[2], errs[3] = e0, e1, e2, e3
	return
}

func TestBytecodeConstantsCannotBeReassigned(t *testing.T) {
	cases := []dualRun{
		{
			name:   "plain reassignment",
			hari:   "'생일'을 [문자열]인 \"1월 1일\"로 고정하자\n'생일'을 [문자열]인 \"2월 2일\"로 정하자\n",
			kanade: "『誕生日』を【文字列】の「1月1日」で固定しよう\n『誕生日』を【文字列】の「2月2日」にしよう\n",
		},
		{
			name:   "list push",
			hari:   "'목록'을 [(숫자)목록]인 [1, 2]로 고정하자\n'목록' 뒤에 3을 추가하자\n",
			kanade: "『リスト』を【(数字)リスト】の【1,2】で固定しよう\n『リスト』の後ろに3を追加しよう\n",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, errs := runAllFour(t, c)
			for i, name := range []string{"hari tree", "hari bytecode", "kanade tree", "kanade bytecode"} {
				requireErrorContains(t, errs[i], "ConstantAssignmentError")
				_ = name
			}
		})
	}
}

func TestBytecodeConstantReadableAndCatchable(t *testing.T) {
	c := dualRun{
		hari: `'원주율'을 [숫자]인 3.14로 고정하자
일단 해보자:
    '원주율'을 3으로 정하자
오류가 발생했다면 ('에러'):
    '에러'의 '메시지'를 출력하자
틀"원주율은 {'원주율'}"을 출력하자
`,
		kanade: `『円周率』を【数字】の3.14で固定しよう
とりあえずやってみよう:
    『円周率』を3にしよう
発生したら(『エラー』):
    『エラー』の『メッセージ』を出力しよう
枠「円周率は{『円周率』}」を出力しよう
`,
	}
	outs, errs := runAllFour(t, c)
	for i, e := range errs {
		if e != nil {
			t.Fatalf("engine %d: %v", i, e)
		}
	}
	wantHari := []string{"ConstantAssignmentError: 상수 '원주율'의 값은 변경할 수 없어요.", "원주율은 3.14"}
	wantKanade := []string{"ConstantAssignmentError: 定数 '円周率' の値は変更できません。", "円周率は3.14"}
	for i, want := range [][]string{wantHari, wantHari, wantKanade, wantKanade} {
		if !reflect.DeepEqual(outs[i], want) {
			t.Errorf("engine %d printed %q, want %q", i, outs[i], want)
		}
	}
}

func TestBytecodeDictionaries(t *testing.T) {
	c := dualRun{
		hari: `'점수'를 [(문자열, 숫자)사전]인 {"국어": 90, "수학": 80}으로 정하자
틀"국어 점수: {'점수'의 "국어"}"를 출력하자
'점수'의 "수학"에 10을 더하자
틀"오른 수학 점수: {'점수'의 "수학"}"을 출력하자
'점수'의 "영어"를 95로 정하자
틀"영어: {'점수'의 "영어"}"를 출력하자
일단 해보자:
    '점수'의 "과학"을 출력하자
오류가 발생했다면 ('에러'):
    '에러'의 '메시지'를 출력하자
`,
		kanade: `『点数』を【(文字列,数字)辞書】の{「国語」:90,「数学」:80}にしよう
枠「国語の点数:{『点数』の「国語」}」を出力しよう
『点数』の「数学」に10を足そう
枠「上がった数学の点数:{『点数』の「数学」}」を出力しよう
『点数』の「英語」を95にしよう
枠「英語:{『点数』の「英語」}」を出力しよう
とりあえずやってみよう:
    『点数』の「科学」を出力しよう
発生したら(『エラー』):
    『エラー』の『メッセージ』を出力しよう
`,
	}
	outs, errs := runAllFour(t, c)
	for i, e := range errs {
		if e != nil {
			t.Fatalf("engine %d: %v", i, e)
		}
	}
	wantHari := []string{"국어 점수: 90", "오른 수학 점수: 90", "영어: 95", "KeyError: 사전에서 '과학' 이름을 찾을 수 없어요."}
	wantKanade := []string{"国語の点数:90", "上がった数学の点数:90", "英語:95", "KeyError: 辞書から '科学' という名前を見つけることができません。"}
	for i, want := range [][]string{wantHari, wantHari, wantKanade, wantKanade} {
		if !reflect.DeepEqual(outs[i], want) {
			t.Errorf("engine %d printed %q, want %q", i, outs[i], want)
		}
	}
}

// Dicts are reference values: a function that mutates its dict argument
// changes the caller's (tutorial/8-functions.md's 사전바꾸기 example).
func TestBytecodeDictionaryIsSharedByReference(t *testing.T) {
	c := dualRun{
		hari: `<사전바꾸기>를 만들자 ([(문자열, 아무거나)사전]인 '사전'):
    '사전'의 "상태"를 "바뀜"으로 정하자

'내사전'을 [(문자열, 아무거나)사전]인 {"상태": "안바뀜"}으로 정하자
<사전바꾸기>('내사전')을 실행하자
틀"상태: {'내사전'의 "상태"}"를 출력하자
`,
		kanade: `〈辞書を変える〉を作ろう(【(文字列,何でも)辞書】の『辞書』):
    『辞書』の「状態」を「変わった」にしよう

『私の辞書』を【(文字列,何でも)辞書】の{「状態」:「変わらない」}にしよう
〈辞書を変える〉(『私の辞書』)を実行しよう
枠「状態:{『私の辞書』の「状態」}」を出力しよう
`,
	}
	outs, errs := runAllFour(t, c)
	for i, e := range errs {
		if e != nil {
			t.Fatalf("engine %d: %v", i, e)
		}
	}
	for i, want := range [][]string{{"상태: 바뀜"}, {"상태: 바뀜"}, {"状態:変わった"}, {"状態:変わった"}} {
		if !reflect.DeepEqual(outs[i], want) {
			t.Errorf("engine %d printed %q, want %q", i, outs[i], want)
		}
	}
}
