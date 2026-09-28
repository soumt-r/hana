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
