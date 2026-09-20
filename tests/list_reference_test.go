package tests

// Runtime spec 1.2: a list is an object. What a function does to the list it was passed
// (push, pop, empty, set an element) happens to the caller's list, and two variables that
// were given the same list share it. Both engines agree.

import (
	"strings"
	"testing"
)

func TestListsAreReferences(t *testing.T) {
	cases := []struct{ name, code, want string }{
		{"pushed and set in a function", `
<채우기>를 만들자 ([(숫자)목록]인 '목'):
    '목' 뒤에 99를 추가하자
    '목'의 1번째를 7로 정하자

'가'를 [(숫자)목록]인 [1, 2]로 정하자
<채우기>('가')를 실행하자
'가'를 출력하자
`, "[7, 2, 99]"},
		{"pushed at the front", `
<채우기>를 만들자 ([(숫자)목록]인 '목'):
    '목' 앞에 0을 추가하자

'가'를 [(숫자)목록]인 [1]로 정하자
<채우기>('가')를 실행하자
'가'를 출력하자
`, "[0, 1]"},
		{"popped in a function", `
<꺼내기>를 만들자 ([(숫자)목록]인 '목'):
    '목' 뒤에서 꺼내자

'가'를 [(숫자)목록]인 [1, 2, 3]으로 정하자
<꺼내기>('가')를 실행하자
'가'를 출력하자
`, "[1, 2]"},
		{"emptied in a function", `
<지우기>를 만들자 ([(숫자)목록]인 '목'):
    '목'의 <비우기>()를 실행하자

'가'를 [(숫자)목록]인 [1, 2]로 정하자
<지우기>('가')를 실행하자
'가'를 출력하자
`, "[]"},
		{"two variables share one list", `
'가'를 [(숫자)목록]인 [1]로 정하자
'나'를 [(숫자)목록]인 '가'로 정하자
'나' 뒤에 2를 추가하자
'가'를 출력하자
`, "[1, 2]"},
		{"a list inside a list", `
'표'를 [(목록)목록]인 [[1], [2]]로 정하자
'안'을 [(숫자)목록]인 '표'의 1번째로 정하자
'안' 뒤에 5를 추가하자
'표'를 출력하자
`, "[[1, 5], [2]]"},
		{"a list field is shared through the object", `
[상자]를 설계하자:
    '안'을 [(숫자)목록]인 [1]로 정하자

<넣기>를 만들자 ([상자]인 '상'):
    '상'의 '안' 뒤에 2를 추가하자

'통'을 [상자]인 새로운 [상자]()로 정하자
'다른통'을 [상자]인 새로운 [상자]()로 정하자
<넣기>('통')를 실행하자
'통'의 '안'을 출력하자
'다른통'의 '안'을 출력하자
`, "[1, 2]|[1]"},
		{"a loop walks the list it began with", `
'수들'을 [(숫자)목록]인 [1, 2, 3]으로 정하자
'수들'의 '수'마다 반복하자:
    '수들' 뒤에 '수'를 추가하자
'수들'을 출력하자
`, "[1, 2, 3, 1, 2, 3]"},
		{"a value popped out", `
'가'를 [(숫자)목록]인 [1, 2, 3]으로 정하자
'끝'을 [숫자]인 '가' 뒤에서 꺼낸 값으로 정하자
'끝'을 출력하자
'가'를 출력하자
`, "3|[1, 2]"},
		{"a refused push leaves the list alone", `
'가'를 [(숫자)목록]인 [1]로 정하자
일단 해보자:
    '가' 뒤에 "글"을 추가하자
오류가 발생했다면 ('e'):
    "실패"를 출력하자
'가'를 출력하자
`, "실패|[1]"},
		{"a refused push onto a field leaves the list alone", `
[상자]를 설계하자:
    '안'을 [(숫자)목록]인 [1]로 정하자

'통'을 [상자]인 새로운 [상자]()로 정하자
일단 해보자:
    '통'의 '안' 뒤에 "글"을 추가하자
오류가 발생했다면 ('e'):
    "실패"를 출력하자
'통'의 '안'을 출력하자
`, "실패|[1]"},
		{"a list that contains itself can be printed", `
'가'를 [목록]인 [1]로 정하자
'가' 뒤에 '가'를 추가하자
'가'를 출력하자
`, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tree, err := runHaja(t, c.code)
			if err != nil {
				t.Fatalf("tree-walker: %v", err)
			}
			bc, err := runBytecode(t, c.code)
			if err != nil {
				t.Fatalf("bytecode: %v", err)
			}
			got, gotBC := strings.Join(tree.Output, "|"), strings.Join(bc.Output, "|")
			if got != gotBC {
				t.Errorf("the engines differ: tree-walker %q, bytecode %q", got, gotBC)
			}
			if c.want != "" && got != c.want {
				t.Errorf("printed %q, want %q", got, c.want)
			}
		})
	}
}

// A constant variable's list cannot be pushed to, popped from or emptied (spec 2.1).
func TestConstantListsCannotBeChanged(t *testing.T) {
	for name, change := range map[string]string{
		"push":  "'목록' 뒤에 3을 추가하자",
		"pop":   "'목록' 뒤에서 꺼내자",
		"clear": "'목록'의 <비우기>()를 실행하자",
	} {
		code := "'목록'을 [(숫자)목록]인 [1, 2]로 고정하자\n" + change + "\n"
		if _, err := runHaja(t, code); err == nil || !strings.Contains(err.Error(), "ConstantAssignmentError") {
			t.Errorf("%s (tree-walker): %v", name, err)
		}
		if _, err := runBytecode(t, code); err == nil || !strings.Contains(err.Error(), "ConstantAssignmentError") {
			t.Errorf("%s (bytecode): %v", name, err)
		}
	}
}
