package hari

import (
	"strings"

	"github.com/soumt-r/hana/ast"
	"github.com/soumt-r/hana/token"
)

// LoopKind classifies which loop AST node a KW_LOOP verb should become.
type LoopKind int

const (
	LoopForEach LoopKind = iota
	LoopWhile
	LoopRange
)

// LangProfile carries the handful of language-specific literals and
// verb-classification rules that parseGenericSOV (and a few other spots)
// need but that aren't already captured by a dedicated token.TokenType.
// The lexer maps surface text to the same abstract token types for every
// locale; LangProfile is what lets one Parser implementation stay correct
// across locales for the small number of places that still inspect a
// token's raw Literal (see listPosition's comment for why that raw text is
// never allowed to leak into the AST itself).
//
// hariProfile below is the locale hana was originally written against;
// other locales (e.g. kanade) construct their own profile and parse tokens
// produced by their own lexer via NewFromTokens.
type LangProfile struct {
	// ErrorLiterals: IDENT literals accepted as an untyped catch-clause
	// marker before KW_CATCH (하자: "오류"/"오류가").
	ErrorLiterals []string

	// PluralSelfWords: literals that refer to the static/class-level
	// self-reference (하자: "우리", "'우리'").
	PluralSelfWords []string

	// ConditionThenWords: trailing IDENT literals that close a condition
	// expression and are discarded (하자: "라면").
	ConditionThenWords []string

	// PoppedValueWord: optional readability IDENT after a front/back-popped
	// expression (하자: "값").
	PoppedValueWord string

	// MemberParticle: the PARTICLE literal that introduces member access
	// (하자: "의").
	MemberParticle string

	// TypeInWord: literal fallback recognized as a TYPE_IN marker when the
	// lexer didn't already tag it as TYPE_IN (하자: "인").
	TypeInWord string

	// TemplatePrefix/TemplateSuffix strip a TEMPLATE_STRING token's literal
	// down to its raw content (하자: `틀"` ... `"`).
	TemplatePrefix string
	TemplateSuffix string

	// ConstructorFunctionName: the FUNCTION literal that denotes a
	// constructor, mapped internally to "__init__" (하자: "처음 만들어질 때").
	ConstructorFunctionName string

	// FrontMarker: substring listPosition looks for in a component's first
	// particle to decide "front" vs "back" (하자: "앞").
	FrontMarker string

	// AccessModifierFromVerb classifies a KW_MAKE/function-declaration verb
	// literal into "public"/"private"/"protected".
	AccessModifierFromVerb func(literal string) string

	// IsConstVerb reports whether a KW_MAKE verb literal declares a constant.
	IsConstVerb func(literal string) bool

	// IsAbstractClassVerb reports whether a KW_CLASS verb literal declares
	// an abstract class (하자: 밑설계하자).
	IsAbstractClassVerb func(literal string) bool

	// IsPrintInlineVerb reports whether a KW_PRINT verb literal means
	// "print without a trailing newline".
	IsPrintInlineVerb func(literal string) bool

	// ClassifyLoop inspects a KW_LOOP verb token plus the parsed SOV
	// components and decides which loop AST node to build.
	ClassifyLoop func(verb token.Token, components []Component) LoopKind

	// NormalizeCompareOpSOV/SVO translate a COMPARE token's raw literal
	// into the BinaryExpression operator symbol, for parseConditionOperand's
	// two word-order branches (SOV: "A B 크다" / SVO: "A 크다 B"). These are
	// two separate rule sets, not one shared function, because that's what
	// 하자 itself already does (SVO only recognizes 같다/다르다/같지 않다,
	// SOV recognizes the full comparison vocabulary) — kept as-is rather
	// than unified, to not change 하자's observable behavior here.
	NormalizeCompareOpSOV func(op string) string
	NormalizeCompareOpSVO func(op string) string

	// ImportAsParticles: particles marking the alias component of
	// `<이름>을 <별칭>으로 가져오자` (하자: "로"/"으로").
	ImportAsParticles []string

	// ImportAllWord is the word that imports a whole module (`[모듈]에서 전부
	// 가져오자`), and ImportJoinParticles join the items of an import list
	// (`<a>와 <b>를 가져오자`).
	ImportAllWord       string
	ImportJoinParticles []string

	// DelimLen: byte length of one side of the STRING/VAR/TYPE/FUNCTION
	// delimiter pair (하자: `"`/`'`/`[`/`<` are all 1 byte; 카나데's
	// full-width「』【〈 etc. are all 3 bytes in UTF-8). Every site that
	// does `literal[1:len(literal)-1]` to strip a token's surrounding
	// delimiters uses this instead, so it doesn't assume 1-byte quotes.
	// Only correct because 하자's four delimiter pairs happen to share one
	// byte length and 카나데's four happen to share another — if a locale
	// ever needed asymmetric ones this would need to split per-kind.
	DelimLen int

	// TypeOpen/TypeClose: the TYPE token's own bracket characters, used
	// where a TYPE literal is checked/stripped by exact prefix/suffix
	// rather than by DelimLen alone (하자: "["/"]").
	TypeOpen  string
	TypeClose string
}

func literalIn(literal string, options []string) bool {
	for _, opt := range options {
		if literal == opt {
			return true
		}
	}
	return false
}

var hariProfile = &LangProfile{
	ErrorLiterals:           []string{"오류", "오류가"},
	PluralSelfWords:         []string{"우리", "'우리'"},
	ConditionThenWords:      []string{"라면"},
	PoppedValueWord:         "값",
	MemberParticle:          "의",
	TypeInWord:              "인",
	TemplatePrefix:          "틀\"",
	TemplateSuffix:          "\"",
	ConstructorFunctionName: "처음 만들어질 때",
	FrontMarker:             "앞",
	AccessModifierFromVerb: func(literal string) string {
		if strings.HasSuffix(literal, "숨기자") {
			return "private"
		}
		if strings.HasSuffix(literal, "물려주자") {
			return "protected"
		}
		return "public"
	},
	IsConstVerb: func(literal string) bool {
		return strings.HasSuffix(literal, "고정하자")
	},
	IsAbstractClassVerb: func(literal string) bool {
		return strings.HasPrefix(literal, "밑")
	},
	IsPrintInlineVerb: func(literal string) bool {
		return strings.HasSuffix(literal, "이어출력하자")
	},
	ClassifyLoop: func(verb token.Token, components []Component) LoopKind {
		if len(components) == 1 {
			return LoopForEach
		}
		if len(components) >= 2 {
			if id, ok := components[len(components)-1].Expr.(*ast.Identifier); ok && (id.Value == "동안" || id.Value == "동안은") {
				return LoopWhile
			}
		}
		return LoopRange
	},
	NormalizeCompareOpSOV: func(op string) string {
		switch {
		case strings.Contains(op, "일종이다"):
			return "instanceof"
		case strings.Contains(op, "같다") && strings.Contains(op, "!"):
			return "!="
		case strings.Contains(op, "같다"):
			return "=="
		case strings.Contains(op, "크다"):
			return ">"
		case strings.Contains(op, "작다"):
			return "<"
		case strings.Contains(op, "이상이다"):
			return ">="
		case strings.Contains(op, "이하이다"):
			return "<="
		case strings.Contains(op, "다르다") || strings.Contains(op, "않다"):
			return "!="
		}
		return op
	},
	NormalizeCompareOpSVO: func(op string) string {
		switch {
		case op == "같다":
			return "=="
		case op == "다르다" || op == "같지 않다":
			return "!="
		}
		return op
	},
	ImportAsParticles:   []string{"로", "으로"},
	ImportAllWord:       "전부",
	ImportJoinParticles: []string{"와", "과"},
	DelimLen:            1,
	TypeOpen:            "[",
	TypeClose:           "]",
}
