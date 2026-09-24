package lsp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// session feeds framed client messages to a Server and returns everything it
// wrote back, decoded.
func session(t *testing.T, msgs ...string) ([]message, error) {
	t.Helper()
	var in bytes.Buffer
	for _, m := range msgs {
		fmt.Fprintf(&in, "Content-Length: %d\r\n\r\n%s", len(m), m)
	}
	var out bytes.Buffer
	srv := NewServer(&in, &out)
	err := srv.Serve()

	var got []message
	r := bufio.NewReader(&out)
	for {
		body, ferr := readFrame(r)
		if ferr != nil {
			break
		}
		var m message
		if err := json.Unmarshal(body, &m); err != nil {
			t.Fatalf("server wrote invalid JSON %q: %v", body, err)
		}
		got = append(got, m)
	}
	return got, err
}

const (
	initMsg     = `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`
	shutdownMsg = `{"jsonrpc":"2.0","id":2,"method":"shutdown"}`
	exitMsg     = `{"jsonrpc":"2.0","method":"exit"}`
)

func TestInitializeAdvertisesFullSync(t *testing.T) {
	got, err := session(t, initMsg, shutdownMsg, exitMsg)
	if err != nil {
		t.Fatalf("clean shutdown should not error: %v", err)
	}
	if len(got) != 2 || string(got[0].ID) != "1" || string(got[1].ID) != "2" {
		t.Fatalf("unexpected replies: %+v", got)
	}
	raw, _ := json.Marshal(got[0].Result)
	if !strings.Contains(string(raw), `"textDocumentSync":1`) {
		t.Errorf("initialize result missing full sync: %s", raw)
	}
}

func TestStringRequestIDIsEchoed(t *testing.T) {
	got, _ := session(t, `{"jsonrpc":"2.0","id":"abc","method":"initialize"}`, exitMsg)
	if len(got) != 1 || string(got[0].ID) != `"abc"` {
		t.Fatalf("string id not echoed: %+v", got)
	}
}

func TestRequestBeforeInitializeIsRejected(t *testing.T) {
	got, _ := session(t, `{"jsonrpc":"2.0","id":7,"method":"textDocument/hover"}`, exitMsg)
	if len(got) != 1 || got[0].Error == nil || got[0].Error.Code != errServerNotInit {
		t.Fatalf("want ServerNotInitialized, got %+v", got)
	}
}

func TestUnknownRequestGetsMethodNotFoundButNotificationsAreIgnored(t *testing.T) {
	got, _ := session(t, initMsg,
		`{"jsonrpc":"2.0","id":3,"method":"nope/request"}`,
		`{"jsonrpc":"2.0","method":"nope/notification"}`,
		shutdownMsg, exitMsg)
	if len(got) != 3 {
		t.Fatalf("want 3 replies (init, method-not-found, shutdown), got %d: %+v", len(got), got)
	}
	if got[1].Error == nil || got[1].Error.Code != errMethodNotFound {
		t.Errorf("unknown request: %+v", got[1])
	}
}

func TestExitWithoutShutdownIsAnError(t *testing.T) {
	if _, err := session(t, initMsg, exitMsg); err == nil {
		t.Error("exit before shutdown should be reported")
	}
}

func TestStreamClosingEndsServeCleanly(t *testing.T) {
	if _, err := session(t, initMsg); err != nil {
		t.Errorf("EOF should end Serve without error, got %v", err)
	}
}

func TestDocumentSyncTracksOpenChangeClose(t *testing.T) {
	var in bytes.Buffer
	for _, m := range []string{
		initMsg,
		`{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///a.hr","text":"one"}}}`,
		`{"jsonrpc":"2.0","method":"textDocument/didChange","params":{"textDocument":{"uri":"file:///a.hr"},"contentChanges":[{"text":"two"},{"text":"three"}]}}`,
	} {
		fmt.Fprintf(&in, "Content-Length: %d\r\n\r\n%s", len(m), m)
	}
	srv := NewServer(&in, &bytes.Buffer{})
	srv.Serve()
	if srv.docs["file:///a.hr"] != "three" {
		t.Fatalf("after open+change want %q, got %q", "three", srv.docs["file:///a.hr"])
	}

	in.Reset()
	closeMsg := `{"jsonrpc":"2.0","method":"textDocument/didClose","params":{"textDocument":{"uri":"file:///a.hr"}}}`
	fmt.Fprintf(&in, "Content-Length: %d\r\n\r\n%s", len(closeMsg), closeMsg)
	srv.in = bufio.NewReader(&in)
	srv.Serve()
	if _, ok := srv.docs["file:///a.hr"]; ok {
		t.Error("didClose should forget the document")
	}
}

func TestKoreanTextSurvivesFraming(t *testing.T) {
	// Content-Length counts bytes, not characters — Korean/Japanese text is where that breaks.
	var in bytes.Buffer
	body := `{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///k.knd","text":"「こんにちは」を出力しよう"}}}`
	for _, m := range []string{initMsg, body} {
		fmt.Fprintf(&in, "Content-Length: %d\r\n\r\n%s", len(m), m)
	}
	srv := NewServer(&in, &bytes.Buffer{})
	srv.Serve()
	if srv.docs["file:///k.knd"] != "「こんにちは」を出力しよう" {
		t.Errorf("multi-byte text mangled: %q", srv.docs["file:///k.knd"])
	}
}
