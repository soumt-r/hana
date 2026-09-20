package tests

// Regression tests for ImportStatement's As field ("<이름>을 <별칭>으로
// 가져오자") on the tree-walking interpreter: two packages exporting the same name need
// a way to bind them under different local names. bcstdlib/bytecode_test.go covers
// the same behavior on the bytecode VM.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestImportAliasResolvesNameCollision(t *testing.T) {
	dir := t.TempDir()
	pkgAPath := filepath.Join(dir, "pkgA.hj")
	pkgBPath := filepath.Join(dir, "pkgB.hj")
	if err := os.WriteFile(pkgAPath, []byte(`
[숫자]를 돌려주는 <계산하기>를 만들자 ([숫자]인 '값'):
    ('값' + 1)를 돌려주자
`), 0o644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}
	if err := os.WriteFile(pkgBPath, []byte(`
[숫자]를 돌려주는 <계산하기>를 만들자 ([숫자]인 '값'):
    ('값' * 2)를 돌려주자
`), 0o644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	pkgAEscaped := strings.ReplaceAll(pkgAPath, `\`, `\\`)
	pkgBEscaped := strings.ReplaceAll(pkgBPath, `\`, `\\`)
	interp, err := runHaja(t, `
"`+pkgAEscaped+`"에서 <계산하기>를 <A계산하기>로 가져오자
"`+pkgBEscaped+`"에서 <계산하기>를 <B계산하기>로 가져오자
틀"A: {<A계산하기>(10)}"를 출력하자
틀"B: {<B계산하기>(10)}"를 출력하자
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"A: 11", "B: 20"}
	if strings.Join(interp.Output, ",") != strings.Join(want, ",") {
		t.Errorf("Output = %v, want %v", interp.Output, want)
	}
}

func TestImportAliasForBuiltinModule(t *testing.T) {
	interp, err := runHaja(t, `
[수학]에서 <올림>을 <반올림>으로 가져오자
틀"{<반올림>(3.2)}"를 출력하자
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"4"}
	if strings.Join(interp.Output, ",") != strings.Join(want, ",") {
		t.Errorf("Output = %v, want %v", interp.Output, want)
	}
}

func TestImportAliasForClass(t *testing.T) {
	dir := t.TempDir()
	utilsPath := filepath.Join(dir, "shape_utils.hj")
	if err := os.WriteFile(utilsPath, []byte(`
[도형]을 설계하자:
    '이름'을 [문자열]인 "도형"으로 정하자
`), 0o644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	utilsEscaped := strings.ReplaceAll(utilsPath, `\`, `\\`)
	interp, err := runHaja(t, `
"`+utilsEscaped+`"에서 '도형'을 '도형유틸'로 가져오자
'x'를 [도형유틸]인 새로운 [도형유틸]()로 정하자
틀"{'x'의 '이름'}"를 출력하자
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"도형"}
	if strings.Join(interp.Output, ",") != strings.Join(want, ",") {
		t.Errorf("Output = %v, want %v", interp.Output, want)
	}
}
