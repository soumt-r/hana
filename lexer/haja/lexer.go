package haja

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

var indentPattern = regexp.MustCompile(`^[ 	]*`)

// Compiled once: the patterns are the same for every line and every lexer.
var tokenSpecs = []struct {
	kind  token.TokenType
	regex *regexp.Regexp
}{
	{token.STRING, regexp.MustCompile(`^"(?:\\[\s\S]|[^"\\])*"`)},
	{token.VAR, regexp.MustCompile(`^'[가-힣a-zA-Z0-9_]+'`)},
	{"FUNCTION", regexp.MustCompile(`^<[^>]+>`)},
	// [모듈]이나 [타입]. 대괄호 안에 git 경로(github.com/owner/repo)도 올 수 있다.
	{"TYPE", regexp.MustCompile(`^\[(?:\([^)]+\))?(?:[가-힣a-zA-Z_][가-힣a-zA-Z0-9_]*|[a-z0-9-]+(?:\.[a-z0-9-]+)+(?:/[A-Za-z0-9][A-Za-z0-9._-]*){2,})\]`)},
	{token.LBRACKET, regexp.MustCompile(`^\[`)},
	{token.RBRACKET, regexp.MustCompile(`^\]`)},
	{token.LPAREN, regexp.MustCompile(`^\(`)},
	{token.RPAREN, regexp.MustCompile(`^\)`)},
	{token.LBRACE, regexp.MustCompile(`^\{`)},
	{token.RBRACE, regexp.MustCompile(`^\}`)},
	{token.COMMA, regexp.MustCompile(`^,`)},
	{token.COLON, regexp.MustCompile(`^:`)},
	{token.KW_RETURN, regexp.MustCompile(`^돌려주자`)},
	{token.KW_BREAK, regexp.MustCompile(`^반복을 끝내자`)},
	{token.KW_LOOP, regexp.MustCompile(`^반복하자`)},
	{token.KW_PUSH, regexp.MustCompile(`^추가하자`)},
	{token.KW_POP, regexp.MustCompile(`^빼내자|^꺼내자`)},
	{token.KW_POPPED, regexp.MustCompile(`^꺼낸`)},
	{token.KW_TRY, regexp.MustCompile(`^일단 해보자`)},
	{token.KW_CATCH, regexp.MustCompile(`^발생했다면`)},
	{token.KW_FINALLY, regexp.MustCompile(`^마무리는 항상`)},
	{token.KW_THROW, regexp.MustCompile(`^던지자|^발생시키자`)},
	{token.KW_CLASS, regexp.MustCompile(`^설계하자|^밑설계하자`)},
	{token.KW_INTERFACE, regexp.MustCompile(`^규정하자`)},
	{token.KW_MUST_HAVE, regexp.MustCompile(`^있어야 한다`)},
	{token.KW_IMPORT, regexp.MustCompile(`^가져오자`)},
	{token.KW_FROM, regexp.MustCompile(`^에서`)},
	{token.KW_IMPLEMENTS, regexp.MustCompile(`^따르는`)},
	{token.KW_IF, regexp.MustCompile(`^만약`)},
	{token.KW_ELSE, regexp.MustCompile(`^그렇지 않다면|^그렇지 않고`)},
	{token.IDENT, regexp.MustCompile(`^라면`)},
	{token.KW_MAKE, regexp.MustCompile(`^만들어 숨기자|^만들어 물려주자|^만들자|^정하여 숨기자|^정하여 물려주자|^정하자|^숨기자|^고정하자|^준비하자`)},
	{token.KW_PRINT, regexp.MustCompile(`^출력하자|^이어출력하자`)},
	{token.KW_INPUT, regexp.MustCompile(`^입력받자`)},
	{token.KW_EXECUTE, regexp.MustCompile(`^실행하자`)},
	{token.KW_NULL, regexp.MustCompile(`^비어있음`)},
	{token.KW_TRUE, regexp.MustCompile(`^참`)},
	{token.KW_FALSE, regexp.MustCompile(`^거짓`)},
	{token.KW_CONSTRUCT, regexp.MustCompile(`^처음 만들어질 때`)},
	{token.KW_DO_AS, regexp.MustCompile(`^다음과 같이 하자`)},
	{token.KW_BASE, regexp.MustCompile(`^바탕으로 하고|^바탕으로`)},
	{token.KW_GETTER, regexp.MustCompile(`^가져올 때`)},
	{token.KW_SETTER, regexp.MustCompile(`^정할 때`)},
	{token.KW_PARENT, regexp.MustCompile(`^부모`)},
	{token.KW_OUTER, regexp.MustCompile(`^바깥`)},
	{token.KW_NEW, regexp.MustCompile(`^새로운`)},
	{token.KW_SWITCH, regexp.MustCompile(`^따라 나누자`)},
	{token.KW_CASE, regexp.MustCompile(`^경우`)},
	{token.KW_DEFAULT, regexp.MustCompile(`^나머지는`)},
	{token.KW_FALLTHROUGH, regexp.MustCompile(`^다음으로 이어가자`)},
	{token.KW_AND, regexp.MustCompile(`^그리고`)},
	{token.KW_OR, regexp.MustCompile(`^또는`)},
	{token.KW_SELF, regexp.MustCompile(`^나`)},
	{token.KW_FRONT, regexp.MustCompile(`^앞에(서)?`)},
	{token.KW_BACK, regexp.MustCompile(`^뒤에(서)?`)},
	{token.KW_ADD, regexp.MustCompile(`^더하자`)},
	{token.KW_SUB, regexp.MustCompile(`^빼자`)},
	{"TEMPLATE_STRING", regexp.MustCompile(`^틀"(?:\{[^{}]*\}|\\[\s\S]|[^"\\{])*"`)},
	{"COMPARE", regexp.MustCompile(`^(==|!=|<=|>=|<|>|와\s*같다|과\s*같다|보다\s*크다|보다\s*작다|이상이다|이하이다|이하다|같지 않다|같다|다르다|크다|작다|의\s*일종이다|이다)`)},
	{token.ASSIGN, regexp.MustCompile(`^=`)},
	{"OP", regexp.MustCompile(`^[+\-*/%]`)},
	{"TYPE_IN", regexp.MustCompile(`^인`)},
	{"PARTICLE", regexp.MustCompile(`^(를|을|가|이|는|은|의|와|과|로|으로|에|에서|보다|만큼|도|번째|부터|까지|마다|앞에서|뒤에서|앞에|뒤에)`)},
	{token.INT, regexp.MustCompile(`^\d+(?:\.\d+)?`)},
	{token.IDENT, regexp.MustCompile(`^[가-힣a-zA-Z_][가-힣a-zA-Z0-9_]*`)},
	{"SPACE", regexp.MustCompile(`^[ \t]+`)},
}

func (l *Lexer) tokenize(input string) {
	lines := strings.Split(input, "\n")
	indents := []int{0}
	lineNum := 1

	for _, line := range lines {
		// 인라인 주석 제거
		if idx := strings.Index(line, "(참고)"); idx != -1 {
			line = line[:idx]
		}
		if idx := strings.Index(line, "(참고:"); idx != -1 {
			line = line[:idx]
		}

		if strings.TrimSpace(line) == "" {
			lineNum++
			continue
		}

		// 들여쓰기 계산
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
			// 가장 긴 문자열부터 매칭하기 위해 정규식 배열 순회 (TS와 동일 방식 채택 - 빠르고 안전한 개발 위해)
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
				// 어느 규칙에도 안 맞는 글자: 버리지 않고 ILLEGAL 토큰으로 남겨 파서가 보고하게 한다.
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
