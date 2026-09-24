package tests

// Declared types are enforced at run time (Runtime spec 2.2): a value must
// match the [타입] it is stored under, the type outlives the declaration,
// generics are checked element by element, 비어있음 fits every type, an omitted
// or [아무거나] type accepts anything, and children satisfy their parents' and
// interfaces' types. Every case runs on tree-walker/bytecode x hari/kanade.

import (
	"strings"
	"testing"
)

type typeCase struct {
	name   string
	hari   string
	kanade string
	want   string // expected joined output, or "" when wantErr is set
	// wantKanade overrides want for the Kanade engines when the output text
	// itself is language-specific (空っぽ vs 비어있음).
	wantKanade string
	// wantErr is a fragment of the English error text every engine must raise.
	wantErr string
	// wantContains, when set, is a fragment the (joined) output must contain
	// instead of equaling want.
	wantContains string
}

func runTypeCases(t *testing.T, cases []typeCase) {
	t.Helper()
	labels := []string{"hari tree", "hari bytecode", "kanade tree", "kanade bytecode"}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			outs, errs := runAllFour(t, dualRun{name: c.name, hari: c.hari, kanade: c.kanade})
			for i, label := range labels {
				if c.wantErr != "" {
					if errs[i] == nil || !strings.Contains(errs[i].Error(), c.wantErr) {
						t.Errorf("%s: want an error containing %q, got %v (output %v)", label, c.wantErr, errs[i], outs[i])
					}
					continue
				}
				if errs[i] != nil {
					t.Errorf("%s: unexpected error %v", label, errs[i])
					continue
				}
				got := strings.Join(outs[i], "|")
				if c.wantContains != "" {
					if !strings.Contains(got, c.wantContains) {
						t.Errorf("%s: output %q should contain %q", label, got, c.wantContains)
					}
					continue
				}
				want := c.want
				if c.wantKanade != "" && i >= 2 {
					want = c.wantKanade
				}
				if got != want {
					t.Errorf("%s: output %q, want %q", label, got, want)
				}
			}
		})
	}
}

func TestDeclaredTypesAreEnforcedOnDeclaration(t *testing.T) {
	runTypeCases(t, []typeCase{
		{
			name:    "a string is not a number",
			hari:    "'a'를 [숫자]인 \"문자\"로 정하자\n",
			kanade:  "『a』を【数字】の「文字」にしよう\n",
			wantErr: "can only hold",
		},
		{
			name:   "a matching value is fine",
			hari:   "'a'를 [숫자]인 3으로 정하자\n'a'를 출력하자\n",
			kanade: "『a』を【数字】の3にしよう\n『a』を出力しよう\n",
			want:   "3",
		},
		{
			name:    "a number is not a boolean",
			hari:    "'a'를 [논리]인 1로 정하자\n",
			kanade:  "『a』を【論理】の1にしよう\n",
			wantErr: "can only hold",
		},
	})
}

func TestTheDeclaredTypeOutlivesTheDeclaration(t *testing.T) {
	runTypeCases(t, []typeCase{
		{
			name:    "reassigning another type fails",
			hari:    "'a'를 [숫자]인 3으로 정하자\n'a'를 \"x\"로 정하자\n",
			kanade:  "『a』を【数字】の3にしよう\n『a』を「x」にしよう\n",
			wantErr: "can only hold",
		},
		{
			name:   "reassigning the same type is fine",
			hari:   "'a'를 [숫자]인 3으로 정하자\n'a'를 4로 정하자\n'a'를 출력하자\n",
			kanade: "『a』を【数字】の3にしよう\n『a』を4にしよう\n『a』を出力しよう\n",
			want:   "4",
		},
		{
			name:   "an untyped variable may change type",
			hari:   "'a'를 3으로 정하자\n'a'를 \"x\"로 정하자\n'a'를 출력하자\n",
			kanade: "『a』を3にしよう\n『a』を「x」にしよう\n『a』を出力しよう\n",
			want:   "x",
		},
		{
			name:   "[아무거나] accepts anything",
			hari:   "'a'를 [아무거나]인 3으로 정하자\n'a'를 \"x\"로 정하자\n'a'를 출력하자\n",
			kanade: "『a』を【何でも】の3にしよう\n『a』を「x」にしよう\n『a』を出力しよう\n",
			want:   "x",
		},
		{
			name:       "비어있음 fits every type",
			hari:       "'a'를 [숫자]인 3으로 정하자\n'a'를 비어있음으로 정하자\n'a'를 출력하자\n",
			kanade:     "『a』を【数字】の3にしよう\n『a』を空っぽにしよう\n『a』を出力しよう\n",
			want:       "비어있음",
			wantKanade: "空っぽ",
		},
	})
}

func TestGenericsAreCheckedWhenElementsAreAdded(t *testing.T) {
	runTypeCases(t, []typeCase{
		{
			name:    "a wrong element in the initial list",
			hari:    "'목록'을 [(숫자)목록]인 [1, \"둘\"]로 정하자\n",
			kanade:  "『目録』を【(数字)リスト】の【1,「二」】にしよう\n",
			wantErr: "can only hold",
		},
		{
			name:    "pushing a wrong element",
			hari:    "'목록'을 [(숫자)목록]인 [1, 2]로 정하자\n'목록' 뒤에 \"셋\"을 추가하자\n",
			kanade:  "『目録』を【(数字)リスト】の【1,2】にしよう\n『目録』の後に「三」を追加しよう\n",
			wantErr: "can only hold",
		},
		{
			name:   "pushing a matching element",
			hari:   "'목록'을 [(숫자)목록]인 [1, 2]로 정하자\n'목록' 뒤에 3을 추가하자\n'목록'을 출력하자\n",
			kanade: "『目録』を【(数字)リスト】の【1,2】にしよう\n『目録』の後に3を追加しよう\n『目録』を出力しよう\n",
			want:   "[1, 2, 3]",
		},
		{
			name:    "dictionary values are checked",
			hari:    "'점수'를 [(문자열, 숫자)사전]인 {\"국어\": \"구십\"}으로 정하자\n",
			kanade:  "『点数』を【(文字列,数字)辞書】の{「国語」:「九十」}にしよう\n",
			wantErr: "can only hold",
		},
		{
			name:   "a well-typed dictionary",
			hari:   "'점수'를 [(문자열, 숫자)사전]인 {\"국어\": 90}으로 정하자\n'점수'의 \"국어\"를 출력하자\n",
			kanade: "『点数』を【(文字列,数字)辞書】の{「国語」:90}にしよう\n『点数』の「国語」を出力しよう\n",
			want:   "90",
		},
	})
}

func TestParameterTypesAreEnforced(t *testing.T) {
	runTypeCases(t, []typeCase{
		{
			name:    "the wrong argument type",
			hari:    "<두배>를 만들자 ([숫자]인 '값'):\n    ('값' * 2)를 출력하자\n<두배>(\"문자\")를 실행하자\n",
			kanade:  "〈二倍〉を作ろう(【数字】の『値』):\n    (『値』 * 2)を出力しよう\n〈二倍〉(「文字」)を実行しよう\n",
			wantErr: "must be",
		},
		{
			name:   "the right argument type",
			hari:   "<두배>를 만들자 ([숫자]인 '값'):\n    ('값' * 2)를 출력하자\n<두배>(4)를 실행하자\n",
			kanade: "〈二倍〉を作ろう(【数字】の『値』):\n    (『値』 * 2)を出力しよう\n〈二倍〉(4)を実行しよう\n",
			want:   "8",
		},
		{
			name:    "the parameter keeps its type inside the body",
			hari:    "<바꾸기>를 만들자 ([숫자]인 '값'):\n    '값'을 \"문자\"로 정하자\n<바꾸기>(1)를 실행하자\n",
			kanade:  "〈変える〉を作ろう(【数字】の『値』):\n    『値』を「文字」にしよう\n〈変える〉(1)を実行しよう\n",
			wantErr: "can only hold",
		},
		{
			name:       "an untyped parameter takes anything",
			hari:       "<보이기>를 만들자 ('값'):\n    '값'을 출력하자\n<보이기>(\"글\")를 실행하자\n",
			kanade:     "〈見せる〉を作ろう(『値』):\n    『値』を出力しよう\n〈見せる〉(「文」)を実行しよう\n",
			want:       "글",
			wantKanade: "文",
		},
	})
}

const (
	hariAnimals   = "[동물]을 설계하자:\n    '이름'을 [문자열]인 \"동물\"으로 정하자\n[동물]을 바탕으로 [강아지]를 설계하자:\n    '재주'를 [문자열]인 \"앉아\"로 정하자\n[고양이]를 설계하자:\n    '이름'을 [문자열]인 \"나비\"로 정하자\n"
	kanadeAnimals = "【動物】を設計しよう:\n    『名前』を【文字列】の「動物」にしよう\n【動物】をもとにして【子犬】を設計しよう:\n    『芸』を【文字列】の「おすわり」にしよう\n【猫】を設計しよう:\n    『名前』を【文字列】の「ミケ」にしよう\n"
)

func TestClassTypesAcceptChildrenAndRejectStrangers(t *testing.T) {
	runTypeCases(t, []typeCase{
		{
			name:       "upcasting: a child fits its parent's type",
			hari:       hariAnimals + "'친구'를 [동물]인 새로운 [강아지]()로 정하자\n'친구'의 '재주'를 출력하자\n",
			kanade:     kanadeAnimals + "『友達』を【動物】の新しい【子犬】()にしよう\n『友達』の『芸』を出力しよう\n",
			want:       "앉아",
			wantKanade: "おすわり",
		},
		{
			name:    "a stranger class is refused",
			hari:    hariAnimals + "'친구'를 [동물]인 새로운 [고양이]()로 정하자\n",
			kanade:  kanadeAnimals + "『友達』を【動物】の新しい【猫】()にしよう\n",
			wantErr: "can only hold",
		},
		{
			name:    "a parent is not a child",
			hari:    hariAnimals + "'친구'를 [강아지]인 새로운 [동물]()로 정하자\n",
			kanade:  kanadeAnimals + "『友達』を【子犬】の新しい【動物】()にしよう\n",
			wantErr: "can only hold",
		},
		{
			name:    "a class-typed parameter refuses a stranger",
			hari:    hariAnimals + "<부르기>를 만들자 ([동물]인 '누구'):\n    '누구'의 '이름'을 출력하자\n<부르기>(새로운 [고양이]())를 실행하자\n",
			kanade:  kanadeAnimals + "〈呼ぶ〉を作ろう(【動物】の『誰』):\n    『誰』の『名前』を出力しよう\n〈呼ぶ〉(新しい【猫】())を実行しよう\n",
			wantErr: "must be",
		},
	})
}

func TestFieldTypesAreEnforced(t *testing.T) {
	runTypeCases(t, []typeCase{
		{
			name:    "writing a wrong type into a typed field",
			hari:    "[사람]을 설계하자:\n    '나이'를 [숫자]인 20으로 정하자\n'홍길동'을 [사람]인 새로운 [사람]()로 정하자\n'홍길동'의 '나이'를 \"스물\"로 정하자\n",
			kanade:  "【人】を設計しよう:\n    『年齢』を【数字】の20にしよう\n『太郎』を【人】の新しい【人】()にしよう\n『太郎』の『年齢』を「二十」にしよう\n",
			wantErr: "can only hold",
		},
		{
			name:    "a default that does not match its own type",
			hari:    "[사람]을 설계하자:\n    '나이'를 [숫자]인 \"스물\"로 정하자\n새로운 [사람]()을 실행하자\n",
			kanade:  "【人】を設計しよう:\n    『年齢』を【数字】の「二十」にしよう\n新しい【人】()を実行しよう\n",
			wantErr: "can only hold",
		},
		{
			name:   "a matching write",
			hari:   "[사람]을 설계하자:\n    '나이'를 [숫자]인 20으로 정하자\n'홍길동'을 [사람]인 새로운 [사람]()로 정하자\n'홍길동'의 '나이'를 21로 정하자\n'홍길동'의 '나이'를 출력하자\n",
			kanade: "【人】を設計しよう:\n    『年齢』を【数字】の20にしよう\n『太郎』を【人】の新しい【人】()にしよう\n『太郎』の『年齢』を21にしよう\n『太郎』の『年齢』を出力しよう\n",
			want:   "21",
		},
	})
}

func TestTypeErrorsAreCatchable(t *testing.T) {
	runTypeCases(t, []typeCase{{
		name:         "일단 해보자 catches a type error and its message is readable",
		hari:         "일단 해보자:\n    'a'를 [숫자]인 \"x\"로 정하자\n오류가 발생했다면 ('에러'):\n    '에러'의 '메시지'를 출력하자\n",
		kanade:       "とりあえずやってみよう:\n    『a』を【数字】の「x」にしよう\n発生したら(『エラー』):\n    『エラー』の『メッセージ』を出力しよう\n",
		wantContains: "TypeError",
	}})
}
