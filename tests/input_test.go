package tests

// `입력받자` reads one line per statement from the interpreter's input source
// and converts it to the requested type, identically on both engines and in
// both languages. With no input source it reads an empty line (tests and tools
// never block).

import (
	"strings"
	"testing"

	"github.com/soumt-r/hana/bcstdlib"
	"github.com/soumt-r/hana/bcvm"
	"github.com/soumt-r/hana/bytecode"
	"github.com/soumt-r/hana/conv"
	lexer "github.com/soumt-r/hana/lexer/hari"
	kanadelexer "github.com/soumt-r/hana/lexer/kanade"
	parser "github.com/soumt-r/hana/parser/hari"
	kanadeparser "github.com/soumt-r/hana/parser/kanade"
	"github.com/soumt-r/hana/stdlib"
	"github.com/soumt-r/hana/vm"
)

func linesOf(stdin string) func() string { return conv.NewLineReader(strings.NewReader(stdin)) }

// runAllFourWithInput runs code on tree-walker/bytecode x hari/kanade, feeding each the same stdin.
func runAllFourWithInput(t *testing.T, hari, kanade, stdin string) (outs [4][]string, errs [4]error) {
	t.Helper()
	{
		p := parser.New(lexer.New(hari))
		prog := p.ParseProgram()
		i := vm.NewInterpreter(prog)
		stdlib.RegisterStandardLibrary(i)
		i.ReadLine = linesOf(stdin)
		errs[0] = i.Run()
		outs[0] = i.Output
		p = parser.New(lexer.New(hari))
		bc := bytecode.NewCompiler().Compile(p.ParseProgram())
		v := bcvm.New(bc)
		bcstdlib.RegisterStandardLibrary(v)
		v.ReadLine = linesOf(stdin)
		errs[1] = v.Run()
		outs[1] = v.Output
	}
	{
		p := kanadeparser.New(kanadelexer.New(kanade))
		prog := p.ParseProgram()
		i := vm.NewInterpreter(prog, vm.JapaneseConfig)
		stdlib.RegisterStandardLibrary(i)
		i.ReadLine = linesOf(stdin)
		errs[2] = i.Run()
		outs[2] = i.Output
		p = kanadeparser.New(kanadelexer.New(kanade))
		bc := bytecode.NewKanadeCompiler().Compile(p.ParseProgram())
		v := bcvm.New(bc)
		v.UseJapaneseWords()
		bcstdlib.RegisterStandardLibrary(v, bcstdlib.Japanese)
		v.ReadLine = linesOf(stdin)
		errs[3] = v.Run()
		outs[3] = v.Output
	}
	return
}

func TestInputReadsAndConvertsOnEveryEngine(t *testing.T) {
	hari := "'이름'을 입력받자\n'나이'를 [숫자]로 입력받자\n'참인가'를 [논리]로 입력받자\n틀\"{'이름'}/{'나이' + 1}/{'참인가'}\"를 출력하자\n"
	kanade := "『名前』を入力してもらおう\n『年齢』を【数字】で入力してもらおう\n『はい』を【論理】で入力してもらおう\n枠「{『名前』}/{『年齢』 + 1}/{『はい』}」を出力しよう\n"
	outs, errs := runAllFourWithInput(t, hari, kanade, "홍길동\n 20 \n참\n")
	// Kanade's boolean word is 真, so 참 is not a boolean there.
	if errs[0] != nil || errs[1] != nil {
		t.Fatalf("hari: tree %v, bytecode %v", errs[0], errs[1])
	}
	for i := 0; i < 2; i++ {
		if strings.Join(outs[i], "|") != "홍길동/21/참" {
			t.Errorf("hari engine %d output = %v", i, outs[i])
		}
	}
	if errs[2] == nil || errs[3] == nil {
		t.Errorf("kanade must reject 참 as a boolean: tree %v, bytecode %v", errs[2], errs[3])
	}

	outs, errs = runAllFourWithInput(t, hari, kanade, "田中\n 20 \n真\n")
	for i := 2; i < 4; i++ {
		if errs[i] != nil || strings.Join(outs[i], "|") != "田中/21/真" {
			t.Errorf("kanade engine %d: out %v err %v", i, outs[i], errs[i])
		}
	}
}

func TestInputConversionFailureIsACatchableError(t *testing.T) {
	hari := "일단 해보자:\n    '나이'를 [숫자]로 입력받자\n오류가 발생했다면 ('에러'):\n    '에러'의 '메시지'를 출력하자\n"
	kanade := "とりあえずやってみよう:\n    『年齢』を【数字】で入力してもらおう\n発生したら(『エラー』):\n    『エラー』の『メッセージ』を出力しよう\n"
	outs, errs := runAllFourWithInput(t, hari, kanade, "스물\n")
	for i := range outs {
		if errs[i] != nil {
			t.Errorf("engine %d: the failure should have been caught, got %v", i, errs[i])
		}
	}
	if len(outs[0]) != 1 || !strings.Contains(outs[0][0], "숫자") || strings.Join(outs[0], "") != strings.Join(outs[1], "") {
		t.Errorf("hari messages: tree %v, bytecode %v", outs[0], outs[1])
	}
}

func TestInputWithoutASourceReadsAnEmptyLine(t *testing.T) {
	i, err := runHari(t, "'이름'을 입력받자\n틀\"[{'이름'}]\"를 출력하자\n")
	if err != nil || strings.Join(i.Output, "|") != "[]" {
		t.Errorf("tree-walker: %v %v", i.Output, err)
	}
	b, err := runBytecode(t, "'이름'을 입력받자\n틀\"[{'이름'}]\"를 출력하자\n")
	if err != nil || strings.Join(b.Output, "|") != "[]" {
		t.Errorf("bytecode: %v %v", b.Output, err)
	}
}
