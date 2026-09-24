// Kanade(카나데) regression tests: lex + parse + run a Japanese-syntax
// script through the same tree-walker/bytecode engines as Hari, via
// parser/kanade + vm.JapaneseConfig / bytecode.NewKanadeCompiler +
// bcstdlib.Japanese, and assert on output. Sources here are lifted
// verbatim or near-verbatim from kanade-docs' actual published examples
// (kanade-docs/src/pages/docs) rather than invented, specifically so a
// regression here means real published documentation would also break —
// see hana/CLAUDE.md for why an earlier ASCII-punctuation version of this
// file passed while matching none of kanade-docs' real syntax.
package tests

import (
	"strings"
	"testing"

	"github.com/soumt-r/hana/bcstdlib"
	"github.com/soumt-r/hana/bcvm"
	"github.com/soumt-r/hana/bytecode"
	kanadelexer "github.com/soumt-r/hana/lexer/kanade"
	kanadeparser "github.com/soumt-r/hana/parser/kanade"
	"github.com/soumt-r/hana/stdlib"
	"github.com/soumt-r/hana/vm"
)

func runKanade(t *testing.T, code string) (*vm.Interpreter, error) {
	t.Helper()
	l := kanadelexer.New(code)
	p := kanadeparser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse error: %v", p.Errors())
	}
	interp := vm.NewInterpreter(prog, vm.JapaneseConfig)
	stdlib.RegisterStandardLibrary(interp)
	err := interp.Run()
	return interp, err
}

// runKanadeBytecode is runBytecode (bytecode_test.go) for a Kanade source
// string: lex+parse via lexer/kanade+parser/kanade, compile via
// bytecode.NewKanadeCompiler, register bcstdlib.Japanese.
func runKanadeBytecode(t *testing.T, code string) (*bcvm.VM, error) {
	t.Helper()
	l := kanadelexer.New(code)
	p := kanadeparser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse error: %v", p.Errors())
	}
	compiler := bytecode.NewKanadeCompiler()
	bcProg := compiler.Compile(prog)
	if len(compiler.Errors()) > 0 {
		t.Fatalf("compile error: %v", compiler.Errors())
	}
	vm := bcvm.New(bcProg)
	vm.UseJapaneseWords()
	bcstdlib.RegisterStandardLibrary(vm, bcstdlib.Japanese)
	err := vm.Run()
	return vm, err
}

// From kanade-docs/src/pages/docs/tutorial/1-variables.md and 4-conditions.md.
func TestKanadeVariablesPrintAndIf(t *testing.T) {
	interp, err := runKanade(t, `『名前』を【文字列】の「田中太郎」にしよう
枠「こんにちは、{『名前』}さん！」を出力しよう

『点数』を【数字】の85にしよう
もし(『点数』が90より大きい)なら:
    「よくできました！」を出力しよう
もしくは(『点数』が80より大きい)なら:
    「もう少し頑張ってみましょう！」を出力しよう
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"こんにちは、田中太郎さん！", "もう少し頑張ってみましょう！"}
	if strings.Join(interp.Output, "|") != strings.Join(want, "|") {
		t.Errorf("Output = %v, want %v", interp.Output, want)
	}
}

// From kanade-docs/src/pages/docs/tutorial/7-loops.md and 6-collections.md.
func TestKanadeLoops(t *testing.T) {
	interp, err := runKanade(t, `『エネルギー』を【数字】の3にしよう
(『エネルギー』が0より大きい)間繰り返そう:
    枠「エネルギー残り:{『エネルギー』}」を出力しよう
    『エネルギー』から1を引こう

『合計』を【数字】の0にしよう
0から3まで繰り返そう(『i』):
    『合計』に『i』を足そう
『合計』を出力しよう

『果物』を【(文字列)リスト】の【「りんご」,「ぶどう」】にしよう
『果物』の後に「バナナ」を追加しよう
『果物』の『実』ごとに繰り返そう:
    『実』を出力しよう
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{
		"エネルギー残り:3", "エネルギー残り:2", "エネルギー残り:1",
		"6", "りんご", "ぶどう", "バナナ",
	}
	if strings.Join(interp.Output, "|") != strings.Join(want, "|") {
		t.Errorf("Output = %v, want %v", interp.Output, want)
	}
}

// From kanade-docs/src/pages/docs/oop/1-classes.md.
const kanadeCarClassSource = `【自動車】を設計しよう:
    『速度』を【数字】の0にしよう

    最初に作られる時(【文字列】の『初期色』)次のようにしよう:
        『私』の『速度』を0にしよう

    〈走る〉を作ろう(【数字】の『増加量』):
        『私』の『速度』に『増加量』を足そう

『マイカー』を【自動車】の新しい【自動車】(「青」)にしよう
『マイカー』の〈走る〉(50)を実行しよう
枠「速度:{『マイカー』の『速度』}」を出力しよう
`

func TestKanadeClassConstructorAndMethod(t *testing.T) {
	interp, err := runKanade(t, kanadeCarClassSource)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"速度:50"}
	if strings.Join(interp.Output, "|") != strings.Join(want, "|") {
		t.Errorf("Output = %v, want %v", interp.Output, want)
	}
}

// From kanade-docs/src/pages/docs/tutorial/9-errors.md.
func TestKanadeTryCatchFinally(t *testing.T) {
	interp, err := runKanade(t, `とりあえずやってみよう:
    「安全に実行を開始します。」を出力しよう
発生したら(『エラー』):
    「エラーが出た時だけ実行されます。」を出力しよう
最後はいつも:
    「エラーの発生に関係なくいつも実行されます！」を出力しよう
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"安全に実行を開始します。", "エラーの発生に関係なくいつも実行されます！"}
	if strings.Join(interp.Output, "|") != strings.Join(want, "|") {
		t.Errorf("Output = %v, want %v", interp.Output, want)
	}

	// The untyped-catch path, and the builtin [エラー] class's 'メッセージ'
	// field, via a script that actually throws.
	interp2, err := runKanade(t, `とりあえずやってみよう:
    「安全網の中に入りました！」を出力しよう
    新しい【エラー】(「あっ！問題が発生しました！」)を発生させよう
    「この部分は実行されません。」を出力しよう
発生したら(『エラー』):
    枠「捕まえた！エラーの内容:{『エラー』の『メッセージ』}」を出力しよう
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want2 := []string{"安全網の中に入りました！", "捕まえた！エラーの内容:あっ！問題が発生しました！"}
	if strings.Join(interp2.Output, "|") != strings.Join(want2, "|") {
		t.Errorf("Output = %v, want %v", interp2.Output, want2)
	}
}

// From kanade-docs/src/pages/docs/tutorial/5-switch.md.
func TestKanadeSwitch(t *testing.T) {
	interp, err := runKanade(t, `『会員ランク』を【文字列】の「VIP」にしよう
『支給品』を【(文字列)リスト】の【】にしよう

『会員ランク』によって分けよう:
    「VIP」の場合:
        『支給品』に「伝説の剣」を追加しよう
        次に続けよう
    「ゴールド」、「シルバー」の場合:
        『支給品』に「魔法のポーション」を追加しよう
        次に続けよう
    残りは:
        『支給品』に「初心者用包帯」を追加しよう

枠「支給完了:{『支給品』}」を出力しよう
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"支給完了:[伝説の剣, 魔法のポーション, 初心者用包帯]"}
	if strings.Join(interp.Output, "|") != strings.Join(want, "|") {
		t.Errorf("Output = %v, want %v", interp.Output, want)
	}
}

// From kanade-docs/src/pages/docs/oop/4-interfaces.md.
func TestKanadeInterface(t *testing.T) {
	interp, err := runKanade(t, `【飛べるもの】を規定しよう:
    〈飛ぶ〉がなければならない()

【飛べるもの】に従う【鳥】を設計しよう:
    〈飛ぶ〉を作ろう():
        「パタパタと飛びます！」を出力しよう

『スズメ』を【鳥】の新しい【鳥】()にしよう
『スズメ』の〈飛ぶ〉()を実行しよう
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"パタパタと飛びます！"}
	if strings.Join(interp.Output, "|") != strings.Join(want, "|") {
		t.Errorf("Output = %v, want %v", interp.Output, want)
	}
}

func TestKanadeImportBuiltinModuleAndAlias(t *testing.T) {
	interp, err := runKanade(t, `【数学】から〈切り上げ〉を持ってこよう
『値』を〈切り上げ〉(3.2)にしよう
『値』を出力しよう
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Join(interp.Output, "|") != "4" {
		t.Errorf("Output = %v, want [4]", interp.Output)
	}

	interp2, err := runKanade(t, `【数学】から〈切り上げ〉を〈丸め〉に持ってこよう
『値』を〈丸め〉(3.2)にしよう
『値』を出力しよう
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Join(interp2.Output, "|") != "4" {
		t.Errorf("Output = %v, want [4]", interp2.Output)
	}
}

func TestKanadeListClearMethod(t *testing.T) {
	interp, err := runKanade(t, `『リスト』を【(文字列)リスト】の【「りんご」,「バナナ」】にしよう
『リスト』を出力しよう
『リスト』の〈空にする〉()を実行しよう
『リスト』を出力しよう
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"[りんご, バナナ]", "[]"}
	if strings.Join(interp.Output, "|") != strings.Join(want, "|") {
		t.Errorf("Output = %v, want %v", interp.Output, want)
	}
}

// The tests below exercise the bytecode path (bytecode.NewKanadeCompiler +
// bcvm + bcstdlib.Japanese) for the spots that were hardcoded to 하자 until
// this pass: self/static-word compilation (LOAD_SELF/LOAD_STATIC_CLASS),
// the builtin [에러] class's own name/fields, template-literal {}
// interpolation, and bcstdlib's builtin names.

func TestKanadeBytecodeClassSelfReference(t *testing.T) {
	vm, err := runKanadeBytecode(t, kanadeCarClassSource)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vm.Output) != 1 || vm.Output[0] != "速度:50" {
		t.Errorf("Output = %v, want [速度:50]", vm.Output)
	}
}

func TestKanadeBytecodeTemplateLiteralInterpolation(t *testing.T) {
	vm, err := runKanadeBytecode(t, `『名前』を【文字列】の「太郎」にしよう
枠「こんにちは、{『名前』}さん」を出力しよう
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vm.Output) != 1 || vm.Output[0] != "こんにちは、太郎さん" {
		t.Errorf("Output = %v, want [こんにちは、太郎さん]", vm.Output)
	}
}

func TestKanadeBytecodeTryCatchUsesJapaneseBuiltinErrorClass(t *testing.T) {
	vm, err := runKanadeBytecode(t, `とりあえずやってみよう:
    新しい【エラー】(「あっ！問題が発生しました！」)を発生させよう
発生したら(『エラー』):
    枠「捕まえた！エラーの内容:{『エラー』の『メッセージ』}」を出力しよう
最後はいつも:
    「後片付け完了。」を出力しよう
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"捕まえた！エラーの内容:あっ！問題が発生しました！", "後片付け完了。"}
	if strings.Join(vm.Output, "|") != strings.Join(want, "|") {
		t.Errorf("Output = %v, want %v", vm.Output, want)
	}
}

func TestKanadeBytecodeBuiltinModuleImport(t *testing.T) {
	vm, err := runKanadeBytecode(t, `【数学】から〈切り上げ〉を持ってこよう
『値』を〈切り上げ〉(3.2)にしよう
『値』を出力しよう
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vm.Output) != 1 || vm.Output[0] != "4" {
		t.Errorf("Output = %v, want [4]", vm.Output)
	}
}

func TestKanadeBytecodeListClearMethod(t *testing.T) {
	vm, err := runKanadeBytecode(t, `『リスト』を【(文字列)リスト】の【「りんご」,「バナナ」】にしよう
『リスト』を出力しよう
『リスト』の〈空にする〉()を実行しよう
『リスト』を出力しよう
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"[りんご, バナナ]", "[]"}
	if strings.Join(vm.Output, "|") != strings.Join(want, "|") {
		t.Errorf("Output = %v, want %v", vm.Output, want)
	}
}

// The tests below each lock in one of the real doctest failures found and
// fixed by running go run ./cmd/doctest against the actual kanade-docs
// tree (see hana/CLAUDE.md) — regression coverage for bugs that unit tests
// alone never would have surfaced, since each needed a real published
// example's exact phrasing to trigger.

// From kanade-docs/src/pages/docs/patterns/1-singleton.md: calling a static
// method via "TYPE의 〈메서드〉()" — parser/hari's TYPE_IN branch used to
// always collapse "TYPE의 X" into just X, discarding the TYPE, which is
// right for "TYPE의 0" (a type-annotated value) but wrong here since the
// TYPE itself is the receiver of a static method call.
func TestKanadeStaticMethodCallViaTypeName(t *testing.T) {
	src := `【データベース】を設計しよう:
    『私たち』の『_インスタンス』を空っぽに隠そう

    【データベース】を返す『私たち』の〈取得する〉を作ろう():
        もし(『私たち』の『_インスタンス』が空っぽと同じだ)ならば:
            『私たち』の『_インスタンス』を【データベース】の新しい【データベース】()にしよう
        『私たち』の『_インスタンス』を返そう

『DB1』を【データベース】の〈取得する〉()にしよう
『DB2』を【データベース】の〈取得する〉()にしよう
もし(『DB1』と『DB2』が同じだ)なら:
    「同じインスタンス」を出力しよう
`
	interp, err := runKanade(t, src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Join(interp.Output, "|") != "同じインスタンス" {
		t.Errorf("Output = %v, want [同じインスタンス]", interp.Output)
	}

	vm, err := runKanadeBytecode(t, src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Join(vm.Output, "|") != "同じインスタンス" {
		t.Errorf("Output = %v, want [同じインスタンス]", vm.Output)
	}
}

// From kanade-docs/src/pages/docs/tutorial/1-variables.md: builtin
// conversions called as "【TYPE】の〈変換〉(...)" — the same TYPE-の
// collapse as above, but here TYPE ("文字列") is *not* a registered class
// at all, so the fix has to fall back to a plain global function call
// rather than a static method lookup (see the CallExpression case in both
// vm/eval_expr.go and bytecode/compiler.go).
func TestKanadeBuiltinConversionViaTypeName(t *testing.T) {
	interp, err := runKanade(t, `『文字』を【文字列】の〈文字列に〉(123)にしよう
『文字』を出力しよう
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Join(interp.Output, "|") != "123" {
		t.Errorf("Output = %v, want [123]", interp.Output)
	}
}

// From kanade-docs/src/pages/docs/tutorial/6-collections.md: indexing with
// "番目" (an ordinal particle missing from lexer/kanade's PARTICLE regex
// until this pass, so it fell through to a generic IDENT token instead —
// silently starting a bogus extra SOV component) and the decorative
// trailing "の値" after an index (kanade-docs connects it with the member
// particle, unlike 하자's bare "1번째 값" with no particle at all).
func TestKanadeIndexingWithOrdinalAndValueSuffix(t *testing.T) {
	interp, err := runKanade(t, `『果物』を【(文字列)リスト】の【「りんご」,「ぶどう」,「みかん」】にしよう
枠「1番目の果物:{『果物』の1番目の値}」を出力しよう
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Join(interp.Output, "|") != "1番目の果物:りんご" {
		t.Errorf("Output = %v, want [1番目の果物:りんご]", interp.Output)
	}
}

// From kanade-docs/src/pages/docs/tutorial/3-strings.md: the four string
// pseudo-methods, under kanade-docs' actual names (not invented ones).
func TestKanadeStringPseudoMethods(t *testing.T) {
	src := `『文章』を「りんごなしりんご」にしよう
『欠片』を『文章』の〈切り取り〉(1,2)にしよう
『変わった文章』を『文章』の〈入れ替え〉(「りんご」,「ぶどう」)にしよう
『リスト』を『文章』の〈分割〉(「な」)にしよう
枠「{『欠片』}|{『変わった文章』}|{『リスト』}|{『文章』の〈含むか確認〉(「りんご」)}」を出力しよう
`
	interp, err := runKanade(t, src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "りん|ぶどうなしぶどう|[りんご, しりんご]|真"
	if strings.Join(interp.Output, "|") != want {
		t.Errorf("Output = %v, want %q", interp.Output, want)
	}

	vm, err := runKanadeBytecode(t, src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Join(vm.Output, "|") != want {
		t.Errorf("Output = %v, want %q", vm.Output, want)
	}
}

// From kanade-docs/src/pages/docs/oop/1-classes.md (spec 3.8 Dynamic
// Reflection): `<'변수'>()`/`〈『変数』〉()` looks up the method to call by
// the *value* of a variable at runtime. Tree-walker only — bytecode
// explicitly doesn't support this (see bytecode.Compiler's doc comment).
func TestKanadeDynamicReflection(t *testing.T) {
	interp, err := runKanade(t, `【注文】を設計しよう:
    〈キャンセルする〉を作ろう():
        「注文がキャンセルされました。」を出力しよう

『注文書』を【注文】の新しい【注文】()にしよう
『行動名』を【文字列】の「キャンセルする」にしよう
『注文書』の〈『行動名』〉()を実行しよう
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Join(interp.Output, "|") != "注文がキャンセルされました。" {
		t.Errorf("Output = %v, want [注文がキャンセルされました。]", interp.Output)
	}
}
