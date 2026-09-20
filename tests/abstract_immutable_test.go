package tests

// Interfaces and abstract classes cannot be instantiated (Runtime spec 3.1.1),
// and a string's character cannot be reassigned (Runtime spec 5.4), on
// tree-walker/bytecode x haja/kanade.

import "testing"

func TestAbstractTypesCannotBeInstantiated(t *testing.T) {
	runTypeCases(t, []typeCase{
		{
			name:    "interface",
			haja:    "[날수있는것]을 규정하자:\n    <날기>가 있어야 한다 ()\n새로운 [날수있는것]()을 실행하자\n\"만들어짐\"을 출력하자\n",
			kanade:  "【飛べるもの】を規定しよう:\n    〈飛ぶ〉がなければならない()\n新しい【飛べるもの】()を実行しよう\n「作られた」を出力しよう\n",
			wantErr: "is an interface",
		},
		{
			name:    "abstract class",
			haja:    "[도형]을 밑설계하자:\n    <넓이>를 만들자 ():\n        1을 출력하자\n새로운 [도형]()을 실행하자\n\"만들어짐\"을 출력하자\n",
			kanade:  "【図形】を下設計しよう:\n    〈面積〉を作ろう():\n        1を出力しよう\n新しい【図形】()を実行しよう\n「作られた」を出力しよう\n",
			wantErr: "is an abstract class",
		},
		{
			name:    "abstract class assigned to a variable",
			haja:    "[도형]을 밑설계하자:\n    <넓이>가 있어야 한다 ()\n'모양'을 [도형]인 새로운 [도형]()으로 정하자\n",
			kanade:  "【図形】を下設計しよう:\n    〈面積〉がなければならない()\n『形』を【図形】の新しい【図形】()にしよう\n",
			wantErr: "is an abstract class",
		},
		{
			name:   "a concrete child of an abstract class is fine",
			haja:   "[도형]을 밑설계하자:\n    <넓이>가 있어야 한다 ()\n[도형]을 바탕으로 [네모]를 설계하자:\n    [숫자]를 돌려주는 <넓이>를 만들자 ():\n        6을 돌려주자\n'모양'을 [도형]인 새로운 [네모]()로 정하자\n('모양'의 <넓이>())를 출력하자\n",
			kanade: "【図形】を下設計しよう:\n    〈面積〉がなければならない()\n【図形】をもとにして【四角形】を設計しよう:\n    【数字】を返す〈面積〉を作ろう():\n        6を返そう\n『形』を【図形】の新しい【四角形】()にしよう\n(『形』の〈面積〉())を出力しよう\n",
			want:   "6",
		},
		{
			name:   "an ordinary class is still fine",
			haja:   "[사람]을 설계하자:\n    '이름'을 \"철수\"로 정하자\n'누구'를 [사람]인 새로운 [사람]()으로 정하자\n'누구'의 '이름'을 출력하자\n",
			kanade: "【人】を設計しよう:\n    『名前』を「太郎」にしよう\n『誰』を【人】の新しい【人】()にしよう\n『誰』の『名前』を出力しよう\n",
			want:   "철수", wantKanade: "太郎",
		},
	})
}

func TestStringCharactersAreImmutable(t *testing.T) {
	runTypeCases(t, []typeCase{
		{
			name:    "reassigning a character",
			haja:    "'이름'을 \"홍길동\"으로 정하자\n'이름'의 1번째를 \"김\"으로 정하자\n'이름'을 출력하자\n",
			kanade:  "『名前』を「山田」にしよう\n『名前』の1番目を「田」にしよう\n『名前』を出力しよう\n",
			wantErr: "immutable",
		},
		{
			name:   "reading a character is fine",
			haja:   "'이름'을 \"홍길동\"으로 정하자\n'이름'의 1번째를 출력하자\n",
			kanade: "『名前』を「山田」にしよう\n『名前』の1番目を出力しよう\n",
			want:   "홍", wantKanade: "山",
		},
		{
			name:   "list elements can still be reassigned",
			haja:   "'목록'을 [1, 2, 3]으로 정하자\n'목록'의 2번째를 9로 정하자\n'목록'의 2번째를 출력하자\n",
			kanade: "『目録』を【1,2,3】にしよう\n『目録』の2番目を9にしよう\n『目録』の2番目を出力しよう\n",
			want:   "9",
		},
	})
}
