package tests

import (
	"reflect"
	"strings"
	"testing"
)

// Running every docs code block on both the tree-walker and the bytecode VM
// (hana run vs hana run --bc) found the differences below. Each case is the smallest program that
// reproduced one, run on all four engine/language combinations, and all four
// must print the expected lines.
func TestBytecodeMatchesTreeWalkerOnFormerlyDivergentFeatures(t *testing.T) {
	cases := []struct {
		name   string
		c      dualRun
		wantKo []string
		wantJa []string
	}{
		{
			name: "quoted 나/私 with no self bound is an ordinary variable",
			c: dualRun{
				hari:   "'가'를 [숫자]인 10으로 정하자\n'나'를 [숫자]인 20으로 정하자\n'가' + '나'를 출력하자\n",
				kanade: "『私』を【数字】の20にしよう\n『私』を出力しよう\n",
			},
			wantKo: []string{"30"}, wantJa: []string{"20"},
		},
		{
			name: "class name as a value reaches its static fields",
			c: dualRun{
				hari: `[자동차]를 설계하자:
    '우리'의 '총생산량'을 [숫자]인 7로 정하자

틀"수: {'자동차'의 '총생산량'}"을 출력하자
`,
				kanade: `【自動車】を設計しよう:
    『私たち』の『総生産量』を【数字】の7にしよう

枠「数:{『自動車』の『総生産量』}」を出力しよう
`,
			},
			wantKo: []string{"수: 7"}, wantJa: []string{"数:7"},
		},
		{
			name: "getter runs on read",
			c: dualRun{
				hari: `[지갑]을 설계하자:
    '_돈'을 [숫자]인 5로 정하여 숨기자

    '돈'을 [숫자]로 정하자:
        가져올 때:
            '나'의 '_돈'을 돌려주자
        정할 때 ('금액'):
            '나'의 '_돈'을 '금액'으로 정하자

'내지갑'을 [지갑]인 새로운 [지갑]()으로 정하자
'내지갑'의 '돈'을 1000으로 정하자
'내지갑'의 '돈'을 출력하자
`,
				kanade: `【財布】を設計しよう:
    『_お金』を【数字】の5に隠そう

    『お金』を【数字】にしよう:
        取得する時:
            『私』の『_お金』を返そう
        決める時(『金額』):
            『私』の『_お金』を『金額』にしよう

『私の財布』を【財布】の新しい【財布】()にしよう
『私の財布』の『お金』を1000にしよう
『私の財布』の『お金』を出力しよう
`,
			},
			wantKo: []string{"1000"}, wantJa: []string{"1000"},
		},
		{
			name: "static field assigned from inside a static method",
			c: dualRun{
				hari: `[싱글]을 설계하자:
    '우리'의 '_값'을 비어있음으로 정하여 숨기자

    [문자열]을 돌려주는 '우리'의 <가져오기>를 만들자 ():
        만약 ('우리'의 '_값'이 비어있음과 같다) 라면:
            '우리'의 '_값'을 "처음"으로 정하자
        '우리'의 '_값'을 돌려주자

[싱글]의 <가져오기>()를 출력하자
[싱글]의 <가져오기>()를 출력하자
`,
				kanade: `【シングル】を設計しよう:
    『私たち』の『_値』を空っぽに隠そう

    【文字列】を返す『私たち』の〈取得する〉を作ろう():
        もし(『私たち』の『_値』が空っぽと同じだ)ならば:
            『私たち』の『_値』を「最初」にしよう
        『私たち』の『_値』を返そう

【シングル】の〈取得する〉()を出力しよう
【シングル】の〈取得する〉()を出力しよう
`,
			},
			wantKo: []string{"처음", "처음"}, wantJa: []string{"最初", "最初"},
		},
		{
			name: "== calls the class's equality magic method",
			c: dualRun{
				hari: `[좌표]를 설계하자:
    'x'를 [숫자]인 0으로 정하자

    처음 만들어질 때 ([숫자]인 '초기x') 다음과 같이 하자:
        '나'의 'x'를 '초기x'로 정하자

    [논리]를 돌려주는 <기호 같다>를 만들자 ([좌표]인 '대상'):
        만약 ('나'의 'x'와 '대상'의 'x'가 다르다) 라면:
            거짓을 돌려주자
        참을 돌려주자

'a'를 [좌표]인 새로운 [좌표](1)로 정하자
'b'를 [좌표]인 새로운 [좌표](1)로 정하자
'c'를 [좌표]인 새로운 [좌표](2)로 정하자
만약 ('a'와 'b'가 같다) 라면:
    "a==b"를 출력하자
만약 ('a'와 'c'가 같다) 라면:
    "a==c"를 출력하자
그렇지 않다면:
    "a!=c"를 출력하자
`,
				kanade: `【座標】を設計しよう:
    『x』を【数字】の0にしよう

    最初に作られる時(【数字】の『初期x』)次のようにしよう:
        『私』の『x』を『初期x』にしよう

    【論理】を返す〈記号 同じだ〉を作ろう(【座標】の『対象』):
        もし(『私』の『x』と『対象』の『x』が違う)なら:
            偽を返そう
        真を返そう

『a』を【座標】の新しい【座標】(1)にしよう
『b』を【座標】の新しい【座標】(1)にしよう
『c』を【座標】の新しい【座標】(2)にしよう
もし(『a』と『b』が同じだ)なら:
    「a==b」を出力しよう
もし(『a』と『c』が同じだ)なら:
    「a==c」を出力しよう
それ以外ならば:
    「a!=c」を出力しよう
`,
			},
			wantKo: []string{"a==b", "a!=c"}, wantJa: []string{"a==b", "a!=c"},
		},
		{
			name: "dynamic reflection <'변수'>() calls the method the variable names",
			c: dualRun{
				hari: `[주문]을 설계하자:
    <취소하기>를 만들자 ():
        "취소됨"을 출력하자

'주문서'를 [주문]인 새로운 [주문]()으로 정하자
'행동명'을 [문자열]인 "취소하기"로 정하자
'주문서'의 <'행동명'>()을 실행하자
`,
				kanade: `【注文】を設計しよう:
    〈キャンセルする〉を作ろう():
        「キャンセル」を出力しよう

『注文書』を【注文】の新しい【注文】()にしよう
『行動名』を【文字列】の「キャンセルする」にしよう
『注文書』の〈『行動名』〉()を実行しよう
`,
			},
			wantKo: []string{"취소됨"}, wantJa: []string{"キャンセル"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			outs, errs := runAllFour(t, tc.c)
			for i, e := range errs {
				if e != nil {
					t.Fatalf("engine %d: %v", i, e)
				}
			}
			for i, want := range [][]string{tc.wantKo, tc.wantKo, tc.wantJa, tc.wantJa} {
				if !reflect.DeepEqual(outs[i], want) {
					t.Errorf("engine %d printed %q, want %q", i, outs[i], want)
				}
			}
		})
	}
}

// A loop over a list with no explicit variable name exposes each element under
// the language's default item name — 아이템 / アイテム — in both engines. The
// bytecode compiler used to hardcode the Korean name, so the Kanade loop below
// failed there with a missing variable.
func TestDefaultLoopItemNameIsTheLanguages(t *testing.T) {
	c := dualRun{
		name:   "default loop item",
		hari:   "'과일들'을 [\"사과\", \"배\"]로 정하자\n'과일들'마다 반복하자:\n    '아이템'을 출력하자\n",
		kanade: "『果物たち』を【「りんご」、「なし」】にしよう\n『果物たち』ごとに繰り返そう:\n    『アイテム』を出力しよう\n",
	}
	outs, errs := runAllFour(t, c)
	for i, label := range []string{"hari tree", "hari bytecode", "kanade tree", "kanade bytecode"} {
		if errs[i] != nil {
			t.Errorf("%s: %v", label, errs[i])
		}
	}
	if strings.Join(outs[0], "|") != "사과|배" || strings.Join(outs[1], "|") != "사과|배" {
		t.Errorf("hari: tree %v, bytecode %v", outs[0], outs[1])
	}
	if strings.Join(outs[2], "|") != "りんご|なし" || strings.Join(outs[3], "|") != "りんご|なし" {
		t.Errorf("kanade: tree %v, bytecode %v", outs[2], outs[3])
	}
}
