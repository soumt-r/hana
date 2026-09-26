package tests

import (
	"reflect"
	"strings"
	"testing"
)

// A 돌려주자 at the top level ends the program there, without an error (the
// tree-walker used to report "return"; the bytecode engine already ended).
// `마무리는 항상` still runs on the way out, and in an imported file it ends
// that file's top-level code: the file is loaded and its functions work.
func TestTopLevelReturnEndsTheProgram(t *testing.T) {
	c := dualRun{
		name:   "top-level return",
		hari:   "\"전\"을 출력하자\n1을 돌려주자\n\"후\"를 출력하자\n",
		kanade: "「前」を出力しよう\n1を返そう\n「後」を出力しよう\n",
	}
	outs, errs := runAllFour(t, c)
	for i, name := range []string{"hari tree", "hari bytecode", "kanade tree", "kanade bytecode"} {
		if errs[i] != nil {
			t.Fatalf("%s: %v", name, errs[i])
		}
	}
	for i, want := range [][]string{{"전"}, {"전"}, {"前"}, {"前"}} {
		if !reflect.DeepEqual(outs[i], want) {
			t.Errorf("run %d: got %q, want %q", i, outs[i], want)
		}
	}
}

func TestTopLevelReturnRunsFinally(t *testing.T) {
	code := "일단 해보자:\n    2를 돌려주자\n마무리는 항상:\n    \"마무리\"를 출력하자\n\"뒤\"를 출력하자\n"
	tree, err := runHari(t, code)
	if err != nil {
		t.Fatalf("tree-walker: %v", err)
	}
	bc, err := runBytecode(t, code)
	if err != nil {
		t.Fatalf("bytecode: %v", err)
	}
	if got := strings.Join(tree.Output, "|") + "#" + strings.Join(bc.Output, "|"); got != "마무리#마무리" {
		t.Errorf("printed %q", got)
	}
}

func TestTopLevelReturnInAnImportedFile(t *testing.T) {
	inTempDir(t, map[string]string{
		"lib.hr": "\"모듈 시작\"을 출력하자\n1을 돌려주자\n\"안 나옴\"을 출력하자\n<도움>을 만들자 ():\n    \"도움\"을 돌려주자\n",
	})
	code := "\"lib.hr\"에서 <도움>을 가져오자\n(<도움>())을 출력하자\n"
	tree, err := runHari(t, code)
	if err != nil {
		t.Fatalf("tree-walker: %v", err)
	}
	bc, err := runBytecode(t, code)
	if err != nil {
		t.Fatalf("bytecode: %v", err)
	}
	if got := strings.Join(tree.Output, "|") + "#" + strings.Join(bc.Output, "|"); got != "모듈 시작|도움#모듈 시작|도움" {
		t.Errorf("printed %q", got)
	}
}
