package lsp

import (
	"strings"

	"github.com/soumt-r/hana/ast"
	"github.com/soumt-r/hana/errs"
	harilexer "github.com/soumt-r/hana/lexer/hari"
	kanadelexer "github.com/soumt-r/hana/lexer/kanade"
	hariparser "github.com/soumt-r/hana/parser/hari"
	kanadeparser "github.com/soumt-r/hana/parser/kanade"
	"github.com/soumt-r/hana/token"
)

// language bundles what differs between Hari and Kanade documents. Everything
// language-specific the server needs (parsing now; keywords and hover text in
// later steps) hangs off this so features never branch on file extensions.
type language struct {
	id    string // LSP languageId
	parse func(text string) (*ast.Program, *hariparser.Parser)

	keywords     []string // suggested as-is; lexes to a keyword token (see TestKeywordsLexAsKeywords)
	comparisons  []string // comparison words; each lexes to a COMPARE token
	words        []string // grammar words that have no token of their own (return marker, plural self, ...)
	builtinTypes []string // names between the type delimiters, e.g. 숫자 -> [숫자]

	// The delimiters that wrap a name of each kind: 'x' <f> [T] in Hari,
	// 『x』 〈f〉 【T】 in Kanade.
	varDelims, funcDelims, typeDelims [2]string

	// detail strings shown next to completion items, in the document's language.
	detailBuiltinType, detailClass, detailVariable, detailFunction, detailMethod, detailField string

	// hover text, all in the document's language
	noteType, noteReturn string            // labels for a declared type / return type
	keywordDocs          map[string]string // keyed by lexer token type, e.g. "KW_IF"
	builtinTypeDocs      map[string]string // keyed by the names in builtinTypes
	lex                  func(text string) []token.Token
	locale               errs.Locale // wording for diagnostics and messages
}

// parseWith runs the parser, turning a parser panic on half-typed code into an
// empty program: an editor sends broken text constantly and must not crash.
func parseWith(p *hariparser.Parser) (prog *ast.Program) {
	defer func() {
		if recover() != nil {
			prog = &ast.Program{}
		}
	}()
	return p.ParseProgram()
}

var hariLang = &language{
	id:     "hari",
	locale: errs.Korean,
	parse: func(text string) (*ast.Program, *hariparser.Parser) {
		p := hariparser.New(harilexer.New(text))
		return parseWith(p), p
	},
	keywords: []string{
		"만약", "그렇지 않다면", "그렇지 않고", "반복하자", "반복을 끝내자", "출력하자", "이어출력하자", "입력받자", "돌려주자", "정하자", "만들자",
		"고정하자", "설계하자", "밑설계하자", "규정하자", "가져오자", "실행하자", "일단 해보자", "발생했다면", "마무리는 항상", "던지자", "참", "거짓",
		"비어있음", "새로운", "처음 만들어질 때", "다음과 같이 하자", "따라 나누자", "경우", "나머지는", "그리고", "또는", "추가하자", "꺼내자", "나",
		"따르는", "바탕으로", "발생시키자", "숨기자", "있어야 한다",
	},
	comparisons:  []string{"같다", "다르다", "이다", "작다", "크다"},
	words:        []string{"돌려주는", "우리", "전부"},
	builtinTypes: []string{"숫자", "문자열", "논리", "아무거나", "비어있음", "목록", "사전"},
	varDelims:    [2]string{"'", "'"},
	funcDelims:   [2]string{"<", ">"},
	typeDelims:   [2]string{"[", "]"},

	detailBuiltinType: "기본 타입", detailClass: "클래스", detailVariable: "변수",
	detailFunction: "함수", detailMethod: "메서드", detailField: "필드",
	noteType: "타입", noteReturn: "반환 타입",
	keywordDocs: hariKeywordDocs, builtinTypeDocs: hariBuiltinTypeDocs,
	lex: func(text string) []token.Token { return harilexer.New(text).Tokens },
}

var kanadeLang = &language{
	id:     "kanade",
	locale: errs.Japanese,
	parse: func(text string) (*ast.Program, *hariparser.Parser) {
		p := kanadeparser.New(kanadelexer.New(text))
		return parseWith(p), p
	},
	keywords: []string{
		"もし", "もしくは", "それ以外なら", "ごとに繰り返そう", "間繰り返そう", "まで繰り返そう", "繰り返しを終わろう", "出力しよう", "続けて出力しよう",
		"入力させよう", "返そう", "作ろう", "固定しよう", "隠そう", "譲ろう", "設計しよう", "規定しよう", "持ってこよう", "実行しよう",
		"とりあえずやってみよう", "発生したら", "最後はいつも", "発生させよう", "真", "偽", "空っぽ", "新しい", "最初に作られる時", "次のようにしよう",
		"によって分けよう", "の場合", "残りは", "かつ", "または", "追加しよう", "取り出そう", "から", "そして", "なければならない", "にしよう",
		"もとにして", "前", "取得する時", "基づいて", "外", "後", "従う", "次に続けよう", "決める時", "準備しよう", "発生したなら", "私", "親",
	},
	comparisons:  []string{"同じだ", "違う"},
	words:        []string{"なら", "ならば", "値", "長さ", "私たち", "全部"},
	builtinTypes: []string{"数字", "文字列", "論理", "何でも", "リスト", "辞書"},
	varDelims:    [2]string{"『", "』"},
	funcDelims:   [2]string{"〈", "〉"},
	typeDelims:   [2]string{"【", "】"},

	detailBuiltinType: "基本型", detailClass: "クラス", detailVariable: "変数",
	detailFunction: "関数", detailMethod: "メソッド", detailField: "フィールド",
	noteType: "型", noteReturn: "戻り値の型",
	keywordDocs: kanadeKeywordDocs, builtinTypeDocs: kanadeBuiltinTypeDocs,
	lex: func(text string) []token.Token { return kanadelexer.New(text).Tokens },
}

// languageFor picks the language from the document URI; unknown extensions
// return nil and get no language features.
func languageFor(uri string) *language {
	switch {
	case strings.HasSuffix(uri, ".hr"):
		return hariLang
	case strings.HasSuffix(uri, ".knd"):
		return kanadeLang
	}
	return nil
}
