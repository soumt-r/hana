package tests

import (
	"fmt"
	"testing"

	lexer "github.com/soumt-r/hana/lexer/haja"
	parser "github.com/soumt-r/hana/parser/haja"
	"github.com/soumt-r/hana/vm"
)

func TestHajaLoops(t *testing.T) {
	input := `
(참고) 반복문 테스트
<반복테스트>를 만들자 ():
    1부터 5까지 반복하자:
        "1"을 출력하자
        반복을 끝내자

    10을 돌려주자

'결과'를 <반복테스트>()로 정하자
'결과'를 출력하자
`

	l := lexer.New(input)
	p := parser.New(l)
	prog := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("파싱 에러: %v", p.Errors())
	}
	fmt.Printf("AST:\n%s\n", prog.String())

	interpreter := vm.NewInterpreter(prog)
	fmt.Println("=== Haja 루프 테스트 ===")
	err := interpreter.Run()
	if err != nil {
		t.Fatalf("런타임 에러: %v", err)
	}
}
