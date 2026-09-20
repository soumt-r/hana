package tests

// `[상자]인 <만들기>()` is a call whose result is checked against [상자]. It used to be
// read as a static call on the class (the reading Kanade needs, where "の" is both the
// type word and the member particle) and failed with "static member not found".
// A Haja static call is spelled with the particle: `[상자]의 <이름짓기>()`.

import (
	"strings"
	"testing"
)

func TestTypedCallToAClassIsACall(t *testing.T) {
	code := `
[상자]를 설계하자:
    처음 만들어질 때 () 다음과 같이 하자:
        '나'의 '값'을 3으로 정하자
    [문자열]을 돌려주는 '우리'의 <이름짓기>를 만들자 ():
        "정적"을 돌려주자
[상자]를 돌려주는 <상자만들기>를 만들자 ():
    새로운 [상자]()를 돌려주자
'a'를 [상자]인 <상자만들기>()로 정하자
'a'의 '값'을 출력하자
[상자]의 <이름짓기>()를 출력하자
'b'를 [문자열]인 [상자]의 <이름짓기>()로 정하자
'b'를 출력하자
`
	for engine, r := range engines(t, code) {
		if r.err != nil || strings.Join(r.out, "|") != "3|정적|정적" {
			t.Errorf("%s: got %v (%v)", engine, r.out, r.err)
		}
	}
}
