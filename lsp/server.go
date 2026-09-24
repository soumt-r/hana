// Package lsp is hana's language server: one server for both Hari (.hr) and
// Kanade (.knd) documents, sharing the lexers/parsers the interpreter uses.
package lsp

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"sync"
)

// Server speaks LSP over a reader/writer pair (stdio in production).
type Server struct {
	in  *bufio.Reader
	out io.Writer

	writeMu sync.Mutex

	docs        map[string]string
	initialized bool
	shutdown    bool
	exited      bool
}

func NewServer(in io.Reader, out io.Writer) *Server {
	return &Server{in: bufio.NewReader(in), out: out, docs: map[string]string{}}
}

// Serve handles messages until the client sends `exit` or closes the stream.
// It returns nil on a clean exit (after `shutdown`) and an error otherwise,
// mirroring the LSP rule that `exit` without `shutdown` is abnormal.
func (s *Server) Serve() error {
	for !s.exited {
		body, err := readFrame(s.in)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		var msg message
		if err := json.Unmarshal(body, &msg); err != nil {
			s.reply(nil, nil, &responseError{Code: errInvalidRequest, Message: "invalid JSON"})
			continue
		}
		s.handle(msg)
	}
	if !s.shutdown {
		return errors.New("exit before shutdown")
	}
	return nil
}

func (s *Server) send(v any) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	writeFrame(s.out, v)
}

func (s *Server) reply(id json.RawMessage, result any, rerr *responseError) {
	if rerr != nil {
		s.send(message{RPC: "2.0", ID: id, Error: rerr})
		return
	}
	if result == nil {
		result = json.RawMessage("null")
	}
	s.send(message{RPC: "2.0", ID: id, Result: result})
}

func (s *Server) notify(method string, params any) {
	raw, _ := json.Marshal(params)
	s.send(message{RPC: "2.0", Method: method, Params: raw})
}

func (s *Server) handle(msg message) {
	isRequest := len(msg.ID) > 0

	if !s.initialized && msg.Method != "initialize" && msg.Method != "exit" {
		if isRequest {
			s.reply(msg.ID, nil, &responseError{Code: errServerNotInit, Message: "server not initialized"})
		}
		return
	}

	switch msg.Method {
	case "initialize":
		s.initialized = true
		s.reply(msg.ID, map[string]any{
			"capabilities": map[string]any{
				"textDocumentSync":       1, // full document on every change
				"hoverProvider":          true,
				"definitionProvider":     true,
				"documentSymbolProvider": true,
				"completionProvider": map[string]any{
					// The opening delimiters of names in either language.
					"triggerCharacters": []string{"'", "<", "[", "『", "〈", "【"},
				},
			},
			"serverInfo": map[string]any{"name": "hana-lsp"},
		}, nil)
	case "initialized":
	case "shutdown":
		s.shutdown = true
		s.reply(msg.ID, nil, nil)
	case "exit":
		s.exited = true
	case "textDocument/didOpen":
		var p struct {
			TextDocument struct {
				URI  string `json:"uri"`
				Text string `json:"text"`
			} `json:"textDocument"`
		}
		if json.Unmarshal(msg.Params, &p) == nil {
			s.docs[p.TextDocument.URI] = p.TextDocument.Text
			s.documentChanged(p.TextDocument.URI)
		}
	case "textDocument/didChange":
		var p struct {
			TextDocument struct {
				URI string `json:"uri"`
			} `json:"textDocument"`
			ContentChanges []struct {
				Text string `json:"text"`
			} `json:"contentChanges"`
		}
		if json.Unmarshal(msg.Params, &p) == nil && len(p.ContentChanges) > 0 {
			// Full sync: the last change carries the whole document.
			s.docs[p.TextDocument.URI] = p.ContentChanges[len(p.ContentChanges)-1].Text
			s.documentChanged(p.TextDocument.URI)
		}
	case "textDocument/didClose":
		var p struct {
			TextDocument struct {
				URI string `json:"uri"`
			} `json:"textDocument"`
		}
		if json.Unmarshal(msg.Params, &p) == nil {
			delete(s.docs, p.TextDocument.URI)
			s.documentClosed(p.TextDocument.URI)
		}
	case "textDocument/definition":
		s.definition(msg.ID, msg.Params)
	case "textDocument/documentSymbol":
		s.documentSymbol(msg.ID, msg.Params)
	case "textDocument/hover":
		s.hover(msg.ID, msg.Params)
	case "textDocument/completion":
		s.completion(msg.ID, msg.Params)
	default:
		if isRequest {
			s.reply(msg.ID, nil, &responseError{Code: errMethodNotFound, Message: "method not found: " + msg.Method})
		}
	}
}

// documentChanged and documentClosed are the hooks later features attach to
// (diagnostics on change, clearing them on close).
func (s *Server) documentChanged(uri string) {
	lang := languageFor(uri)
	if lang == nil {
		return
	}
	s.publish(uri, diagnosticsFor(lang, s.docs[uri]))
}

// documentClosed clears the document's diagnostics so they don't linger in
// the editor's Problems panel after the file is closed.
func (s *Server) documentClosed(uri string) {
	if languageFor(uri) == nil {
		return
	}
	s.publish(uri, []diagnostic{})
}

func (s *Server) publish(uri string, diags []diagnostic) {
	s.notify("textDocument/publishDiagnostics", map[string]any{"uri": uri, "diagnostics": diags})
}
