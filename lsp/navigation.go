package lsp

import (
	"encoding/json"
	"unicode/utf8"

	"github.com/soumt-r/hana/token"
)

type location struct {
	URI   string   `json:"uri"`
	Range lspRange `json:"range"`
}

type symbolInformation struct {
	Name     string   `json:"name"`
	Kind     int      `json:"kind"`
	Location location `json:"location"`
}

// LSP SymbolKind values.
const (
	symClass    = 5
	symMethod   = 6
	symField    = 8
	symFunction = 12
	symVariable = 13
)

func symbolKindOf(completionKind int) int {
	switch completionKind {
	case kindClass:
		return symClass
	case kindMethod:
		return symMethod
	case kindField:
		return symField
	case kindFunction:
		return symFunction
	}
	return symVariable
}

// declaringTypes returns, for a declarable name's token type (variable,
// function or type), the keyword tokens that mark a line as declaring it; nil
// for tokens that are not names.
func declaringTypes(tokType string) map[string]bool {
	switch tokType {
	case string(token.VAR), "FUNCTION":
		return map[string]bool{string(token.KW_MAKE): true}
	case "TYPE":
		return map[string]bool{string(token.KW_CLASS): true, string(token.KW_INTERFACE): true}
	}
	return nil
}

// definitionOf finds where a name is declared: the first occurrence on a line
// that has the declaring keyword (`정하자`/`만들자` for variables and functions,
// `설계하자`/`규정하자` for types), or, failing that, its first occurrence at
// all. Tokens carry positions but the AST does not, so this walks the token
// stream — which is also exactly what the interpreter read.
func definitionOf(tokens []token.Token, name token.Token) (token.Token, bool) {
	declaring := declaringTypes(string(name.Type))
	if declaring == nil {
		return token.Token{}, false
	}
	linesWithDeclarer := map[int]bool{}
	for _, t := range tokens {
		if declaring[string(t.Type)] {
			linesWithDeclarer[t.Line] = true
		}
	}
	var first *token.Token
	for i := range tokens {
		t := tokens[i]
		if t.Type != name.Type || t.Literal != name.Literal {
			continue
		}
		if first == nil {
			first = &tokens[i]
		}
		if linesWithDeclarer[t.Line] {
			return t, true
		}
	}
	if first != nil {
		return *first, true
	}
	return token.Token{}, false
}

func (s *Server) definition(id json.RawMessage, params json.RawMessage) {
	var p struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
		Position position `json:"position"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		s.reply(id, nil, &responseError{Code: errInvalidRequest, Message: "bad definition params"})
		return
	}
	lang := languageFor(p.TextDocument.URI)
	if lang == nil {
		s.reply(id, nil, nil)
		return
	}
	text := s.docs[p.TextDocument.URI]
	tok, ok := tokenAt(lang, text, p.Position.Line, p.Position.Character)
	if !ok {
		s.reply(id, nil, nil)
		return
	}
	def, ok := definitionOf(lang.lex(text), tok)
	if !ok {
		s.reply(id, nil, nil)
		return
	}
	s.reply(id, location{URI: p.TextDocument.URI, Range: tokenRange(text, def)}, nil)
}

func tokenRange(text string, tok token.Token) lspRange {
	lines := splitLines(text)
	return rangeOfSpan(lines[tok.Line-1], tok.Line-1, tok.Col, utf8.RuneCountInString(tok.Literal))
}

func (s *Server) documentSymbol(id json.RawMessage, params json.RawMessage) {
	var p struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		s.reply(id, nil, &responseError{Code: errInvalidRequest, Message: "bad documentSymbol params"})
		return
	}
	out := []symbolInformation{}
	if lang := languageFor(p.TextDocument.URI); lang != nil {
		text := s.docs[p.TextDocument.URI]
		tokens := lang.lex(text)
		for _, sym := range lang.documentSymbols(text) {
			tokType := lang.tokenTypeForLabel(sym.label)
			for _, t := range tokens {
				if t.Literal == sym.label && string(t.Type) == tokType {
					if def, ok := definitionOf(tokens, t); ok {
						out = append(out, symbolInformation{
							Name:     sym.label,
							Kind:     symbolKindOf(sym.kind),
							Location: location{URI: p.TextDocument.URI, Range: tokenRange(text, def)},
						})
					}
					break
				}
			}
		}
	}
	s.reply(id, out, nil)
}

// tokenTypeForLabel tells which token type a symbol label lexes to, from the
// language's own delimiters.
func (l *language) tokenTypeForLabel(label string) string {
	switch {
	case hasPrefix(label, l.varDelims[0]):
		return string(token.VAR)
	case hasPrefix(label, l.funcDelims[0]):
		return "FUNCTION"
	case hasPrefix(label, l.typeDelims[0]):
		return "TYPE"
	}
	return ""
}

func hasPrefix(s, prefix string) bool { return len(s) >= len(prefix) && s[:len(prefix)] == prefix }
