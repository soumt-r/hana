package tests

// [목록]'s tools that take a function of the program (변환하기, 걸러내기, 접기,
// 찾기, 하나라도, 모두, 기준정렬). The basic calls are in stdCases; these are the
// edges, run on both Go engines.

import (
	"strings"
	"testing"
)

const callbackPrelude = `
[목록]에서 전부 가져오자

<두배>를 만들자 ('x'):
    ('x' * 2)를 돌려주자
<짝수>를 만들자 ('x'):
    만약 ('x' % 2 == 0) 라면:
        참을 돌려주자
    거짓을 돌려주자
<더하기>를 만들자 ('합', 'x'):
    ('합' + 'x')를 돌려주자
<자기자신>를 만들자 ('x'):
    'x'를 돌려주자
<숫자아님>를 만들자 ('x'):
    "네"를 돌려주자
<터짐>를 만들자 ('x'):
    새로운 [오류]("콜백이 터졌어요")를 발생시키자
<길이>를 만들자 ('글'):
    '글'의 '길이'를 돌려주자
<아무것도>를 만들자 ('x'):
    'y'를 [숫자]인 1로 정하자
`

func TestListCallbacksOnEmptyLists(t *testing.T) {
	code := callbackPrelude + `
<변환하기>([], <두배>)를 출력하자
<걸러내기>([], <짝수>)를 출력하자
<접기>([], <더하기>, 7)를 출력하자
<찾기>([], <짝수>)를 출력하자
<하나라도>([], <짝수>)를 출력하자
<모두>([], <짝수>)를 출력하자
<기준정렬>([], <두배>)를 출력하자
`
	for engine, r := range engines(t, code) {
		if r.err != nil {
			t.Fatalf("%s: %v", engine, r.err)
		}
		if got, want := strings.Join(r.out, "|"), "[]|[]|7|비어있음|거짓|참|[]"; got != want {
			t.Errorf("%s: got %q, want %q", engine, got, want)
		}
	}
}

func TestListCallbacksKeepValuesAndOrder(t *testing.T) {
	code := callbackPrelude + `
<접기>(["가", "나", "다"], <더하기>, "")를 출력하자
<기준정렬>(["나무", "가", "하늘하늘"], <길이>)를 출력하자
<기준정렬>(["b", "a", "c"], <자기자신>)를 출력하자
<기준정렬>([3, 1, 2], <자기자신>)를 출력하자
<변환하기>([1, 2], <아무것도>)를 출력하자
<찾기>([1, 3, 4, 6], <짝수>)를 출력하자
<하나라도>([1, 3], <짝수>)를 출력하자
<모두>([2, 3], <짝수>)를 출력하자
`
	for engine, r := range engines(t, code) {
		if r.err != nil {
			t.Fatalf("%s: %v", engine, r.err)
		}
		if got, want := strings.Join(r.out, "|"), "가나다|[가, 나무, 하늘하늘]|[a, b, c]|[1, 2, 3]|[비어있음, 비어있음]|4|거짓|거짓"; got != want {
			t.Errorf("%s: got %q, want %q", engine, got, want)
		}
	}
}

func TestListCallbacksRefuseWhatTheyCannotUse(t *testing.T) {
	cases := []struct{ call, want string }{
		{`<걸러내기>([1, 2], <숫자아님>)`, "must return true or false"},
		{`<찾기>([1, 2], <숫자아님>)`, "must return true or false"},
		{`<하나라도>([1, 2], <숫자아님>)`, "must return true or false"},
		{`<모두>([1, 2], <숫자아님>)`, "must return true or false"},
		{`<변환하기>([1, 2], 5)`, "callable"},
		{`<접기>([1, 2], "함수아님", 0)`, "callable"},
		{`<변환하기>("글", <두배>)`, "list"},
		{`<변환하기>([1, 2])`, "Expected 2 argument"},
		{`<접기>([1, 2], <더하기>)`, "Expected 3 argument"},
		{`<기준정렬>([1, "a"], <자기자신>)`, "sorted"},
		{`<기준정렬>([[1], [2]], <자기자신>)`, "sorted"},
		{`<변환하기>([1, 2], <터짐>)`, "콜백이 터졌어요"},
	}
	for _, c := range cases {
		for engine, r := range engines(t, callbackPrelude+c.call+"를 출력하자\n") {
			if r.err == nil || !strings.Contains(r.err.Error(), c.want) {
				t.Errorf("%s %s: want an error containing %q, got %v", engine, c.call, c.want, r.err)
			}
		}
	}
}
