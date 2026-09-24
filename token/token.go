package token

type TokenType string

type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Col     int // 0-based column in characters (runes), for editor diagnostics
}

const (
	EOF = "EOF"

	// ILLEGAL is a character no lexer rule accepts. It stays in the stream (rather
	// than being skipped) so the parser reports it as an unexpected token.
	ILLEGAL = "ILLEGAL"

	IDENT  = "IDENT"
	INT    = "INT"
	STRING = "STRING"
	VAR    = "VAR"

	COLON  = ":"
	ASSIGN = "=" // 함수 파라미터 기본값 구분자 (예: '이름' = "손님")

	LBRACKET = "["
	RBRACKET = "]"
	LPAREN   = "("
	RPAREN   = ")"
	LBRACE   = "{"
	RBRACE   = "}"
	COMMA    = ","

	KW_PRINT       = "KW_PRINT"      // 출력하자
	KW_INPUT       = "KW_INPUT"      // 입력받자
	KW_MAKE        = "KW_MAKE"       // 만들자, 정하자
	KW_INTERFACE   = "KW_INTERFACE"  // 규정하자
	KW_CLASS       = "KW_CLASS"      // 설계하자
	KW_IMPLEMENTS  = "KW_IMPLEMENTS" // 따르는
	KW_IF          = "KW_IF"         // 만약
	KW_ELSE        = "KW_ELSE"       // 그렇지 않다면
	KW_CONSTRUCT   = "KW_CONSTRUCT"
	KW_DO_AS       = "KW_DO_AS"
	KW_BASE        = "KW_BASE"
	KW_GETTER      = "KW_GETTER"
	KW_SETTER      = "KW_SETTER"
	KW_FRONT       = "KW_FRONT"
	KW_BACK        = "KW_BACK"
	KW_ADD         = "KW_ADD"
	KW_SUB         = "KW_SUB"
	KW_EXECUTE     = "KW_EXECUTE"   // 실행하자
	KW_MUST_HAVE   = "KW_MUST_HAVE" // 있어야 한다
	KW_NULL        = "KW_NULL"      // 비어있음
	KW_RETURN      = "KW_RETURN"    // 돌려주자
	KW_BREAK       = "KW_BREAK"     // 반복을 끝내자
	KW_LOOP        = "KW_LOOP"      // 반복하자
	KW_PUSH        = "KW_PUSH"      // 추가하자
	KW_POP         = "KW_POP"       // 꺼내자
	KW_POPPED      = "KW_POPPED"    // 꺼낸 (표현식 형태: '목록' 뒤에서 꺼낸 값)
	KW_TRY         = "KW_TRY"       // 일단 해보자
	KW_CATCH       = "KW_CATCH"     // 발생했다면
	KW_FINALLY     = "KW_FINALLY"   // 마무리는 항상
	KW_THROW       = "KW_THROW"     // 던지자
	KW_IMPORT      = "KW_IMPORT"
	KW_FROM        = "KW_FROM"
	KW_TRUE        = "KW_TRUE"
	KW_FALSE       = "KW_FALSE"
	KW_PARENT      = "KW_PARENT"
	KW_OUTER       = "KW_OUTER"
	KW_NEW         = "KW_NEW"
	KW_SELF        = "KW_SELF"
	KW_SWITCH      = "KW_SWITCH"
	KW_CASE        = "KW_CASE"
	KW_DEFAULT     = "KW_DEFAULT"
	KW_FALLTHROUGH = "KW_FALLTHROUGH"
	KW_AND         = "KW_AND" // 그리고
	KW_OR          = "KW_OR"  // 또는

	// KW_ELIF: an "else if" fused into one word (카나데: "もしくは"). 하자
	// has no equivalent — it chains else-if by writing KW_ELSE immediately
	// followed by a separate KW_IF token ("그렇지 않고 만약"). Added
	// alongside parser/hari's parseIf support for it.
	KW_ELIF = "KW_ELIF"
)
