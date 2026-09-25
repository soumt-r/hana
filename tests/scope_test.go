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

		// A loop or handler variable is declared in the loop's (handler's) own scope, so a
		// variable of the same name outside is hidden, not changed, and is back afterwards.
		// The bytecode VM used to assign the outer variable instead.
		{"a range variable hides an outer variable of the same name", `
'가'를 100으로 정하자
1부터 2까지 반복하자 ('가'):
    '가'를 이어출력하자
'가'를 출력하자
`, "12100", ""},
		{"a range's ends read the outer variable the loop variable hides", `
'가'를 3으로 정하자
1부터 '가'까지 반복하자 ('가'):
    '가'를 이어출력하자
'가'부터 1까지 반복하자 ('가'):
    '가'를 이어출력하자
'가'를 출력하자
`, "1233213", ""},
		{"a hidden variable's constness and type do not bind the loop variable", `
'가'를 "글"로 고정하자
1부터 2까지 반복하자 ('가'):
    '가'를 이어출력하자
'가'를 출력하자
`, "12글", ""},
		{"a loop variable that hides a constant can be changed", `
'가'를 "글"로 고정하자
1부터 2까지 반복하자 ('가'):
    '가'에 10을 더하자
    '가'를 이어출력하자
'가'를 출력하자
`, "1112글", ""},
		{"a list item that hides a constant list can be pushed to", `
'목'을 [1]로 고정하자
[[5]]의 '목'마다 반복하자:
    '목'에 6을 추가하자
    '목'을 이어출력하자
'목'을 출력하자
`, "[5, 6][1]", ""},
		{"the outer constant is still a constant after the loop", `
'가'를 1로 고정하자
1부터 2까지 반복하자 ('가'):
    '가'에 10을 더하자
'가'에 1을 더하자
`, "", "상수 '가'의 값은 변경할 수 없어요"},
		{"a loop variable the body assigns still hides the outer one", `
'가'를 100으로 정하자
1부터 2까지 반복하자 ('가'):
    '가'에 10을 더하자
    '가'를 이어출력하자
'가'를 출력하자
`, "1112100", ""},
		{"a range to a variable end hides the outer variable", `
'가'를 100으로 정하자
'끝'을 2로 정하자
1부터 '끝'까지 반복하자 ('가'):
    '가'를 이어출력하자
'가'를 출력하자
`, "12100", ""},
		{"nested loops with the same variable", `
1부터 2까지 반복하자 ('가'):
    1부터 2까지 반복하자 ('가'):
        '가'를 이어출력하자
    '가'를 이어출력하자
"."를 출력하자
`, "121122.", ""},
		{"a list item hides an outer variable of the same name", `
'과일'을 "원래"로 정하자
'과일들'을 ["사과", "배"]로 정하자
'과일들'의 '과일'마다 반복하자:
    '과일'을 이어출력하자
'과일'을 출력하자
`, "사과배원래", ""},
		{"a handler's error variable hides an outer variable of the same name", `
'에러'를 "원래"로 정하자
일단 해보자:
    새로운 [오류]("실패")를 발생시키자
오류가 발생했다면 ('에러'):
    '에러'의 '메시지'를 이어출력하자
'에러'를 출력하자
`, "실패원래", ""},
		{"a loop variable hides a parameter in a function", `
<보기>를 만들자 ('가'):
    1부터 2까지 반복하자 ('가'):
        '가'를 이어출력하자
    '가'를 출력하자
<보기>("매개변수")를 실행하자
`, "12매개변수", ""},
		{"breaking out of the loop brings the outer variable back", `
'가'를 100으로 정하자
1부터 5까지 반복하자 ('가'):
    만약 ('가' == 2) 라면:
        반복을 끝내자
    '가'를 이어출력하자
'가'를 출력하자
`, "1100", ""},
		{"a top level with many variables", `
'v1'을 1로 정하자
'v2'를 2로 정하자
'v3'을 3으로 정하자
'v4'를 4로 정하자
'v5'를 5로 정하자
'v6'을 6으로 정하자
'v7'을 7로 정하자
'v8'을 8로 정하자
'v9'를 9로 정하자
'v10'을 10으로 정하자
'v11'을 11로 정하자
'v12'를 12로 정하자
'v13'을 13으로 정하자
1부터 2까지 반복하자 ('v5'):
    'v5'를 이어출력하자
    '안'을 'v5'로 정하자
'v5'를 출력하자
`, "125", ""},

		// An error that leaves a loop for a 일단 해보자 around it ends the loop's scope too.
		{"an error out of a loop ends the loop's scope", `
'가'를 100으로 정하자
일단 해보자:
    1부터 3까지 반복하자 ('가'):
        '임시'를 5로 정하자
        새로운 [오류]("멈춤")를 발생시키자
오류가 발생했다면 ('e'):
    'e'의 '메시지'를 이어출력하자
'가'를 이어출력하자
'임시'를 출력하자
`, "멈춤100", "'임시' 변수를 찾을 수 없어요"},
		{"what the try block itself declares stays after an error", `
일단 해보자:
    '밖'을 1로 정하자
    1부터 3까지 반복하자 ('수'):
        '안'을 2로 정하자
        새로운 [오류]("멈춤")를 발생시키자
오류가 발생했다면 ('e'):
    '밖'을 이어출력하자
'밖'을 출력하자
'안'을 출력하자
`, "11", "'안' 변수를 찾을 수 없어요"},
		{"an error out of a function's loop, caught in the function", `
<보기>를 만들자 ():
    '가'를 100으로 정하자
    일단 해보자:
        '목록'을 [1, 2]로 정하자
        '목록'의 '가'마다 반복하자:
            새로운 [오류]("멈춤")를 발생시키자
    오류가 발생했다면 ('e'):
        '가'를 출력하자
<보기>()를 실행하자
`, "100", ""},
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
