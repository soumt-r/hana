package tests

import (
	"fmt"
	"testing"

	lexer "github.com/soumt-r/hana/lexer/haja"
	parser "github.com/soumt-r/hana/parser/haja"
	"github.com/soumt-r/hana/vm"
)

func TestHajaTryCatch(t *testing.T) {
	input := `
(참고) 예외처리 테스트
일단 해보자:
    "치명적인 오류 발생!"을 던지자
    "이건 실행안됨"을 출력하자
오류가 발생했다면 ('에러내용'):
    '에러내용'을 출력하자
마무리는 항상:
    "정리작업 완료"를 출력하자
`

	l := lexer.New(input)
	p := parser.New(l)
	prog := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("파싱 에러: %v", p.Errors())
	}

	interpreter := vm.NewInterpreter(prog)
	fmt.Println("=== Haja 예외처리 테스트 ===")
	err := interpreter.Run()
	if err != nil {
		t.Fatalf("런타임 에러: %v", err)
	}
}
