package lsp

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// diagnosticsAfterOpen opens one document and returns what the server
// published for it.
func diagnosticsAfterOpen(t *testing.T, uri, text string) []diagnostic {
	t.Helper()
	open, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "method": "textDocument/didOpen",
		"params": map[string]any{"textDocument": map[string]any{"uri": uri, "text": text}},
	})
	got, _ := session(t, initMsg, string(open), exitMsg)
	for _, m := range got {
		if m.Method != "textDocument/publishDiagnostics" {
			continue
		}
		var p struct {
			URI         string       `json:"uri"`
			Diagnostics []diagnostic `json:"diagnostics"`
		}
		if err := json.Unmarshal(m.Params, &p); err != nil || p.URI != uri {
			t.Fatalf("bad publish: %s (%v)", m.Params, err)
		}
		return p.Diagnostics
	}
	t.Fatalf("nothing published for %s", uri)
	return nil
}

func TestHajaDiagnosticPointsAtTheBadToken(t *testing.T) {
	ds := diagnosticsAfterOpen(t, "file:///a.hj", "\"a\"를 출력하자\n  ) 만약")
	if len(ds) == 0 {
		t.Fatal("expected a diagnostic")
	}
	d := ds[0]
	if d.Range.Start != (position{1, 2}) || d.Range.End != (position{1, 3}) {
		t.Errorf("range = %+v, want line 1 cols 2-3", d.Range)
	}
	if !strings.Contains(d.Message, "')'") {
		t.Errorf("message should quote the token: %q", d.Message)
	}
	if !strings.HasSuffix(d.Message, "요.") {
		t.Errorf("haja message should be 해요체: %q", d.Message)
	}
}

func TestKanadeDiagnosticsAreJapanese(t *testing.T) {
	ds := diagnosticsAfterOpen(t, "file:///a.knd", "「あ」を出力しよう )")
	if len(ds) != 1 {
		t.Fatalf("want exactly one diagnostic (no end-of-input echo), got %+v", ds)
	}
	if ds[0].Range.Start != (position{0, 10}) {
		t.Errorf("range = %+v, want line 0 col 10", ds[0].Range)
	}
	if !strings.HasSuffix(ds[0].Message, "ません。") {
		t.Errorf("kanade message should be です・ます調: %q", ds[0].Message)
	}
}

func TestCleanDocumentPublishesEmptyListNotNull(t *testing.T) {
	open, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "method": "textDocument/didOpen",
		"params": map[string]any{"textDocument": map[string]any{"uri": "file:///ok.hj", "text": "\"안녕\"을 출력하자\n"}},
	})
	got, _ := session(t, initMsg, string(open), exitMsg)
	for _, m := range got {
		if m.Method == "textDocument/publishDiagnostics" {
			if !bytes.Contains(m.Params, []byte(`"diagnostics":[]`)) {
				t.Errorf("want an empty array so the editor clears old errors: %s", m.Params)
			}
			return
		}
	}
	t.Fatal("clean documents must still publish (to clear earlier errors)")
}

func TestUnknownExtensionGetsNoDiagnostics(t *testing.T) {
	open := `{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///a.txt","text":")"}}}`
	got, _ := session(t, initMsg, open, exitMsg)
	for _, m := range got {
		if m.Method == "textDocument/publishDiagnostics" {
			t.Errorf("no language for .txt, yet published: %s", m.Params)
		}
	}
}

func TestColumnsAreUTF16UnitsAfterWideCharacters(t *testing.T) {
	// The emoji is one character but two UTF-16 units, so the bad token's
	// editor column is one more than its character column.
	ds := diagnosticsAfterOpen(t, "file:///w.hj", "\"😀\"를 출력하자 ,\n")
	if len(ds) == 0 {
		t.Fatal("expected a diagnostic")
	}
	// characters: " 😀 " 를 ␠ 출력하자 ␠ , -> ',' is char index 10, UTF-16 index 11
	if ds[0].Range.Start.Character != 11 {
		t.Errorf("start.character = %d, want 11", ds[0].Range.Start.Character)
	}
}

func TestDiagnosticPastTheEndIsPinnedToLastLine(t *testing.T) {
	ds := diagnosticsAfterOpen(t, "file:///e.hj", "1 + )")
	for _, d := range ds {
		if d.Range.Start.Line != 0 {
			t.Errorf("single-line doc, diagnostic on line %d: %+v", d.Range.Start.Line, d)
		}
	}
}

func TestDidCloseClearsDiagnostics(t *testing.T) {
	open := `{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///c.hj","text":")"}}}`
	closeMsg := `{"jsonrpc":"2.0","method":"textDocument/didClose","params":{"textDocument":{"uri":"file:///c.hj"}}}`
	got, _ := session(t, initMsg, open, closeMsg, exitMsg)
	var last message
	for _, m := range got {
		if m.Method == "textDocument/publishDiagnostics" {
			last = m
		}
	}
	if !bytes.Contains(last.Params, []byte(`"diagnostics":[]`)) {
		t.Errorf("close should publish an empty list, last publish: %s", last.Params)
	}
}
