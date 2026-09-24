package tests

// A switch with no matching case and no default does nothing. Spec 5.4: a string can be walked with 마다 반복하자, one character
// (code point) at a time. Both engines agree.

import (
	"strings"
	"testing"
)

func TestSwitchWithoutAMatchDoesNothing(t *testing.T) {
	code := "'x'를 [문자열]인 \"수\"로 정하자\n'x'에 따라 나누자:\n    \"월\" 인 경우:\n        \"가\"를 출력하자\n\"끝\"을 출력하자\n"
	tree, err := runHari(t, code)
	if err != nil {
		t.Fatal(err)
	}
	bc, err := runBytecode(t, code)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(tree.Output, "|") + "#" + strings.Join(bc.Output, "|"); got != "끝#끝" {
		t.Errorf("printed %q", got)
	}
}

func TestSwitchWithADefaultOrAMatchIsFine(t *testing.T) {
	code := "'x'를 [문자열]인 \"수\"로 정하자\n'x'에 따라 나누자:\n    \"월\" 인 경우:\n        \"가\"를 출력하자\n    나머지는:\n        \"나\"를 출력하자\n" +
		"'x'에 따라 나누자:\n    \"수\" 인 경우:\n        \"다\"를 출력하자\n"
	tree, err := runHari(t, code)
	if err != nil {
		t.Fatal(err)
	}
	bc, err := runBytecode(t, code)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(tree.Output, "|") + "#" + strings.Join(bc.Output, "|"); got != "나|다#나|다" {
		t.Errorf("printed %q", got)
	}
}

func TestAStringCanBeWalkedCharacterByCharacter(t *testing.T) {
	code := "'인사'를 [문자열]인 \"안녕😀\"으로 정하자\n'인사'의 '글자'마다 반복하자:\n    '글자'를 출력하자\n"
	tree, err := runHari(t, code)
	if err != nil {
		t.Fatal(err)
	}
	bc, err := runBytecode(t, code)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(tree.Output, "|") + "#" + strings.Join(bc.Output, "|"); got != "안|녕|😀#안|녕|😀" {
		t.Errorf("printed %q", got)
	}
	number := "'n'을 [숫자]인 3으로 정하자\n'n'의 '글자'마다 반복하자:\n    '글자'를 출력하자\n"
	_, err = runHari(t, number)
	requireErrorContains(t, err, "Not iterable")
	_, err = runBytecode(t, number)
	requireErrorContains(t, err, "Not iterable")
}
