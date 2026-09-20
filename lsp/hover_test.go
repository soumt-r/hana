package lsp

import (
	"encoding/json"
	"strings"
	"testing"
)

const hajaFunc = "[숫자]를 돌려주는 <더하기>를 만들자 ([숫자]인 '가', [숫자]인 '나'):\n    '결과값'을 [숫자]인 '가' + '나' 로 정하자\n    '결과값'을 돌려주자\n"

// hoverText returns the hover markdown at (line, character) or "" for none.
func hoverText(lang *language, text string, line, character int) string {
	if h := hoverAt(lang, text, line, character); h != nil {
		return h.Contents.Value
	}
	return ""
}

func TestEveryKeywordHasHoverDocsInItsOwnLanguage(t *testing.T) {
	for _, l := range []*language{hajaLang, kanadeLang} {
		for _, kw := range l.keywords {
			if got := hoverText(l, kw, 0, 0); got == "" || !strings.Contains(got, kw) {
				t.Errorf("%s: keyword %q has no hover docs (got %q)", l.id, kw, got)
			}
		}
		for _, typ := range l.builtinTypes {
			label := l.wrap(l.typeDelims, typ)
			if got := hoverText(l, label, 0, 1); got == "" {
				t.Errorf("%s: builtin type %q has no hover docs", l.id, label)
			}
		}
	}
}

func TestKeywordDocsOnlyUseRealTokenTypes(t *testing.T) {
	// A typo'd key would silently never match; every key must be produced by
	// some suggested keyword.
	for _, l := range []*language{hajaLang, kanadeLang} {
		used := map[string]bool{}
		for _, kw := range l.keywords {
			used[string(l.lex(kw)[0].Type)] = true
		}
		for key := range l.keywordDocs {
			if !used[key] {
				t.Errorf("%s: keywordDocs key %q matches no suggested keyword", l.id, key)
			}
		}
	}
}

func TestHoverParameterShowsDeclaredType(t *testing.T) {
	got := hoverText(hajaLang, hajaFunc, 0, 30) // on the '가' parameter
	if !strings.Contains(got, "'가'") || !strings.Contains(got, "변수") || !strings.Contains(got, "타입: [숫자]") {
		t.Errorf("hover = %q", got)
	}
}

func TestHoverLocalVariableShowsItsAnnotatedType(t *testing.T) {
	got := hoverText(hajaLang, hajaFunc, 1, 6)
	if !strings.Contains(got, "'결과값'") || !strings.Contains(got, "타입: [숫자]") {
		t.Errorf("hover = %q", got)
	}
}

func TestHoverKanadeVariableShowsItsAnnotatedType(t *testing.T) {
	got := hoverText(kanadeLang, "『名前』を【文字列】の「田中」にしよう\n", 0, 1)
	if !strings.Contains(got, "『名前』") || !strings.Contains(got, "型: 【文字列】") {
		t.Errorf("hover = %q", got)
	}
}

func TestUnannotatedVariableHasNoTypeLine(t *testing.T) {
	got := hoverText(hajaLang, "'나이'를 3으로 정하자\n", 0, 1)
	if !strings.Contains(got, "'나이'") || strings.Contains(got, "타입") {
		t.Errorf("hover = %q", got)
	}
}

func TestHoverFunctionShowsReturnType(t *testing.T) {
	got := hoverText(hajaLang, hajaFunc, 0, 12) // on <더하기>
	if !strings.Contains(got, "<더하기>") || !strings.Contains(got, "반환 타입: [숫자]") {
		t.Errorf("hover = %q", got)
	}
}

func TestHoverNothingOnPlainText(t *testing.T) {
	if got := hoverText(hajaLang, "\"안녕\"을 출력하자\n", 0, 2); got != "" {
		t.Errorf("string literal should have no hover, got %q", got)
	}
	if got := hoverText(hajaLang, "\"안녕\"을 출력하자\n", 5, 0); got != "" {
		t.Errorf("line past the end should have no hover, got %q", got)
	}
}

func TestKanadeHoverIsJapaneseAndTypeAware(t *testing.T) {
	src := "『年齢』を【数字】で3にしよう\n"
	if got := hoverText(kanadeLang, src, 0, 1); !strings.Contains(got, "変数") || !strings.Contains(got, "『年齢』") {
		t.Errorf("variable hover = %q", got)
	}
	if got := hoverText(kanadeLang, src, 0, 6); !strings.Contains(got, "数字") || !strings.Contains(got, "基本型") && !strings.Contains(got, "です") {
		t.Errorf("type hover = %q", got)
	}
	if got := hoverText(kanadeLang, "「あ」を出力しよう\n", 0, 5); !strings.Contains(got, "出力") {
		t.Errorf("keyword hover = %q", got)
	}
}

func TestHoverRangeCoversTheTokenInUTF16Units(t *testing.T) {
	// The emoji occupies two UTF-16 units, shifting everything after it.
	src := "\"😀\"를 출력하자\n"
	h := hoverAt(hajaLang, src, 0, 8)
	if h == nil {
		t.Fatal("expected hover on 출력하자")
	}
	// "😀" is chars 0-3 (quote, emoji, quote) + 를 + space: 출력하자 starts at char 5, UTF-16 6.
	if h.Range.Start.Character != 6 || h.Range.End.Character != 10 {
		t.Errorf("range = %+v, want UTF-16 columns 6-10", h.Range)
	}
	// A UTF-16 offset that lands inside the emoji pair must not shift the answer.
	if hoverAt(hajaLang, src, 0, 2) != nil {
		t.Errorf("no keyword sits on the emoji")
	}
}

func TestHoverOverTheWire(t *testing.T) {
	open := `{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///a.hj","text":"\"a\"를 출력하자\n"}}}`
	req := `{"jsonrpc":"2.0","id":5,"method":"textDocument/hover","params":{"textDocument":{"uri":"file:///a.hj"},"position":{"line":0,"character":5}}}`
	miss := `{"jsonrpc":"2.0","id":6,"method":"textDocument/hover","params":{"textDocument":{"uri":"file:///a.hj"},"position":{"line":0,"character":1}}}`
	got, _ := session(t, initMsg, open, req, miss, exitMsg)
	for _, m := range got {
		raw, _ := json.Marshal(m.Result)
		switch string(m.ID) {
		case "5":
			if !strings.Contains(string(raw), "화면에 값을") && !strings.Contains(string(raw), "출력") {
				t.Errorf("hover reply = %s", raw)
			}
		case "6":
			if string(m.ID) == "6" && string(raw) != "null" && m.Result != nil {
				t.Errorf("miss should be null, got %s", raw)
			}
		}
	}
}
