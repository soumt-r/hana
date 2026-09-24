package tests

// The canonical way to write a length is the quoted property: '글'의 '길이'
// (Kanade: 『文』の『長さ』), which reads as a plain phrase before a particle
// (를, 로 정하자) on every engine. Docs and examples use this form.

import "testing"

func TestQuotedLengthWorksBeforeParticles(t *testing.T) {
	runTypeCases(t, []typeCase{
		{
			name:   "string and list length, printed and stored",
			hari:   "'글'을 \"가나다\"로 정하자\n'글'의 '길이'를 출력하자\n'n'을 [숫자]인 '글'의 '길이'로 정하자\n'n'을 출력하자\n'목록'을 [1, 2]로 정하자\n'목록'의 '길이'를 출력하자\n",
			kanade: "『文』を「あいう」にしよう\n『文』の『長さ』を出力しよう\n『n』を【数字】の『文』の『長さ』にしよう\n『n』を出力しよう\n『目録』を【1, 2】にしよう\n『目録』の『長さ』を出力しよう\n",
			want:   "3|3|2",
		},
		{
			name:   "in a condition and in a template",
			hari:   "'글'을 \"가나다\"로 정하자\n만약 ('글'의 '길이' > 2) 라면:\n    틀\"길이 {'글'의 '길이'}\"를 출력하자\n",
			kanade: "『文』を「あいう」にしよう\nもし(『文』の『長さ』が2より大きい)なら:\n    枠「長さ {『文』の『長さ』}」を出力しよう\n",
			want:   "길이 3", wantKanade: "長さ 3",
		},
	})
}
