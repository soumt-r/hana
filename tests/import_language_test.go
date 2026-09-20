package tests

// Imports run each module in its own language: a local file by its extension
// (.hj / .knd), a package folder by the importing file's language
// (packages/<모듈>/haja/index.hj or packages/<모듈>/kanade/index.knd).

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// inTempDir writes files (relative path -> content) into a fresh directory and
// makes it the working directory for the test, since package lookup is
// relative to it. It returns the directory.
func inTempDir(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(old) })
	return dir
}

func TestKanadeFileCanImportAKanadeFile(t *testing.T) {
	inTempDir(t, map[string]string{
		"lib.knd": "〈挨拶〉を作ろう (【文字列】の『名前』):\n    枠「こんにちは、{『名前』}！」を出力しよう\n",
	})
	main := "「lib.knd」から〈挨拶〉を持ってこよう\n〈挨拶〉(「カナデ」)を実行しよう\n"

	tree, err := runKanade(t, main)
	if err != nil {
		t.Fatalf("tree-walker: %v", err)
	}
	bc, err := runKanadeBytecode(t, main)
	if err != nil {
		t.Fatalf("bytecode: %v", err)
	}
	want := "こんにちは、カナデ！"
	if strings.Join(tree.Output, "|") != want || strings.Join(bc.Output, "|") != want {
		t.Errorf("tree-walker %v, bytecode %v, want [%s]", tree.Output, bc.Output, want)
	}
}

func TestHajaFileCanImportAKanadeFileAndViceVersa(t *testing.T) {
	inTempDir(t, map[string]string{
		"jp.knd": "〈挨拶〉を作ろう ():\n    「こんにちは」を出力しよう\n",
		"kr.hj":  "<인사>를 만들자 ():\n    \"안녕하세요\"를 출력하자\n",
	})
	hajaMain, err := runHaja(t, "\"jp.knd\"에서 <挨拶>를 가져오자\n<挨拶>()를 실행하자\n")
	if err != nil {
		t.Fatalf("haja importing kanade: %v", err)
	}
	if got := strings.Join(hajaMain.Output, "|"); got != "こんにちは" {
		t.Errorf("haja importing kanade printed %q", got)
	}
	kanadeMain, err := runKanade(t, "「kr.hj」から〈인사〉を持ってこよう\n〈인사〉()を実行しよう\n")
	if err != nil {
		t.Fatalf("kanade importing haja: %v", err)
	}
	if got := strings.Join(kanadeMain.Output, "|"); got != "안녕하세요" {
		t.Errorf("kanade importing haja printed %q", got)
	}
}

func TestImportedModuleGetsTheStandardLibraryInItsOwnLanguage(t *testing.T) {
	// Before sub-interpreters inherited the stdlib, the module below could not
	// call the conversion builtin at all.
	inTempDir(t, map[string]string{
		"conv.hj":  "<변환>를 만들자 ():\n    <문자로>(42)를 출력하자\n",
		"conv.knd": "〈変換〉を作ろう ():\n    〈文字列に〉(42)を出力しよう\n",
	})
	h, err := runHaja(t, "\"conv.hj\"에서 <변환>를 가져오자\n<변환>()를 실행하자\n")
	if err != nil {
		t.Fatalf("haja module: %v", err)
	}
	k, err := runKanade(t, "「conv.knd」から〈変換〉を持ってこよう\n〈変換〉()を実行しよう\n")
	if err != nil {
		t.Fatalf("kanade module: %v", err)
	}
	if strings.Join(h.Output, "|") != "42" || strings.Join(k.Output, "|") != "42" {
		t.Errorf("haja %v, kanade %v, want [42] each", h.Output, k.Output)
	}
}

func TestPackagePicksTheEntryPointForTheImportersLanguage(t *testing.T) {
	inTempDir(t, map[string]string{
		"packages/greet/haja/index.hj":    "<인사>를 만들자 ():\n    \"안녕하세요\"를 출력하자\n",
		"packages/greet/kanade/index.knd": "〈挨拶〉を作ろう ():\n    「こんにちは」を出力しよう\n",
	})
	h, err := runHaja(t, "[greet]에서 <인사>를 가져오자\n<인사>()를 실행하자\n")
	if err != nil {
		t.Fatalf("haja: %v", err)
	}
	k, err := runKanade(t, "【greet】から〈挨拶〉を持ってこよう\n〈挨拶〉()を実行しよう\n")
	if err != nil {
		t.Fatalf("kanade: %v", err)
	}
	if strings.Join(h.Output, "|") != "안녕하세요" || strings.Join(k.Output, "|") != "こんにちは" {
		t.Errorf("haja %v, kanade %v", h.Output, k.Output)
	}
}

func TestPackageWithoutAnEntryForTheLanguageIsAnError(t *testing.T) {
	inTempDir(t, map[string]string{
		"packages/onlyko/haja/index.hj": "<인사>를 만들자 ():\n    \"안녕하세요\"를 출력하자\n",
	})
	_, err := runKanade(t, "【onlyko】から〈인사〉を持ってこよう\n")
	if err == nil {
		t.Fatal("importing a package that has no kanade entry point must fail")
	}
	requireErrorContains(t, err, "no 'kanade' entry point")
}

func TestImportedFileWithASyntaxErrorIsReported(t *testing.T) {
	inTempDir(t, map[string]string{"broken.hj": "\"안녕\"을 출력하자 )\n"})
	_, err := runHaja(t, "\"broken.hj\"에서 <인사>를 가져오자\n")
	requireErrorContains(t, err, "Syntax error in file")
}

func TestPackageEntryWithASyntaxErrorIsReported(t *testing.T) {
	inTempDir(t, map[string]string{"packages/broken/haja/index.hj": "\"안녕\"을 출력하자 )\n"})
	_, err := runHaja(t, "[broken]에서 <인사>를 가져오자\n")
	requireErrorContains(t, err, "of package 'broken'")
}
