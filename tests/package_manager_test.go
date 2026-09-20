package tests

// Importing packages that `hana add` installed: [github.com/owner/repo] finds the
// version hana-lock.json names in the cache (or the folder hana.json replaces it
// with), in both languages and on both engines. The downloading itself is in
// pkg/install_test.go; here the cache is laid out by hand.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/soumt-r/hana/errs"
)

const (
	greetPath = "example.test/owner/greet"
)

// installed lays out a cache holding the given package files
// ("<path>@<version>/<file>") and returns the project folder, which is the working
// directory of the test. The project's own files come from project.
func installed(t *testing.T, cache, project map[string]string) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HANA_HOME", home)
	for name, content := range cache {
		path := filepath.Join(home, "pkg", filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return inTempDir(t, project)
}

func TestInstalledPackageImportsInBothLanguagesAndEngines(t *testing.T) {
	installed(t, map[string]string{
		greetPath + "@1.0.0/hana.pkg.json":    `{"name": "` + greetPath + `"}`,
		greetPath + "@1.0.0/haja/index.hj":    "<인사하기>를 만들자 ('이름'):\n    '글'을 틀\"안녕, {'이름'}!\"으로 정하자\n    '글'을 돌려주자\n",
		greetPath + "@1.0.0/kanade/index.knd": "〈挨拶〉を作ろう(『名前』):\n    『文』を枠「こんにちは、{『名前』}！」にしよう\n    『文』を返そう\n",
	}, map[string]string{
		"hana.json":      `{"dependencies": {"` + greetPath + `": "1.0.0"}}`,
		"hana-lock.json": `{"` + greetPath + `": {"version": "1.0.0", "commit": "abc"}}`,
	})

	haja := "[" + greetPath + "]에서 <인사하기>를 가져오자\n<인사하기>(\"하나\")를 출력하자\n"
	kanade := "【" + greetPath + "】から〈挨拶〉を持ってこよう\n〈挨拶〉(「カナデ」)を出力しよう\n"

	tree, err := runHaja(t, haja)
	if err != nil {
		t.Fatalf("haja tree-walker: %v", err)
	}
	bc, err := runBytecode(t, haja)
	if err != nil {
		t.Fatalf("haja bytecode: %v", err)
	}
	if got := strings.Join(tree.Output, "|") + "#" + strings.Join(bc.Output, "|"); got != "안녕, 하나!#안녕, 하나!" {
		t.Errorf("haja printed %q", got)
	}

	tree2, err := runKanade(t, kanade)
	if err != nil {
		t.Fatalf("kanade tree-walker: %v", err)
	}
	bc2, err := runKanadeBytecode(t, kanade)
	if err != nil {
		t.Fatalf("kanade bytecode: %v", err)
	}
	if got := strings.Join(tree2.Output, "|") + "#" + strings.Join(bc2.Output, "|"); got != "こんにちは、カナデ！#こんにちは、カナデ！" {
		t.Errorf("kanade printed %q", got)
	}
}

func TestOnlyOneLanguageEntryIsRequired(t *testing.T) {
	installed(t, map[string]string{
		greetPath + "@1.0.0/haja/index.hj": "<인사하기>를 만들자 ('이름'):\n    '이름'을 돌려주자\n",
	}, map[string]string{
		"hana.json":      `{"dependencies": {"` + greetPath + `": "1.0.0"}}`,
		"hana-lock.json": `{"` + greetPath + `": {"version": "1.0.0", "commit": "a"}}`,
	})
	_, err := runKanade(t, "【"+greetPath+"】から〈인사하기〉を持ってこよう\n")
	requireErrorContains(t, err, "has no 'kanade' entry point")
}

func TestReplaceUsesALocalFolderAndTheLockIsNotNeeded(t *testing.T) {
	dir := installed(t, nil, map[string]string{
		"hana.json":                    `{"dependencies": {"` + greetPath + `": "1.0.0"}, "replace": {"` + greetPath + `": "../local-greet"}}`,
		"../local-greet/haja/index.hj": "<인사하기>를 만들자 ():\n    \"로컬\"을 돌려주자\n",
	})
	_ = dir
	interp, err := runHaja(t, "["+greetPath+"]에서 <인사하기>를 가져오자\n<인사하기>()를 출력하자\n")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(interp.Output, "|"); got != "로컬" {
		t.Errorf("printed %q", got)
	}
}

func TestAPackageThatIsNotInstalledSaysSo(t *testing.T) {
	installed(t, nil, map[string]string{
		"hana.json": `{"dependencies": {"` + greetPath + `": "1.0.0"}}`,
	})
	code := "[" + greetPath + "]에서 <인사하기>를 가져오자\n"
	_, err := runHaja(t, code)
	requireErrorContains(t, err, "is not installed")
	_, err = runBytecode(t, code)
	requireErrorContains(t, err, "is not installed")
	if got := errs.Message(errs.Korean, errs.New(errs.ImportPackageNotInstalled, greetPath)); !strings.Contains(got, "hana install") {
		t.Errorf("the message does not say what to do: %q", got)
	}
}
