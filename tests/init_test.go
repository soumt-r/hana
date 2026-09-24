package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitMakesARunnableProject(t *testing.T) {
	hana, _ := packTools(t)
	for lang, want := range map[string]string{"hari": "안녕, 하리!\n", "kanade": "こんにちは、カナデ！\n"} {
		dir := filepath.Join(t.TempDir(), "새프로젝트")
		if out, err := exec.Command(hana, "init", dir, "--lang", lang).CombinedOutput(); err != nil {
			t.Fatalf("%s: %v\n%s", lang, err, out)
		}
		main := filepath.Join(dir, map[string]string{"hari": "main.hr", "kanade": "main.knd"}[lang])
		out, err := exec.Command(hana, "run", main).CombinedOutput()
		if err != nil || string(out) != want {
			t.Errorf("%s: running the new project gave %q (%v), want %q", lang, out, err, want)
		}
		if _, err := os.Stat(filepath.Join(dir, ".gitignore")); err != nil {
			t.Errorf("%s: no .gitignore: %v", lang, err)
		}
	}
}

func TestInitDoesNotOverwrite(t *testing.T) {
	hana, _ := packTools(t)
	dir := t.TempDir()
	mine := filepath.Join(dir, "main.hr")
	os.WriteFile(mine, []byte("\"내 코드\"를 출력하자\n"), 0o644)
	os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("내 설정\n"), 0o644)
	refuse := exec.Command(hana, "init", dir)
	refuse.Env = append(os.Environ(), "HANA_LANG=ko")
	out, err := refuse.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "이미 있어요") {
		t.Errorf("want a refusal, got %q (%v)", out, err)
	}
	if data, _ := os.ReadFile(mine); !strings.Contains(string(data), "내 코드") {
		t.Errorf("main.hr was overwritten: %q", data)
	}
	// Without a main file, an existing .gitignore is kept as it is.
	os.Remove(mine)
	if out, err := exec.Command(hana, "init", dir).CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if data, _ := os.ReadFile(filepath.Join(dir, ".gitignore")); string(data) != "내 설정\n" {
		t.Errorf(".gitignore was changed: %q", data)
	}
}

func TestInitRejectsUnknownLanguage(t *testing.T) {
	hana, _ := packTools(t)
	unknown := exec.Command(hana, "init", t.TempDir(), "--lang", "cobol")
	unknown.Env = append(os.Environ(), "HANA_LANG=ko")
	out, err := unknown.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "hari나 kanade") {
		t.Errorf("got %q (%v)", out, err)
	}
}
