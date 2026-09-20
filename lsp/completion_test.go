package lsp

import (
	"encoding/json"
	"os"
	"testing"

	hajalexer "github.com/soumt-r/hana/lexer/haja"
	kanadelexer "github.com/soumt-r/hana/lexer/kanade"
)

func labels(items []completionItem) map[string]completionItem {
	m := map[string]completionItem{}
	for _, it := range items {
		m[it.Label] = it
	}
	return m
}

// Guards the keyword tables against drifting from the lexers: every suggested
// keyword must actually lex as a keyword token in its own language.
func TestKeywordsLexAsKeywords(t *testing.T) {
	first := func(l *language, kw string) (typ, literal string) {
		if l == hajaLang {
			tok := hajalexer.New(kw).Tokens[0]
			return string(tok.Type), tok.Literal
		}
		tok := kanadelexer.New(kw).Tokens[0]
		return string(tok.Type), tok.Literal
	}
	for _, l := range []*language{hajaLang, kanadeLang} {
		for _, kw := range l.keywords {
			typ, lit := first(l, kw)
			// The keyword must be exactly one keyword token: a typo such as
			// "거짓짓" would otherwise still lex (as 거짓 plus leftovers).
			if len(typ) < 3 || typ[:3] != "KW_" || lit != kw {
				t.Errorf("%s keyword %q lexes as %s %q, want a single KW_ token", l.id, kw, typ, lit)
			}
		}
	}
}

func TestHajaCompletionOffersKeywordsBuiltinTypesAndDeclarations(t *testing.T) {
	src := "'나이'를 3으로 정하자\n"
	got := labels(completionsFor(hajaLang, src))
	for _, want := range []string{"출력하자", "[숫자]", "'나이'"} {
		if _, ok := got[want]; !ok {
			t.Errorf("missing %q in %v", want, got)
		}
	}
	if got["'나이'"].Kind != kindVariable || got["[숫자]"].Kind != kindClass || got["출력하자"].Kind != kindKeyword {
		t.Errorf("wrong kinds: %+v", got)
	}
}

func TestKanadeCompletionUsesKanadeVocabularyAndDelimiters(t *testing.T) {
	got := labels(completionsFor(kanadeLang, "『年齢』を3にしよう\n"))
	for _, want := range []string{"出力しよう", "【数字】", "『年齢』"} {
		if _, ok := got[want]; !ok {
			t.Errorf("missing %q", want)
		}
	}
	for label := range got {
		if label == "출력하자" || label == "[숫자]" {
			t.Errorf("Korean entry %q leaked into Kanade completion", label)
		}
	}
	if got["『年齢』"].Detail != "変数" {
		t.Errorf("detail should be Japanese: %+v", got["『年齢』"])
	}
}

func TestClassMembersAndParametersAreOffered(t *testing.T) {
	src := "[자동차]를 설계하자:\n    '색상'을 [문자열]인 \"하양\"으로 정하자\n\n    처음 만들어질 때 ([문자열]인 '초기색상') 다음과 같이 하자:\n        '나'의 '색상'을 '초기색상'으로 정하자\n"
	got := labels(completionsFor(hajaLang, src))
	if got["[자동차]"].Kind != kindClass {
		t.Errorf("class missing: %v", got)
	}
	if got["'색상'"].Kind != kindField {
		t.Errorf("field should be Field kind: %+v", got["'색상'"])
	}
	if _, ok := got["'초기색상'"]; !ok {
		t.Errorf("constructor parameter missing")
	}
}

func TestCompletionSurvivesBrokenCode(t *testing.T) {
	got := completionsFor(hajaLang, "'a'를 정하자 )) [[ <")
	if len(got) == 0 {
		t.Error("keywords should still be offered for half-typed code")
	}
}

func TestNoDuplicateCompletionItems(t *testing.T) {
	seen := map[string]bool{}
	for _, it := range completionsFor(hajaLang, "'a'를 1로 정하자\n'a'를 2로 정하자\n") {
		k := it.Label + "|" + it.Detail
		if seen[k] {
			t.Errorf("duplicate %q", k)
		}
		seen[k] = true
	}
}

func TestCompletionRequestOverTheWire(t *testing.T) {
	open := `{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///a.knd","text":"『年齢』を3にしよう\n"}}}`
	req := `{"jsonrpc":"2.0","id":9,"method":"textDocument/completion","params":{"textDocument":{"uri":"file:///a.knd"},"position":{"line":0,"character":0}}}`
	got, _ := session(t, initMsg, open, req, exitMsg)
	for _, m := range got {
		if string(m.ID) != "9" {
			continue
		}
		raw, _ := json.Marshal(m.Result)
		var items []completionItem
		if err := json.Unmarshal(raw, &items); err != nil || len(labels(items)) == 0 {
			t.Fatalf("bad completion result %s (%v)", raw, err)
		}
		if _, ok := labels(items)["『年齢』"]; !ok {
			t.Errorf("document variable missing over the wire")
		}
		return
	}
	t.Fatal("no reply to completion request")
}

func TestCompletionForUnknownExtensionIsEmptyList(t *testing.T) {
	req := `{"jsonrpc":"2.0","id":4,"method":"textDocument/completion","params":{"textDocument":{"uri":"file:///a.txt"}}}`
	got, _ := session(t, initMsg, req, exitMsg)
	for _, m := range got {
		if string(m.ID) == "4" {
			raw, _ := json.Marshal(m.Result)
			if string(raw) != "[]" {
				t.Errorf("want [] for unknown language, got %s", raw)
			}
			return
		}
	}
	t.Fatal("no reply")
}

func TestCompletionOffersComparisonsAndGrammarWords(t *testing.T) {
	haja := labels(completionsFor(hajaLang, ""))
	for _, w := range []string{"같다", "크다", "돌려주는", "우리", "가져오자", "밑설계하자"} {
		if _, ok := haja[w]; !ok {
			t.Errorf("haja completion is missing %q", w)
		}
	}
	kanade := labels(completionsFor(kanadeLang, ""))
	for _, w := range []string{"同じだ", "値", "長さ", "なら", "私"} {
		if _, ok := kanade[w]; !ok {
			t.Errorf("kanade completion is missing %q", w)
		}
	}
}

// Each comparison word must really lex as a comparison, or the table would
// suggest text the language does not read that way.
func TestComparisonWordsLexAsComparisons(t *testing.T) {
	for _, l := range []*language{hajaLang, kanadeLang} {
		for _, w := range l.comparisons {
			toks := l.lex(w)
			if len(toks) == 0 || toks[0].Type != "COMPARE" || toks[0].Literal != w {
				t.Errorf("%s: %q does not lex as a single COMPARE token: %+v", l.id, w, toks)
			}
		}
	}
}

// The docs sites keep a generated copy of these tables for their in-browser
// editors; a stale copy would suggest different words than the language server.
func TestGeneratedTypeScriptKeywordsAreCurrent(t *testing.T) {
	for lang, rel := range map[string]string{
		"haja":   "../../haja-docs/src/utils/haja/lspKeywords.ts",
		"kanade": "../../kanade-docs/src/utils/kanade/lspKeywords.ts",
	} {
		got, err := os.ReadFile(rel)
		if err != nil {
			t.Logf("skipping %s: %v", rel, err)
			continue
		}
		if string(got) != TypeScriptKeywords(lang) {
			t.Errorf("%s is stale — run: go run ./cmd/lspgen -haja <file> -kanade <file>", rel)
		}
	}
}
