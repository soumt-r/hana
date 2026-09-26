package tests

import (
	"reflect"
	"testing"
)

// Inside a method, getter or setter a name that is not a variable of the call
// is the object's field of that name (the tree-walker's Environment rule): the
// bytecode VM used to report it as a missing variable. The call's own
// variables (a parameter of the same name) come first, writes keep the field's
// declared type, and a name that is no field becomes a local variable.
func TestMethodsReadAndWriteFieldsByName(t *testing.T) {
	src := `[상자]를 설계하자:
    '값'을 [숫자]인 1로 정하자
    '목록'을 [(숫자)목록]인 [1]로 정하자
    '이름'을 "상자"로 정하자
    '별칭'을 [문자열]로 정하자:
        가져올 때:
            ('이름' + "!")을 돌려주자
        정할 때 ('새값'):
            '이름'을 '새값'으로 정하자
    <보기>를 만들자 ():
        '값'을 돌려주자
    <늘리기>를 만들자 ():
        '값'에 1을 더하자
    <가림>을 만들자 ('값'):
        '값'에 100을 더하자
        '값'을 출력하자
    <타입위반>을 만들자 ():
        '값'을 "글자"로 정하자
    <목록추가>를 만들자 ('x'):
        '목록'에 'x'를 추가하자
    <새지역>을 만들자 ():
        '없던것'을 5로 정하자
        '없던것'을 출력하자
'b'를 새로운 [상자]()로 정하자
('b'의 <보기>())를 출력하자
'b'의 <늘리기>()를 실행하자
'b'의 <가림>(7)을 실행하자
('b'의 '값')을 출력하자
'b'의 '별칭'을 "새이름"으로 정하자
('b'의 '별칭')을 출력하자
'b'의 <목록추가>(2)를 실행하자
일단 해보자:
    'b'의 <목록추가>("셋")을 실행하자
오류가 발생했다면 ('e'):
    "목록 타입"을 출력하자
('b'의 '목록')을 출력하자
일단 해보자:
    'b'의 <타입위반>()을 실행하자
오류가 발생했다면 ('e'):
    "값 타입"을 출력하자
'b'의 <새지역>()을 실행하자
`
	want := []string{"1", "107", "2", "새이름!", "목록 타입", "[1, 2]", "값 타입", "5"}
	tree, err := runHari(t, src)
	if err != nil {
		t.Fatalf("tree-walker: %v", err)
	}
	if !reflect.DeepEqual(tree.Output, want) {
		t.Errorf("tree-walker: got %q, want %q", tree.Output, want)
	}
	bc, err := runBytecode(t, src)
	if err != nil {
		t.Fatalf("bytecode: %v", err)
	}
	if !reflect.DeepEqual(bc.Output, want) {
		t.Errorf("bytecode: got %q, want %q", bc.Output, want)
	}
}
