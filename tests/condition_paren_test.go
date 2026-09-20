package tests

// A parenthesized group at the start of a comparison — (('x' % 2) == 0) — used to
// be read as the whole condition, so the comparison after it was dropped and the
// condition silently came out false.

import (
	"strings"
	"testing"

	"github.com/soumt-r/hana/errs"
)

const hajaParenConditions = `
'x'를 [숫자]인 4로 정하자
만약 (('x' % 2) == 0) 라면:
    "가"를 출력하자
만약 ('x' % 2 == 0) 라면:
    "나"를 출력하자
만약 (('x' + 1) == 5) 그리고 ('x' > 3) 라면:
    "다"를 출력하자
만약 (('x' + 1) == 6) 또는 (('x' * 2) == 8) 라면:
    "라"를 출력하자
만약 (('x' + 1)이 5와 같다) 라면:
    "마"를 출력하자
만약 (('x' * 2) != 8) 라면:
    "안나와요"를 출력하자
만약 ((('x' * 2) + 1) >= 9) 라면:
    "바"를 출력하자
만약 (('x' % 2) == 1) 라면:
    "안나와요2"를 출력하자
만약 ('x' > 3) 그리고 (('x' % 2) == 0) 라면:
    "사"를 출력하자
`

const kanadeParenConditions = `
『x』を【数字】の4にしよう
もし((『x』 % 2) == 0)なら:
    「あ」を出力しよう
もし(『x』 % 2 == 0)なら:
    「い」を出力しよう
もし(((『x』 * 2) + 1) >= 9)なら:
    「う」を出力しよう
もし(((『x』 % 2) == 1))なら:
    「出ません」を出力しよう
`

func TestParenthesizedComparisonsAreNotDropped(t *testing.T) {
	for engine, r := range engines(t, hajaParenConditions) {
		if r.err != nil {
			t.Fatalf("%s: %v", engine, r.err)
		}
		if got, want := strings.Join(r.out, ""), "가나다라마바사"; got != want {
			t.Errorf("%s: got %q, want %q", engine, got, want)
		}
	}
	tw, err := runKanade(t, kanadeParenConditions)
	if err != nil {
		t.Fatalf("kanade tree-walker: %v", err)
	}
	bc, err := runKanadeBytecode(t, kanadeParenConditions)
	if err != nil {
		t.Fatalf("kanade bytecode: %v", err)
	}
	for label, out := range map[string][]string{"tree-walker": tw.Output, "bytecode": bc.Output} {
		if got, want := strings.Join(out, ""), "あいう"; got != want {
			t.Errorf("kanade %s: got %q, want %q", label, got, want)
		}
	}
}

// Only true and false are conditions: anything else is a TypeError, in if, while
// and on either side of 그리고/또는, instead of silently counting as false.
func TestConditionsMustBeBooleans(t *testing.T) {
	cases := []struct{ name, code, want string }{
		{"숫자", "'x'를 [숫자]인 4로 정하자\n만약 ('x') 라면:\n    \"안나와요\"를 출력하자\n", "숫자 값은 조건으로 쓸 수 없어요"},
		{"글", "'x'를 [문자열]인 \"글\"로 정하자\n만약 ('x') 라면:\n    \"안나와요\"를 출력하자\n", "문자열 값은 조건으로 쓸 수 없어요"},
		{"목록", "'x'를 [목록]인 [1]로 정하자\n만약 ('x') 라면:\n    \"안나와요\"를 출력하자\n", "목록 값은 조건으로 쓸 수 없어요"},
		{"비어있음", "만약 (비어있음) 라면:\n    \"안나와요\"를 출력하자\n", "비어있음 값은 조건으로 쓸 수 없어요"},
		{"반복", "'x'를 [숫자]인 1로 정하자\n('x') 인 동안 반복하자:\n    'x'를 0으로 정하자\n", "숫자 값은 조건으로 쓸 수 없어요"},
		{"그리고 왼쪽", "'x'를 [숫자]인 1로 정하자\n만약 ('x') 그리고 (1 < 2) 라면:\n    \"안나와요\"를 출력하자\n", "숫자 값은 조건으로 쓸 수 없어요"},
		{"그리고 오른쪽", "'x'를 [숫자]인 1로 정하자\n만약 (1 < 2) 그리고 ('x') 라면:\n    \"안나와요\"를 출력하자\n", "숫자 값은 조건으로 쓸 수 없어요"},
		{"또는 오른쪽", "'x'를 [숫자]인 1로 정하자\n만약 (1 > 2) 또는 ('x') 라면:\n    \"안나와요\"를 출력하자\n", "숫자 값은 조건으로 쓸 수 없어요"},
	}
	for _, c := range cases {
		for engine, r := range engines(t, c.code) {
			if r.err == nil || !strings.Contains(errs.Localize(errs.Korean, r.err), c.want) || len(r.out) != 0 {
				t.Errorf("%s %s: want a TypeError %q and no output, got %v %v", engine, c.name, c.want, r.err, r.out)
			}
		}
	}
	// A condition that is true or false keeps working, and 그리고/또는 still short-circuit.
	ok := "'x'를 [숫자]인 1로 정하자\n만약 (1 > 2) 그리고 ('x') 라면:\n    \"안나와요\"를 출력하자\n만약 (1 < 2) 또는 ('x') 라면:\n    \"나와요\"를 출력하자\n"
	for engine, r := range engines(t, ok) {
		if r.err != nil || strings.Join(r.out, "") != "나와요" {
			t.Errorf("%s: short-circuit should not look at the right side, got %v %v", engine, r.out, r.err)
		}
	}
}
