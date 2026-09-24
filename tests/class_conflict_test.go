package tests

// Two modules that declare a class of the same name cannot both be imported under it:
// the second import is an error that says so, instead of one class quietly replacing
// the other. Importing one of them under another name (<이름>을 <별칭>으로) is the way out.
// Both engines agree.

import (
	"strings"
	"testing"

	"github.com/soumt-r/hana/bytecode"
	lexer "github.com/soumt-r/hana/lexer/hari"
	parser "github.com/soumt-r/hana/parser/hari"
)

// requireCompileErrorContaining compiles code for the bytecode VM and expects the
// compiler to refuse it with a message that contains substr.
func requireCompileErrorContaining(t *testing.T, code, substr string) {
	t.Helper()
	p := parser.New(lexer.New(code))
	prog := p.ParseProgram()
	compiler := bytecode.NewCompiler()
	compiler.Compile(prog)
	if got := strings.Join(compiler.Errors(), "\n"); !strings.Contains(got, substr) {
		t.Fatalf("bytecode compiler errors %q do not contain %q", got, substr)
	}
}

const (
	boxA = "[상자]를 설계하자:\n    '뜻'을 [문자열]인 \"A상자\"로 정하자\n"
	boxB = "[상자]를 설계하자:\n    '뜻'을 [문자열]인 \"B상자\"로 정하자\n"
)

func TestTwoModulesWithTheSameClassNameConflict(t *testing.T) {
	inTempDir(t, map[string]string{"a.hr": boxA, "b.hr": boxB})
	code := "\"a.hr\"에서 '상자'를 가져오자\n\"b.hr\"에서 '상자'를 가져오자\n"
	_, err := runHari(t, code)
	requireErrorContains(t, err, "has the same name as a class of")
	requireCompileErrorContaining(t, code, "has the same name as a class of")
}

func TestAnAliasResolvesAClassNameConflict(t *testing.T) {
	inTempDir(t, map[string]string{"a.hr": boxA, "b.hr": boxB})
	code := "\"a.hr\"에서 '상자'를 가져오자\n\"b.hr\"에서 '상자'를 '다른상자'로 가져오자\n" +
		"'첫째'를 [상자]인 새로운 [상자]()로 정하자\n'둘째'를 [다른상자]인 새로운 [다른상자]()로 정하자\n" +
		"'첫째'의 '뜻'을 출력하자\n'둘째'의 '뜻'을 출력하자\n"
	tree, err := runHari(t, code)
	if err != nil {
		t.Fatalf("tree-walker: %v", err)
	}
	bc, err := runBytecode(t, code)
	if err != nil {
		t.Fatalf("bytecode: %v", err)
	}
	if got := strings.Join(tree.Output, "|") + "#" + strings.Join(bc.Output, "|"); got != "A상자|B상자#A상자|B상자" {
		t.Errorf("printed %q", got)
	}
}

func TestAModuleClassConflictsWithAClassOfTheProgram(t *testing.T) {
	inTempDir(t, map[string]string{"a.hr": boxA})
	code := "\"a.hr\"에서 '상자'를 가져오자\n" + boxB
	_, err := runHari(t, code)
	requireErrorContains(t, err, "already has")
	requireCompileErrorContaining(t, code, "already has")
}

func TestTheSameModuleImportedTwiceIsNoConflict(t *testing.T) {
	inTempDir(t, map[string]string{"a.hr": boxA + "<이름>을 만들자 ():\n    \"a\"를 돌려주자\n"})
	code := "\"a.hr\"에서 '상자'를 가져오자\n\"a.hr\"에서 <이름>을 가져오자\n<이름>()을 출력하자\n"
	if _, err := runHari(t, code); err != nil {
		t.Fatalf("tree-walker: %v", err)
	}
	if _, err := runBytecode(t, code); err != nil {
		t.Fatalf("bytecode: %v", err)
	}
}
