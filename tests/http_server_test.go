package tests

// The http_server package end to end: the real package folder (manifest, Hari and
// Kanade entries) plus its native library built from packages/http_server/native
// (Go's net/http), driven by real HTTP requests. Needs a C toolchain, like
// native_plugin_test.go, and is skipped without one.

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/soumt-r/hana/pkg"
)

var (
	httpLibOnce sync.Once
	httpLibPath string
	httpLibErr  error
	httpRuns    int
)

func httpServerLibrary(t *testing.T) string {
	t.Helper()
	httpLibOnce.Do(func() {
		if _, err := exec.LookPath("gcc"); err != nil {
			if _, err := exec.LookPath("clang"); err != nil {
				httpLibErr = fmt.Errorf("no C compiler")
				return
			}
		}
		dir, err := filepath.Abs(filepath.Join("testdata", "httpbuild"))
		if err != nil {
			httpLibErr = err
			return
		}
		os.MkdirAll(dir, 0o755)
		httpLibPath = filepath.Join(dir, "http_server-"+pkg.Platform()+pkg.LibraryExt(runtime.GOOS))
		cmd := exec.Command("go", "build", "-buildmode=c-shared", "-o", httpLibPath, ".")
		cmd.Dir = filepath.Join("..", "packages", "http_server", "native")
		cmd.Env = append(os.Environ(), "CGO_ENABLED=1")
		if out, err := cmd.CombinedOutput(); err != nil {
			httpLibErr = fmt.Errorf("building the library: %v\n%s", err, out)
		}
	})
	if httpLibErr != nil {
		t.Skipf("http_server tests need a C toolchain: %v", httpLibErr)
	}
	return httpLibPath
}

// httpServerPackage copies the real package into a fresh working folder and
// puts the freshly built library where its manifest expects it for this platform.
func httpServerPackage(t *testing.T) {
	t.Helper()
	lib, err := os.ReadFile(httpServerLibrary(t))
	if err != nil {
		t.Fatal(err)
	}
	src := filepath.Join("..", "packages", "http_server")
	httpRuns++
	dir := filepath.Join(filepath.Dir(httpLibPath), fmt.Sprintf("run-%d-%d", os.Getpid(), httpRuns))
	pkgDir := filepath.Join(dir, "packages", "http_server")
	for _, rel := range []string{"hana.pkg.json", filepath.Join("hari", "index.hr"), filepath.Join("kanade", "index.knd")} {
		data, err := os.ReadFile(filepath.Join(src, rel))
		if err != nil {
			t.Fatal(err)
		}
		dst := filepath.Join(pkgDir, rel)
		os.MkdirAll(filepath.Dir(dst), 0o755)
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	manifest, err := pkg.Load(pkgDir)
	if err != nil {
		t.Fatal(err)
	}
	native, ok := manifest.NativeFor(pkg.Platform())
	if !ok {
		t.Skipf("the manifest declares no library for %s", pkg.Platform())
	}
	dst := filepath.Join(pkgDir, filepath.FromSlash(native.File))
	os.MkdirAll(filepath.Dir(dst), 0o755)
	if err := os.WriteFile(dst, lib, 0o755); err != nil {
		t.Fatal(err)
	}
	old, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(old) })
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// serveProgram runs a hana program that starts the server in the background and
// waits until the port answers. The returned function reports how the program
// ended once the server has been shut down.
func serveProgram(t *testing.T, port int, run func() error) (base string, finished func() error) {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- run() }()
	base = fmt.Sprintf("http://127.0.0.1:%d", port)
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if resp, err := http.Get(base + "/ping"); err == nil {
			resp.Body.Close()
			return base, func() error {
				select {
				case err := <-done:
					return err
				case <-time.After(10 * time.Second):
					return fmt.Errorf("the program did not end after /stop")
				}
			}
		}
		select {
		case err := <-done:
			t.Fatalf("the program ended before the server was up: %v", err)
		case <-time.After(50 * time.Millisecond):
		}
	}
	t.Fatal("the server never came up")
	return "", nil
}

func fetch(t *testing.T, method, url, body string) (int, string, http.Header) {
	t.Helper()
	req, _ := http.NewRequest(method, url, strings.NewReader(body))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(data), resp.Header
}

const hariServer = `
[http_server]에서 <GET>과 <POST>와 <서버열기>와 <서버닫기>와 <응답>을 가져오자

<핑>를 만들자 ('요청'):
    "퐁"을 돌려주자
<안녕>를 만들자 ('요청'):
    틀"안녕 {'요청'의 "query"의 "이름"}"을 돌려주자
<메아리>를 만들자 ('요청'):
    <응답>(201, '요청'의 "body")를 돌려주자
<망가짐>를 만들자 ('요청'):
    (1 / 0)를 돌려주자
<끝>를 만들자 ('요청'):
    <서버닫기>()를 실행하자
    "안녕히"를 돌려주자

<GET>("/ping", <핑>)를 실행하자
<GET>("/hello", <안녕>)를 실행하자
<POST>("/echo", <메아리>)를 실행하자
<GET>("/broken", <망가짐>)를 실행하자
<GET>("/stop", <끝>)를 실행하자
<서버열기>(PORT)를 실행하자
"서버 끝"을 출력하자
`

const kanadeServer = `
【http_server】から〈GET〉と〈POST〉と〈サーバー起動〉と〈サーバー停止〉と〈応答〉を持ってこよう

〈ピン〉を作ろう(『リクエスト』):
    「ポン」を返そう
〈こんにちは〉を作ろう(『リクエスト』):
    枠「こんにちは {『リクエスト』の「query」の「名前」}」を返そう
〈エコー〉を作ろう(『リクエスト』):
    〈応答〉(201, 『リクエスト』の「body」)を返そう
〈壊れた〉を作ろう(『リクエスト』):
    (1 / 0)を返そう
〈終わり〉を作ろう(『リクエスト』):
    〈サーバー停止〉()を実行しよう
    「さようなら」を返そう

〈GET〉(「/ping」, 〈ピン〉)を実行しよう
〈GET〉(「/hello」, 〈こんにちは〉)を実行しよう
〈POST〉(「/echo」, 〈エコー〉)を実行しよう
〈GET〉(「/broken」, 〈壊れた〉)を実行しよう
〈GET〉(「/stop」, 〈終わり〉)を実行しよう
〈サーバー起動〉(PORT)を実行しよう
「サーバー終了」を出力しよう
`

// exercise sends the same requests to either language's server and checks it.
func exercise(t *testing.T, base, hello, helloWant, stopWant string) {
	t.Helper()
	if code, body, _ := fetch(t, "GET", base+"/hello?"+hello, ""); code != 200 || body != helloWant {
		t.Errorf("GET /hello: %d %q, want 200 %q", code, body, helloWant)
	}
	if code, _, _ := fetch(t, "GET", base+"/nope", ""); code != 404 {
		t.Errorf("an unknown path should be 404, got %d", code)
	}
	if code, _, _ := fetch(t, "POST", base+"/hello", ""); code != 404 {
		t.Errorf("a route only answers its own method, got %d", code)
	}
	if code, body, _ := fetch(t, "POST", base+"/echo", "받은 본문"); code != 201 || body != "받은 본문" {
		t.Errorf("POST /echo: %d %q, want 201 with the body back", code, body)
	}
	if code, body, _ := fetch(t, "GET", base+"/broken", ""); code != 500 || !strings.Contains(body, "Division by zero") {
		t.Errorf("a failing handler should be 500 with the reason, got %d %q", code, body)
	}

	// Many requests at once: the handlers run one at a time but none is lost.
	var wg sync.WaitGroup
	var mu sync.Mutex
	bad := 0
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, _ := http.NewRequest("GET", base+"/hello?"+hello, nil)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				mu.Lock()
				bad++
				mu.Unlock()
				return
			}
			data, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if string(data) != helloWant {
				mu.Lock()
				bad++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if bad != 0 {
		t.Errorf("%d of 30 concurrent requests went wrong", bad)
	}

	if code, body, _ := fetch(t, "GET", base+"/stop", ""); code != 200 || body != stopWant {
		t.Errorf("GET /stop: %d %q", code, body)
	}
}

type serverRunner struct {
	name string
	run  func(t *testing.T, code string) ([]string, error)
}

var hariRunners = []serverRunner{
	{"tree-walker", func(t *testing.T, code string) ([]string, error) {
		i, err := runHari(t, code)
		if i == nil {
			return nil, err
		}
		return i.Output, err
	}},
	{"bytecode", func(t *testing.T, code string) ([]string, error) {
		vm, err := runBytecode(t, code)
		if vm == nil {
			return nil, err
		}
		return vm.Output, err
	}},
}

var kanadeRunners = []serverRunner{
	{"tree-walker", func(t *testing.T, code string) ([]string, error) {
		i, err := runKanade(t, code)
		if i == nil {
			return nil, err
		}
		return i.Output, err
	}},
	{"bytecode", func(t *testing.T, code string) ([]string, error) {
		vm, err := runKanadeBytecode(t, code)
		if vm == nil {
			return nil, err
		}
		return vm.Output, err
	}},
}

// runServerTest starts the program on each engine in turn and drives it over HTTP.
func runServerTest(t *testing.T, runners []serverRunner, program, query, helloWant, stopWant, endLine string) {
	for _, r := range runners {
		t.Run(r.name, func(t *testing.T) {
			httpServerPackage(t)
			port := freePort(t)
			var out []string
			base, finished := serveProgram(t, port, func() error {
				var err error
				out, err = r.run(t, strings.Replace(program, "PORT", fmt.Sprint(port), 1))
				return err
			})
			exercise(t, base, query, helloWant, stopWant)
			if err := finished(); err != nil {
				t.Fatalf("the program should end cleanly after /stop: %v", err)
			}
			if got := strings.Join(out, "|"); got != endLine {
				t.Errorf("the line after the server call should run once the server stops, output %q, want %q", got, endLine)
			}
		})
	}
}

func TestHariHTTPServerPackage(t *testing.T) {
	runServerTest(t, hariRunners, hariServer, url.Values{"이름": {"하나"}}.Encode(), "안녕 하나", "안녕히", "서버 끝")
}

func TestKanadeHTTPServerPackage(t *testing.T) {
	runServerTest(t, kanadeRunners, kanadeServer, url.Values{"名前": {"はな"}}.Encode(), "こんにちは はな", "さようなら", "サーバー終了")
}
