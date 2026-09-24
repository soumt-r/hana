package tests

import (
	"strings"
	"testing"

	"github.com/soumt-r/hana/errs"
)

// Each case throws one engine-raised error, catches it with an untyped
// handler, and prints the caught message. Every case runs on both VMs in both
// languages and must produce the identical, native-language text:
//   - guards the errs catalog wiring at both catch boundaries (vm's
//     TryStatement and bcvm's errorObjectFor);
//   - guards the bug this refactor fixed, where the bytecode VM bound caught
//     errors to a hardcoded Korean 오류/메시지 so a Kanade handler read a
//     property that didn't exist and printed 空っぽ.
type caughtErrorCase struct {
	name   string
	ko     string // Hari source: the body of the `일단 해보자:` block
	ja     string // Kanade source: the body of the `とりあえずやってみよう:` block
	wantKo string
	wantJa string
}

func TestCaughtEngineErrorsAreLocalizedIdenticallyOnBothVMs(t *testing.T) {
	cases := []caughtErrorCase{
		{
			name:   "divide by zero",
			ko:     "'결과'를 10 / 0으로 정하자",
			ja:     "『結果』を10/0にしよう",
			wantKo: "DivideByZeroError: 0으로 나눌 수 없어요.",
			wantJa: "DivideByZeroError: 0で割ることはできません。",
		},
		{
			name:   "pop from empty list",
			ko:     "'목록'을 [(숫자)목록]인 []로 정하자\n    '목록' 뒤에서 꺼내자",
			ja:     "『リスト』を【(数字)リスト】の【】にしよう\n    『リスト』の後ろから取り出そう",
			wantKo: "IndexOutOfBoundsError: 목록이 비어 있어요.",
			wantJa: "IndexOutOfBoundsError: リストが空です。",
		},
		{
			name:   "list index out of range",
			ko:     "'목록'을 [(숫자)목록]인 [1, 2]로 정하자\n    '목록'의 5번째를 출력하자",
			ja:     "『リスト』を【(数字)リスト】の【1,2】にしよう\n    『リスト』の5番目を出力しよう",
			wantKo: "IndexOutOfBoundsError: 목록의 길이를 벗어난 위치(인덱스)예요.",
			wantJa: "IndexOutOfBoundsError: リストの長さを超えた位置（インデックス）です。",
		},
		{
			name:   "unknown variable",
			ko:     "'없는변수'를 출력하자",
			ja:     "『ない変数』を出力しよう",
			wantKo: "ReferenceError: '없는변수' 변수를 찾을 수 없어요.",
			wantJa: "ReferenceError: 変数『ない変数』が見つかりません。",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ko := "일단 해보자:\n    " + c.ko + "\n오류가 발생했다면 ('에러'):\n    '에러'의 '메시지'를 출력하자\n"
			ja := "とりあえずやってみよう:\n    " + c.ja + "\n発生したら(『エラー』):\n    『エラー』の『メッセージ』を出力しよう\n"

			interp, err := runHari(t, ko)
			if err != nil {
				t.Fatalf("hari tree-walker: %v", err)
			}
			assertLastLine(t, "hari tree-walker", interp.Output, c.wantKo)

			bc, err := runBytecode(t, ko)
			if err != nil {
				t.Fatalf("hari bytecode: %v", err)
			}
			assertLastLine(t, "hari bytecode", bc.Output, c.wantKo)

			kinterp, err := runKanade(t, ja)
			if err != nil {
				t.Fatalf("kanade tree-walker: %v", err)
			}
			assertLastLine(t, "kanade tree-walker", kinterp.Output, c.wantJa)

			kbc, err := runKanadeBytecode(t, ja)
			if err != nil {
				t.Fatalf("kanade bytecode: %v", err)
			}
			assertLastLine(t, "kanade bytecode", kbc.Output, c.wantJa)
		})
	}
}

func assertLastLine(t *testing.T, engine string, output []string, want string) {
	t.Helper()
	if len(output) == 0 {
		t.Errorf("%s printed nothing, want %q", engine, want)
		return
	}
	if got := output[len(output)-1]; got != want {
		t.Errorf("%s printed %q, want %q", engine, got, want)
	}
}

// An uncaught error surfaces from Run() as a Go error. It stays language
// neutral there (English wording, same Kind prefix) — localizing is the
// caller's job via errs.Localize, which is what cmd/run.go does.
func TestUncaughtErrorIsLocalizableByTheCaller(t *testing.T) {
	_, err := runHari(t, "'목록'을 [(숫자)목록]인 []로 정하자\n'목록' 뒤에서 꺼내자\n")
	requireErrorContains(t, err, "IndexOutOfBoundsError")
	if got := errs.Localize(errs.Korean, err); got != "IndexOutOfBoundsError: 목록이 비어 있어요." {
		t.Errorf("ko: %q", got)
	}
	if got := errs.Localize(errs.Japanese, err); got != "IndexOutOfBoundsError: リストが空です。" {
		t.Errorf("ja: %q", got)
	}
	if !strings.Contains(err.Error(), "The list is empty.") {
		t.Errorf("Error() should stay English, got %q", err.Error())
	}
}
