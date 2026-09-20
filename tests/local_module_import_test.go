package tests

// A local file can import from standard modules, in both engines: what it brings
// in is its own, and its functions use it.

import (
	"strings"
	"testing"
)

func TestALocalFileCanUseAStandardModule(t *testing.T) {
	inTempDir(t, map[string]string{
		"lib.hj": "[수학]에서 <올림>을 가져오자\n<위로>를 만들자 ('값'):\n    <올림>('값')를 돌려주자\n",
	})
	code := "\"lib.hj\"에서 <위로>를 가져오자\n<위로>(2.1)을 출력하자\n<위로>(7.9)를 출력하자\n"
	tree, err := runHaja(t, code)
	if err != nil {
		t.Fatalf("tree-walker: %v", err)
	}
	bc, err := runBytecode(t, code)
	if err != nil {
		t.Fatalf("bytecode: %v", err)
	}
	if got := strings.Join(tree.Output, "|") + "#" + strings.Join(bc.Output, "|"); got != "3|8#3|8" {
		t.Errorf("printed %q", got)
	}
}

func TestAPackageCanUseALocalFileThatUsesAStandardModule(t *testing.T) {
	installed(t, map[string]string{
		greetPath + "@1.0.0/haja/index.hj": "\"helper.hj\"에서 <위로>를 가져오자\n<크게>를 만들자 ('값'):\n    <위로>('값')를 돌려주자\n",
	}, map[string]string{
		"hana.json":      `{"dependencies": {"` + greetPath + `": "1.0.0"}}`,
		"hana-lock.json": `{"` + greetPath + `": {"version": "1.0.0", "commit": "a"}}`,
		"helper.hj":      "[수학]에서 <올림>을 가져오자\n<위로>를 만들자 ('값'):\n    <올림>('값')를 돌려주자\n",
	})
	code := "[" + greetPath + "]에서 <크게>를 가져오자\n<크게>(1.2)를 출력하자\n"
	tree, err := runHaja(t, code)
	if err != nil {
		t.Fatalf("tree-walker: %v", err)
	}
	bc, err := runBytecode(t, code)
	if err != nil {
		t.Fatalf("bytecode: %v", err)
	}
	if got := strings.Join(tree.Output, "|") + "#" + strings.Join(bc.Output, "|"); got != "2#2" {
		t.Errorf("printed %q", got)
	}
}
