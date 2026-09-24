package vm

import (
	"github.com/soumt-r/hana/typecheck"
	"strings"

	"github.com/soumt-r/hana/ast"
	"github.com/soumt-r/hana/errs"
	hari_lexer "github.com/soumt-r/hana/lexer/hari"
	kanade_lexer "github.com/soumt-r/hana/lexer/kanade"
	hari_parser "github.com/soumt-r/hana/parser/hari"
	kanade_parser "github.com/soumt-r/hana/parser/kanade"
	"github.com/soumt-r/hana/symbol"
)

type LangConfig struct {
	// Name is the language's directory name inside a package ("hari",
	// "kanade"), SourceExt its source file extension. A package provides one
	// entry point per language at <package>/<Name>/index<SourceExt>.
	Name      string
	SourceExt string

	// Types names the built-in types: declared-type checks (Runtime spec 2.2)
	// and `입력받자` use them.
	Types typecheck.Names

	// ParseProgram lexes and parses a whole source file in this language,
	// returning the parser's error messages (empty when it parsed cleanly).
	ParseProgram func(source string) (*ast.Program, []string)

	BuiltinToString     string
	BuiltinToNumber     string
	BuiltinToCode       string
	BuiltinToText       string
	NativePrefix        string
	DefaultItemName     string
	DefaultIndexName    string
	NullString          string
	ObjectFormat        string
	TrueString          string
	FalseString         string
	SelfWords           []string
	PluralSelfWords     []string
	selfSyms            []symbol.Symbol // SelfWords/PluralSelfWords as Symbols (see resolveSymbols)
	pluralSelfSyms      []symbol.Symbol
	BuiltinErrorClass   string
	BuiltinErrorMessage string
	BuiltinErrorCtorArg string

	// ListClearMethod is the one mutating list pseudo-method the runtime
	// spec defines (spec 2.6 — lists are otherwise manipulated via native
	// syntax, not method calls). Checked directly against a
	// BoundListMethod.FuncName in eval_expr.go's CallExpression case.
	ListClearMethod string

	// LengthWord is the property name a list/string's "길이"/length access
	// compares against in eval_expr.go's MemberExpression case (checked
	// there since, unlike a bound method name, a plain property access has
	// no dedicated call site to special-case at compile/parse time).
	LengthWord string

	// VarQuoteOpen/VarQuoteClose are the VAR token's own quote characters
	// (하자: "'"/"'", 카나데: "『"/"』"), needed at runtime for dynamic
	// reflection (spec 3.8: `<'변수'>()`) — a FUNCTION token's Name still
	// carries its *inner* VAR quoting after parsing strips the outer
	// FUNCTION delimiters (that's how eval_expr.go's FunctionReference case
	// tells "call the method named by this variable's value" apart from
	// "call the method literally named X").
	VarQuoteOpen  string
	VarQuoteClose string

	// StringSliceMethod/StringReplaceMethod/StringSplitMethod/
	// StringContainsMethod are the four string pseudo-methods (자르기/
	// 바꾸기/분리하기/포함확인), checked against a BoundStringMethod.FuncName
	// in eval_expr.go's CallExpression case — same shape as
	// ListClearMethod, just four of them instead of one.
	StringSliceMethod    string
	StringReplaceMethod  string
	StringSplitMethod    string
	StringContainsMethod string

	// Locale selects the wording errs.Localize uses wherever a runtime error
	// becomes user-visible text (a `발생했다면` handler's caught message, the
	// CLI's top-level print). Throwing code never consults it.
	Locale errs.Locale

	// EqualsMethodName is the magic method eval_expr.go's BinaryExpression
	// case looks for on a *HariObject operand of == / != (operator
	// overloading — spec 3.5's "매직 메서드"). This was hardcoded to the
	// Korean literal "기호 같다" regardless of language until a real
	// kanade-docs example (oop/1-classes.md's 〈記号 同じだ〉) was found to
	// silently never match — verified by running it through hana.exe.
	EqualsMethodName string

	// ParseEmbeddedExpr lexes+parses a `{...}` template-string interpolation's
	// inner code. It's a func, not a lexer/parser pair of strings, because
	// the interpolated code needs the locale's *full* grammar (any
	// expression is legal inside {}), not just a couple of literals — a
	// nil value keeps eval_expr.go's original 하자-only fallback so every
	// existing caller of KoreanConfig is unaffected.
	ParseEmbeddedExpr func(code string) ast.Expression
}

func stringInList(v string, list []string) bool {
	for _, item := range list {
		if v == item {
			return true
		}
	}
	return false
}

func (c *LangConfig) IsSelfWord(v string) bool       { return stringInList(v, c.SelfWords) }
func (c *LangConfig) IsPluralSelfWord(v string) bool { return stringInList(v, c.PluralSelfWords) }

// resolveSymbols works out the Symbols of the self words once, when an
// interpreter takes the config, so the evaluator can test a name by number.
func (c *LangConfig) resolveSymbols() {
	c.selfSyms = symbolsOf(c.SelfWords)
	c.pluralSelfSyms = symbolsOf(c.PluralSelfWords)
}

func symbolsOf(words []string) []symbol.Symbol {
	syms := make([]symbol.Symbol, len(words))
	for i, w := range words {
		syms[i] = symbol.Intern(w)
	}
	return syms
}

func symbolIn(s symbol.Symbol, list []symbol.Symbol) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func (c *LangConfig) IsSelfSym(s symbol.Symbol) bool       { return symbolIn(s, c.selfSyms) }
func (c *LangConfig) IsPluralSelfSym(s symbol.Symbol) bool { return symbolIn(s, c.pluralSelfSyms) }

var KoreanConfig = LangConfig{
	Name:      "hari",
	SourceExt: ".hr",
	Types:     typecheck.Names{Number: "숫자", String: "문자열", Boolean: "논리", Any: "아무거나", List: "목록", Dict: "사전", Null: "비어있음"},
	ParseProgram: func(source string) (*ast.Program, []string) {
		p := hari_parser.New(hari_lexer.New(source))
		prog := p.ParseProgram()
		return prog, p.Errors()
	},
	BuiltinToString:      "문자로",
	BuiltinToNumber:      "숫자로",
	BuiltinToCode:        "코드로",
	BuiltinToText:        "글자로",
	NativePrefix:         "네이티브_",
	DefaultItemName:      "아이템",
	DefaultIndexName:     "인덱스",
	NullString:           "비어있음",
	ObjectFormat:         "[%s 객체]",
	TrueString:           "참",
	FalseString:          "거짓",
	SelfWords:            []string{"나"},
	PluralSelfWords:      []string{"우리"},
	BuiltinErrorClass:    "오류",
	BuiltinErrorMessage:  "메시지",
	BuiltinErrorCtorArg:  "초기메시지",
	ListClearMethod:      "비우기",
	LengthWord:           "길이",
	VarQuoteOpen:         "'",
	VarQuoteClose:        "'",
	StringSliceMethod:    "자르기",
	StringReplaceMethod:  "바꾸기",
	StringSplitMethod:    "분리하기",
	StringContainsMethod: "포함확인",
	EqualsMethodName:     "기호 같다",
	Locale:               errs.Korean,
}

// JapaneseConfig is the LangConfig for 카나데(Kanade) scripts, parsed via
// parser/kanade. It must stay in sync with that package's kanadeProfile —
// LangConfig covers runtime-side (VM/stdlib) naming, kanadeProfile covers
// parse-side grammar, and both exist because they were added at different
// layers of hana (see hana/CLAUDE.md's LangConfig note).
var JapaneseConfig = LangConfig{
	Name:      "kanade",
	SourceExt: ".knd",
	Types:     typecheck.Names{Number: "数字", String: "文字列", Boolean: "論理", Any: "何でも", List: "リスト", Dict: "辞書", Null: "空っぽ"},
	ParseProgram: func(source string) (*ast.Program, []string) {
		p := kanade_parser.New(kanade_lexer.New(source))
		prog := p.ParseProgram()
		return prog, p.Errors()
	},
	// kanade-docs/src/pages/docs/tutorial/1-variables.md uses all four —
	// checked against real doc content, not guessed (BuiltinToString was
	// "文字に" and BuiltinToNumber "数に" here before, which don't match).
	BuiltinToString:     "文字列に",
	BuiltinToNumber:     "数字に",
	BuiltinToCode:       "コードに",
	BuiltinToText:       "文字に",
	NativePrefix:        "ネイティブ_",
	DefaultItemName:     "アイテム",
	DefaultIndexName:    "インデックス",
	NullString:          "空っぽ",
	ObjectFormat:        "[%s オブジェクト]",
	TrueString:          "真",
	FalseString:         "偽",
	SelfWords:           []string{"私"},
	PluralSelfWords:     []string{"私たち"},
	BuiltinErrorClass:   "エラー",
	BuiltinErrorMessage: "メッセージ",
	BuiltinErrorCtorArg: "初期メッセージ",
	ListClearMethod:     "空にする",
	LengthWord:          "長さ",
	VarQuoteOpen:        "『",
	VarQuoteClose:       "』",
	// kanade-docs/src/pages/docs/tutorial/3-strings.md's actual wording.
	StringSliceMethod:    "切り取り",
	StringReplaceMethod:  "入れ替え",
	StringSplitMethod:    "分割",
	StringContainsMethod: "含むか確認",
	// kanade-docs/src/pages/docs/oop/1-classes.md's actual method name
	// (〈記号 同じだ〉, delimiters stripped).
	EqualsMethodName: "記号 同じだ",
	Locale:           errs.Japanese,
	ParseEmbeddedExpr: func(code string) ast.Expression {
		l := kanade_lexer.New(code)
		p := kanade_parser.New(l)
		return p.ParseExpression()
	},
}

// allConfigs lists every language the runtime can load source for.
var allConfigs = []LangConfig{KoreanConfig, JapaneseConfig}

// ConfigForFile picks the language of a source file from its extension, so
// a Hari file can import a Kanade file and the other way round: the imported
// file always runs in its own language.
func ConfigForFile(filename string) LangConfig {
	for _, c := range allConfigs {
		if strings.HasSuffix(filename, c.SourceExt) {
			return c
		}
	}
	return KoreanConfig
}
