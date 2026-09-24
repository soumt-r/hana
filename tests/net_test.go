package tests

// [소켓] and [HTTP] talk to real peers, so these tests start them on the
// loopback interface: a Go listener for the script to connect to, a script that
// listens for a Go client, and an httptest server for the HTTP calls. Every case
// runs on both engines.

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/soumt-r/hana/std/stdimpl"
)

// echoServer answers each connection's first line with "받았어요: <line>" and
// closes it; it serves as many connections as come.
func echoServer(t *testing.T) (port int) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				line, _ := bufio.NewReader(conn).ReadString('\n')
				fmt.Fprintf(conn, "받았어요: %s", line)
			}()
		}
	}()
	return ln.Addr().(*net.TCPAddr).Port
}

func TestSocketClientTalksToAServer(t *testing.T) {
	port := echoServer(t)
	code := fmt.Sprintf(`
[소켓]에서 전부 가져오자
'통로'를 <연결하기>("127.0.0.1", %d)로 정하자
<보내기>('통로', "안녕\n")를 실행하자
<줄받기>('통로')를 출력하자
<줄받기>('통로')를 출력하자
<받기>('통로', 10)을 출력하자
<닫기>('통로')를 실행하자
`, port)
	for engine, r := range engines(t, code) {
		if r.err != nil {
			t.Fatalf("%s: %v", engine, r.err)
		}
		if got, want := strings.Join(r.out, "|"), "받았어요: 안녕|비어있음|"; got != want {
			t.Errorf("%s: got %q, want %q", engine, got, want)
		}
	}
}

// serveScript runs a script that listens on port, and once it is up lets talk
// use a client connection to it. It returns what the script printed.
func serveScript(t *testing.T, engine, code string, port int, talk func(net.Conn)) []string {
	t.Helper()
	type result struct {
		out []string
		err error
	}
	done := make(chan result, 1)
	go func() {
		var r engineResult
		if engine == "bytecode" {
			vm, err := runBytecode(t, code)
			r.err = err
			if vm != nil {
				r.out = vm.Output
			}
		} else {
			interp, err := runHari(t, code)
			r.err = err
			if interp != nil {
				r.out = interp.Output
			}
		}
		done <- result{r.out, r.err}
	}()
	var conn net.Conn
	var err error
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		if conn, err = net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port)); err == nil {
			break
		}
	}
	if err != nil {
		t.Fatalf("%s: the script never started listening: %v", engine, err)
	}
	talk(conn)
	conn.Close()
	select {
	case r := <-done:
		if r.err != nil {
			t.Fatalf("%s: %v", engine, r.err)
		}
		return r.out
	case <-time.After(5 * time.Second):
		t.Fatalf("%s: the script did not finish", engine)
		return nil
	}
}

func TestSocketServerTalksToAClient(t *testing.T) {
	for _, engine := range []string{"tree-walker", "bytecode"} {
		port := freePort(t)
		code := fmt.Sprintf(`
[소켓]에서 전부 가져오자
'서버'를 <듣기>(%d)로 정하자
'손님'을 <받아들이기>('서버')로 정하자
'줄'을 [문자열]인 <줄받기>('손님')로 정하자
<보내기>('손님', 틀"메아리: {'줄'}\n")를 실행하자
<닫기>('손님')를 실행하자
<닫기>('서버')를 실행하자
"끝"을 출력하자
`, port)
		var reply string
		out := serveScript(t, engine, code, port, func(conn net.Conn) {
			fmt.Fprint(conn, "하나\n")
			reply, _ = bufio.NewReader(conn).ReadString('\n')
		})
		if reply != "메아리: 하나\n" || strings.Join(out, "|") != "끝" {
			t.Errorf("%s: client got %q, script printed %v", engine, reply, out)
		}
	}
}

func TestSocketReceiveKeepsCharactersWhole(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			fmt.Fprint(conn, "가나다라")
			conn.Close()
		}
	}()
	code := fmt.Sprintf(`
[소켓]에서 전부 가져오자
'통로'를 <연결하기>("127.0.0.1", %d)로 정하자
<받기>('통로', 2)를 출력하자
<받기>('통로', 100)을 출력하자
<받기>('통로', 100)을 출력하자
`, ln.Addr().(*net.TCPAddr).Port)
	for engine, r := range engines(t, code) {
		if r.err != nil {
			t.Fatalf("%s: %v", engine, r.err)
		}
		if got, want := strings.Join(r.out, "|"), "가나|다라|"; got != want {
			t.Errorf("%s: got %q, want %q", engine, got, want)
		}
	}
}

func TestSocketTimeoutAndAddress(t *testing.T) {
	quiet, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer quiet.Close()
	var held sync.WaitGroup
	held.Add(1)
	go func() {
		defer held.Done()
		for {
			conn, err := quiet.Accept()
			if err != nil {
				return
			}
			defer conn.Close() // never answers
		}
	}()
	code := fmt.Sprintf(`
[소켓]에서 전부 가져오자
'서버'를 <듣기>(0)로 정하자
<주소>('서버')를 출력하자
<닫기>('서버')를 실행하자
'통로'를 <연결하기>("127.0.0.1", %d)로 정하자
<시간제한>('통로', 0.2)를 실행하자
<줄받기>('통로')를 출력하자
`, quiet.Addr().(*net.TCPAddr).Port)
	for engine, r := range engines(t, code) {
		if r.err == nil || !strings.Contains(r.err.Error(), "did not finish in the time set") {
			t.Errorf("%s: want a timeout, got %v", engine, r.err)
		}
		if len(r.out) != 1 || !strings.HasPrefix(r.out[0], "127.0.0.1:") {
			t.Errorf("%s: the listening address should be printed first, got %v", engine, r.out)
		}
	}
	quiet.Close()
	held.Wait()
}

func TestSocketMisuseIsReported(t *testing.T) {
	closed := freePort(t)
	cases := []struct{ call, want string }{
		{`<보내기>(999, "x")`, "Socket 999 does not exist"},
		{`<닫기>(999)`, "Socket 999 does not exist"},
		{`<연결하기>("127.0.0.1", 70000)`, "port must be a whole number"},
		{`<연결하기>("127.0.0.1", 1.5)`, "must be a whole number"},
		{fmt.Sprintf(`<연결하기>("127.0.0.1", %d)`, closed), "Could not connect to"},
		{`<시간제한>(<듣기>(0), 4000)`, "0 and 3600"},
		{`<받기>(<듣기>(0), 1)`, "listening socket cannot send or receive"},
		{`<받아들이기>(<연결하기>("127.0.0.1", ` + fmt.Sprint(echoServer(t)) + `))`, "connected socket cannot accept"},
	}
	for _, c := range cases {
		for engine, r := range engines(t, "[소켓]에서 전부 가져오자\n"+c.call+"를 출력하자\n") {
			if r.err == nil || !strings.Contains(r.err.Error(), c.want) {
				t.Errorf("%s %s: want an error containing %q, got %v", engine, c.call, c.want, r.err)
			}
		}
	}
	// A closed socket is gone.
	port := echoServer(t)
	code := fmt.Sprintf("[소켓]에서 전부 가져오자\n'통로'를 <연결하기>(\"127.0.0.1\", %d)로 정하자\n<닫기>('통로')를 실행하자\n<닫기>('통로')를 실행하자\n", port)
	for engine, r := range engines(t, code) {
		if r.err == nil || !strings.Contains(r.err.Error(), "does not exist or is already closed") {
			t.Errorf("%s: closing twice: got %v", engine, r.err)
		}
	}
}

func TestNetworkAccessCanBeTurnedOff(t *testing.T) {
	old := stdimpl.NetAccess
	stdimpl.NetAccess = stdimpl.DenyNet
	t.Cleanup(func() { stdimpl.NetAccess = old })
	for _, code := range []string{
		"[소켓]에서 <연결하기>를 가져오자\n<연결하기>(\"127.0.0.1\", 80)를 출력하자\n",
		"[소켓]에서 <듣기>를 가져오자\n<듣기>(0)을 출력하자\n",
		"[HTTP]에서 <가져오기>를 가져오자\n<가져오기>(\"http://127.0.0.1/\")를 출력하자\n",
	} {
		for engine, r := range engines(t, code) {
			if r.err == nil || !strings.Contains(r.err.Error(), "Network access is turned off") {
				t.Errorf("%s %q: want a refusal, got %v", engine, code, r.err)
			}
		}
	}
}

func httpTestServer(t *testing.T) string {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("X-Agent", r.Header.Get("User-Agent")+"/"+r.Header.Get("X-Mine"))
		fmt.Fprint(w, "안녕, "+r.URL.Query().Get("이름"))
	})
	mux.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 1<<10)
		n, _ := r.Body.Read(buf)
		w.WriteHeader(201)
		fmt.Fprintf(w, "%s %s %s", r.Method, r.Header.Get("Content-Type"), buf[:n])
	})
	mux.HandleFunc("/moved", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/hello?이름=이사", 302) })
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server.URL
}

func TestHTTPRequests(t *testing.T) {
	base := httpTestServer(t)
	code := fmt.Sprintf(`
[HTTP]에서 <가져오기>와 <보내기>와 <요청>을 가져오자
'응답'을 <가져오기>("%[1]s/hello?%%EC%%9D%%B4%%EB%%A6%%84=%%ED%%95%%98%%EB%%82%%98", {"X-Mine": "값"})로 정하자
'응답'의 "status"를 출력하자
'응답'의 "body"를 출력하자
'응답'의 "headers"의 "content-type"을 출력하자
'응답'의 "headers"의 "x-agent"를 출력하자
'글'을 <보내기>("%[1]s/echo", "본문", {"Content-Type": "text/plain"})로 정하자
'글'의 "status"를 출력하자
'글'의 "body"를 출력하자
<요청>("put", "%[1]s/echo", "바꿈")의 "body"를 출력하자
<가져오기>("%[1]s/moved")의 "body"를 출력하자
<가져오기>("%[1]s/없는곳")의 "status"를 출력하자
`, base)
	for engine, r := range engines(t, code) {
		if r.err != nil {
			t.Fatalf("%s: %v", engine, r.err)
		}
		want := "200|안녕, 하나|text/plain; charset=utf-8|hari/값|201|POST text/plain 본문|PUT  바꿈|안녕, 이사|404"
		if got := strings.Join(r.out, "|"); got != want {
			t.Errorf("%s:\n got %q\nwant %q", engine, got, want)
		}
	}
}

func TestHTTPProblemsAreReported(t *testing.T) {
	base := httpTestServer(t)
	dead := freePort(t)
	cases := []struct{ call, want string }{
		{`<가져오기>("ftp://example.com/")`, "is not an http or https address"},
		{`<가져오기>("주소아님")`, "is not an http or https address"},
		{fmt.Sprintf(`<가져오기>("http://127.0.0.1:%d/")`, dead), "Could not send a request to"},
		{`<요청>("파쇄", "` + base + `/hello")`, "is not a method that can be used"},
		{`<가져오기>("` + base + `/hello", 5)`, "must be a dictionary"},
		{`<가져오기>("` + base + `/hello", {"a": 1})`, "dictionary of strings"},
		{`<보내기>("` + base + `/echo")`, "between 2 and 3"},
	}
	for _, c := range cases {
		for engine, r := range engines(t, "[HTTP]에서 전부 가져오자\n"+c.call+"를 출력하자\n") {
			if r.err == nil || !strings.Contains(r.err.Error(), c.want) {
				t.Errorf("%s %s: want an error containing %q, got %v", engine, c.call, c.want, r.err)
			}
		}
	}
}

func TestNetworkModulesInKanade(t *testing.T) {
	base := httpTestServer(t)
	code := "【HTTP】から〈取得〉を持ってこよう\n〈取得〉(「" + base + "/hello」)の「status」を出力しよう\n"
	tw, err := runKanade(t, code)
	if err != nil {
		t.Fatalf("tree-walker: %v", err)
	}
	bc, err := runKanadeBytecode(t, code)
	if err != nil {
		t.Fatalf("bytecode: %v", err)
	}
	if strings.Join(tw.Output, "") != "200" || strings.Join(bc.Output, "") != "200" {
		t.Errorf("got %v and %v", tw.Output, bc.Output)
	}
	sockets := "【ソケット】から〈待ち受け〉と〈アドレス〉と〈閉じる〉を持ってこよう\n『待機』を【数字】の〈待ち受け〉(0)にしよう\n〈アドレス〉(『待機』)を出力しよう\n〈閉じる〉(『待機』)を実行しよう\n"
	tw, err = runKanade(t, sockets)
	if err != nil || len(tw.Output) != 1 || !strings.HasPrefix(tw.Output[0], "127.0.0.1:") {
		t.Errorf("sockets in Kanade: %v %v", tw.Output, err)
	}
}

// rawHTTPServer answers the n-th connection with the n-th canned response (last
// one repeated) and records the request head and body it received. The answers
// are written by hand so the client's parsing is tested against exact bytes.
func rawHTTPServer(t *testing.T, responses ...string) (base string, requests func() []string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	var mu sync.Mutex
	var got []string
	go func() {
		for n := 0; ; n++ {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			reply := responses[min(n, len(responses)-1)]
			go func() {
				defer conn.Close()
				r := bufio.NewReader(conn)
				var head strings.Builder
				length := 0
				for {
					line, err := r.ReadString('\n')
					if err != nil {
						return
					}
					head.WriteString(line)
					if strings.HasPrefix(strings.ToLower(line), "content-length:") {
						fmt.Sscanf(strings.TrimSpace(line[len("content-length:"):]), "%d", &length)
					}
					if line == "\r\n" {
						break
					}
				}
				body := make([]byte, length)
				r.Read(body)
				mu.Lock()
				got = append(got, head.String()+string(body))
				mu.Unlock()
				fmt.Fprint(conn, reply)
			}()
		}
	}()
	return "http://" + ln.Addr().String(), func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), got...)
	}
}

func TestHTTPClientReadsEveryKindOfBody(t *testing.T) {
	cases := []struct{ name, response, want string }{
		{"길이가 정해진 본문", "HTTP/1.1 200 OK\r\nContent-Length: 6\r\n\r\n안녕", "200|안녕"},
		{"조각으로 나뉜 본문", "HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\n\r\n3;ext=1\r\n안\r\n3\r\n녕\r\n0\r\nX-Trailer: 1\r\n\r\n", "200|안녕"},
		{"끝까지 읽는 본문", "HTTP/1.0 200 OK\r\nConnection: close\r\n\r\n닫힐 때까지", "200|닫힐 때까지"},
		{"100 Continue 뒤의 답", "HTTP/1.1 100 Continue\r\n\r\nHTTP/1.1 201 Created\r\nContent-Length: 2\r\n\r\nok", "201|ok"},
		{"본문 없는 답", "HTTP/1.1 204 No Content\r\n\r\n", "204|"},
		{"오류 상태도 답이에요", "HTTP/1.1 500 Oops\r\nContent-Length: 6\r\n\r\n실패", "500|실패"},
	}
	for _, c := range cases {
		base, _ := rawHTTPServer(t, c.response)
		code := fmt.Sprintf("[HTTP]에서 <가져오기>를 가져오자\n'답'을 <가져오기>(\"%s/\")로 정하자\n틀\"{'답'의 \"status\"}|{'답'의 \"body\"}\"를 출력하자\n", base)
		for engine, r := range engines(t, code) {
			if r.err != nil || strings.Join(r.out, "") != c.want {
				t.Errorf("%s %s: got %q (%v), want %q", engine, c.name, r.out, r.err, c.want)
			}
		}
	}
}

func TestHTTPClientFollowsRedirectsLikeABrowser(t *testing.T) {
	base, requests := rawHTTPServer(t,
		"HTTP/1.1 303 See Other\r\nLocation: /done\r\nContent-Length: 0\r\n\r\n",
		"HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\nok",
	)
	code := fmt.Sprintf("[HTTP]에서 <보내기>를 가져오자\n<보내기>(\"%s/start\", \"내용\")의 \"body\"를 출력하자\n", base)
	tw, err := runHari(t, code)
	if err != nil || strings.Join(tw.Output, "") != "ok" {
		t.Fatalf("got %v %v", tw.Output, err)
	}
	got := requests()
	if len(got) != 2 || !strings.HasPrefix(got[0], "POST /start HTTP/1.1") || !strings.HasPrefix(got[1], "GET /done HTTP/1.1") || strings.Contains(got[1], "내용") {
		t.Errorf("a 303 should turn the POST into a GET without the body, got %q", got)
	}

	base, requests = rawHTTPServer(t,
		"HTTP/1.1 307 Temporary Redirect\r\nLocation: "+"/again\r\nContent-Length: 0\r\n\r\n",
		"HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\nok",
	)
	code = fmt.Sprintf("[HTTP]에서 <보내기>를 가져오자\n<보내기>(\"%s/start\", \"내용\")의 \"body\"를 출력하자\n", base)
	if _, err := runHari(t, code); err != nil {
		t.Fatal(err)
	}
	got = requests()
	if len(got) != 2 || !strings.HasPrefix(got[1], "POST /again HTTP/1.1") || !strings.HasSuffix(got[1], "내용") {
		t.Errorf("a 307 should repeat the POST with its body, got %q", got)
	}

	// A redirect loop ends after ten hops with the last answer, not forever.
	base, requests = rawHTTPServer(t, "HTTP/1.1 302 Found\r\nLocation: /loop\r\nContent-Length: 0\r\n\r\n")
	code = fmt.Sprintf("[HTTP]에서 <가져오기>를 가져오자\n<가져오기>(\"%s/loop\")의 \"status\"를 출력하자\n", base)
	if tw, err := runHari(t, code); err != nil || strings.Join(tw.Output, "") != "302" || len(requests()) != 11 {
		t.Errorf("a redirect loop: got %v %v after %d requests", tw.Output, err, len(requests()))
	}
}

func TestHTTPClientRequestText(t *testing.T) {
	base, requests := rawHTTPServer(t, "HTTP/1.1 200 OK\r\nContent-Length: 0\r\n\r\n")
	code := fmt.Sprintf("[HTTP]에서 <요청>을 가져오자\n<요청>(\"head\", \"%s/a b?x=1\", \"\", {\"user-agent\": \"내 프로그램\", \"X-Mine\": \"값\"})의 \"status\"를 출력하자\n", base)
	if _, err := runHari(t, code); err != nil {
		t.Fatal(err)
	}
	head := requests()[0]
	for _, want := range []string{"HEAD /a%20b?x=1 HTTP/1.1\r\n", "Host: 127.0.0.1:", "user-agent: 내 프로그램\r\n", "X-Mine: 값\r\n", "Connection: close\r\n", "Accept-Encoding: identity\r\n"} {
		if !strings.Contains(head, want) {
			t.Errorf("the request lacks %q:\n%s", want, head)
		}
	}
	if strings.Contains(head, "User-Agent: hari") {
		t.Errorf("a header the script gives should replace the default:\n%s", head)
	}
}

func TestHTTPClientRefusesWhatItCannotRead(t *testing.T) {
	cases := []struct{ name, response, want string }{
		{"너무 큰 본문", "HTTP/1.1 200 OK\r\nContent-Length: 999999999\r\n\r\n", "too large to read"},
		{"이상한 상태 줄", "이상한 답\r\n\r\n", "Could not send a request to"},
		{"중간에 끊긴 본문", "HTTP/1.1 200 OK\r\nContent-Length: 100\r\n\r\n짧음", "Could not send a request to"},
	}
	for _, c := range cases {
		base, _ := rawHTTPServer(t, c.response)
		code := fmt.Sprintf("[HTTP]에서 <가져오기>를 가져오자\n<가져오기>(\"%s/\")를 출력하자\n", base)
		for engine, r := range engines(t, code) {
			if r.err == nil || !strings.Contains(r.err.Error(), c.want) {
				t.Errorf("%s %s: want an error containing %q, got %v", engine, c.name, c.want, r.err)
			}
		}
	}
}
