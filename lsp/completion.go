package lsp

import (
	"encoding/json"

	"github.com/soumt-r/hana/ast"
)

// LSP CompletionItemKind values.
const (
	kindMethod   = 2
	kindFunction = 3
	kindField    = 5
	kindVariable = 6
	kindClass    = 7
	kindKeyword  = 14
)

// symbol is one name the document declares; completion lists it and hover
// describes it. typeNote is preformatted in the document's language ("" if the
// declaration names no type).
type symbol struct {
	label    string
	kind     int
	detail   string
	typeNote string
}

type completionItem struct {
	Label  string `json:"label"`
	Kind   int    `json:"kind"`
	Detail string `json:"detail,omitempty"`
}

// completionsFor lists everything the user could type at a bare position:
// the language's keywords and builtin types plus every name the document
// itself declares. The client filters by what has been typed so far, so no
// position analysis is needed (and the results stay valid mid-edit).
func completionsFor(lang *language, text string) []completionItem {
	var items []completionItem
	seen := map[string]bool{}
	add := func(sym symbol) {
		key := sym.label + "\x00" + sym.detail
		if seen[key] {
			return
		}
		seen[key] = true
		items = append(items, completionItem{Label: sym.label, Kind: sym.kind, Detail: sym.detail})
	}

	for _, list := range [][]string{lang.keywords, lang.comparisons, lang.words} {
		for _, kw := range list {
			add(symbol{label: kw, kind: kindKeyword})
		}
	}
	for _, t := range lang.builtinTypes {
		add(symbol{label: lang.wrap(lang.typeDelims, t), kind: kindClass, detail: lang.detailBuiltinType})
	}

	for _, sym := range lang.documentSymbols(text) {
		add(sym)
	}
	return items
}

// documentSymbols returns the names declared anywhere in text.
func (l *language) documentSymbols(text string) []symbol {
	prog, _ := l.parse(text)
	if prog == nil {
		return nil
	}
	var syms []symbol
	l.declarations(prog.Statements, false, func(s symbol) { syms = append(syms, s) })
	return syms
}

func (l *language) typeNote(label string, t *ast.TypeReference) string {
	if t == nil {
		return ""
	}
	return label + ": " + l.wrap(l.typeDelims, t.Name)
}

func (l *language) wrap(d [2]string, name string) string { return d[0] + name + d[1] }

// declarations walks statements collecting declared names. Inside a class body
// variables are fields and functions are methods.
func (l *language) declarations(stmts []ast.Statement, inClass bool, add func(symbol)) {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *ast.VariableDeclaration:
			if s.Name == nil {
				continue
			}
			kind, detail := kindVariable, l.detailVariable
			if inClass {
				kind, detail = kindField, l.detailField
			}
			add(symbol{l.wrap(l.varDelims, s.Name.Value), kind, detail, l.typeNote(l.noteType, s.TypeRef)})
		case *ast.FunctionDeclaration:
			if s.Name == nil {
				continue
			}
			kind, detail := kindFunction, l.detailFunction
			if inClass {
				kind, detail = kindMethod, l.detailMethod
			}
			add(symbol{l.wrap(l.funcDelims, s.Name.Value), kind, detail, l.typeNote(l.noteReturn, s.ReturnType)})
			l.parameters(s.Params, add)
			if s.Body != nil {
				l.declarations(s.Body.Statements, false, add)
			}
		case *ast.ClassDeclaration:
			if s.Name != nil {
				add(symbol{label: l.wrap(l.typeDelims, s.Name.Name), kind: kindClass, detail: l.detailClass})
			}
			l.declarations(s.Body, true, add)
		case *ast.ConstructorDeclaration:
			l.parameters(s.Params, add)
			l.declarations(s.Body, false, add)
		case *ast.InterfaceDeclaration:
			if s.Name != nil {
				add(symbol{label: l.wrap(l.typeDelims, s.Name.Name), kind: kindClass, detail: l.detailClass})
			}
		}
	}
}

func (l *language) parameters(params []*ast.Parameter, add func(symbol)) {
	for _, p := range params {
		if p != nil && p.Name != nil {
			add(symbol{l.wrap(l.varDelims, p.Name.Value), kindVariable, l.detailVariable, l.typeNote(l.noteType, p.TypeAnnotation)})
		}
	}
}

func (s *Server) completion(id json.RawMessage, params json.RawMessage) {
	var p struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		s.reply(id, nil, &responseError{Code: errInvalidRequest, Message: "bad completion params"})
		return
	}
	lang := languageFor(p.TextDocument.URI)
	if lang == nil {
		s.reply(id, []completionItem{}, nil)
		return
	}
	items := completionsFor(lang, s.docs[p.TextDocument.URI])
	if items == nil {
		items = []completionItem{}
	}
	s.reply(id, items, nil)
}
