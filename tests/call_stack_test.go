package tests

// The tree-walker builds call arguments on a shared stack and pools loop scopes; these
// programs would show a slot being overwritten or a scope leaking into the next pass.

import (
	"strings"
	"testing"
)

func TestNestedCallArgumentsKeepTheirSlots(t *testing.T) {
	code := `
[숫자]를 돌려주는 <더하기>를 만들자 ([숫자]인 '가', [숫자]인 '나'):
    '가' + '나'를 돌려주자

[숫자]를 돌려주는 <기본>를 만들자 ([숫자]인 '가', [숫자]인 '나' = <더하기>(100, 200), [숫자]인 '다' = 1):
    '가' + '나' + '다'를 돌려주자

<기본>(<더하기>(1, 2), <더하기>(<더하기>(3, 4), 5))를 출력하자
<기본>(<기본>(1, 2, 3), 10)를 출력하자
<기본>(7)를 출력하자
`
	tree, err := runHaja(t, code)
	if err != nil {
		t.Fatal(err)
	}
	bc, err := runBytecode(t, code)
	if err != nil {
		t.Fatal(err)
	}
	want := "16|17|308"
	if got := strings.Join(tree.Output, "|") + "#" + strings.Join(bc.Output, "|"); got != want+"#"+want {
		t.Errorf("printed %q", got)
	}
}

func TestLoopScopesDoNotLeakBetweenPasses(t *testing.T) {
	code := `
1부터 3까지 반복하자 ('가'):
    '임시'를 '가' * 10으로 정하자
    '임시'를 출력하자
`
	tree, err := runHaja(t, code)
	if err != nil {
		t.Fatal(err)
	}
	bc, err := runBytecode(t, code)
	if err != nil {
		t.Fatal(err)
	}
	want := "10|20|30"
	if got := strings.Join(tree.Output, "|") + "#" + strings.Join(bc.Output, "|"); got != want+"#"+want {
		t.Errorf("printed %q", got)
	}
}

// A range loop runs one pass per number of its range: what the body does to the loop
// variable or to the variable the end came from changes nothing about the passes.
func TestRangeLoopIgnoresWhatTheBodyDoesToItsVariables(t *testing.T) {
	cases := map[string]string{
		"1|2|3":  "'끝'을 [숫자]인 3으로 정하자\n1부터 '끝'까지 반복하자 ('가'):\n    '가'를 출력하자\n    '끝'에 10을 더하자\n    '가'에 5를 더하자\n",
		"1|2|3#": "1부터 3까지 반복하자 ('가'):\n    '가'를 출력하자\n    '가'를 100으로 정하자\n",
		"3|2|1":  "3부터 1까지 반복하자 ('가'):\n    '가'를 출력하자\n    '가'에 5를 더하자\n",
	}
	for want, code := range cases {
		want = strings.TrimSuffix(want, "#")
		tree, err := runHaja(t, code)
		if err != nil {
			t.Fatal(err)
		}
		bc, err := runBytecode(t, code)
		if err != nil {
			t.Fatal(err)
		}
		if got := strings.Join(tree.Output, "|") + "#" + strings.Join(bc.Output, "|"); got != want+"#"+want {
			t.Errorf("%q printed %q, want %q on both", code, got, want)
		}
	}
}
