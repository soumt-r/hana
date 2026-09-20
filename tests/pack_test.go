package tests

// `hana pack` makes one executable out of a program: hana-runtime with the
// bytecode (and the native libraries the program loads) appended. These tests
// build the real hana and hana-runtime, pack programs, and run the results from
// folders that have nothing else in them.

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/soumt-r/hana/pkg"
)

var (
	packOnce  sync.Once
	packHana  string
	packRun   string
	packError error
)

// packTools builds hana and hana-runtime once.
func packTools(t *testing.T) (hana, hanaRuntime string) {
	t.Helper()
	packOnce.Do(func() {
		// Tests change the working folder, so find the module from this file's place.
		_, thisFile, _, _ := runtime.Caller(0)
		root := filepath.Join(filepath.Dir(thisFile), "..")
		dir := filepath.Join(root, "tests", "testdata", "packbuild")
		os.MkdirAll(dir, 0o755)
		ext := ""
		if runtime.GOOS == "windows" {
			ext = ".exe"
		}
		packHana = filepath.Join(dir, "hana"+ext)
		packRun = filepath.Join(dir, "hana-runtime"+ext)
		for out, pkgPath := range map[string]string{packHana: ".", packRun: "./cmd/hana-runtime"} {
			// The release build, so the size checks below mean what users get.
			cmd := exec.Command("go", "build", "-ldflags=-s -w", "-trimpath", "-o", out, pkgPath)
			cmd.Dir = root
			if msg, err := cmd.CombinedOutput(); err != nil {
				packError = fmt.Errorf("building %s: %v\n%s", pkgPath, err, msg)
				return
			}
		}
	})
	if packError != nil {
		t.Fatal(packError)
	}
	return packHana, packRun
}

// packProgram compiles source (named file, in the current folder) into an
// executable in a new empty folder and returns its path; flags go to hana pack.
func packProgram(t *testing.T, file, source string, flags ...string) (exe string, err error) {
	t.Helper()
	hana, _ := packTools(t)
	if err := os.WriteFile(file, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	exe = filepath.Join(t.TempDir(), "packed"+filepath.Ext(hana))
	out, err := exec.Command(hana, append([]string{"pack", file, "-o", exe}, flags...)...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%v: %s", err, out)
	}
	return exe, nil
}

func runPacked(t *testing.T, exe, stdin string) (stdout string, err error) {
	t.Helper()
	cmd := exec.Command(exe)
	cmd.Dir = filepath.Dir(exe)
	cmd.Stdin = strings.NewReader(stdin)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err = cmd.Run()
	return out.String(), err
}

func TestPackedProgramRunsOnItsOwn(t *testing.T) {
	inTempDir(t, nil)
	exe, err := packProgram(t, "hello.hj", "'이름'을 [문자열]로 입력받자\n틀\"안녕, {'이름'}!\"를 출력하자\n")
	if err != nil {
		t.Fatal(err)
	}
	if out, err := runPacked(t, exe, "하나\n"); err != nil || out != "안녕, 하나!\n" {
		t.Errorf("got %q (%v)", out, err)
	}
	if info, _ := os.Stat(exe); info.Size() > 8<<20 {
		t.Errorf("a packed program should stay small, this one is %d bytes", info.Size())
	}
}

func TestPackedKanadeProgram(t *testing.T) {
	inTempDir(t, nil)
	exe, err := packProgram(t, "hello.knd", "「こんにちは」を出力しよう\n")
	if err != nil {
		t.Fatal(err)
	}
	if out, err := runPacked(t, exe, ""); err != nil || out != "こんにちは\n" {
		t.Errorf("got %q (%v)", out, err)
	}
}

func TestPackedProgramReportsRuntimeErrors(t *testing.T) {
	inTempDir(t, nil)
	exe, err := packProgram(t, "boom.hj", "새로운 [오류](\"터졌어요\")를 발생시키자\n")
	if err != nil {
		t.Fatal(err)
	}
	out, err := runPacked(t, exe, "")
	if err == nil || !strings.Contains(out, "터졌어요") {
		t.Errorf("want a failing exit and the message, got %q (%v)", out, err)
	}
}

const echoMain = "[echo]에서 <네이티브_Echo>를 가져오자\n<네이티브_Echo>(1, \"가나\", [2, 참])를 출력하자\n"

// By default the native library is written beside the executable, as
// libraries/<package><extension>.
func TestPackWritesNativeLibrariesBesideTheExecutable(t *testing.T) {
	echoPackage(t, "")
	exe, err := packProgram(t, "main.hj", echoMain)
	if err != nil {
		t.Fatal(err)
	}
	library := filepath.Join(filepath.Dir(exe), "libraries", "echo"+pkg.LibraryExt(runtime.GOOS))
	if _, err := os.Stat(library); err != nil {
		t.Fatalf("the library should be written beside the executable: %v", err)
	}
	if info, _ := os.Stat(exe); info.Size() > 8<<20 {
		t.Errorf("without the library inside, the executable stays small; it is %d bytes", info.Size())
	}
	if out, err := runPacked(t, exe, ""); err != nil || strings.TrimSpace(out) != "[1, 가나, [2, 참]]" {
		t.Errorf("got %q (%v)", out, err)
	}
	// Without its libraries/ folder the program cannot load the library.
	if err := os.RemoveAll(filepath.Join(filepath.Dir(exe), "libraries")); err != nil {
		t.Fatal(err)
	}
	if out, err := runPacked(t, exe, ""); err == nil || !strings.Contains(out, "echo") {
		t.Errorf("want a failure naming the package, got %q (%v)", out, err)
	}
}

func TestPackEmbedsNativeLibrariesOnRequest(t *testing.T) {
	echoPackage(t, "")
	exe, err := packProgram(t, "main.hj", echoMain, "--embed")
	if err != nil {
		t.Fatal(err)
	}
	if entries, _ := os.ReadDir(filepath.Dir(exe)); len(entries) != 1 {
		t.Errorf("--embed should leave only the executable, found %d files", len(entries))
	}
	// The folder the program runs in has no packages/ of its own.
	if out, err := runPacked(t, exe, ""); err != nil || strings.TrimSpace(out) != "[1, 가나, [2, 참]]" {
		t.Errorf("got %q (%v)", out, err)
	}
}

func TestPackNeedsTheLibraryOfTheTargetPlatform(t *testing.T) {
	echoPackage(t, `{"name": "echo", "native": {"plan9-mips": {"file": "native/echo.bin"}}}`)
	_, err := packProgram(t, "main.hj", "[echo]에서 <네이티브_Echo>를 가져오자\n<네이티브_Echo>(1)을 출력하자\n")
	if err == nil || !strings.Contains(err.Error(), "네이티브 라이브러리가 없어요") {
		t.Errorf("want a message about the missing library, got %v", err)
	}
}

func TestRuntimeWithoutAProgramSaysSo(t *testing.T) {
	_, hanaRuntime := packTools(t)
	out, err := exec.Command(hanaRuntime).CombinedOutput()
	if err == nil || !strings.Contains(string(out), "does not contain a program") {
		t.Errorf("got %q (%v)", out, err)
	}
}
