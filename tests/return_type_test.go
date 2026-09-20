package tests

// A function may say what it returns (`[숫자]를 돌려주는 <제곱>을 만들자`). That is
// never required, but when it is written the returned value is checked against it:
// null (a function that ends without returning) fits every type, a subclass fits
// its parent, and a value of another type is a TypeError.

import (
	"strings"
	"testing"

	"github.com/soumt-r/hana/errs"
)

const returnTypePrelude = `
[숫자]를 돌려주는 <제곱>을 만들자 ([숫자]인 'x'):
    ('x' * 'x')를 돌려주자
[숫자]를 돌려주는 <글을돌려줌>을 만들자 ():
    "글"을 돌려주자
[숫자]를 돌려주는 <아무것도>를 만들자 ():
    'y'를 [숫자]인 1로 정하자
<선언없음>을 만들자 ():
    "무엇이든"을 돌려주자
[(숫자)목록]를 돌려주는 <숫자목록>을 만들자 ():
    [1, 2, 3]을 돌려주자
[(숫자)목록]를 돌려주는 <섞인목록>을 만들자 ():
    [1, "둘"]을 돌려주자
[논리]를 돌려주는 <논리아님>을 만들자 ():
    1을 돌려주자

[동물]을 설계하자:
    처음 만들어질 때 () 다음과 같이 하자:
        'y'를 [숫자]인 1로 정하자
[개]를 [동물]을 바탕으로 하고 설계하자:
    처음 만들어질 때 () 다음과 같이 하자:
        'y'를 [숫자]인 2로 정하자
[동물]을 돌려주는 <강아지>를 만들자 ():
    새로운 [개]()를 돌려주자
[개]를 돌려주는 <동물이었음>을 만들자 ():
    새로운 [동물]()을 돌려주자
`

func TestReturnTypesAreCheckedWhenDeclared(t *testing.T) {
	ok := returnTypePrelude + `
<제곱>(4)를 출력하자
<아무것도>()를 출력하자
<선언없음>()를 출력하자
<숫자목록>()을 출력하자
'멍멍'을 <강아지>()로 정하자
"통과"를 출력하자
`
	for engine, r := range engines(t, ok) {
		if r.err != nil {
			t.Fatalf("%s: %v", engine, r.err)
		}
		if got, want := strings.Join(r.out, "|"), "16|비어있음|무엇이든|[1, 2, 3]|통과"; got != want {
			t.Errorf("%s: got %q, want %q", engine, got, want)
		}
	}
}

func TestReturnTypeMismatchesAreErrors(t *testing.T) {
	cases := []struct{ call, want string }{
		{"<글을돌려줌>()", "'글을돌려줌' 함수는 '숫자' 타입을 돌려줘야 하는데 '문자열' 값을 돌려줬어요."},
		{"<섞인목록>()", "'섞인목록' 함수는 '(숫자)목록' 타입을 돌려줘야 하는데 '목록' 값을 돌려줬어요."},
		{"<논리아님>()", "'논리아님' 함수는 '논리' 타입을 돌려줘야 하는데 '숫자' 값을 돌려줬어요."},
		{"<동물이었음>()", "'동물이었음' 함수는 '개' 타입을 돌려줘야 하는데 '동물' 값을 돌려줬어요."},
	}
	for _, c := range cases {
		for engine, r := range engines(t, returnTypePrelude+c.call+"를 출력하자\n") {
			if r.err == nil || errs.Localize(errs.Korean, r.err) != "TypeError: "+c.want {
				t.Errorf("%s %s: got %v", engine, c.call, r.err)
			}
		}
	}
}

func TestReturnTypesOfMethodsAndCaughtErrors(t *testing.T) {
	code := `
[계산기]를 설계하자:
    [숫자]를 돌려주는 <더하기>를 만들자 ([숫자]인 'a', [숫자]인 'b'):
        ('a' + 'b')를 돌려주자
    [숫자]를 돌려주는 <엉뚱하게>를 만들자 ():
        "엉뚱"을 돌려주자
'기계'를 [계산기]인 새로운 [계산기]()로 정하자
'기계'의 <더하기>(2, 3)을 출력하자
일단 해보자:
    '기계'의 <엉뚱하게>()를 출력하자
오류가 발생했다면 ('에러'):
    "잡힘: "을 이어출력하자
    '에러'의 '메시지'를 출력하자
`
	for engine, r := range engines(t, code) {
		if r.err != nil {
			t.Fatalf("%s: %v", engine, r.err)
		}
		if len(r.out) != 3 || r.out[0] != "5" || r.out[1] != "잡힘: " || !strings.Contains(r.out[2], "'엉뚱하게' 함수는 '숫자' 타입을 돌려줘야 하는데 '문자열' 값을 돌려줬어요.") {
			t.Errorf("%s: got %q", engine, r.out)
		}
	}
}

func TestReturnTypesInKanade(t *testing.T) {
	ok := "【数字】を返す〈二倍〉を作ろう(【数字】の『x』):\n    (『x』 * 2)を返そう\n【数字】を返す〈違う〉を作ろう():\n    「文字」を返そう\n〈二倍〉(4)を出力しよう\n〈違う〉()を出力しよう\n"
	tw, err1 := runKanade(t, ok)
	bc, err2 := runKanadeBytecode(t, ok)
	for label, run := range map[string]struct {
		out []string
		err error
	}{"tree-walker": {tw.Output, err1}, "bytecode": {bc.Output, err2}} {
		if len(run.out) != 1 || run.out[0] != "8" || run.err == nil || errs.Localize(errs.Japanese, run.err) != "TypeError: 関数『違う』は『数字』型を返す必要がありますが、『文字列』の値を返しました。" {
			t.Errorf("kanade %s: got %v %v", label, run.out, run.err)
		}
	}
}
