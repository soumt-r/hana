package cmd

import (
	"regexp"
	"strings"
	"testing"

	"github.com/soumt-r/hana/errs"
)

var verb = regexp.MustCompile(`%[sv]`)

// Every message has both languages, and both take the same number of values, so a
// translation can never print the wrong argument (or none).
func TestCatalogHasBothLanguagesWithTheSameVerbs(t *testing.T) {
	for key, m := range catalog {
		if strings.TrimSpace(m.ko) == "" || strings.TrimSpace(m.ja) == "" {
			t.Errorf("%s: a language is missing", key)
		}
		if len(verb.FindAllString(m.ko, -1)) != len(verb.FindAllString(m.ja, -1)) {
			t.Errorf("%s: the two languages take a different number of values:\n  %s\n  %s", key, m.ko, m.ja)
		}
	}
	for _, h := range help {
		if strings.TrimSpace(h.ko) == "" || strings.TrimSpace(h.ja) == "" {
			t.Errorf("the words for %q are missing a language", h.english)
		}
	}
}

func TestDetectUILocale(t *testing.T) {
	clear := func(t *testing.T) {
		for _, name := range []string{"HANA_LANG", "LC_ALL", "LC_MESSAGES", "LANG", "LANGUAGE"} {
			t.Setenv(name, "")
		}
	}
	cases := []struct {
		name string
		env  map[string]string
		args []string
		want errs.Locale
	}{
		{"the flag wins over everything", map[string]string{"HANA_LANG": "ko"}, []string{"--locale", "ja", "run", "a.hj"}, errs.Japanese},
		{"the flag with =", nil, []string{"run", "a.hj", "--locale=ja"}, errs.Japanese},
		{"the environment variable", map[string]string{"HANA_LANG": "ja"}, []string{"run", "a.hj"}, errs.Japanese},
		{"the environment variable beats the script", map[string]string{"HANA_LANG": "ko"}, []string{"run", "a.knd"}, errs.Korean},
		{"a Kanade script means Japanese", nil, []string{"run", "a.knd"}, errs.Japanese},
		{"init for Kanade means Japanese", nil, []string{"init", "dir", "--lang", "kanade"}, errs.Japanese},
		{"the system language", map[string]string{"LANG": "ja_JP.UTF-8"}, []string{"--help"}, errs.Japanese},
		{"the system language, Korean", map[string]string{"LANG": "ko_KR.UTF-8"}, []string{"--help"}, errs.Korean},
		{"LC_ALL comes before LANG", map[string]string{"LC_ALL": "ko_KR.UTF-8", "LANG": "ja_JP.UTF-8"}, []string{"--help"}, errs.Korean},
		{"a language that has no translation is ignored", map[string]string{"HANA_LANG": "fr"}, []string{"run", "a.knd"}, errs.Japanese},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			clear(t)
			for name, value := range c.env {
				t.Setenv(name, value)
			}
			if got := detectUILocale(c.args); got != c.want {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}

func TestTFallsBackToTheKeyAndFormats(t *testing.T) {
	saved := uiLocale
	defer func() { uiLocale = saved }()

	uiLocale = errs.Korean
	if got := T("build.saveFail", "x"); got != "바이트코드 저장 오류: x" {
		t.Errorf("Korean: %q", got)
	}
	uiLocale = errs.Japanese
	if got := T("build.saveFail", "x"); got != "バイトコードを保存できませんでした: x" {
		t.Errorf("Japanese: %q", got)
	}
	if got := T("no.such.key"); got != "no.such.key" {
		t.Errorf("a missing key should show itself, got %q", got)
	}
}
