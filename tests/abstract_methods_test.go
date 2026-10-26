package tests

import "testing"

// A class that is not abstract must implement every abstract method
// (<m>가 있어야 한다) its chain declares — checked before the program runs, like a
// 규정's methods (Runtime spec 3.5), on tree-walker/bytecode x hari/kanade.
// Regression test for a bug where nothing checked it and the program ran until
// the missing method was called (MethodNotFoundError).

const abstractDrinkHari = `"시작"을 출력하자

[음료]를 밑설계하자:
    <만들기>를 만들자 ():
        '나'의 <우리기>()를 실행하자
        '나'의 <곁들이기>()를 실행하자

    <우리기>가 있어야 한다 ()
    <곁들이기>가 있어야 한다 ()

[음료]를 바탕으로 [코코아]를 설계하자:
    <우리기>를 만들자 ():
        "코코아 가루를 녹여요."를 출력하자
`

const abstractDrinkKanade = `「始め」を出力しよう

【飲み物】を下設計しよう:
    〈作る〉を作ろう():
        私の〈淹れる〉()を実行しよう
        私の〈添える〉()を実行しよう

    〈淹れる〉がなければならない()
    〈添える〉がなければならない()

【飲み物】をもとにして【ココア】を設計しよう:
    〈淹れる〉を作ろう():
        「ココアを溶かす」を出力しよう
`

func TestUnimplementedAbstractMethod(t *testing.T) {
	runTypeCases(t, []typeCase{
		{
			name:    "a missing method is an error even when nothing calls it",
			hari:    abstractDrinkHari,
			kanade:  abstractDrinkKanade,
			wantErr: "must implement method '",
		},
		{
			name:    "and when the object is made and used",
			hari:    abstractDrinkHari + "새로운 [코코아]()의 <만들기>()를 실행하자\n",
			kanade:  abstractDrinkKanade + "新しい【ココア】()の〈作る〉()を実行しよう\n",
			wantErr: "of abstract class '",
		},
		{
			name: "an abstract child may leave it to its own child",
			hari: `[음료]를 밑설계하자:
    <만들기>를 만들자 ():
        '나'의 <우리기>()를 실행하자
        '나'의 <곁들이기>()를 실행하자

    <우리기>가 있어야 한다 ()
    <곁들이기>가 있어야 한다 ()

[음료]를 바탕으로 [차]를 밑설계하자:
    <우리기>를 만들자 ():
        "찻잎을 우려요."를 출력하자

[차]를 바탕으로 [레몬차]를 설계하자:
    <곁들이기>를 만들자 ():
        "레몬을 띄워요."를 출력하자

새로운 [레몬차]()의 <만들기>()를 실행하자
`,
			kanade: `【飲み物】を下設計しよう:
    〈作る〉を作ろう():
        私の〈淹れる〉()を実行しよう
        私の〈添える〉()を実行しよう

    〈淹れる〉がなければならない()
    〈添える〉がなければならない()

【飲み物】をもとにして【茶】を下設計しよう:
    〈淹れる〉を作ろう():
        「茶葉を淹れる」を出力しよう

【茶】をもとにして【レモン茶】を設計しよう:
    〈添える〉を作ろう():
        「レモンを浮かべる」を出力しよう

新しい【レモン茶】()の〈作る〉()を実行しよう
`,
			want:       "찻잎을 우려요.|레몬을 띄워요.",
			wantKanade: "茶葉を淹れる|レモンを浮かべる",
		},
		{
			name: "a middle class's method counts, an abstract one higher up is still checked",
			hari: `[음료]를 밑설계하자:
    <우리기>가 있어야 한다 ()
    <곁들이기>가 있어야 한다 ()

[음료]를 바탕으로 [차]를 밑설계하자:
    <우리기>를 만들자 ():
        "찻잎을 우려요."를 출력하자

[차]를 바탕으로 [레몬차]를 설계하자:
    <곁들이기>를 만들자 ():
        "레몬을 띄워요."를 출력하자

'잔'을 새로운 [레몬차]()로 정하자
'잔'의 <우리기>()를 실행하자
'잔'의 <곁들이기>()를 실행하자
`,
			kanade: `【飲み物】を下設計しよう:
    〈淹れる〉がなければならない()
    〈添える〉がなければならない()

【飲み物】をもとにして【茶】を下設計しよう:
    〈淹れる〉を作ろう():
        「茶葉を淹れる」を出力しよう

【茶】をもとにして【レモン茶】を設計しよう:
    〈添える〉を作ろう():
        「レモンを浮かべる」を出力しよう

『杯』を新しい【レモン茶】()にしよう
『杯』の〈淹れる〉()を実行しよう
『杯』の〈添える〉()を実行しよう
`,
			want:       "찻잎을 우려요.|레몬을 띄워요.",
			wantKanade: "茶葉を淹れる|レモンを浮かべる",
		},
		{
			name: "two abstract levels, the grandchild misses the top one",
			hari: `[음료]를 밑설계하자:
    <곁들이기>가 있어야 한다 ()

[음료]를 바탕으로 [차]를 밑설계하자:
    <우리기>가 있어야 한다 ()

[차]를 바탕으로 [레몬차]를 설계하자:
    <우리기>를 만들자 ():
        "찻잎을 우려요."를 출력하자
`,
			kanade: `【飲み物】を下設計しよう:
    〈添える〉がなければならない()

【飲み物】をもとにして【茶】を下設計しよう:
    〈淹れる〉がなければならない()

【茶】をもとにして【レモン茶】を設計しよう:
    〈淹れる〉を作ろう():
        「茶葉を淹れる」を出力しよう
`,
			wantErr: "of abstract class '",
		},
	})

	// Checked before anything runs: "시작" is never printed.
	outs, errs := runAllFour(t, dualRun{hari: abstractDrinkHari, kanade: abstractDrinkKanade})
	for i, label := range []string{"hari tree", "hari bytecode", "kanade tree", "kanade bytecode"} {
		if errs[i] == nil || len(outs[i]) != 0 {
			t.Errorf("%s: want the error before any output, got %v (output %v)", label, errs[i], outs[i])
		}
	}
}
