package lsp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The VS Code extension keeps its own TextMate keyword lists. This makes sure
// every keyword the server suggests is also highlighted, so the two cannot
// drift apart silently.
func grammarKeywords(t *testing.T, file string) map[string]bool {
	t.Helper()
	path := filepath.Join("..", "..", "vscode-haja", "syntaxes", file)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("extension sources not next to hana: %v", err)
	}
	var g struct {
		Repository struct {
			Keywords struct {
				Patterns []struct {
					Match string `json:"match"`
				} `json:"patterns"`
			} `json:"keywords"`
		} `json:"repository"`
	}
	if err := json.Unmarshal(data, &g); err != nil {
		t.Fatalf("%s is not valid JSON: %v", file, err)
	}
	if len(g.Repository.Keywords.Patterns) == 0 {
		t.Fatalf("%s has no keywords pattern", file)
	}
	match := g.Repository.Keywords.Patterns[0].Match
	match = strings.TrimSuffix(strings.TrimPrefix(match, `\b(`), `)\b`)
	match = strings.TrimSuffix(strings.TrimPrefix(match, `(`), `)`)
	set := map[string]bool{}
	for _, kw := range strings.Split(match, "|") {
		set[kw] = true
	}
	return set
}

func TestGrammarHighlightsEverySuggestedKeyword(t *testing.T) {
	for file, l := range map[string]*language{"haja.tmLanguage.json": hajaLang, "kanade.tmLanguage.json": kanadeLang} {
		have := grammarKeywords(t, file)
		for _, kw := range l.keywords {
			if !have[kw] {
				t.Errorf("%s: keyword %q is suggested by the server but not highlighted", file, kw)
			}
		}
	}
}
