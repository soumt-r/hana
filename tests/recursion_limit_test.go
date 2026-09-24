package tests

// Runaway recursion used to end the whole process with Go's fatal "stack
// overflow" (about 1 GB of memory, uncatchable). It is now a RecursionError
// that a 일단 해보자 can catch, on both engines and in both languages, while
// ordinary deep recursion still works.

import (
	"fmt"
	"strings"
	"testing"

	"github.com/soumt-r/hana/vm"
)

const (
	hariSum = "[숫자]를 돌려주는 <합>을 만들자 ([숫자]인 'n'):\n    만약 ('n'이 1 이하이다) 라면:\n        1를 돌려주자\n    ('n' + <합>('n' - 1))를 돌려주자\n"
)

func TestDeepButFiniteRecursionStillWorks(t *testing.T) {
	code := hariSum + fmt.Sprintf("<합>(%d)를 출력하자\n", vm.MaxCallDepth/2)
	want := fmt.Sprint((vm.MaxCallDepth / 2) * (vm.MaxCallDepth/2 + 1) / 2)
	tree, err := runHari(t, code)
	if err != nil {
		t.Fatalf("tree-walker: %v", err)
	}
	bc, err := runBytecode(t, code)
	if err != nil {
		t.Fatalf("bytecode: %v", err)
	}
	if strings.Join(tree.Output, "|") != want || strings.Join(bc.Output, "|") != want {
		t.Errorf("tree %v, bytecode %v, want [%s]", tree.Output, bc.Output, want)
	}
}

func TestRunawayRecursionIsARecursionErrorOnEveryEngine(t *testing.T) {
	hari := "<f>를 만들자 ():\n    <f>()를 실행하자\n<f>()를 실행하자\n"
	kanade := "〈f〉を作ろう ():\n    〈f〉()を実行しよう\n〈f〉()を実行しよう\n"
	_, e0 := runHari(t, hari)
	_, e1 := runBytecode(t, hari)
	_, e2 := runKanade(t, kanade)
	_, e3 := runKanadeBytecode(t, kanade)
	for i, err := range []error{e0, e1, e2, e3} {
		if err == nil || !strings.Contains(err.Error(), "RecursionError") {
			t.Errorf("engine %d: want a RecursionError, got %v", i, err)
		}
	}
}

func TestRunawayRecursionCanBeCaught(t *testing.T) {
	code := "<f>를 만들자 ():\n    <f>()를 실행하자\n일단 해보자:\n    <f>()를 실행하자\n오류가 발생했다면 ('에러'):\n    '에러'의 '메시지'를 출력하자\n"
	for name, run := range map[string]func(*testing.T, string) ([]string, error){
		"tree-walker": func(t *testing.T, c string) ([]string, error) { i, err := runHari(t, c); return i.Output, err },
		"bytecode":    func(t *testing.T, c string) ([]string, error) { v, err := runBytecode(t, c); return v.Output, err },
	} {
		out, err := run(t, code)
		if err != nil {
			t.Errorf("%s: the recursion error should have been caught, got %v", name, err)
			continue
		}
		if len(out) != 1 || !strings.Contains(out[0], "10000") {
			t.Errorf("%s: caught message = %v", name, out)
		}
	}
}

// An error inside an operand used to be discarded by the tree-walker: with the
// recursion limit it showed up as "10001100009999...<nil>" (each level
// concatenating its n with a nil), and any failing call inside an expression
// could turn into a silent nil the same way.
func TestErrorsInsideOperandsAreNotSwallowed(t *testing.T) {
	for name, code := range map[string]string{
		"call as right operand":       "<f>를 만들자 ():\n    (1 + <없는함수>())를 돌려주자\n<f>()를 실행하자\n",
		"missing variable as operand": "'a'를 (1 + '없는변수')로 정하자\n",
	} {
		i, err := runHari(t, code)
		b, berr := runBytecode(t, code)
		if err == nil || berr == nil {
			t.Errorf("%s: tree-walker err=%v (out %v), bytecode err=%v (out %v) — both must fail", name, err, i.Output, berr, b.Output)
		}
	}
}
