package tests

// Variables made inside a loop body or an error handler belong to that iteration
// or handler (Runtime spec 1.1); the tree-walker, the bytecode VM and the browser
// engine agree. `만약` blocks do not open a scope: what they declare stays visible.
// The bytecode VM used to keep loop variables for the whole function, so a program
// could work there and fail in the tree-walker.

import (
	"strings"
	"testing"

	"github.com/soumt-r/hana/errs"
)

func TestBlockScopes(t *testing.T) {
	cases := []struct {
		name, code, out, err string
	}{
		{"a loop body variable is new every iteration", `
'i'를 [숫자]인 0으로 정하자
('i' < 3) 인 동안 반복하자:
    만약 ('i' == 0) 라면:
        '값'을 [숫자]인 5로 정하자
    '값'을 출력하자
    'i'에 1을 더하자
`, "5", "'값' 변수를 찾을 수 없어요"},
		{"a loop body variable is gone after the loop", `
'n'을 [숫자]인 0으로 정하자
('n' < 2) 인 동안 반복하자:
    '본문'을 [숫자]인 'n' + 10으로 정하자
    'n'에 1을 더하자
'본문'을 출력하자
`, "", "'본문' 변수를 찾을 수 없어요"},
		{"an if block does not open a scope", `
만약 (1 == 1) 라면:
    '안'을 [숫자]인 7로 정하자
'안'을 출력하자
`, "7", ""},
		{"an outer variable is assigned, not shadowed", `
'합'을 [숫자]인 0으로 정하자
1부터 3까지 반복하자 ('횟수'):
    '합'에 '횟수'를 더하자
'합'을 출력하자
`, "6", ""},
		{"the range variable is gone after the loop", `
1부터 2까지 반복하자 ('횟수'):
    '횟수'를 출력하자
'횟수'를 출력하자
`, "12", "'횟수' 변수를 찾을 수 없어요"},
		{"the list item is gone after the loop", `
'과일들'을 [(문자열)목록]인 ["사과", "배"]로 정하자
'과일들'의 '과일'마다 반복하자:
    '과일'을 출력하자
'과일'을 출력하자
`, "사과배", "'과일' 변수를 찾을 수 없어요"},
		{"a constant may be declared again in every iteration", `
'i'를 [숫자]인 0으로 정하자
('i' < 3) 인 동안 반복하자:
    '고정값'을 [숫자]인 'i' * 2로 고정하자
    '고정값'을 출력하자
    'i'에 1을 더하자
`, "024", ""},
		{"a declared type does not carry over to the next iteration", `
'i'를 [숫자]인 0으로 정하자
('i' < 2) 인 동안 반복하자:
    만약 ('i' == 0) 라면:
        '값'을 [숫자]인 1로 정하자
    그렇지 않다면:
        '값'을 [문자열]인 "둘"로 정하자
    '값'을 출력하자
    'i'에 1을 더하자
`, "1둘", ""},
		{"break leaves the loop scope and the code after it runs", `
'i'를 [숫자]인 0으로 정하자
('i' < 10) 인 동안 반복하자:
    '안쪽'을 [숫자]인 'i'로 정하자
    만약 ('i' == 2) 라면:
        반복을 끝내자
    'i'에 1을 더하자
"끝 "을 이어출력하자
'i'를 출력하자
'안쪽'을 출력하자
`, "끝 2", "'안쪽' 변수를 찾을 수 없어요"},
		{"nested loops keep their own variables", `
'합'을 [숫자]인 0으로 정하자
1부터 3까지 반복하자 ('가'):
    '곱'을 [숫자]인 '가' * 10으로 정하자
    1부터 2까지 반복하자 ('나'):
        '더'을 [숫자]인 '곱' + '나'로 정하자
        '합'에 '더'를 더하자
'합'을 출력하자
`, "129", ""},
		{"an error handler's variables end with it", `
일단 해보자:
    새로운 [오류]("실패")를 발생시키자
오류가 발생했다면 ('에러'):
    '내용'을 [문자열]인 '에러'의 '메시지'로 정하자
    '내용'을 출력하자
'에러'를 출력하자
`, "실패", "'에러' 변수를 찾을 수 없어요"},
		{"a loop inside a function", `
<세기>를 만들자 ():
    'i'를 [숫자]인 0으로 정하자
    ('i' < 2) 인 동안 반복하자:
        '안'을 [숫자]인 'i'로 정하자
        'i'에 1을 더하자
    '안'을 출력하자
<세기>()를 실행하자
`, "", "'안' 변수를 찾을 수 없어요"},
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
