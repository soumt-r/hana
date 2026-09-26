package tests

// Regression tests for the bytecode compiler+VM, a second execution path
// alongside the tree-walking interpreter covered by the rest of this package:
// the core language plus switch/for-each/classes/interfaces/try-catch/import.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/soumt-r/hana/bcstdlib"
	"github.com/soumt-r/hana/bcvm"
	"github.com/soumt-r/hana/bytecode"
	lexer "github.com/soumt-r/hana/lexer/hari"
	parser "github.com/soumt-r/hana/parser/hari"
)

// runBytecode lexes, parses, compiles, and runs a hari source string
// through the bytecode VM (with the standard library registered, same as
// the real CLI), returning the VM (for Output) and any error (parse,
// compile, or runtime — the caller usually only cares that there is or
// isn't one).
func runBytecode(t *testing.T, code string) (*bcvm.VM, error) {
	t.Helper()
	l := lexer.New(code)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse error: %v", p.Errors())
	}
	compiler := bytecode.NewCompiler()
	bcProg := compiler.Compile(prog)
	if len(compiler.Errors()) > 0 {
		t.Fatalf("compile error: %v", compiler.Errors())
	}
	vm := bcvm.New(bcProg)
	bcstdlib.RegisterStandardLibrary(vm)
	err := vm.Run()
	return vm, err
}

func TestBytecodeArithmeticAndPrint(t *testing.T) {
	vm, err := runBytecode(t, `
'결과'를 (2 + 3) * 4로 정하자
'결과'를 출력하자
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vm.Output) != 1 || vm.Output[0] != "20" {
		t.Errorf("Output = %v, want [\"20\"]", vm.Output)
	}
}

func TestBytecodeIfElse(t *testing.T) {
	vm, err := runBytecode(t, `
'숫자'를 42로 정하자
만약 ('숫자'가 10보다 크다) 라면:
    '숫자'를 출력하자
그렇지 않다면:
    0을 출력하자
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vm.Output) != 1 || vm.Output[0] != "42" {
		t.Errorf("Output = %v, want [\"42\"]", vm.Output)
	}
}

func TestBytecodeWhileLoop(t *testing.T) {
	vm, err := runBytecode(t, `
'n'을 0으로 정하자
('n'이 3보다 작다) 인 동안 반복하자:
    'n'을 출력하자
    'n'에 1을 더하자
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"0", "1", "2"}
	if strings.Join(vm.Output, ",") != strings.Join(want, ",") {
		t.Errorf("Output = %v, want %v", vm.Output, want)
	}
}

// ForRangeStatement must be inclusive of the end value and count down when
// start > end (Runtime spec 5), and 반복을 끝내자 must exit the loop.
func TestBytecodeForRange(t *testing.T) {
	vm, err := runBytecode(t, `1부터 5까지 반복하자 ('i'):
    만약 ('i'가 3보다 크다) 라면:
        반복을 끝내자
    'i'를 출력하자
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"1", "2", "3"}
	if strings.Join(vm.Output, ",") != strings.Join(want, ",") {
		t.Errorf("Output = %v, want %v", vm.Output, want)
	}

	vm, err = runBytecode(t, `5부터 1까지 반복하자 ('i'):
    'i'를 출력하자
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want = []string{"5", "4", "3", "2", "1"}
	if strings.Join(vm.Output, ",") != strings.Join(want, ",") {
		t.Errorf("Output = %v, want %v", vm.Output, want)
	}
}

func TestBytecodeFunctionDefaultsAndOverride(t *testing.T) {
	vm, err := runBytecode(t, `
[숫자]를 돌려주는 <더하기>를 만들자 ([숫자]인 '가', [숫자]인 '다' = 10):
    '가' + '다'를 돌려주자
<더하기>(1, 2)를 출력하자
<더하기>(1)를 출력하자
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"3", "11"}
	if strings.Join(vm.Output, ",") != strings.Join(want, ",") {
		t.Errorf("Output = %v, want %v", vm.Output, want)
	}
}

func TestBytecodeMissingAndTooManyArguments(t *testing.T) {
	_, err := runBytecode(t, `
<더하기>를 만들자 ([숫자]인 '가', [숫자]인 '다'):
    '가' + '다'를 돌려주자
<더하기>(1)를 실행하자
`)
	if err == nil || !strings.Contains(err.Error(), "MissingArgumentError") {
		t.Errorf("error = %v, want MissingArgumentError", err)
	}

	_, err = runBytecode(t, `
<더하기>를 만들자 ([숫자]인 '가', [숫자]인 '다'):
    '가' + '다'를 돌려주자
<더하기>(1, 2, 3)를 실행하자
`)
	if err == nil || !strings.Contains(err.Error(), "ArgumentError") {
		t.Errorf("error = %v, want ArgumentError", err)
	}
}

func TestBytecodeRecursion(t *testing.T) {
	vm, err := runBytecode(t, `
[숫자]를 돌려주는 <팩토리얼>을 만들자 ([숫자]인 'n'):
    만약 ('n'이 1 이하이다) 라면:
        1을 돌려주자
    ('n' * <팩토리얼>('n' - 1))를 돌려주자
<팩토리얼>(5)를 출력하자
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vm.Output) != 1 || vm.Output[0] != "120" {
		t.Errorf("Output = %v, want [\"120\"]", vm.Output)
	}
}

func TestBytecodeListOps(t *testing.T) {
	vm, err := runBytecode(t, `
'과일'을 ["사과", "포도"]로 정하자
'과일' 앞에 "바나나"를 추가하자
'과일' 뒤에 "수박"을 추가하자
'과일' 앞에서 꺼내자
'과일'을 출력하자
'과일'의 '길이'를 출력하자
'과일'의 1번째를 출력하자
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"[사과, 포도, 수박]", "3", "사과"}
	if strings.Join(vm.Output, ",") != strings.Join(want, ",") {
		t.Errorf("Output = %v, want %v", vm.Output, want)
	}
}

func TestBytecodeListPopExpressionAndEmptyError(t *testing.T) {
	vm, err := runBytecode(t, `
'과일'을 ["사과", "포도", "수박"]으로 정하자
'꺼낸것'을 '과일' 뒤에서 꺼낸 값으로 정하자
'꺼낸것'을 출력하자
'과일'을 출력하자
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"수박", "[사과, 포도]"}
	if strings.Join(vm.Output, ",") != strings.Join(want, ",") {
		t.Errorf("Output = %v, want %v", vm.Output, want)
	}

	_, err = runBytecode(t, `
'빈목록'을 []로 정하자
'빈목록' 뒤에서 꺼내자
`)
	if err == nil || !strings.Contains(err.Error(), "IndexOutOfBoundsError") {
		t.Errorf("error = %v, want IndexOutOfBoundsError", err)
	}
}

func TestBytecodeTemplateLiteral(t *testing.T) {
	vm, err := runBytecode(t, `
'이름'을 "철수"로 정하자
'나이'를 25로 정하자
틀"{'이름'}님은 {'나이'}살입니다"를 출력하자
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vm.Output) != 1 || vm.Output[0] != "철수님은 25살입니다" {
		t.Errorf("Output = %v, want [\"철수님은 25살입니다\"]", vm.Output)
	}
}

// 그리고/또는 must short-circuit: the right operand's side effect must not
// run once the left already determines the result.
func TestBytecodeLogicalShortCircuit(t *testing.T) {
	vm, err := runBytecode(t, `
<부작용>을 만들자 ():
    "호출됨"을 출력하자
    참을 돌려주자

만약 (거짓) 그리고 (<부작용>()) 라면:
    "실행안됨"을 출력하자
"끝1"을 출력하자

만약 (참) 또는 (<부작용>()) 라면:
    "실행됨"을 출력하자
"끝2"를 출력하자
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"끝1", "실행됨", "끝2"}
	if strings.Join(vm.Output, ",") != strings.Join(want, ",") {
		t.Errorf("Output = %v, want %v (부작용 must never run)", vm.Output, want)
	}
}

func TestBytecodeDivideByZero(t *testing.T) {
	_, err := runBytecode(t, `'x'를 (10 / 0)으로 정하자`)
	if err == nil || !strings.Contains(err.Error(), "DivideByZeroError") {
		t.Errorf("error = %v, want DivideByZeroError", err)
	}
}

func TestBytecodeSwitchFallthrough(t *testing.T) {
	vm, err := runBytecode(t, `
'등급'을 "VIP"로 정하자
'등급'에 따라 나누자:
    "VIP" 인 경우:
        "환영합니다"를 출력하자
        다음으로 이어가자
    "일반" 인 경우:
        "안녕하세요"를 출력하자
    나머지는:
        "손님"을 출력하자
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"환영합니다", "안녕하세요"}
	if strings.Join(vm.Output, ",") != strings.Join(want, ",") {
		t.Errorf("Output = %v, want %v", vm.Output, want)
	}
}

func TestBytecodeSwitchMultiTestAndDefault(t *testing.T) {
	vm, err := runBytecode(t, `
'점수'를 3으로 정하자
'점수'에 따라 나누자:
    1, 2 인 경우:
        "낮음"을 출력하자
    3, 4 인 경우:
        "중간"을 출력하자
    나머지는:
        "높음"을 출력하자

'미매치'를 99로 정하자
'미매치'에 따라 나누자:
    1 인 경우:
        "하나"를 출력하자
    나머지는:
        "기타"를 출력하자
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"중간", "기타"}
	if strings.Join(vm.Output, ",") != strings.Join(want, ",") {
		t.Errorf("Output = %v, want %v", vm.Output, want)
	}
}

func TestBytecodeForEachDefaultItemName(t *testing.T) {
	vm, err := runBytecode(t, `
'과일들'을 [(문자열)목록]인 ["사과", "포도", "배"]로 정하자
'과일들' 마다 반복하자:
    '아이템'을 출력하자
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"사과", "포도", "배"}
	if strings.Join(vm.Output, ",") != strings.Join(want, ",") {
		t.Errorf("Output = %v, want %v", vm.Output, want)
	}
}

func TestBytecodeForEachNamedItemAndBreak(t *testing.T) {
	vm, err := runBytecode(t, `
'이름들'을 [(문자열)목록]인 ["철수", "영희"]로 정하자
'이름들'의 '이름'마다 반복하자:
    '이름'을 출력하자

'숫자들'을 [(숫자)목록]인 [1, 2, 3, 4, 5]로 정하자
'합계'를 0으로 정하자
'숫자들' 마다 반복하자:
    만약 ('아이템'이 3과 같다) 라면:
        반복을 끝내자
    '합계'를 ('합계' + '아이템')으로 정하자
'합계'를 출력하자
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"철수", "영희", "3"}
	if strings.Join(vm.Output, ",") != strings.Join(want, ",") {
		t.Errorf("Output = %v, want %v", vm.Output, want)
	}
}

func TestBytecodeClassConstructorAndInheritance(t *testing.T) {
	vm, err := runBytecode(t, `
[자동차]를 설계하자:
    '연료'를 [숫자]인 100으로 정하여 숨기자
    '색상'을 [문자열]인 "빨강"으로 정하자

    처음 만들어질 때 ([문자열]인 '초기색상') 다음과 같이 하자:
        '나'의 '색상'을 '초기색상'으로 정하자

    [문자열]을 돌려주는 <색깔>을 만들자 ():
        '나'의 '색상'을 돌려주자

[자동차]를 바탕으로 [전기차]를 설계하자:
    처음 만들어질 때 ([문자열]인 '초기색상') 다음과 같이 하자:
        부모의 <처음 만들어질 때>('초기색상')을 실행하자

'차'를 [전기차]인 새로운 [전기차]("파랑")으로 정하자
틀"{'차'의 <색깔>()}"를 출력하자
만약 ('차'가 [자동차]의 일종이다) 라면:
    "상속됨"을 출력하자
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"파랑", "상속됨"}
	if strings.Join(vm.Output, ",") != strings.Join(want, ",") {
		t.Errorf("Output = %v, want %v", vm.Output, want)
	}
}

func TestBytecodeClassOverrideAndSuperCall(t *testing.T) {
	vm, err := runBytecode(t, `
[동물]을 설계하자:
    [문자열]을 돌려주는 <소리>를 만들자 ():
        "..."를 돌려주자

[동물]을 바탕으로 [개]를 설계하자:
    [문자열]을 돌려주는 <소리>를 만들자 ():
        틀"멍멍 ({부모의 <소리>()})"를 돌려주자

'개'를 [개]인 새로운 [개]()로 정하자
틀"{'개'의 <소리>()}"를 출력하자
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"멍멍 (...)"}
	if strings.Join(vm.Output, ",") != strings.Join(want, ",") {
		t.Errorf("Output = %v, want %v", vm.Output, want)
	}
}

func TestBytecodeStaticFieldAndMethod(t *testing.T) {
	vm, err := runBytecode(t, `
[상자]를 설계하자:
    '우리'의 '개수'를 [숫자]인 0으로 정하자

    [숫자]를 돌려주는 '우리'의 <증가>를 만들자 ():
        '우리'의 '개수'에 1을 더하자
        '우리'의 '개수'를 돌려주자

'a'를 [상자]의 <증가>()로 정하자
'b'를 [상자]의 <증가>()로 정하자
틀"{'a'}, {'b'}, {[상자]의 '개수'}"를 출력하자
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"1, 2, 2"}
	if strings.Join(vm.Output, ",") != strings.Join(want, ",") {
		t.Errorf("Output = %v, want %v", vm.Output, want)
	}
}

func TestBytecodePrivateAccessViolation(t *testing.T) {
	_, err := runBytecode(t, `
[금고]를 설계하자:
    '비밀번호'를 [문자열]인 "1234"로 정하여 숨기자

'금고1'을 [금고]인 새로운 [금고]()로 정하자
'금고1'의 '비밀번호'를 출력하자
`)
	if err == nil || !strings.Contains(err.Error(), "AccessViolationError") {
		t.Errorf("error = %v, want AccessViolationError", err)
	}
}

func TestBytecodeInterfacePreflightRejectsIncompleteClass(t *testing.T) {
	_, err := runBytecode(t, `
[움직이는것]을 규정하자:
    <달리기>가 있어야 한다 ([숫자]인 '속도')

[움직이는것]을 따르는 [가짜]를 설계하자:
    '이름'을 [문자열]인 "가짜"로 정하자
`)
	if err == nil || !strings.Contains(err.Error(), "InterfaceImplementationError") {
		t.Errorf("error = %v, want InterfaceImplementationError", err)
	}
}

func TestBytecodeInterfaceImplementedPasses(t *testing.T) {
	vm, err := runBytecode(t, `
[소리내는것]을 규정하자:
    <소리내기>가 있어야 한다 ()

[소리내는것]을 따르는 [고양이]를 설계하자:
    <소리내기>를 만들자 ():
        "야옹"을 출력하자

'냥'을 [고양이]인 새로운 [고양이]()로 정하자
'냥'의 <소리내기>()를 실행하자
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"야옹"}
	if strings.Join(vm.Output, ",") != strings.Join(want, ",") {
		t.Errorf("Output = %v, want %v", vm.Output, want)
	}
}

func TestBytecodeSetter(t *testing.T) {
	vm, err := runBytecode(t, `
[온도계]를 설계하자:
    '섭씨'를 [숫자]인 0으로 정하자

    '화씨' 정하자:
        정할 때 ('값'):
            '나'의 '섭씨'를 '값'으로 정하자

'온도'를 [온도계]인 새로운 [온도계]()로 정하자
'온도'의 '화씨'를 100으로 정하자
틀"{'온도'의 '섭씨'}"를 출력하자
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"100"}
	if strings.Join(vm.Output, ",") != strings.Join(want, ",") {
		t.Errorf("Output = %v, want %v", vm.Output, want)
	}
}

func TestBytecodeCatchUntypedCatchesEngineError(t *testing.T) {
	vm, err := runBytecode(t, `
일단 해보자:
    '결과'를 10 / 0으로 정하자
오류가 발생했다면 ('에러'):
    틀"잡힘: {'에러'의 '메시지'}"를 출력하자
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"잡힘: DivideByZeroError: 0으로 나눌 수 없어요."}
	if strings.Join(vm.Output, ",") != strings.Join(want, ",") {
		t.Errorf("Output = %v, want %v", vm.Output, want)
	}
}

func TestBytecodeCatchTypedHandlerDoesNotMatchEngineError(t *testing.T) {
	vm, err := runBytecode(t, `
일단 해보자:
    '결과'를 10 / 0으로 정하자
[수학오류]가 발생했다면 ('에러'):
    "여기 오면 버그"를 출력하자
오류가 발생했다면 ('에러'):
    "폴백"을 출력하자
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"폴백"}
	if strings.Join(vm.Output, ",") != strings.Join(want, ",") {
		t.Errorf("Output = %v, want %v", vm.Output, want)
	}
}

func TestBytecodeThrowAndTypedCatchViaUpcasting(t *testing.T) {
	vm, err := runBytecode(t, `
[오류]를 바탕으로 [내오류]를 설계하자:
    처음 만들어질 때 ([문자열]인 '메시지') 다음과 같이 하자:
        부모의 <처음 만들어질 때>('메시지')을 실행하자

일단 해보자:
    새로운 [내오류]("커스텀 실패")를 던지자
[내오류]가 발생했다면 ('에러'):
    틀"잡힘: {'에러'의 '메시지'}"를 출력하자
오류가 발생했다면 ('에러'):
    "여기 오면 버그"를 출력하자
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"잡힘: 커스텀 실패"}
	if strings.Join(vm.Output, ",") != strings.Join(want, ",") {
		t.Errorf("Output = %v, want %v", vm.Output, want)
	}
}

func TestBytecodeFinallyRunsOnNormalCompletion(t *testing.T) {
	vm, err := runBytecode(t, `
일단 해보자:
    "try"를 출력하자
마무리는 항상:
    "finally"를 출력하자
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"try", "finally"}
	if strings.Join(vm.Output, ",") != strings.Join(want, ",") {
		t.Errorf("Output = %v, want %v", vm.Output, want)
	}
}

// Runtime spec 4.2: finally must run and the error must still propagate
// even when no catch clause matches it (or there are no catch clauses at
// all) — it's not enough for finally to run only on a successful catch.
func TestBytecodeFinallyRunsThenRethrowsOnNoMatchingHandler(t *testing.T) {
	vm, err := runBytecode(t, `
일단 해보자:
    '결과'를 10 / 0으로 정하자
마무리는 항상:
    "finally"를 출력하자
`)
	if err == nil || !strings.Contains(err.Error(), "DivideByZeroError") {
		t.Fatalf("error = %v, want DivideByZeroError", err)
	}
	want := []string{"finally"}
	if strings.Join(vm.Output, ",") != strings.Join(want, ",") {
		t.Errorf("Output = %v, want %v", vm.Output, want)
	}
}

func TestBytecodeFinallyRunsBeforeReturn(t *testing.T) {
	vm, err := runBytecode(t, `
[숫자]를 돌려주는 <테스트>를 만들자 ():
    일단 해보자:
        7을 돌려주자
    마무리는 항상:
        "finally"를 출력하자

틀"결과: {<테스트>()}"를 출력하자
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"finally", "결과: 7"}
	if strings.Join(vm.Output, ",") != strings.Join(want, ",") {
		t.Errorf("Output = %v, want %v", vm.Output, want)
	}
}

func TestBytecodeFinallyRunsBeforeBreakAndLoopStillExits(t *testing.T) {
	vm, err := runBytecode(t, `
'i'를 1로 정하자
(('i'가 5보다 작다) 또는 ('i'와 5가 같다)) 인 동안 반복하자:
    일단 해보자:
        만약 ('i'가 3과 같다) 라면:
            반복을 끝내자
    마무리는 항상:
        틀"finally(i={'i'})"를 출력하자
    'i'에 1을 더하자
틀"끝: {'i'}"를 출력하자
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"finally(i=1)", "finally(i=2)", "finally(i=3)", "끝: 3"}
	if strings.Join(vm.Output, ",") != strings.Join(want, ",") {
		t.Errorf("Output = %v, want %v", vm.Output, want)
	}
}

func TestBytecodeNestedTryOuterCatchesAfterInnerFinally(t *testing.T) {
	vm, err := runBytecode(t, `
일단 해보자:
    일단 해보자:
        '결과'를 10 / 0으로 정하자
    마무리는 항상:
        "inner-finally"를 출력하자
오류가 발생했다면 ('에러'):
    틀"outer-caught: {'에러'의 '메시지'}"를 출력하자
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"inner-finally", "outer-caught: DivideByZeroError: 0으로 나눌 수 없어요."}
	if strings.Join(vm.Output, ",") != strings.Join(want, ",") {
		t.Errorf("Output = %v, want %v", vm.Output, want)
	}
}

// List push/pop on an object field, not just a plain variable — this was a
// standing 1차 restriction (ListPushStatement etc. only accepted a plain
// Identifier target) that classes/GET_MEMBER/SET_MEMBER now make possible
// to lift properly: read the current list via the same general expression
// path as any other read, write the result back via compileAssignTarget,
// same as a compound assignment.
func TestBytecodeListPushPopOnObjectField(t *testing.T) {
	vm, err := runBytecode(t, `
[장바구니]를 설계하자:
    '항목들'을 [(문자열)목록]인 []로 정하자

    <담기>를 만들자 ([문자열]인 '상품'):
        '나'의 '항목들' 뒤에 '상품'을 추가하자

    [문자열]을 돌려주는 <꺼내기>를 만들자 ():
        '나'의 '항목들' 뒤에서 꺼낸 값을 돌려주자

'장바구니1'을 [장바구니]인 새로운 [장바구니]()로 정하자
'장바구니1'의 <담기>("사과")를 실행하자
'장바구니1'의 <담기>("바나나")를 실행하자
틀"{'장바구니1'의 '항목들'}"를 출력하자
틀"{'장바구니1'의 <꺼내기>()}"를 출력하자
틀"{'장바구니1'의 '항목들'}"를 출력하자
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"[사과, 바나나]", "바나나", "[사과]"}
	if strings.Join(vm.Output, ",") != strings.Join(want, ",") {
		t.Errorf("Output = %v, want %v", vm.Output, want)
	}
}

func TestBytecodeConversionBuiltinsAlwaysAvailable(t *testing.T) {
	vm, err := runBytecode(t, `
틀"{<숫자로>("42")}"를 출력하자
틀"{<문자로>(3.5)}"를 출력하자
틀"{<코드로>("A")}"를 출력하자
틀"{<글자로>(65)}"를 출력하자
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"42", "3.5", "65", "A"}
	if strings.Join(vm.Output, ",") != strings.Join(want, ",") {
		t.Errorf("Output = %v, want %v", vm.Output, want)
	}
}

func TestBytecodeMathModuleGatedByImport(t *testing.T) {
	_, err := runBytecode(t, `틀"{<올림>(3.2)}"를 출력하자`)
	if err == nil || !strings.Contains(err.Error(), "MethodNotFoundError") {
		t.Fatalf("error = %v, want MethodNotFoundError (올림 unimported)", err)
	}

	vm, err := runBytecode(t, `
[수학]에서 <올림>을 가져오자
[수학]에서 <버림>을 가져오자
틀"{<올림>(3.2)}"를 출력하자
틀"{<버림>(3.8)}"를 출력하자
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"4", "3"}
	if strings.Join(vm.Output, ",") != strings.Join(want, ",") {
		t.Errorf("Output = %v, want %v", vm.Output, want)
	}
}

func TestBytecodeLocalFileImportFunctionAndClass(t *testing.T) {
	dir := t.TempDir()
	utilsPath := filepath.Join(dir, "utils.hr")
	if err := os.WriteFile(utilsPath, []byte(`
[숫자]를 돌려주는 <제곱>을 만들자 ([숫자]인 '값'):
    ('값' * '값')를 돌려주자

[도형]을 설계하자:
    '이름'을 [문자열]인 "도형"으로 정하자

    [문자열]을 돌려주는 <소개>를 만들자 ():
        '나'의 '이름'을 돌려주자
`), 0o644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	utilsPathEscaped := strings.ReplaceAll(utilsPath, `\`, `\\`)
	vm, err := runBytecode(t, `
"`+utilsPathEscaped+`"에서 <제곱>을 가져오자
틀"제곱: {<제곱>(7)}"를 출력하자

"`+utilsPathEscaped+`"에서 '도형'을 가져오자
'도형1'을 [도형]인 새로운 [도형]()로 정하자
틀"소개: {'도형1'의 <소개>()}"를 출력하자
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"제곱: 49", "소개: 도형"}
	if strings.Join(vm.Output, ",") != strings.Join(want, ",") {
		t.Errorf("Output = %v, want %v", vm.Output, want)
	}
}

func TestBytecodeLocalImportMissingFile(t *testing.T) {
	l := lexer.New(`"이런파일없음.hr"에서 <아무거나>를 가져오자`)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse error: %v", p.Errors())
	}
	compiler := bytecode.NewCompiler()
	compiler.Compile(prog)
	errs := compiler.Errors()
	if len(errs) == 0 || !strings.Contains(errs[0], "ImportError") {
		t.Fatalf("compile errors = %v, want an ImportError", errs)
	}
}

func TestBytecodeCircularLocalImportIsRejected(t *testing.T) {
	dir := t.TempDir()
	aPath := filepath.Join(dir, "a.hr")
	bPath := filepath.Join(dir, "b.hr")
	bPathEscaped := strings.ReplaceAll(bPath, `\`, `\\`)
	aPathEscaped := strings.ReplaceAll(aPath, `\`, `\\`)

	if err := os.WriteFile(aPath, []byte(`"`+bPathEscaped+`"에서 <함수B>를 가져오자
<함수A>을 만들자 ():
    "A"를 출력하자
`), 0o644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}
	if err := os.WriteFile(bPath, []byte(`"`+aPathEscaped+`"에서 <함수A>를 가져오자
<함수B>을 만들자 ():
    "B"를 출력하자
`), 0o644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	l := lexer.New(`"` + aPathEscaped + `"에서 <함수A>를 가져오자`)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse error: %v", p.Errors())
	}
	compiler := bytecode.NewCompiler()
	compiler.Compile(prog)
	errs := compiler.Errors()
	if len(errs) == 0 || !strings.Contains(errs[0], "circular import") {
		t.Fatalf("compile errors = %v, want a circular import error", errs)
	}
}

func TestBytecodeImportAliasResolvesNameCollision(t *testing.T) {
	dir := t.TempDir()
	pkgAPath := filepath.Join(dir, "pkgA.hr")
	pkgBPath := filepath.Join(dir, "pkgB.hr")
	if err := os.WriteFile(pkgAPath, []byte(`
[숫자]를 돌려주는 <계산하기>를 만들자 ([숫자]인 '값'):
    ('값' + 1)를 돌려주자
`), 0o644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}
	if err := os.WriteFile(pkgBPath, []byte(`
[숫자]를 돌려주는 <계산하기>를 만들자 ([숫자]인 '값'):
    ('값' * 2)를 돌려주자
`), 0o644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	pkgAEscaped := strings.ReplaceAll(pkgAPath, `\`, `\\`)
	pkgBEscaped := strings.ReplaceAll(pkgBPath, `\`, `\\`)
	vm, err := runBytecode(t, `
"`+pkgAEscaped+`"에서 <계산하기>를 <A계산하기>로 가져오자
"`+pkgBEscaped+`"에서 <계산하기>를 <B계산하기>로 가져오자
틀"A: {<A계산하기>(10)}"를 출력하자
틀"B: {<B계산하기>(10)}"를 출력하자
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"A: 11", "B: 20"}
	if strings.Join(vm.Output, ",") != strings.Join(want, ",") {
		t.Errorf("Output = %v, want %v", vm.Output, want)
	}
}

func TestBytecodeImportAliasForBuiltinModule(t *testing.T) {
	vm, err := runBytecode(t, `
[수학]에서 <올림>을 <반올림>으로 가져오자
틀"{<반올림>(3.2)}"를 출력하자
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"4"}
	if strings.Join(vm.Output, ",") != strings.Join(want, ",") {
		t.Errorf("Output = %v, want %v", vm.Output, want)
	}
}

// InputStatement never actually reads stdin in either engine — see
// vm/exec_stmt.go's comment ("테스트 환경에서 hang 방지"): it always binds
// the target to an empty string. Bytecode was missing this case entirely
// (compileStatement's default error), so any 입력받자 statement failed to
// compile.
func TestBytecodeInputStatementBindsEmptyStringWithoutBlocking(t *testing.T) {
	vm, err := runBytecode(t, `
'이름'을 [문자열]로 입력받자
틀"이름: '{'이름'}'"를 출력하자
"끝"을 출력하자
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"이름: ''", "끝"}
	if strings.Join(vm.Output, ",") != strings.Join(want, ",") {
		t.Errorf("Output = %v, want %v", vm.Output, want)
	}
}

// String builtin pseudo-methods (자르기/바꾸기/분리하기/포함확인) — these
// were explicitly out of scope during the class work (GET_MEMBER's string
// case just raised MethodNotFoundError for any method call), but a user
// hit 포함확인 in real code, so they're implemented now. Mirrors
// vm.BoundStringMethod's dispatch table.
func TestBytecodeStringPseudoMethods(t *testing.T) {
	vm, err := runBytecode(t, `
'문장'을 [문자열]인 "안녕하세요 세계"로 정하자
틀"{'문장'의 <자르기>(1, 2)}"를 출력하자
틀"{'문장'의 <바꾸기>("세계", "하리")}"를 출력하자
틀"{'문장'의 <분리하기>(" ")}"를 출력하자
틀"{'문장'의 <포함확인>("세계")}"를 출력하자
틀"{'문장'의 <포함확인>("없음")}"를 출력하자
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"안녕", "안녕하세요 하리", "[안녕하세요, 세계]", "참", "거짓"}
	if strings.Join(vm.Output, ",") != strings.Join(want, ",") {
		t.Errorf("Output = %v, want %v", vm.Output, want)
	}
}

func TestBytecodeStringPseudoMethodWrongArgCountErrors(t *testing.T) {
	_, err := runBytecode(t, `'문장'을 [문자열]인 "test"로 정하자
'문장'의 <포함확인>()을 실행하자`)
	if err == nil || !strings.Contains(err.Error(), "ArgumentError") {
		t.Errorf("error = %v, want ArgumentError", err)
	}
}

// TestBytecodeListClearMethod covers LIST_CLEAR (비우기), compiled via a
// compile-time-detected-but-runtime-verified pattern (see LIST_CLEAR's own
// doc comment in opcode.go) rather than through GET_MEMBER/CALL_METHOD's
// generic bound-method dispatch, since the mutation needs to write its
// result back to wherever the list came from and a runtime call value has
// no memory of that.
func TestBytecodeListClearMethod(t *testing.T) {
	vm, err := runBytecode(t, `'목록'을 ["사과", "포도"]로 정하자
'목록'의 '길이'를 출력하자
'목록'의 <비우기>()를 실행하자
'목록'의 '길이'를 출력하자
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"2", "0"}
	if strings.Join(vm.Output, ",") != strings.Join(want, ",") {
		t.Errorf("Output = %v, want %v", vm.Output, want)
	}
}

func TestBytecodeListClearMethodOnObjectField(t *testing.T) {
	vm, err := runBytecode(t, `[상자]를 설계하자:
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
	if strings.Join(vm.Output, ",") != strings.Join(want, ",") {
		t.Errorf("Output = %v, want %v", vm.Output, want)
	}
}

// A string has no <비우기>: the ordinary method call reports it, as the tree-walker does.
func TestBytecodeListClearMethodOnNonListErrors(t *testing.T) {
	_, err := runBytecode(t, `'값'을 "문자열"로 정하자
'값'의 <비우기>()를 실행하자
`)
	if err == nil || !strings.Contains(err.Error(), "MethodNotFoundError") {
		t.Errorf("error = %v, want MethodNotFoundError", err)
	}
}

// A class's own <비우기> is an ordinary method, not the list one.
func TestBytecodeClassOwnClearMethod(t *testing.T) {
	vm, err := runBytecode(t, `[통]을 설계하자:
    '양'을 3으로 정하자
    <비우기>를 만들자 ():
        '나'의 '양'을 0으로 정하자
        "비웠다"를 출력하자

'통'을 새로운 [통]()으로 정하자
'통'의 <비우기>()를 실행하자
('통'의 '양')을 출력하자
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"비웠다", "0"}
	if strings.Join(vm.Output, ",") != strings.Join(want, ",") {
		t.Errorf("Output = %v, want %v", vm.Output, want)
	}
}

// A list's <비우기> given arguments: they are evaluated, then the count is reported.
func TestBytecodeListClearMethodArgCount(t *testing.T) {
	_, err := runBytecode(t, `'목록'을 ["사과"]로 정하자
'목록'의 <비우기>(1)를 실행하자
`)
	if err == nil || !strings.Contains(err.Error(), "ArgumentError") {
		t.Errorf("error = %v, want ArgumentError", err)
	}
	_, err = runBytecode(t, `'목록'을 ["사과"]로 정하자
'목록'의 <비우기>(1 / 0)를 실행하자
`)
	if err == nil || !strings.Contains(err.Error(), "DivideByZero") {
		t.Errorf("error = %v, want DivideByZeroError", err)
	}
}

// [이름] is a variable of that name, else the class, else the name itself (a
// built-in type), as in the tree-walker; a class shows as its objects do.
func TestBytecodeTypeReferenceValue(t *testing.T) {
	vm, err := runBytecode(t, `[상자]를 설계하자:
    '값'을 1로 정하자

[참]을 출력하자
([숫자] == "숫자")를 출력하자
'문자열'을 5로 정하자
[문자열]을 출력하자
[상자]를 출력하자
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"참", "참", "5", "[상자 객체]"}
	if strings.Join(vm.Output, ",") != strings.Join(want, ",") {
		t.Errorf("Output = %v, want %v", vm.Output, want)
	}
}
