package tests

import (
	"fmt"
	"testing"

	lexer "github.com/soumt-r/hana/lexer/hari"
	parser "github.com/soumt-r/hana/parser/hari"
	"github.com/soumt-r/hana/vm"
)

func TestHariInterpreter(t *testing.T) {
	input := `
(참고) 모든 무기가 공통으로 가질 인터페이스를 만들어요
[무기]를 규정하자:
    <공격하기>가 있어야 한다 ()

[무기]를 따르는 [칼]을 설계하자:
    <공격하기>를 만들자 ():
        "칼로 강하게 찌르기!"를 출력하자

[무기]를 따르는 [활]을 설계하자:
    <공격하기>를 만들자 ():
        "멀리서 활 쏘기!"를 출력하자

(참고) 무기를 쥐고 휘두를 플레이어 클래스예요
[용사]를 설계하자:
    '_장착무기'를 비어있음으로 정하여 숨기자

    <무기바꾸기>를 만들자 ('새로운무기'):
        '나'의 '_장착무기'를 '새로운무기'로 정하자
        "무기를 교체했습니다!"를 출력하자

    <공격>을 만들자 ():
        만약 ('나'의 '_장착무기'가 비어있음과 같다) 라면:
            "맨손 공격! 퍽퍽!"을 출력하자
        그렇지 않다면:
            '나'의 '_장착무기'의 <공격하기>()를 실행하자

(참고) 게임 시작! 무기 없이 맨손으로 공격해 봐요
'주인공'을 새로운 [용사]()로 정하자
'주인공'의 <공격>()을 실행하자

(참고) 칼을 장착하고 다시 공격!
'주인공'의 <무기바꾸기>(새로운 [칼]())를 실행하자
'주인공'의 <공격>()을 실행하자

(참고) 활을 장착하고 다시 공격!
'주인공'의 <무기바꾸기>(새로운 [활]())를 실행하자
'주인공'의 <공격>()을 실행하자
`

	l := lexer.New(input)
	p := parser.New(l)
	prog := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("파싱 에러: %v", p.Errors())
	}

	interpreter := vm.NewInterpreter(prog)
	fmt.Println("=== Hari 가상머신 실행 결과 ===")
	err := interpreter.Run()
	if err != nil {
		t.Fatalf("런타임 에러: %v", err)
	}
	fmt.Println("===============================")
}
