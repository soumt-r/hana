// Package magic names the operator methods a class can declare (spec 3.5):
// `A + B` with an object on the left whose class (or an ancestor) has
// `<기호 더하기>` runs `A의 <기호 더하기>(B)`. Both engines (vm, bcvm) read
// these tables; the browser engines and Haru have the same words.
//
// `==` and `!=` share one method, keyed "==": `!=` turns a true/false answer
// around, and an answer of 비어있음 (the method returned nothing) means "not
// equal".
//
// The arithmetic operators also have a method for an object on the right,
// keyed "r" + the operator ("r+"): `3 * B`, when the left has no `*` method,
// runs `B의 <기호 오른쪽 곱하기>(3)`. Comparisons have none (`3 < B` is
// `B > 3`).
package magic

// Hari is the method name for each operator, by the operator's symbol.
var Hari = map[string]string{
	"==": "기호 같다",
	"+":  "기호 더하기",
	"-":  "기호 빼기",
	"*":  "기호 곱하기",
	"/":  "기호 나누기",
	"%":  "기호 나머지",
	">":  "기호 크다",
	"<":  "기호 작다",
	">=": "기호 이상",
	"<=": "기호 이하",
	"r+": "기호 오른쪽 더하기",
	"r-": "기호 오른쪽 빼기",
	"r*": "기호 오른쪽 곱하기",
	"r/": "기호 오른쪽 나누기",
	"r%": "기호 오른쪽 나머지",
}

// Kanade is Hari's table in Kanade's words.
var Kanade = map[string]string{
	"==": "記号 同じだ",
	"+":  "記号 足す",
	"-":  "記号 引く",
	"*":  "記号 掛ける",
	"/":  "記号 割る",
	"%":  "記号 余り",
	">":  "記号 大きい",
	"<":  "記号 小さい",
	">=": "記号 以上",
	"<=": "記号 以下",
	"r+": "記号 右から足す",
	"r-": "記号 右から引く",
	"r*": "記号 右から掛ける",
	"r/": "記号 右から割る",
	"r%": "記号 右から余り",
}
