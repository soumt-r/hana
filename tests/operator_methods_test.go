package tests

import (
	"strings"
	"testing"

	"github.com/soumt-r/hana/errs"
)

// Operator overloading (spec 3.5): `A + B` with an object on the left whose
// class (or an ancestor) has `<기호 더하기>` runs `A의 <기호 더하기>(B)`, and
// likewise `- * / % > < >= <=`. The call is an ordinary method call
// (arguments and the result are checked), `'x'에 B를 더하자` goes through it,
// and a comparison's result decides a condition. Without the method the
// operator is the usual TypeError.

const operatorHari = `[벡터]를 설계하자:
    'x'를 0으로 정하자
    처음 만들어질 때 ('x') 다음과 같이 하자:
        '나'의 'x'를 'x'로 정하자
    [벡터]를 돌려주는 <기호 더하기>를 만들자 ([벡터]인 '대상'):
        새로운 [벡터](('나'의 'x' + '대상'의 'x'))를 돌려주자
    <기호 빼기>를 만들자 ('대상'):
        새로운 [벡터](('나'의 'x' - '대상'의 'x'))를 돌려주자
    <기호 곱하기>를 만들자 ('배'):
        ('나'의 'x' * '배')를 돌려주자
    <기호 나누기>를 만들자 ('배'):
        ('나'의 'x' / '배')를 돌려주자
    <기호 나머지>를 만들자 ('배'):
        ('나'의 'x' % '배')를 돌려주자
    <기호 크다>를 만들자 ('대상'):
        ('나'의 'x'가 '대상'의 'x'보다 크다)를 돌려주자
    <기호 작다>를 만들자 ('대상'):
        ('나'의 'x'가 '대상'의 'x'보다 작다)를 돌려주자
    <기호 이상>을 만들자 ('대상'):
        ('나'의 'x'가 '대상'의 'x' 이상이다)를 돌려주자
    <기호 이하>를 만들자 ('대상'):
        ('나'의 'x'가 '대상'의 'x' 이하이다)를 돌려주자
[점]은 [벡터]를 바탕으로 하고 설계하자:
    'y'를 0으로 정하자
'a'를 새로운 [벡터](7)로 정하자
'b'를 새로운 [벡터](2)로 정하자
(('a' + 'b')의 'x')를 출력하자
(('a' - 'b')의 'x')를 출력하자
('a' * 3)을 출력하자
('a' / 2)를 출력하자
('a' % 4)를 출력하자
('a'가 'b'보다 크다)를 출력하자
('a'가 'b'보다 작다)를 출력하자
('a'가 'a' 이상이다)를 출력하자
('a'가 'b' 이하이다)를 출력하자
'c'를 'a'로 정하자
'c'에 'b'를 더하자
'c'에서 'a'를 빼자
('c'의 'x')를 출력하자
('a'의 'x')를 출력하자
'n'을 0으로 정하자
('a'가 'b'보다 크다)인 동안 반복하자:
    'a'에서 'b'를 빼자
    'n'에 1을 더하자
'n'을 출력하자
'p'를 새로운 [점](5)로 정하자
(('p' + 'p')의 'x')를 출력하자
`

const operatorKanade = `【ベクトル】を設計しよう:
    『x』を0にしよう
    最初に作られる時(『x』)次のようにしよう:
        『私』の『x』を『x』にしよう
    【ベクトル】を返す〈記号 足す〉を作ろう(【ベクトル】の『対象』):
        新しい【ベクトル】((『私』の『x』 + 『対象』の『x』))を返そう
    〈記号 引く〉を作ろう(『対象』):
        新しい【ベクトル】((『私』の『x』 - 『対象』の『x』))を返そう
    〈記号 掛ける〉を作ろう(『倍』):
        (『私』の『x』 * 『倍』)を返そう
    〈記号 割る〉を作ろう(『倍』):
        (『私』の『x』 / 『倍』)を返そう
    〈記号 余り〉を作ろう(『倍』):
        (『私』の『x』 % 『倍』)を返そう
    〈記号 大きい〉を作ろう(『対象』):
        (『私』の『x』が『対象』の『x』より大きい)を返そう
    〈記号 小さい〉を作ろう(『対象』):
        (『私』の『x』が『対象』の『x』より小さい)を返そう
    〈記号 以上〉を作ろう(『対象』):
        (『私』の『x』が『対象』の『x』以上だ)を返そう
    〈記号 以下〉を作ろう(『対象』):
        (『私』の『x』が『対象』の『x』以下だ)を返そう
【ベクトル】をもとにして【点】を設計しよう:
    『y』を0にしよう
『a』を新しい【ベクトル】(7)にしよう
『b』を新しい【ベクトル】(2)にしよう
((『a』 + 『b』)の『x』)を出力しよう
((『a』 - 『b』)の『x』)を出力しよう
(『a』 * 3)を出力しよう
(『a』 / 2)を出力しよう
(『a』 % 4)を出力しよう
(『a』が『b』より大きい)を出力しよう
(『a』が『b』より小さい)を出力しよう
(『a』が『a』以上だ)を出力しよう
(『a』が『b』以下だ)を出力しよう
『c』を『a』にしよう
『c』に『b』を足そう
『c』から『a』を引こう
(『c』の『x』)を出力しよう
(『a』の『x』)を出力しよう
『n』を0にしよう
(『a』が『b』より大きい)間繰り返そう:
    『a』から『b』を引こう
    『n』に1を足そう
『n』を出力しよう
『p』を新しい【点】(5)にしよう
((『p』 + 『p』)の『x』)を出力しよう
`

func TestOperatorMethods(t *testing.T) {
	outs, errors := runAllFour(t, dualRun{name: "operators", hari: operatorHari, kanade: operatorKanade})
	wants := []string{
		"9|5|21|3.5|3|참|거짓|참|거짓|2|7|3|10",
		"9|5|21|3.5|3|真|偽|真|偽|2|7|3|10",
	}
	for k, name := range []string{"tree-walker", "bytecode", "kanade tree-walker", "kanade bytecode"} {
		if errors[k] != nil {
			t.Fatalf("%s: %v", name, errors[k])
		}
		if got := strings.Join(outs[k], "|"); got != wants[k/2] {
			t.Errorf("%s: got %q, want %q", name, got, wants[k/2])
		}
	}
}

// The operator's method is called as a method is: its argument and result
// types are checked, a comparison's result must be a boolean where it decides
// a condition, it counts toward the call nesting limit, and a class without
// the method (or a value on the right) keeps the usual TypeError.
func TestOperatorMethodsAreMethodCalls(t *testing.T) {
	prelude := `[수]를 설계하자:
    'v'를 1로 정하자
    <기호 더하기>를 만들자 ([수]인 'o'):
        새로운 [수]()를 돌려주자
    [숫자]를 돌려주는 <기호 빼기>를 만들자 ('o'):
        "글자"를 돌려주자
    <기호 크다>를 만들자 ('o'):
        1을 돌려주자
    <기호 곱하기>를 만들자 ('o'):
        ('나' * 'o')를 돌려주자
[빈것]을 설계하자:
    'v'를 1로 정하자
'하나'를 새로운 [수]()로 정하자
`
	cases := []struct {
		code string
		want errs.Code
	}{
		{"('하나' + 1)을 출력하자\n", errs.ArgumentTypeMismatch},
		{"('하나' - '하나')를 출력하자\n", errs.ReturnTypeMismatch},
		{"만약 ('하나'가 '하나'보다 크다) 라면:\n    1을 출력하자\n", errs.ConditionNotBoolean},
		{"('하나' * 2)를 출력하자\n", errs.CallTooDeep},
		{"(1 + '하나')를 출력하자\n", errs.OperandTypeMismatch},
		{"(새로운 [빈것]() + 1)을 출력하자\n", errs.OperandTypeMismatch},
	}
	for _, c := range cases {
		for engine, r := range engines(t, prelude+c.code) {
			e, ok := r.err.(*errs.Error)
			if !ok || e.Code != c.want {
				t.Errorf("%s: %q: got %v, want %s", engine, c.code, r.err, c.want)
			}
		}
	}
}
