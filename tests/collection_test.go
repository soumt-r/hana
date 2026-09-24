package tests

import (
	"fmt"
	"strings"
	"testing"

	lexer "github.com/soumt-r/hana/lexer/hari"
	parser "github.com/soumt-r/hana/parser/hari"
	"github.com/soumt-r/hana/vm"
)

func TestHariCollections(t *testing.T) {
	input := `
(참고) 컬렉션 테스트
'인벤토리'를 ["검", "방패", "포션"]으로 정하자
'아이템'을 '인벤토리'의 1번째로 정하자
'아이템'을 출력하자

'인벤토리' 뒤에 "활"을 추가하자
'인벤토리'의 4번째를 출력하자

'인벤토리' 뒤에서 꺼내자
'인벤토리' 마다 반복하자:
    '아이템'을 출력하자

'능력치'를 {"체력": 100, "마나": 50}으로 정하자
'능력치'의 "마나"를 출력하자
`

	l := lexer.New(input)
	p := parser.New(l)
	prog := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("파싱 에러: %v", p.Errors())
	}

	interpreter := vm.NewInterpreter(prog)
	fmt.Println("=== Hari 컬렉션 테스트 ===")
	err := interpreter.Run()
	if err != nil {
		t.Fatalf("런타임 에러: %v", err)
	}
}

// TestListClearMethod covers <비우기>, the one list pseudo-method the
// runtime spec actually defines (spec 2.6: list mutation is otherwise
// done via native syntax — 추가하자/꺼내자 — not method calls). This was a
// real bug, not a documentation gap: vm/eval_expr.go already built a
// BoundListMethod for any FunctionReference property on a list but the
// CallExpression switch had no case for it at all, so every call fell
// through to "TypeError: Not callable." regardless of method name.
func TestListClearMethod(t *testing.T) {
	interp, err := runHari(t, `'목록'을 ["사과", "포도"]로 정하자
'목록'의 '길이'를 출력하자
'목록'의 <비우기>()를 실행하자
'목록'의 '길이'를 출력하자
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"2", "0"}
	if strings.Join(interp.Output, "|") != strings.Join(want, "|") {
		t.Errorf("Output = %v, want %v", interp.Output, want)
	}
}

// TestListClearMethodOnObjectField covers emptying a list held in an object field: the
// list is changed through the object, not just a plain variable.
func TestListClearMethodOnObjectField(t *testing.T) {
	interp, err := runHari(t, `[상자]를 설계하자:
    '내용물'을 ["사과", "포도"]로 정하자

'상자'를 새로운 [상자]()로 정하자
'상자'의 '내용물'의 '길이'를 출력하자
'상자'의 '내용물'의 <비우기>()를 실행하자
'상자'의 '내용물'의 '길이'를 출력하자
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"2", "0"}
	if strings.Join(interp.Output, "|") != strings.Join(want, "|") {
		t.Errorf("Output = %v, want %v", interp.Output, want)
	}
}

func TestListClearMethodArgumentCountError(t *testing.T) {
	_, err := runHari(t, `'목록'을 ["사과"]로 정하자
'목록'의 <비우기>(1)를 실행하자
`)
	requireErrorContains(t, err, "ArgumentError")
}

func TestListClearMethodUnknownNameError(t *testing.T) {
	_, err := runHari(t, `'목록'을 ["사과"]로 정하자
'목록'의 <정렬하기>()를 실행하자
`)
	requireErrorContains(t, err, "MethodNotFoundError")
}
