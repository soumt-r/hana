package lsp

import (
	"encoding/json"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/soumt-r/hana/token"
)

type hoverResult struct {
	Contents markupContent `json:"contents"`
	Range    lspRange      `json:"range"`
}

type markupContent struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

// hoverAt describes the token under an LSP position (0-based line, UTF-16
// character), or returns nil when there is nothing worth showing. It works from
// the lexer's tokens, so it needs no per-language regexes and always agrees with
// what the interpreter would read at that spot.
func hoverAt(lang *language, text string, line, character int) *hoverResult {
	tok, ok := tokenAt(lang, text, line, character)
	if !ok {
		return nil
	}
	body := lang.describe(tok, text)
	if body == "" {
		return nil
	}
	lines := splitLines(text)
	return &hoverResult{
		Contents: markupContent{Kind: "markdown", Value: body},
		Range:    rangeOfSpan(lines[line], line, tok.Col, utf8.RuneCountInString(tok.Literal)),
	}
}

// tokenAt finds the lexer token under an LSP position (0-based line, UTF-16
// character).
func tokenAt(lang *language, text string, line, character int) (token.Token, bool) {
	lines := splitLines(text)
	if line < 0 || line >= len(lines) {
		return token.Token{}, false
	}
	col := runeIndexOfUTF16(lines[line], character)
	for _, tok := range lang.lex(text) {
		width := utf8.RuneCountInString(tok.Literal)
		if tok.Line == line+1 && width > 0 && col >= tok.Col && col < tok.Col+width {
			return tok, true
		}
	}
	return token.Token{}, false
}

// describe renders the hover markdown for one token.
func (l *language) describe(tok token.Token, text string) string {
	typ := string(tok.Type)
	switch {
	case strings.HasPrefix(typ, "KW_"):
		if doc, ok := l.keywordDocs[typ]; ok {
			return "**" + tok.Literal + "**\n\n" + doc
		}
	case typ == string(token.VAR):
		return l.describeSymbol(tok.Literal, text)
	case typ == "FUNCTION":
		return l.describeSymbol(tok.Literal, text)
	case typ == "TYPE":
		return l.describeType(tok.Literal, text)
	}
	return ""
}

func (l *language) describeSymbol(literal, text string) string {
	for _, sym := range l.documentSymbols(text) {
		if sym.label != literal {
			continue
		}
		out := sym.detail + " **" + literal + "**"
		if sym.typeNote != "" {
			out += "\n\n" + sym.typeNote
		}
		return out
	}
	return ""
}

func (l *language) describeType(literal, text string) string {
	// Declared classes win over builtins of the same name.
	for _, sym := range l.documentSymbols(text) {
		if sym.label == literal && sym.detail == l.detailClass {
			return sym.detail + " **" + literal + "**"
		}
	}
	name := strings.TrimSuffix(strings.TrimPrefix(literal, l.typeDelims[0]), l.typeDelims[1])
	// Generic form [(숫자)목록]: the type name follows the argument list.
	if strings.HasPrefix(name, "(") {
		if i := strings.Index(name, ")"); i >= 0 {
			name = name[i+1:]
		}
	}
	if doc, ok := l.builtinTypeDocs[name]; ok {
		return "**" + literal + "**\n\n" + doc
	}
	return ""
}

// runeIndexOfUTF16 converts an LSP character offset (UTF-16 units) into an
// index in runes; offsets past the end map to the end of the line.
func runeIndexOfUTF16(line string, character int) int {
	units, idx := 0, 0
	for _, r := range line {
		if units >= character {
			break
		}
		units += len(utf16.Encode([]rune{r}))
		idx++
	}
	return idx
}

func rangeOfSpan(line string, lineNo, col, width int) lspRange {
	runes := []rune(line)
	col = min(col, len(runes))
	end := min(col+width, len(runes))
	return lspRange{
		Start: position{lineNo, utf16Len(runes[:col])},
		End:   position{lineNo, utf16Len(runes[:end])},
	}
}

func (s *Server) hover(id json.RawMessage, params json.RawMessage) {
	var p struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
		Position position `json:"position"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		s.reply(id, nil, &responseError{Code: errInvalidRequest, Message: "bad hover params"})
		return
	}
	lang := languageFor(p.TextDocument.URI)
	if lang == nil {
		s.reply(id, nil, nil)
		return
	}
	res := hoverAt(lang, s.docs[p.TextDocument.URI], p.Position.Line, p.Position.Character)
	if res == nil {
		s.reply(id, nil, nil)
		return
	}
	s.reply(id, res, nil)
}
