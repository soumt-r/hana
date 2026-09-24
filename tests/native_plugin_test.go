package tests

// Native libraries follow hana's ABI v1 (native/native.go). These tests build
// tests/testdata/nativeecho into a real shared library (.dll/.so/.dylib) and call
// it through a package, on both engines, so they need a C toolchain (gcc or
// clang); without one they are skipped.

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/soumt-r/hana/pkg"
)

var (
	echoOnce sync.Once
	echoPath string
	echoErr  error
)

// echoLibrary builds the test library once and returns its path.
func echoLibrary(t *testing.T) string {
	t.Helper()
	echoOnce.Do(func() {
		if _, err := exec.LookPath("gcc"); err != nil {
			if _, err := exec.LookPath("clang"); err != nil {
				echoErr = fmt.Errorf("no C compiler")
				return
			}
		}
		dir, err := filepath.Abs(filepath.Join("testdata", "nativeecho", "build"))
		if err != nil {
			echoErr = err
			return
		}
		os.MkdirAll(dir, 0o755)
		echoPath = filepath.Join(dir, "echo"+pkg.LibraryExt(runtime.GOOS))
		cmd := exec.Command("go", "build", "-buildmode=c-shared", "-o", echoPath, "./testdata/nativeecho")
		cmd.Env = append(os.Environ(), "CGO_ENABLED=1")
		if out, err := cmd.CombinedOutput(); err != nil {
			echoErr = fmt.Errorf("building the test library: %v\n%s", err, out)
		}
	})
	if echoErr != nil {
		t.Skipf("native library tests need a C toolchain: %v", echoErr)
	}
	return echoPath
}

const echoHari = `
[echo]에서 <네이티브_Echo>를 가져오자
[echo]에서 <네이티브_Fail>를 가져오자
[echo]에서 <네이티브_Nothing>를 가져오자
[echo]에서 <네이티브_Garbage>를 가져오자
[echo]에서 <네이티브_CallHost>를 가져오자
[echo]에서 <네이티브_CallHostFromThread>를 가져오자
[echo]에서 <네이티브_CallLater>를 가져오자
[echo]에서 <네이티브_Join>를 가져오자
`

// echoPackage lays out packages/echo/ in a fresh folder and makes it the working
// directory: a manifest that declares the built library for this platform, and
// its native file. The folder is not deleted afterwards — Windows cannot delete
// a library that is still loaded — so it lives under the ignored build folder.
func echoPackage(t *testing.T, manifest string) {
	t.Helper()
	lib, err := os.ReadFile(echoLibrary(t))
	if err != nil {
		t.Fatal(err)
	}
	if manifest == "" {
		manifest = fmt.Sprintf(`{"name": "echo", "native": {%q: {"file": "native/echo%s"}}}`, pkg.Platform(), pkg.LibraryExt(runtime.GOOS))
	}
	echoRuns++
	dir := filepath.Join(filepath.Dir(echoPath), fmt.Sprintf("run-%d-%d", os.Getpid(), echoRuns))
	files := map[string]string{
		"packages/echo/hana.pkg.json":                              manifest,
		"packages/echo/native/echo" + pkg.LibraryExt(runtime.GOOS): string(lib),
	}
	for name, content := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	old, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(old) })
}

var echoRuns int

type engineResult struct {
	out []string
	err error
}

// engines runs a Hari program on the tree-walker and on the bytecode VM.
func engines(t *testing.T, code string) map[string]engineResult {
	t.Helper()
	res := map[string]engineResult{}
	interp, err := runHari(t, code)
	res["tree-walker"] = engineResult{err: err}
	if interp != nil {
		res["tree-walker"] = engineResult{out: interp.Output, err: err}
	}
	vm, err := runBytecode(t, code)
	res["bytecode"] = engineResult{err: err}
	if vm != nil {
		res["bytecode"] = engineResult{out: vm.Output, err: err}
	}
	return res
}

func TestNativeLibraryRoundTripsValues(t *testing.T) {
	echoPackage(t, "")
	for engine, r := range engines(t, echoHari+"<네이티브_Echo>(1, \"가나\", [2, 참, 비어있음], {\"k\": 3.5})를 출력하자\n") {
		if r.err != nil {
			t.Fatalf("%s: %v", engine, r.err)
		}
		if got, want := strings.Join(r.out, "|"), "[1, 가나, [2, 참, 비어있음], {k: 3.5}]"; got != want {
			t.Errorf("%s: got %q, want %q", engine, got, want)
		}
	}
}

func TestNativeLibraryErrorsAndNull(t *testing.T) {
	echoPackage(t, "")
	for engine, r := range engines(t, echoHari+"<네이티브_Fail>()를 출력하자\n") {
		if r.err == nil || !strings.Contains(r.err.Error(), "'Fail' failed: boom") {
			t.Errorf("%s: a library error should surface with its message, got %v", engine, r.err)
		}
	}
	for engine, r := range engines(t, echoHari+"<네이티브_Nothing>()을 출력하자\n") {
		if r.err != nil || strings.Join(r.out, "|") != "비어있음" {
			t.Errorf("%s: NULL should read as 비어있음, got %v %v", engine, r.out, r.err)
		}
	}
	for engine, r := range engines(t, echoHari+"<네이티브_Garbage>()를 출력하자\n") {
		if r.err == nil || !strings.Contains(r.err.Error(), "valid JSON") {
			t.Errorf("%s: a non-JSON answer should be reported, got %v", engine, r.err)
		}
	}
}

func TestNativeLibraryCallsBackIntoHana(t *testing.T) {
	echoPackage(t, "")
	program := echoHari + `
<두배>를 만들자 ([숫자]인 '값'):
    ('값' * 2)를 돌려주자
<네이티브_CallHost>(<두배>, 21)를 출력하자
<네이티브_CallHostFromThread>(<두배>, 5)를 출력하자
`
	for engine, r := range engines(t, program) {
		if r.err != nil {
			t.Fatalf("%s: %v", engine, r.err)
		}
		if got := strings.Join(r.out, "|"); got != "42|10" {
			t.Errorf("%s: callbacks: got %q, want 42|10", engine, got)
		}
	}
}

func TestNativeCallbackErrorsComeBackAsErrors(t *testing.T) {
	echoPackage(t, "")
	program := echoHari + `
<나누기>를 만들자 ([숫자]인 '값'):
    ('값' / 0)를 돌려주자
<네이티브_CallHost>(<나누기>, 1)을 출력하자
`
	for engine, r := range engines(t, program) {
		if r.err == nil || !strings.Contains(r.err.Error(), "Division by zero") {
			t.Errorf("%s: an error inside the callback should reach the caller, got %v", engine, r.err)
		}
	}
}

// Passing the same function again reuses its id, so a program that hands a
// callback to native code over and over does not register a new one each time.
func TestTheSameFunctionKeepsItsCallbackID(t *testing.T) {
	echoPackage(t, "")
	program := echoHari + `
<두배>를 만들자 ([숫자]인 '값'):
    ('값' * 2)를 돌려주자
<네이티브_Echo>(<두배>, <두배>)를 출력하자
<네이티브_Echo>(<두배>)를 출력하자
`
	same := regexp.MustCompile(`^\[\{\$fn: (cb_\d+)\}, \{\$fn: (cb_\d+)\}\]$`)
	for engine, r := range engines(t, program) {
		if r.err != nil || len(r.out) != 2 {
			t.Fatalf("%s: %v %v", engine, r.out, r.err)
		}
		m := same.FindStringSubmatch(r.out[0])
		if m == nil || m[1] != m[2] {
			t.Errorf("%s: the two arguments should share one id: %q", engine, r.out[0])
			continue
		}
		if !strings.Contains(r.out[1], m[1]) {
			t.Errorf("%s: a later call should reuse the id %s: %q", engine, m[1], r.out[1])
		}
	}
}

// A callback from another thread must not run hana code while the program itself
// is running: it waits until the program is inside a native call (Join here).
func TestCallbacksWaitForTheProgramToBlock(t *testing.T) {
	echoPackage(t, "")
	program := echoHari + `
<틱>를 만들자 ():
    "콜백"을 출력하자

<네이티브_CallLater>(<틱>, 3)를 실행하자
'i'를 0으로 정하자
('i' < 150000)인 동안 반복하자:
    'i'에 1을 더하자
"메인 끝"을 출력하자
<네이티브_Join>()을 실행하자
"끝"을 출력하자
`
	for engine, r := range engines(t, program) {
		if r.err != nil {
			t.Fatalf("%s: %v", engine, r.err)
		}
		if got, want := strings.Join(r.out, "|"), "메인 끝|콜백|콜백|콜백|끝"; got != want {
			t.Errorf("%s: got %q, want %q", engine, got, want)
		}
	}
}

func TestNativeLibraryMissingFunction(t *testing.T) {
	echoPackage(t, "")
	for engine, r := range engines(t, "[echo]에서 <네이티브_Nope>를 가져오자\n") {
		if r.err == nil || !strings.Contains(r.err.Error(), "'Nope' not found") {
			t.Errorf("%s: a function the library lacks: %v", engine, r.err)
		}
	}
}

func TestPackageManifestProblems(t *testing.T) {
	echoPackage(t, `{"name": "echo", "native": {"plan9-mips": {"file": "native/x.so"}}}`)
	for engine, r := range engines(t, echoHari) {
		if r.err == nil || !strings.Contains(r.err.Error(), "no native library for "+pkg.Platform()) {
			t.Errorf("%s: a platform without a native library: %v", engine, r.err)
		}
	}
	echoPackage(t, `{"name": "echo", "native": {"`+pkg.Platform()+`": {"file": "native/missing.bin"}}}`)
	for engine, r := range engines(t, echoHari) {
		if r.err == nil || !strings.Contains(r.err.Error(), "no native library for") {
			t.Errorf("%s: a declared file that is not there: %v", engine, r.err)
		}
	}
	echoPackage(t, `{"nome": "echo"}`)
	for engine, r := range engines(t, echoHari) {
		if r.err == nil || !strings.Contains(r.err.Error(), "not valid") {
			t.Errorf("%s: an invalid manifest: %v", engine, r.err)
		}
	}
}

func TestPackageManifestChoosesTheEntryPoint(t *testing.T) {
	inTempDir(t, map[string]string{
		"packages/greeter/hana.pkg.json": `{"name": "greeter", "entry": {"hari": "src/main.hr", "kanade": "src/main.knd"}}`,
		"packages/greeter/src/main.hr":   "<인사>를 만들자 ():\n    \"안녕\"을 출력하자\n",
		"packages/greeter/src/main.knd":  "〈挨拶〉を作ろう():\n    「こんにちは」を出力しよう\n",
		"packages/greeter/hari/index.hr": "<인사>를 만들자 ():\n    \"틀렸어요\"를 출력하자\n",
	})
	for engine, r := range engines(t, "[greeter]에서 <인사>를 가져오자\n<인사>()를 실행하자\n") {
		if r.err != nil || strings.Join(r.out, "|") != "안녕" {
			t.Errorf("hari %s: %v %v", engine, r.out, r.err)
		}
	}
	k, err := runKanade(t, "【greeter】から〈挨拶〉を持ってこよう\n〈挨拶〉()を実行しよう\n")
	if err != nil || strings.Join(k.Output, "|") != "こんにちは" {
		t.Errorf("kanade tree-walker: %v %v", k.Output, err)
	}
	kb, err := runKanadeBytecode(t, "【greeter】から〈挨拶〉を持ってこよう\n〈挨拶〉()を実行しよう\n")
	if err != nil || strings.Join(kb.Output, "|") != "こんにちは" {
		t.Errorf("kanade bytecode: %v %v", kb.Output, err)
	}
}

func TestPackageImportsSeveralItemsAndAll(t *testing.T) {
	inTempDir(t, map[string]string{
		"packages/tools/hari/index.hr": "<하나>를 만들자 ():\n    \"1\"을 출력하자\n<둘>을 만들자 ():\n    \"2\"를 출력하자\n",
	})
	for engine, r := range engines(t, "[tools]에서 <하나>와 <둘>을 가져오자\n<하나>()를 실행하자\n<둘>()을 실행하자\n") {
		if r.err != nil || strings.Join(r.out, "|") != "1|2" {
			t.Errorf("%s list: %v %v", engine, r.out, r.err)
		}
	}
	for engine, r := range engines(t, "[tools]에서 전부 가져오자\n<둘>()을 실행하자\n") {
		if r.err != nil || strings.Join(r.out, "|") != "2" {
			t.Errorf("%s all: %v %v", engine, r.out, r.err)
		}
	}
}

// A package that is not in the project's ./packages is found in the shared
// roots ($HANA_PACKAGES here, the folder next to the executable in a real
// install), so shipped packages work from any folder; a project's own package of
// the same name wins.
func TestSharedPackagesAreFoundFromAnyFolder(t *testing.T) {
	shared := t.TempDir()
	if err := os.MkdirAll(filepath.Join(shared, "greeter", "hari"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(shared, "greeter", "hari", "index.hr"), []byte("<인사>를 만들자 ():\n    \"공유 패키지\"를 출력하자\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(pkg.EnvPackages, shared)

	inTempDir(t, map[string]string{})
	program := "[greeter]에서 <인사>를 가져오자\n<인사>()를 실행하자\n"
	for engine, r := range engines(t, program) {
		if r.err != nil || strings.Join(r.out, "|") != "공유 패키지" {
			t.Errorf("%s from a folder without packages: %v %v", engine, r.out, r.err)
		}
	}

	if err := os.MkdirAll(filepath.Join("packages", "greeter", "hari"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join("packages", "greeter", "hari", "index.hr"), []byte("<인사>를 만들자 ():\n    \"프로젝트 패키지\"를 출력하자\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for engine, r := range engines(t, program) {
		if r.err != nil || strings.Join(r.out, "|") != "프로젝트 패키지" {
			t.Errorf("%s: the project's own package should win: %v %v", engine, r.out, r.err)
		}
	}
}
