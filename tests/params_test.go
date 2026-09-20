package tests

import (
	"testing"

	lexer "github.com/soumt-r/hana/lexer/haja"
	parser "github.com/soumt-r/hana/parser/haja"
	"github.com/soumt-r/hana/vm"
)

// Runtime spec 1.2: missing a required argument (no default) must raise
// MissingArgumentError, never silently leave the parameter unbound.
func TestMissingArgumentErrorOnFunctionCall(t *testing.T) {
	_, err := runHaja(t, `
<더하기>를 만들자 ([숫자]인 '가', [숫자]인 '나'):
    '가' + '나'를 돌려주자
<더하기>(1)를 실행하자
`)
	requireErrorContains(t, err, "MissingArgumentError")
}

// Runtime spec 1.2: more arguments than declared parameters must raise
// ArgumentError, never silently drop the extras.
func TestTooManyArgumentsErrorOnFunctionCall(t *testing.T) {
	_, err := runHaja(t, `
<더하기>를 만들자 ([숫자]인 '가', [숫자]인 '나'):
    '가' + '나'를 돌려주자
<더하기>(1, 2, 3)를 실행하자
`)
	requireErrorContains(t, err, "ArgumentError")
}

// A parameter with a default (`= 값`) must fall back to it when the
// argument is omitted, and accept an explicit override.
func TestDefaultParameterValue(t *testing.T) {
	interp, err := runHaja(t, `
[숫자]를 돌려주는 <더하기>를 만들자 ([숫자]인 '가', [숫자]인 '나' = 10):
    '가' + '나'를 돌려주자

'기본값사용'을 <더하기>(1)로 정하자
'값전달'을 <더하기>(1, 2)로 정하자
`)
	if err != nil {
		t.Fatalf("unexpected runtime error: %v", err)
	}
	if got, _ := interp.GlobalEnv.Get("기본값사용"); got != 11.0 {
		t.Errorf("기본값사용 = %v, want 11 (1 + default 10)", got)
	}
	if got, _ := interp.GlobalEnv.Get("값전달"); got != 3.0 {
		t.Errorf("값전달 = %v, want 3 (1 + 2, default overridden)", got)
	}
}

// Constructors go through the same bindParams path as functions/methods:
// a missing required constructor argument must also raise
// MissingArgumentError, and a constructor default must also be usable.
func TestConstructorParameterBinding(t *testing.T) {
	_, err := runHaja(t, `
[상자]를 설계하자:
    '값'을 [숫자]인 0으로 정하자
    처음 만들어질 때 ([숫자]인 '초기값') 다음과 같이 하자:
        '나'의 '값'을 '초기값'으로 정하자

'실패'를 [상자]인 새로운 [상자]()로 정하자
`)
	requireErrorContains(t, err, "MissingArgumentError")

	interp, err := runHaja(t, `
[상자]를 설계하자:
    '값'을 [숫자]인 0으로 정하자
    처음 만들어질 때 ([숫자]인 '초기값' = 99) 다음과 같이 하자:
        '나'의 '값'을 '초기값'으로 정하자

'기본'을 [상자]인 새로운 [상자]()로 정하자
'지정'을 [상자]인 새로운 [상자](5)로 정하자
'기본값'을 '기본'의 '값'으로 정하자
'지정값'을 '지정'의 '값'으로 정하자
`)
	if err != nil {
		t.Fatalf("unexpected runtime error: %v", err)
	}
	if got, _ := interp.GlobalEnv.Get("기본값"); got != 99.0 {
		t.Errorf("기본값 = %v, want 99 (constructor default)", got)
	}
	if got, _ := interp.GlobalEnv.Get("지정값"); got != 5.0 {
		t.Errorf("지정값 = %v, want 5 (constructor argument overriding default)", got)
	}
}

// Regression test for a bug where CallExpression evaluated every argument
// expression twice when the callee resolved directly to a *vm.BuiltinFunction
// value (reachable via a quoted-identifier call like '이름'(...) on a name
// bound straight to a BuiltinFunction, as opposed to <이름>(...) which goes
// through a different resolution path). An argument with a side effect must
// only run once.
func TestBuiltinFunctionCallDoesNotDoubleEvaluateArgs(t *testing.T) {
	l := lexer.New(`
'카운터'를 [숫자]인 0으로 정하자
<증가>를 만들자 ():
    '카운터'를 ('카운터' + 1)로 정하자
    '카운터'를 돌려주자

'더미'(<증가>())를 실행하자
`)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse error: %v", p.Errors())
	}

	interp := vm.NewInterpreter(prog)
	interp.RegisterBuiltin("더미", &vm.BuiltinFunction{Name: "더미", Fn: func(i *vm.Interpreter, env *vm.Environment, args ...interface{}) (interface{}, error) {
		return nil, nil
	}})
	if err := interp.Run(); err != nil {
		t.Fatalf("unexpected runtime error: %v", err)
	}

	got, ok := interp.GlobalEnv.Get("카운터")
	if !ok {
		t.Fatalf("'카운터' not found in global env")
	}
	if got != 1.0 {
		t.Errorf("카운터 = %v, want 1 (argument side effect ran more than once)", got)
	}
}
