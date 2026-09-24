package tests

import (
	"strings"
	"testing"

	"github.com/soumt-r/hana/errs"

	lexer "github.com/soumt-r/hana/lexer/hari"
	kanadelexer "github.com/soumt-r/hana/lexer/kanade"
	parser "github.com/soumt-r/hana/parser/hari"
	kanadeparser "github.com/soumt-r/hana/parser/kanade"
)

func firstDiag(t *testing.T, ds []parser.Diagnostic) parser.Diagnostic {
	t.Helper()
	if len(ds) == 0 {
		t.Fatal("expected a diagnostic, got none")
	}
	return ds[0]
}

func TestDiagnosticsCarryPosition(t *testing.T) {
	p := parser.New(lexer.New("\"a\"를 출력하자\n  ) 만약"))
	p.ParseProgram()
	d := firstDiag(t, p.Diagnostics())
	if d.Kind != parser.DiagUnknownToken || d.Line != 2 || d.Col != 2 || d.Length != 1 || d.Literal != ")" {
		t.Errorf("hari diagnostic = %+v", d)
	}
}

func TestDiagnosticsColumnCountsCharactersNotBytes(t *testing.T) {
	p := parser.New(lexer.New("'가나다' 정하자 ,\n"))
	p.ParseProgram()
	d := firstDiag(t, p.Diagnostics())
	if d.Line != 1 || d.Col != 10 {
		t.Errorf("column should count characters (want line 1 col 10), got %+v", d)
	}
}

func TestDiagnosticsWorkForKanade(t *testing.T) {
	p := kanadeparser.New(kanadelexer.New("「あ」を出力しよう\n  ）"))
	p.ParseProgram()
	// Kanade uses the same shared parser, so positions must flow through too.
	for _, d := range p.Diagnostics() {
		if d.Line < 1 {
			t.Errorf("diagnostic without a line: %+v", d)
		}
	}
}

func TestCleanProgramHasNoDiagnostics(t *testing.T) {
	p := parser.New(lexer.New("\"안녕\"을 출력하자\n"))
	p.ParseProgram()
	if ds := p.Diagnostics(); len(ds) != 0 {
		t.Errorf("clean program produced diagnostics: %+v", ds)
	}
}

// The Hari lexer once counted a backslash or the letter 't' as indentation
// (the regex was written as [ \t]), so a line starting with 't' got a bogus
// INDENT and shifted every column on it.
func TestLineStartingWithLatinTIsNotIndented(t *testing.T) {
	for _, tok := range lexer.New("total을 출력하자\n").Tokens {
		if tok.Type == "INDENT" {
			t.Fatalf("a line starting with 't' must not produce INDENT: %+v", tok)
		}
	}
	if tok := lexer.New("total을 출력하자\n").Tokens[0]; tok.Col != 0 {
		t.Errorf("first token column = %d, want 0", tok.Col)
	}
}

func TestParserErrorsCarryTheDiagnosticsInEveryLanguage(t *testing.T) {
	p := parser.New(lexer.New("\"a\"를 출력하자 )\n"))
	p.ParseProgram()
	if len(p.Errors()) == 0 || !strings.Contains(p.Errors()[0], "Unexpected ')' at line 1") {
		t.Fatalf("English errors = %v", p.Errors())
	}
	if ko := p.LocalizedErrors(errs.Korean); !strings.HasSuffix(ko[0], "요.") {
		t.Errorf("Korean errors should be 해요체: %v", ko)
	}
	if ja := p.LocalizedErrors(errs.Japanese); !strings.HasSuffix(ja[0], "ません。") {
		t.Errorf("Japanese errors should be です・ます調: %v", ja)
	}
}

func TestRunningOffTheEndIsReportedOnceOnTheLastLineWithCode(t *testing.T) {
	p := parser.New(lexer.New("\"a\"를 출력하자\n1 +\n\n"))
	p.ParseProgram()
	ds := p.Diagnostics()
	if len(ds) != 1 || ds[0].Literal != "" || ds[0].Line != 2 {
		t.Errorf("want one end-of-input diagnostic on line 2, got %+v", ds)
	}
}

func TestEndOfInputFalloutIsDroppedWhenABadTokenExplainsIt(t *testing.T) {
	p := parser.New(lexer.New("\"a\"를 출력하자 )\n"))
	p.ParseProgram()
	for _, d := range p.Diagnostics() {
		if d.Literal == "" {
			t.Errorf("end-of-input echo should be hidden next to a real bad token: %+v", p.Diagnostics())
		}
	}
}

// A character no lexer rule accepts used to be dropped silently, so
// `"a"를 출력하자 @` ran as if the @ were not there.
func TestUnknownCharactersAreReportedNotDropped(t *testing.T) {
	p := parser.New(lexer.New("\"a\"를 출력하자 @\n"))
	p.ParseProgram()
	ds := p.Diagnostics()
	if len(ds) == 0 || ds[0].Literal != "@" || ds[0].Line != 1 || ds[0].Col != 10 {
		t.Errorf("want an unexpected '@' at line 1 col 10, got %+v", ds)
	}

	k := kanadeparser.New(kanadelexer.New("「あ」を出力しよう ＠\n"))
	k.ParseProgram()
	if len(k.Diagnostics()) == 0 || k.Diagnostics()[0].Literal != "＠" {
		t.Errorf("kanade should report the full-width @, got %+v", k.Diagnostics())
	}
}
