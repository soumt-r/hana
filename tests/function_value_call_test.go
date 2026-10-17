package tests

import (
	"strings"
	"testing"
)

// `<함수>(…)` where '함수' is a variable holding a function of the program (a
// callback passed as an argument) calls that function, after the program's
// functions of that name, as vm/eval_expr.go's FunctionReference does. The
// bytecode engine used to look only at functions by name.
func TestCallingAFunctionValueByItsVariable(t *testing.T) {
	code := `<두배>를 만들자 ('x'):
    ('x' * 2)를 돌려주자
<세배>를 만들자 ('x'):
    ('x' * 3)를 돌려주자
<두번적용>을 만들자 ('함수', '값'):
    (<함수>(<함수>('값')))를 돌려주자
<고른것>을 만들자 ('두배'):
    (<두배>(5))를 돌려주자
(<두번적용>(<두배>, 3))을 출력하자
(<두번적용>(<세배>, 1))을 출력하자
(<고른것>(<세배>))을 출력하자
`
	want := "12,9,10"
	tree, err := runHari(t, code)
	if err != nil {
		t.Fatalf("tree: %v", err)
	}
	bc, err := runBytecode(t, code)
	if err != nil {
		t.Fatalf("bytecode: %v", err)
	}
	if got := strings.Join(tree.Output, ","); got != want {
		t.Errorf("tree: %s, want %s", got, want)
	}
	if got := strings.Join(bc.Output, ","); got != want {
		t.Errorf("bytecode: %s, want %s", got, want)
	}
}

// A variable named like a built-in hides it: after `'문자로'를 5로 정하자`,
// `<문자로>(1)` finds the number, which is no function. The tree-walker keeps
// built-ins among the global variables; the bytecode engine used to call the
// built-in anyway.
func TestAVariableHidesABuiltInOfItsName(t *testing.T) {
	code := `<그냥>을 만들자 ('n'):
    (<문자로>('n'))를 돌려주자
(<그냥>(7))을 출력하자
'문자로'를 5로 정하자
일단 해보자:
    (<그냥>(1))을 출력하자
오류가 발생했다면 ('e'):
    ('e'의 '메시지')를 출력하자
`
	for engine, r := range engines(t, code) {
		if r.err != nil {
			t.Fatalf("%s: %v", engine, r.err)
		}
		if got, want := strings.Join(r.out, "|"), "7|MethodNotFoundError: '문자로' 함수를 찾을 수 없어요."; got != want {
			t.Errorf("%s: got %q, want %q", engine, got, want)
		}
	}
}
