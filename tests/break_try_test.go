package tests

// 반복을 끝내자 leaves only the 일단 해보자 blocks entered inside its own loop. The
// bytecode compiler used to pop every enclosing try, including the ones the loop sits
// in, and the VM crashed popping an empty try stack. A range whose start or end is not a
// number is RangeMustBeNumbers in both engines.

import (
	"strings"
	"testing"

	"github.com/soumt-r/hana/errs"
)

func TestBreakAndTry(t *testing.T) {
	cases := []struct {
		name, code, out, err string
	}{
		{"a break in a loop inside a try", `
일단 해보자:
    1부터 5까지 반복하자 ('수'):
        만약 ('수' == 2) 라면:
            반복을 끝내자
        '수'를 이어출력하자
    "뒤"를 이어출력하자
오류가 발생했다면 ('e'):
    "잡음"을 이어출력하자
"끝"을 출력하자
`, "1뒤끝", ""},
		{"the try around the loop still catches after a break", `
일단 해보자:
    1부터 5까지 반복하자 ('수'):
        반복을 끝내자
    새로운 [오류]("밖")를 발생시키자
오류가 발생했다면 ('e'):
    'e'의 '메시지'를 이어출력하자
"끝"을 출력하자
`, "밖끝", ""},
		{"a finally around the loop runs once, after the loop", `
일단 해보자:
    1부터 5까지 반복하자 ('수'):
        만약 ('수' == 3) 라면:
            반복을 끝내자
        '수'를 이어출력하자
    "뒤"를 이어출력하자
마무리는 항상:
    "F"를 이어출력하자
"끝"을 출력하자
`, "12뒤F끝", ""},
		{"a break out of a try inside the loop runs its finally", `
1부터 5까지 반복하자 ('수'):
    일단 해보자:
        만약 ('수' == 2) 라면:
            반복을 끝내자
        '수'를 이어출력하자
    마무리는 항상:
        "F"를 이어출력하자
"끝"을 출력하자
`, "1FF끝", ""},
		{"a try left by a break no longer catches", `
1부터 5까지 반복하자 ('수'):
    일단 해보자:
        반복을 끝내자
    오류가 발생했다면 ('e'):
        "잘못 잡음"을 이어출력하자
새로운 [오류]("밖")를 발생시키자
`, "", "밖"},
		{"a break from a handler inside the loop", `
일단 해보자:
    1부터 5까지 반복하자 ('수'):
        일단 해보자:
            새로운 [오류]("안")를 발생시키자
        오류가 발생했다면 ('e'):
            'e'의 '메시지'를 이어출력하자
            반복을 끝내자
    새로운 [오류]("밖")를 발생시키자
오류가 발생했다면 ('e'):
    'e'의 '메시지'를 이어출력하자
"끝"을 출력하자
`, "안밖끝", ""},
		{"nested loops and tries", `
일단 해보자:
    1부터 2까지 반복하자 ('가'):
        일단 해보자:
            1부터 3까지 반복하자 ('나'):
                만약 ('나' == 2) 라면:
                    반복을 끝내자
                '나'를 이어출력하자
            "|"를 이어출력하자
        오류가 발생했다면 ('e'):
            "잡음"을 이어출력하자
        만약 ('가' == 1) 라면:
            반복을 끝내자
오류가 발생했다면 ('e'):
    "잡음"을 이어출력하자
"끝"을 출력하자
`, "1|끝", ""},
		{"a break in a loop inside a try in a function", `
<세기>를 만들자 ():
    일단 해보자:
        '목록'을 [1, 2, 3]으로 정하자
        '목록'의 '값'마다 반복하자:
            만약 ('값' == 3) 라면:
                반복을 끝내자
            '값'을 이어출력하자
    오류가 발생했다면 ('e'):
        "잡음"을 이어출력하자
    "함수끝"을 돌려주자
<세기>()를 출력하자
`, "12함수끝", ""},

		// Runtime spec 4.4: a 반복을 끝내자 run outside any loop is an IllegalBreakError.
		// The loops around a call do not count: the function's body is not in them.
		{"a break at the top level", `
"가"를 이어출력하자
반복을 끝내자
"나"를 이어출력하자
`, "가", "'반복을 끝내자'는 반복 안에서만 쓸 수 있어요"},
		{"a break outside a loop that never runs", `
만약 (1 == 2) 라면:
    반복을 끝내자
"끝"을 출력하자
`, "끝", ""},
		{"a break in a function does not leave the caller's loop", `
<끊기>를 만들자 ():
    "f"를 이어출력하자
    반복을 끝내자
1부터 3까지 반복하자 ('수'):
    '수'를 이어출력하자
    일단 해보자:
        <끊기>()를 실행하자
    오류가 발생했다면 ('e'):
        "!"를 이어출력하자
"끝"을 출력하자
`, "1f!2f!3f!끝", ""},
		{"a break outside a loop can be caught where it runs", `
일단 해보자:
    반복을 끝내자
오류가 발생했다면 ('e'):
    'e'의 '메시지'를 출력하자
`, "IllegalBreakError: '반복을 끝내자'는 반복 안에서만 쓸 수 있어요.", ""},
		{"a break in a handler outside a loop", `
일단 해보자:
    새로운 [오류]("가")를 발생시키자
오류가 발생했다면 ('e'):
    반복을 끝내자
`, "", "'반복을 끝내자'는 반복 안에서만 쓸 수 있어요"},

		// An error inside a handler still runs the try's finally on its way up.
		{"an error in a handler runs the finally", `
일단 해보자:
    새로운 [오류]("가")를 발생시키자
오류가 발생했다면 ('e'):
    새로운 [오류]("나")를 발생시키자
마무리는 항상:
    "F"를 이어출력하자
`, "F", "나"},
		{"an error in a handler, the finally, then an outer handler", `
일단 해보자:
    일단 해보자:
        새로운 [오류]("가")를 발생시키자
    오류가 발생했다면 ('e'):
        새로운 [오류]("나")를 발생시키자
    마무리는 항상:
        "F안"을 이어출력하자
오류가 발생했다면 ('e'):
    'e'의 '메시지'를 이어출력하자
마무리는 항상:
    "F밖"을 이어출력하자
"끝"을 출력하자
`, "F안나F밖끝", ""},
		{"a finally that fails after a failed handler", `
일단 해보자:
    일단 해보자:
        새로운 [오류]("가")를 발생시키자
    오류가 발생했다면 ('e'):
        새로운 [오류]("나")를 발생시키자
    마무리는 항상:
        새로운 [오류]("다")를 발생시키자
오류가 발생했다면 ('e'):
    'e'의 '메시지'를 출력하자
`, "다", ""},
		{"a handler that completes runs the finally once", `
일단 해보자:
    새로운 [오류]("가")를 발생시키자
오류가 발생했다면 ('e'):
    "잡음"을 이어출력하자
마무리는 항상:
    "F"를 이어출력하자
새로운 [오류]("밖")를 발생시키자
`, "잡음F", "밖"},
		{"a return from a handler runs the finally once", `
<값>을 만들자 ():
    일단 해보자:
        새로운 [오류]("가")를 발생시키자
    오류가 발생했다면 ('e'):
        "돌려줌"을 돌려주자
    마무리는 항상:
        "F"를 이어출력하자
<값>()를 출력하자
새로운 [오류]("밖")를 발생시키자
`, "F돌려줌", "밖"},
		{"a break from a handler runs the finally once", `
1부터 3까지 반복하자 ('수'):
    일단 해보자:
        새로운 [오류]("가")를 발생시키자
    오류가 발생했다면 ('e'):
        반복을 끝내자
    마무리는 항상:
        "F"를 이어출력하자
"끝"을 출력하자
새로운 [오류]("밖")를 발생시키자
`, "F끝", "밖"},

		{"a range whose end is not a number", `
'n'을 "다섯"으로 정하자
1부터 'n'까지 반복하자 ('수'):
    '수'를 출력하자
`, "", "범위는 숫자로 정해야 해요"},
		{"a range whose start is not a number", `
'n'을 비어있음으로 정하자
'n'부터 3까지 반복하자 ('수'):
    '수'를 출력하자
`, "", "범위는 숫자로 정해야 해요"},
		{"a range that is not numbers can be caught", `
일단 해보자:
    "가"부터 3까지 반복하자 ('수'):
        '수'를 출력하자
오류가 발생했다면 ('e'):
    'e'의 '메시지'를 출력하자
`, "TypeError: 범위는 숫자로 정해야 해요.", ""},
	}
	for _, c := range cases {
		for engine, r := range engines(t, c.code) {
			out := strings.Join(r.out, "")
			gotErr := ""
			if r.err != nil {
				gotErr = errs.Localize(errs.Korean, r.err)
			}
			if out != c.out || (c.err == "") != (r.err == nil) || (c.err != "" && !strings.Contains(gotErr, c.err)) {
				t.Errorf("%s: %s: out %q err %q, want out %q err containing %q", c.name, engine, out, gotErr, c.out, c.err)
			}
		}
	}
}
