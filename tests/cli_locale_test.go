package tests

// The command line speaks Korean or Japanese: help texts, flag descriptions and the
// messages it prints. The language comes from --locale, then HANA_LANG, then the
// script (a .knd file means Japanese), then the system; these run the real binary.

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// hanaSays runs hana with the given environment additions and returns everything it
// printed. HANA_LANG is cleared first so the machine's own setting cannot leak in.
func hanaSays(t *testing.T, env []string, args ...string) string {
	t.Helper()
	hana, _ := packTools(t)
	cmd := exec.Command(hana, args...)
	cmd.Env = append(os.Environ(), "HANA_LANG=")
	cmd.Env = append(cmd.Env, env...)
	out, _ := cmd.CombinedOutput()
	return string(out)
}

func TestHelpIsInBothLanguages(t *testing.T) {
	ja := hanaSays(t, []string{"HANA_LANG=ja"}, "--help")
	for _, want := range []string{"使えるコマンド:", "ハジャ", "「hana [コマンド] --help」"} {
		if !strings.Contains(ja, want) {
			t.Errorf("Japanese help lacks %q:\n%s", want, ja)
		}
	}
	if strings.Contains(ja, "사용할 수") {
		t.Errorf("Japanese help still has Korean:\n%s", ja)
	}
	ko := hanaSays(t, []string{"HANA_LANG=ko"}, "--help")
	for _, want := range []string{"사용할 수 있는 명령:", "하자(한국어)", `"hana [명령] --help"`} {
		if !strings.Contains(ko, want) {
			t.Errorf("Korean help lacks %q:\n%s", want, ko)
		}
	}

	run := hanaSays(t, []string{"HANA_LANG=ja"}, "run", "--help")
	for _, want := range []string{"使い方:", "--allow-file", "【ファイル】モジュール", "フラグ:", "runのヘルプ"} {
		if !strings.Contains(run, want) {
			t.Errorf("Japanese `run --help` lacks %q:\n%s", want, run)
		}
	}
}

func TestTheLanguageComesFromTheFlagThenTheEnvironmentThenTheScript(t *testing.T) {
	dir := t.TempDir()
	hj := filepath.Join(dir, "a.hj")
	knd := filepath.Join(dir, "a.knd")
	os.WriteFile(hj, []byte("\"안녕\"을 출력하자\n"), 0o644)
	os.WriteFile(knd, []byte("「こんにちは」を出力しよう\n"), 0o644)

	cases := []struct {
		name string
		env  []string
		args []string
		want string
	}{
		{"a Haja script and nothing else says Korean", []string{"HANA_LANG=ko"}, []string{"run", "-t", hj}, "실행 시간"},
		{"a Kanade script means Japanese", nil, []string{"run", "-t", knd}, "実行時間"},
		{"the environment beats the script", []string{"HANA_LANG=ko"}, []string{"run", "-t", knd}, "실행 시간"},
		{"the flag beats the environment", []string{"HANA_LANG=ko"}, []string{"--locale", "ja", "run", "-t", hj}, "実行時間"},
		{"the flag after the command works too", nil, []string{"run", "-t", hj, "--locale=ja"}, "実行時間"},
	}
	for _, c := range cases {
		if out := hanaSays(t, c.env, c.args...); !strings.Contains(out, c.want) {
			t.Errorf("%s: want %q in\n%s", c.name, c.want, out)
		}
	}
}

func TestInitAndItsProblemsAreInBothLanguages(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "새프로젝트")
	if out := hanaSays(t, []string{"HANA_LANG=ja"}, "init", dir, "--lang", "kanade"); !strings.Contains(out, "次のコマンドで実行してみましょう") {
		t.Errorf("Japanese init: %s", out)
	}
	if out := hanaSays(t, []string{"HANA_LANG=ja"}, "init", t.TempDir(), "--lang", "cobol"); !strings.Contains(out, "--langはhajaかkanadeにしてください") {
		t.Errorf("Japanese refusal: %s", out)
	}
	// `init --lang kanade` alone means Japanese, when nothing else is said
	other := filepath.Join(t.TempDir(), "다른프로젝트")
	if out := hanaSays(t, nil, "init", other, "--lang", "kanade"); !strings.Contains(out, "次のコマンドで実行してみましょう") {
		t.Errorf("Kanade init without a language setting: %s", out)
	}
}
