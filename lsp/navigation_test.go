package lsp

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf16"
)

const navSample = "[숫자]를 돌려주는 <더하기>를 만들자 ([숫자]인 '가', [숫자]인 '나'):\n" +
	"    '결과값'을 [숫자]인 '가' + '나' 로 정하자\n" +
	"    '결과값'을 돌려주자\n" +
	"'값'을 <더하기>(1, 2)로 정하자\n" +
	"'값'을 출력하자\n"

// at returns the (line, UTF-16 column) of the nth (0-based) occurrence of needle.
func at(t *testing.T, text, needle string, nth int) (int, int) {
	t.Helper()
	idx := -1
	for i := 0; i <= nth; i++ {
		next := strings.Index(text[idx+1:], needle)
		if next < 0 {
			t.Fatalf("%q occurrence %d not found", needle, nth)
		}
		idx += 1 + next
	}
	before := text[:idx]
	line := strings.Count(before, "\n")
	lineStart := strings.LastIndex(before, "\n") + 1
	col := len(utf16.Encode([]rune(before[lineStart:])))
	return line, col
}

func definitionAt(t *testing.T, lang *language, text string, line, col int) *location {
	t.Helper()
	uri := "file:///x." + map[string]string{"haja": "hj", "kanade": "knd"}[lang.id]
	req := func(method string, id int, extra string) string {
		return `{"jsonrpc":"2.0","id":` + string(rune('0'+id)) + `,"method":"` + method + `","params":{"textDocument":{"uri":"` + uri + `"}` + extra + `}}`
	}
	open, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "method": "textDocument/didOpen",
		"params": map[string]any{"textDocument": map[string]any{"uri": uri, "text": text}},
	})
	pos, _ := json.Marshal(position{line, col})
	got, _ := session(t, initMsg, string(open), req("textDocument/definition", 5, `,"position":`+string(pos)), exitMsg)
	for _, m := range got {
		if string(m.ID) != "5" {
			continue
		}
		if m.Result == nil {
			return nil
		}
		raw, _ := json.Marshal(m.Result)
		var loc location
		if err := json.Unmarshal(raw, &loc); err != nil || loc.URI == "" {
			return nil
		}
		return &loc
	}
	t.Fatal("no reply to definition")
	return nil
}

func TestDefinitionOfAVariableUseIsItsDeclarationLine(t *testing.T) {
	line, col := at(t, navSample, "'결과값'", 1) // the use on line 2
	loc := definitionAt(t, hajaLang, navSample, line, col+2)
	if loc == nil || loc.Range.Start.Line != 1 || loc.Range.Start.Character != 4 {
		t.Errorf("definition = %+v, want line 1 col 4", loc)
	}
	line, col = at(t, navSample, "'값'", 1)
	loc = definitionAt(t, hajaLang, navSample, line, col+1)
	if loc == nil || loc.Range.Start.Line != 3 || loc.Range.Start.Character != 0 {
		t.Errorf("'값' definition = %+v, want line 3 col 0", loc)
	}
}

func TestDefinitionOfAFunctionCallIsItsMakeLine(t *testing.T) {
	line, col := at(t, navSample, "<더하기>", 1)
	loc := definitionAt(t, hajaLang, navSample, line, col+2)
	if loc == nil || loc.Range.Start.Line != 0 {
		t.Errorf("definition = %+v, want line 0", loc)
	}
}

func TestDefinitionOfAParameterIsTheSignature(t *testing.T) {
	line, col := at(t, navSample, "'가'", 1) // used in the body
	loc := definitionAt(t, hajaLang, navSample, line, col+1)
	if loc == nil || loc.Range.Start.Line != 0 {
		t.Errorf("definition = %+v, want the parameter on line 0", loc)
	}
}

func TestDefinitionOfATypeIsItsClassLine(t *testing.T) {
	src := "[개]를 설계하자:\n    '이름'을 \"멍\"으로 정하자\n'나'를 새로운 [개]()로 정하자\n"
	line, col := at(t, src, "[개]", 1)
	loc := definitionAt(t, hajaLang, src, line, col+1)
	if loc == nil || loc.Range.Start.Line != 0 {
		t.Errorf("definition = %+v, want line 0", loc)
	}
}

func TestDefinitionWorksForKanade(t *testing.T) {
	src := "『年齢』を3にしよう\n『年齢』を出力しよう\n"
	line, col := at(t, src, "『年齢』", 1)
	loc := definitionAt(t, kanadeLang, src, line, col+1)
	if loc == nil || loc.Range.Start.Line != 0 || loc.Range.Start.Character != 0 {
		t.Errorf("definition = %+v, want line 0 col 0", loc)
	}
}

func TestNoDefinitionForKeywordsOrNothing(t *testing.T) {
	line, col := at(t, navSample, "출력하자", 0)
	if loc := definitionAt(t, hajaLang, navSample, line, col+1); loc != nil {
		t.Errorf("a keyword has no definition, got %+v", loc)
	}
	if loc := definitionAt(t, hajaLang, navSample, 40, 0); loc != nil {
		t.Errorf("past the end has no definition, got %+v", loc)
	}
}

func TestDocumentSymbolsListDeclarationsWithPositions(t *testing.T) {
	open, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "method": "textDocument/didOpen",
		"params": map[string]any{"textDocument": map[string]any{"uri": "file:///s.hj", "text": navSample}},
	})
	req := `{"jsonrpc":"2.0","id":8,"method":"textDocument/documentSymbol","params":{"textDocument":{"uri":"file:///s.hj"}}}`
	got, _ := session(t, initMsg, string(open), req, exitMsg)
	for _, m := range got {
		if string(m.ID) != "8" {
			continue
		}
		raw, _ := json.Marshal(m.Result)
		var syms []symbolInformation
		if err := json.Unmarshal(raw, &syms); err != nil {
			t.Fatal(err)
		}
		byName := map[string]symbolInformation{}
		for _, s := range syms {
			byName[s.Name] = s
		}
		if s, ok := byName["<더하기>"]; !ok || s.Kind != symFunction || s.Location.Range.Start.Line != 0 {
			t.Errorf("function symbol = %+v", byName["<더하기>"])
		}
		if s, ok := byName["'값'"]; !ok || s.Kind != symVariable || s.Location.Range.Start.Line != 3 {
			t.Errorf("variable symbol = %+v", byName["'값'"])
		}
		return
	}
	t.Fatal("no reply to documentSymbol")
}
