package tests

// Module state: what a module declares at its top level (variables, and the code that
// sets them up) belongs to the module. Its functions see and change those variables,
// the module is set up once however many times it is imported, and the importer's own
// variables are not what a module's functions see. Both engines agree.

import (
	"strings"
	"testing"
)

func TestModuleFunctionsSeeTheModulesOwnVariables(t *testing.T) {
	inTempDir(t, map[string]string{
		"lib.hj": "'배수'를 [숫자]인 3으로 정하자\n<곱하기>를 만들자 ('값'):\n    ('값' * '배수')를 돌려주자\n",
	})
	// the importer has a variable of the same name: the module's function still sees its own
	code := "\"lib.hj\"에서 <곱하기>를 가져오자\n'배수'를 [숫자]인 100으로 정하자\n<곱하기>(2)를 출력하자\n'배수'를 출력하자\n"
	tree, err := runHaja(t, code)
	if err != nil {
		t.Fatalf("tree-walker: %v", err)
	}
	bc, err := runBytecode(t, code)
	if err != nil {
		t.Fatalf("bytecode: %v", err)
	}
	if got := strings.Join(tree.Output, "|") + "#" + strings.Join(bc.Output, "|"); got != "6|100#6|100" {
		t.Errorf("printed %q", got)
	}
}

func TestAModuleIsSetUpOnceAndItsStateIsShared(t *testing.T) {
	inTempDir(t, map[string]string{
		"counter.hj": "\"카운터를 준비해요\"를 출력하자\n'횟수'를 [숫자]인 0으로 정하자\n<세기>를 만들자 ():\n    '횟수'에 1을 더하자\n    '횟수'를 돌려주자\n",
		"mid.hj":     "\"counter.hj\"에서 <세기>를 가져오자\n<중간세기>를 만들자 ():\n    <세기>()를 돌려주자\n",
	})
	code := "\"counter.hj\"에서 <세기>를 가져오자\n\"mid.hj\"에서 <중간세기>를 가져오자\n<세기>()를 출력하자\n<중간세기>()를 출력하자\n<세기>()를 출력하자\n"
	tree, err := runHaja(t, code)
	if err != nil {
		t.Fatalf("tree-walker: %v", err)
	}
	bc, err := runBytecode(t, code)
	if err != nil {
		t.Fatalf("bytecode: %v", err)
	}
	want := "카운터를 준비해요|1|2|3"
	if got := strings.Join(tree.Output, "|"); got != want {
		t.Errorf("tree-walker printed %q, want %q", got, want)
	}
	if got := strings.Join(bc.Output, "|"); got != want {
		t.Errorf("bytecode printed %q, want %q", got, want)
	}
}

func TestAModuleDoesNotSeeTheImportersVariables(t *testing.T) {
	inTempDir(t, map[string]string{
		"lib.hj": "<보기>를 만들자 ():\n    '내것'을 출력하자\n",
	})
	code := "\"lib.hj\"에서 <보기>를 가져오자\n'내것'을 [숫자]인 1로 정하자\n<보기>()를 실행하자\n"
	_, err := runHaja(t, code)
	requireErrorContains(t, err, "ReferenceError")
	_, err = runBytecode(t, code)
	requireErrorContains(t, err, "ReferenceError")
}

func TestAPackageKeepsItsStateToo(t *testing.T) {
	installed(t, map[string]string{
		greetPath + "@1.0.0/haja/index.hj": "'인사말'을 [문자열]인 \"안녕\"으로 정하자\n<말하기>를 만들자 ('이름'):\n    틀\"{'인사말'}, {'이름'}!\"을 돌려주자\n<바꾸기>를 만들자 ('새말'):\n    '인사말'을 '새말'로 정하자\n",
	}, map[string]string{
		"hana.json":      `{"dependencies": {"` + greetPath + `": "1.0.0"}}`,
		"hana-lock.json": `{"` + greetPath + `": {"version": "1.0.0", "commit": "a"}}`,
	})
	code := "[" + greetPath + "]에서 <말하기>를 가져오자\n[" + greetPath + "]에서 <바꾸기>를 가져오자\n<말하기>(\"하나\")를 출력하자\n<바꾸기>(\"반가워\")를 실행하자\n<말하기>(\"하나\")를 출력하자\n"
	tree, err := runHaja(t, code)
	if err != nil {
		t.Fatalf("tree-walker: %v", err)
	}
	bc, err := runBytecode(t, code)
	if err != nil {
		t.Fatalf("bytecode: %v", err)
	}
	want := "안녕, 하나!|반가워, 하나!"
	if got := strings.Join(tree.Output, "|"); got != want {
		t.Errorf("tree-walker printed %q, want %q", got, want)
	}
	if got := strings.Join(bc.Output, "|"); got != want {
		t.Errorf("bytecode printed %q, want %q", got, want)
	}
}
