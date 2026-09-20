// Package kanade is a thin locale wrapper: it feeds a kanade lexer's token
// stream into the shared parser/haja.Parser (see LangProfile there) along
// with kanadeProfile, so Kanade(카나데) scripts produce the exact same
// *ast.Program shape as Haja scripts, ready for the tree-walker or
// bytecode compiler unchanged.
package kanade

import (
	"strings"

	kanadelexer "github.com/soumt-r/hana/lexer/kanade"
	"github.com/soumt-r/hana/parser/haja"
	"github.com/soumt-r/hana/token"
)

func New(l *kanadelexer.Lexer) *haja.Parser {
	return haja.NewFromTokens(l.Tokens, kanadeProfile)
}

var kanadeProfile = &haja.LangProfile{
	ErrorLiterals: []string{"エラー", "エラーが"},
	// 『私たち』(quoted, matching kanade-docs literally) — not "私達": see
	// hana/CLAUDE.md for how this was caught (a failing doctest run against
	// real kanade-docs content, not a guess).
	PluralSelfWords:         []string{"私たち", "『私たち』"},
	ConditionThenWords:      []string{"なら", "ならば"},
	PoppedValueWord:         "値",
	MemberParticle:          "の",
	TypeInWord:              "の", // kanade-docs reuses the member particle for this — see TYPE_IN's comment in lexer/kanade
	TemplatePrefix:          "枠「",
	TemplateSuffix:          "」",
	ConstructorFunctionName: "最初に作られる時",
	FrontMarker:             "前",
	DelimLen:                len("「"),
	TypeOpen:                "【",
	TypeClose:               "】",
	AccessModifierFromVerb: func(literal string) string {
		if strings.HasSuffix(literal, "隠そう") {
			return "private"
		}
		if strings.HasSuffix(literal, "譲ろう") {
			return "protected"
		}
		return "public"
	},
	IsConstVerb: func(literal string) bool {
		return strings.HasSuffix(literal, "固定しよう")
	},
	IsAbstractClassVerb: func(literal string) bool {
		return strings.HasPrefix(literal, "下")
	},
	IsPrintInlineVerb: func(literal string) bool {
		return strings.HasSuffix(literal, "続けて出力しよう")
	},
	// Unlike 하자 (single "반복하자" verb, kind read off the trailing
	// component), the loop kind here is fused into the verb's own literal
	// (see lexer/kanade's KW_LOOP regex) since unspaced Japanese text can't
	// carry a separate trailing marker word without it merging into the
	// verb token.
	ClassifyLoop: func(verb token.Token, components []haja.Component) haja.LoopKind {
		switch {
		case strings.HasPrefix(verb.Literal, "ごとに"):
			return haja.LoopForEach
		case strings.HasPrefix(verb.Literal, "間"):
			return haja.LoopWhile
		default:
			return haja.LoopRange
		}
	},
	NormalizeCompareOpSOV: func(op string) string {
		switch {
		case strings.Contains(op, "一種"):
			return "instanceof"
		case strings.Contains(op, "同じ") || strings.Contains(op, "等しい"):
			return "=="
		case strings.Contains(op, "大きい"):
			return ">"
		case strings.Contains(op, "小さい"):
			return "<"
		case strings.Contains(op, "以上"):
			return ">="
		case strings.Contains(op, "以下"):
			return "<="
		case strings.Contains(op, "異なる") || strings.Contains(op, "違う"):
			return "!="
		}
		return op
	},
	NormalizeCompareOpSVO: func(op string) string {
		switch {
		case op == "同じだ" || op == "同じ" || op == "等しい":
			return "=="
		case op == "異なる" || op == "違う":
			return "!="
		}
		return op
	},
	ImportAsParticles:   []string{"に"},
	ImportAllWord:       "全部",
	ImportJoinParticles: []string{"と"},
}
