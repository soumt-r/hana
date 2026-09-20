// Package kanade is a lexer for 카나데(Kanade), the planned Japanese-syntax
// locale of Haja. It produces the exact same token.TokenType stream as
// lexer/haja for the same grammar (see parser/haja's LangProfile), just
// matched against Japanese vocabulary and Japanese full-width punctuation
// instead of Korean/ASCII. The punctuation and vocabulary here are meant to
// match kanade-docs' actual published examples byte-for-byte — see
// hana/CLAUDE.md for the story of why (an earlier ASCII-punctuation version
// of this file didn't match any real kanade-docs content at all).
package kanade

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/soumt-r/hana/token"
)

type Lexer struct {
	Tokens []token.Token
	pos    int
}

func New(input string) *Lexer {
	l := &Lexer{
		Tokens: []token.Token{},
		pos:    0,
	}
	l.tokenize(input)
	return l
}

func (l *Lexer) NextToken() token.Token {
	if l.pos >= len(l.Tokens) {
		return token.Token{Type: token.EOF, Literal: ""}
	}
	tok := l.Tokens[l.pos]
	l.pos++
	return tok
}

// jaWord matches an identifier made of Korean/Latin/digit characters (kept
// for mixed-script identifiers) plus hiragana/katakana/kanji. It must not
// include any of the full-width delimiter characters below (「」『』【】〈〉),
// which it doesn't — none of them fall in the ぁ-んァ-ヶ一-龯ー ranges.
const jaWord = `[가-힣a-zA-Zぁ-んァ-ヶ一-龯ー_][가-힣a-zA-Zぁ-んァ-ヶ一-龯ー0-9_]*`

var indentPattern = regexp.MustCompile(`^[ 	]*`)

// Compiled once: the patterns are the same for every line and every lexer.
var tokenSpecs = []struct {
	kind  token.TokenType
	regex *regexp.Regexp
}{
	// TEMPLATE_STRING must be tried before STRING/bare-枠 so
	// "枠「...」" isn't lexed as a bare IDENT "枠" + STRING.
	{"TEMPLATE_STRING", regexp.MustCompile(`^枠「(?:\{[^{}]*\}|\\[\s\S]|[^」\\{])*」`)},
	{token.STRING, regexp.MustCompile(`^「(?:\\[\s\S]|[^」\\])*」`)},
	{token.VAR, regexp.MustCompile(`^『` + jaWord + `』`)},
	{"FUNCTION", regexp.MustCompile(`^〈[^〉]+〉`)},
	// TYPE: 【(genericArgs)Name】 — generic arg list keeps ASCII
	// parens (kanade-docs: 【(文字列)リスト】, 【(文字列,数字)辞書】).
	{"TYPE", regexp.MustCompile(`^【(?:\([^)]+\))?` + jaWord + `】`)},
	// List/type brackets share 【】 in kanade-docs (a list literal
	// is 【要素,要素】, not a distinct ASCII-bracket form) — TYPE
	// above only matches when what's inside looks like a type
	// name, so a literal like 【「りんご」,「バナナ」】 correctly
	// falls through to bare LBRACKET here instead.
	{token.LBRACKET, regexp.MustCompile(`^【`)},
	{token.RBRACKET, regexp.MustCompile(`^】`)},
	{token.LPAREN, regexp.MustCompile(`^\(`)},
	{token.RPAREN, regexp.MustCompile(`^\)`)},
	{token.LBRACE, regexp.MustCompile(`^\{`)},
	{token.RBRACE, regexp.MustCompile(`^\}`)},
	{token.COMMA, regexp.MustCompile(`^[,、]`)},
	{token.COLON, regexp.MustCompile(`^:`)},
	{token.KW_RETURN, regexp.MustCompile(`^返そう`)},
	{token.KW_BREAK, regexp.MustCompile(`^繰り返しを終わろう`)},
	// Loop kind is fused into the verb itself (not a trailing
	// marker word like 하자's "동안"): unspaced Japanese text
	// means a bare marker word immediately before a bare loop
	// verb would just get swallowed into one long generic
	// IDENT token, so the lexer has to commit to a loop kind
	// at the verb's own regex, matching kanade-docs' own forms.
	{token.KW_LOOP, regexp.MustCompile(`^ごとに繰り返そう|^間繰り返そう|^まで繰り返そう`)},
	{token.KW_PUSH, regexp.MustCompile(`^追加しよう`)},
	// KW_FRONT/KW_BACK include a "…から" form (前から/後ろから/
	// 後から) fused as one token: kanade-docs writes the popped-
	// value-expression form as "の前から取り出した値" — with
	// "から" split off as a separate KW_FROM token, the parser's
	// (FRONT|BACK) + peek(1)==POPPED lookahead would see KW_FROM
	// in between and miss it.
	{token.KW_POP, regexp.MustCompile(`^取り出そう`)},
	{token.KW_POPPED, regexp.MustCompile(`^取り出した`)},
	{token.KW_TRY, regexp.MustCompile(`^とりあえずやってみよう`)},
	{token.KW_CATCH, regexp.MustCompile(`^発生したら|^発生したなら`)},
	{token.KW_FINALLY, regexp.MustCompile(`^最後はいつも|^締めくくりはいつも`)},
	{token.KW_THROW, regexp.MustCompile(`^発生させよう`)},
	{token.KW_CLASS, regexp.MustCompile(`^設計しよう|^下設計しよう`)},
	{token.KW_INTERFACE, regexp.MustCompile(`^規定しよう`)},
	{token.KW_MUST_HAVE, regexp.MustCompile(`^なければならない`)},
	{token.KW_IMPORT, regexp.MustCompile(`^持ってこよう`)},
	{token.KW_FROM, regexp.MustCompile(`^から`)},
	{token.KW_IMPLEMENTS, regexp.MustCompile(`^従う`)},
	// もしくは(なら(ば)?) doubles as "else if" (KW_ELIF) — see
	// parser/haja's KW_ELIF handling; それ以外なら(ば)? is plain
	// else. Must be checked before bare KW_IF ("もし"), which is
	// a prefix of "もしくは" and would otherwise steal the first
	// two characters and leave "くは" as a dangling identifier.
	{token.KW_ELIF, regexp.MustCompile(`^もしくは`)},
	{token.KW_IF, regexp.MustCompile(`^もし`)},
	{token.KW_ELSE, regexp.MustCompile(`^それ以外ならば|^それ以外なら|^それ以外で`)},
	{token.IDENT, regexp.MustCompile(`^ならば|^なら`)},
	// "値" (PoppedValueWord, the optional readability word after
	// "前/後から取り出した") needs its own early match: unspaced
	// Japanese means "値にしよう" would otherwise fall through to
	// the generic IDENT catch-all below and get swallowed whole
	// (value+verb as one token) before parser/haja ever gets a
	// chance to recognize "値" as its own word and stop there.
	// Same root cause as the もしくは/もし ordering fix above.
	{token.IDENT, regexp.MustCompile(`^値`)},
	// "全部" (ImportAllWord: `【数学】から全部持ってこよう`) for the same reason: it is
	// followed directly by the verb.
	{token.IDENT, regexp.MustCompile(`^全部`)},
	{token.KW_MAKE, regexp.MustCompile(`^作って隠そう|^作って譲ろう|^作ろう|^にしよう|^固定しよう|^隠そう|^譲ろう|^準備しよう`)},
	{token.KW_PRINT, regexp.MustCompile(`^続けて出力しよう|^出力しよう`)},
	{token.KW_INPUT, regexp.MustCompile(`^入力させよう|^入力してもらおう`)},
	{token.KW_EXECUTE, regexp.MustCompile(`^実行しよう`)},
	{token.KW_NULL, regexp.MustCompile(`^空っぽ`)},
	{token.KW_TRUE, regexp.MustCompile(`^真`)},
	{token.KW_FALSE, regexp.MustCompile(`^偽`)},
	{token.KW_CONSTRUCT, regexp.MustCompile(`^最初に作られる時`)},
	{token.KW_DO_AS, regexp.MustCompile(`^次のようにしよう`)},
	{token.KW_BASE, regexp.MustCompile(`^基づいて|^もとにして`)},
	{token.KW_GETTER, regexp.MustCompile(`^取得する時`)},
	{token.KW_SETTER, regexp.MustCompile(`^決める時`)},
	{token.KW_PARENT, regexp.MustCompile(`^親`)},
	{token.KW_OUTER, regexp.MustCompile(`^外`)},
	{token.KW_NEW, regexp.MustCompile(`^新しい`)},
	{token.KW_SWITCH, regexp.MustCompile(`^によって分けよう`)},
	{token.KW_CASE, regexp.MustCompile(`^の場合`)},
	{token.KW_DEFAULT, regexp.MustCompile(`^残りは`)},
	{token.KW_FALLTHROUGH, regexp.MustCompile(`^次に続けよう`)},
	{token.KW_AND, regexp.MustCompile(`^かつ|^そして`)},
	// "もしくは" is KW_ELIF only (checked earlier, above) —
	// not repeated here since it would be unreachable dead code.
	{token.KW_OR, regexp.MustCompile(`^または`)},
	{token.KW_SELF, regexp.MustCompile(`^私`)},
	{token.KW_FRONT, regexp.MustCompile(`^前から|^前で|^前に|^前`)},
	{token.KW_BACK, regexp.MustCompile(`^後ろから|^後ろで|^後ろに|^後から|^後で|^後に|^後`)},
	{token.KW_ADD, regexp.MustCompile(`^足そう`)},
	{token.KW_SUB, regexp.MustCompile(`^引こう`)},
	{"COMPARE", regexp.MustCompile(`^(==|!=|<=|>=|<|>|と同じだ|と同じ|と等しい|より大きい|より小さい|以上だ|以下だ|以上|以下|の一種だ|の一種|一種だ|一種|同じだ|同じ|等しい|異なる|違う|小さい|大きい)`)},
	{token.ASSIGN, regexp.MustCompile(`^=`)},
	{"OP", regexp.MustCompile(`^[+\-*/%]`)},
	// TYPE_IN has no dedicated regex/token here: kanade-docs
	// reuses "の" for it (【数字】の0), the same word as member
	// access. parser/haja's TYPE_IN checks already fall back to
	// comparing a PARTICLE token's Literal against
	// LangProfile.TypeInWord, so a bare PARTICLE "の" is enough
	// — no ambiguity in practice since the TYPE-branch check
	// runs (and consumes it) before member-access ever sees it.
	{"PARTICLE", regexp.MustCompile(`^(を|に|で|は|が|の|と|も|から|へ|より|くらい|まで|ずつ|など|番目)`)},
	{token.INT, regexp.MustCompile(`^\d+(?:\.\d+)?`)},
	{token.IDENT, regexp.MustCompile(`^` + jaWord)},
	{"SPACE", regexp.MustCompile(`^[ \t　]+`)},
}

func (l *Lexer) tokenize(input string) {
	lines := strings.Split(input, "\n")
	indents := []int{0}
	lineNum := 1

	for _, line := range lines {
		// kanade-docs' actual comment marker is "(参考...)" — either
		// "(参考)" (rest of line is prose) or "(参考:...)" (colon then note).
		if idx := strings.Index(line, "(参考)"); idx != -1 {
			line = line[:idx]
		}
		if idx := strings.Index(line, "(参考:"); idx != -1 {
			line = line[:idx]
		}

		if strings.TrimSpace(line) == "" {
			lineNum++
			continue
		}

		indentMatch := regexp.MustCompile(`^[ \t]*`).FindString(line)
		currentIndent := len(indentMatch)

		if currentIndent > indents[len(indents)-1] {
			indents = append(indents, currentIndent)
			l.Tokens = append(l.Tokens, token.Token{Type: "INDENT", Line: lineNum})
		} else if currentIndent < indents[len(indents)-1] {
			for currentIndent < indents[len(indents)-1] {
				indents = indents[:len(indents)-1]
				l.Tokens = append(l.Tokens, token.Token{Type: "DEDENT", Line: lineNum})
			}
		}

		remaining := strings.TrimSpace(line)
		col := currentIndent

		for len(remaining) > 0 {
			matched := false
			for _, spec := range tokenSpecs {
				loc := spec.regex.FindStringIndex(remaining)
				if loc != nil && loc[0] == 0 {
					val := remaining[:loc[1]]
					if spec.kind != "SPACE" {
						l.Tokens = append(l.Tokens, token.Token{Type: spec.kind, Literal: val, Line: lineNum, Col: col})
					}
					remaining = remaining[loc[1]:]
					col += utf8.RuneCountInString(val)
					matched = true
					break
				}
			}

			if !matched {
				r, size := utf8.DecodeRuneInString(remaining)
				l.Tokens = append(l.Tokens, token.Token{Type: token.ILLEGAL, Literal: string(r), Line: lineNum, Col: col})
				remaining = remaining[size:]
				col++
			}
		}
		lineNum++
	}

	for len(indents) > 1 {
		indents = indents[:len(indents)-1]
		l.Tokens = append(l.Tokens, token.Token{Type: "DEDENT", Line: lineNum})
	}

	l.Tokens = append(l.Tokens, token.Token{Type: token.EOF, Line: lineNum})
}
