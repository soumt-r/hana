package tests

import "testing"

// A constructor is inside its class, so it may use the object's hidden (숨기자)
// and protected (물려주자) members like any method (Runtime spec 3.5), on
// tree-walker/bytecode x hari/kanade. Regression test for a tree-walker bug
// where the constructor's scope had no bound 'this', so the access check saw
// every hidden call from a constructor as coming from outside.
func TestConstructorUsesHiddenMembers(t *testing.T) {
	runTypeCases(t, []typeCase{
		{
			name: "own hidden method and field, a parent's constructor, a protected member",
			hari: `
[조상]을 설계하자:
    '씨'를 1로 정하여 물려주자
    <물림>을 만들어 물려주자 ():
        "물림"을 돌려주자
    처음 만들어질 때 () 다음과 같이 하자:
        '나'의 '씨'를 '나'의 <비밀>()로 정하자
    <비밀>을 만들어 숨기자 ():
        10을 돌려주자

[조상]을 바탕으로 [가]를 설계하자:
    '값'을 0으로 정하여 숨기자

    처음 만들어질 때 ('수') 다음과 같이 하자:
        부모의 <처음 만들어질 때>()를 실행하자
        '나'의 '값'을 '나'의 <두배>('수')로 정하자
        틀"{'나'의 '값'} {'나'의 '씨'} {'나'의 <물림>()}"을 출력하자

    <보기>를 만들자 ():
        '나'의 <두배>('나'의 '값')를 돌려주자

    <두배>를 만들어 숨기자 ('수'):
        ('수' * 2)를 돌려주자

'하나'를 새로운 [가](3)으로 정하자
'하나'의 <보기>()를 출력하자
`,
			kanade: `
【祖先】を設計しよう:
    『種』を1に譲ろう
    〈継承〉を作って譲ろう():
        「継承」を返そう
    最初に作られる時 () 次のようにしよう:
        私の『種』を私の〈秘密〉()にしよう
    〈秘密〉を作って隠そう():
        10を返そう

【祖先】をもとにして【甲】を設計しよう:
    『値』を0に隠そう

    最初に作られる時 (『数』) 次のようにしよう:
        親の〈最初に作られる時〉()を実行しよう
        私の『値』を私の〈二倍〉(『数』)にしよう
        枠「{私の『値』} {私の『種』} {私の〈継承〉()}」を出力しよう

    〈見る〉を作ろう():
        私の〈二倍〉(私の『値』)を返そう

    〈二倍〉を作って隠そう(『数』):
        (『数』 * 2)を返そう

『一つ』を新しい【甲】(3)にしよう
『一つ』の〈見る〉()を出力しよう
`,
			want:       "6 10 물림|12",
			wantKanade: "6 10 継承|12",
		},
		{
			name:    "another object's hidden method is still refused in a constructor",
			hari:    "[가]를 설계하자:\n    처음 만들어질 때 ('남') 다음과 같이 하자:\n        만약 ('남'이 비어있음과 다르다)라면:\n            '남'의 <비밀>()을 출력하자\n    <비밀>을 만들어 숨기자 ():\n        7을 돌려주자\n'a'를 새로운 [가](비어있음)으로 정하자\n'b'를 새로운 [가]('a')로 정하자\n",
			kanade:  "【甲】を設計しよう:\n    最初に作られる時 (『他』) 次のようにしよう:\n        もし (『他』が空っぽと違う)ならば:\n            『他』の〈秘密〉()を出力しよう\n    〈秘密〉を作って隠そう():\n        7を返そう\n『a』を新しい【甲】(空っぽ)にしよう\n『b』を新しい【甲】(『a』)にしよう\n",
			wantErr: "private",
		},
	})
}

// Code outside a class may not assign to its hidden (숨기자) or protected
// (물려주자) fields, by any form of write, while the class's own constructor
// and methods may (Runtime spec 3.5). Regression test for a bug where only
// reads were checked, so an outside assignment silently changed the value.
func TestHiddenFieldWrites(t *testing.T) {
	runTypeCases(t, []typeCase{
		{
			name:   "a constructor reads a hidden field and appends to a hidden list",
			hari:   "[가]를 설계하자:\n    '값'을 5로 정하여 숨기자\n    '보임'을 0으로 정하자\n    '목록'을 []로 정하여 숨기자\n\n    처음 만들어질 때 ('처음') 다음과 같이 하자:\n        '나'의 '보임'을 ('나'의 '값' + 1)로 정하자\n        '처음'의 '하나'마다 반복하자:\n            '나'의 '목록'에 '하나'를 추가하자\n\n    <목록>을 만들자 ():\n        '나'의 '목록'을 돌려주자\n\n'하나'를 새로운 [가]([1, 2])로 정하자\n틀\"{'하나'의 '보임'} {'하나'의 <목록>()}\"을 출력하자\n",
			kanade: "【甲】を設計しよう:\n    『値』を5に隠そう\n    『見える』を0にしよう\n    『リスト』を【】に隠そう\n\n    最初に作られる時 (『最初』) 次のようにしよう:\n        私の『見える』を(私の『値』 + 1)にしよう\n        『最初』の『一つ』ごとに繰り返そう:\n            私の『リスト』に『一つ』を追加しよう\n\n    〈リスト〉を作ろう():\n        私の『リスト』を返そう\n\n『一つ』を新しい【甲】(【1, 2】)にしよう\n枠「{『一つ』の『見える』} {『一つ』の〈リスト〉()}」を出力しよう\n",
			want:   "6 [1, 2]",
		},
		{
			name:       "outside writes are refused, the value stays",
			hari:       "[저장점]을 설계하자:\n    '체력'을 [숫자]인 0으로 정하여 숨기자\n    '목록'을 []로 정하여 숨기자\n    '가보'를 1로 정하여 물려주자\n    처음 만들어질 때 ([숫자]인 '체력', '처음') 다음과 같이 하자:\n        '나'의 '체력'을 '체력'으로 정하자\n        '처음'의 '하나'마다 반복하자:\n            '나'의 '목록' 뒤에 '하나'를 추가하자\n        '나'의 '체력'에 ('나'의 '체력' + 1)을 더하자\n    [숫자]를 돌려주는 <체력>을 만들자 ():\n        '나'의 '체력'을 돌려주자\n    <목록>을 만들자 ():\n        '나'의 '목록'을 돌려주자\n\n'점'을 새로운 [저장점](100, [1, 2])으로 정하자\n틀\"{'점'의 <체력>()} {'점'의 <목록>()}\"을 출력하자\n일단 해보자:\n    '점'의 '체력'을 9999로 정하자\n오류가 발생했다면 ('e'):\n    ('e'의 '메시지')를 출력하자\n일단 해보자:\n    '점'의 '체력'에 1을 더하자\n오류가 발생했다면 ('e'):\n    ('e'의 '메시지')를 출력하자\n일단 해보자:\n    '점'의 '가보'를 2로 정하자\n오류가 발생했다면 ('e'):\n    ('e'의 '메시지')를 출력하자\n일단 해보자:\n    '점'의 '목록' 뒤에 3을 추가하자\n오류가 발생했다면 ('e'):\n    ('e'의 '메시지')를 출력하자\n틀\"{'점'의 <체력>()} {'점'의 <목록>()}\"을 출력하자\n",
			kanade:     "【保存点】を設計しよう:\n    『体力』を【数字】の0に隠そう\n    『リスト』を【】に隠そう\n    『家宝』を1に譲ろう\n    最初に作られる時 (【数字】の『体力』, 『最初』) 次のようにしよう:\n        私の『体力』を『体力』にしよう\n        『最初』の『一つ』ごとに繰り返そう:\n            私の『リスト』に『一つ』を追加しよう\n        私の『体力』に(私の『体力』 + 1)を足そう\n    【数字】を返す〈体力〉を作ろう():\n        私の『体力』を返そう\n    〈リスト〉を作ろう():\n        私の『リスト』を返そう\n\n『点』を新しい【保存点】(100, 【1, 2】)にしよう\n枠「{『点』の〈体力〉()} {『点』の〈リスト〉()}」を出力しよう\nとりあえずやってみよう:\n    『点』の『体力』を9999にしよう\n発生したら(『e』):\n    (『e』の『メッセージ』)を出力しよう\nとりあえずやってみよう:\n    『点』の『体力』に1を足そう\n発生したら(『e』):\n    (『e』の『メッセージ』)を出力しよう\nとりあえずやってみよう:\n    『点』の『家宝』を2にしよう\n発生したら(『e』):\n    (『e』の『メッセージ』)を出力しよう\nとりあえずやってみよう:\n    『点』の『リスト』に3を追加しよう\n発生したら(『e』):\n    (『e』の『メッセージ』)を出力しよう\n枠「{『点』の〈体力〉()} {『点』の〈リスト〉()}」を出力しよう\n",
			want:       "201 [1, 2]|AccessViolationError: '체력' 속성은 내부 전용(private)이라 외부에서 부를 수 없어요.|AccessViolationError: '체력' 속성은 내부 전용(private)이라 외부에서 부를 수 없어요.|AccessViolationError: '가보' 속성은 상속된 클래스 전용(protected)이라 외부에서 부를 수 없어요.|AccessViolationError: '목록' 속성은 내부 전용(private)이라 외부에서 부를 수 없어요.|201 [1, 2]",
			wantKanade: "201 [1, 2]|AccessViolationError: '体力' 属性は内部専用(private)なので外部から呼ぶことはできません。|AccessViolationError: '体力' 属性は内部専用(private)なので外部から呼ぶことはできません。|AccessViolationError: '家宝' 属性は継承されたクラス専用(protected)なので外部から呼ぶことはできません。|AccessViolationError: 'リスト' 属性は内部専用(private)なので外部から呼ぶことはできません。|201 [1, 2]",
		},
		{
			name:    "an outside assignment is an error when uncaught",
			hari:    "[가]를 설계하자:\n    '체력'을 0으로 정하여 숨기자\n'x'를 새로운 [가]()로 정하자\n'x'의 '체력'을 9999로 정하자\n",
			kanade:  "【甲】を設計しよう:\n    『体力』を0に隠そう\n『x』を新しい【甲】()にしよう\n『x』の『体力』を9999にしよう\n",
			wantErr: "private",
		},
	})
}
