package lsp

import (
	"strings"
	"unicode/utf16"

	"github.com/soumt-r/hana/errs"
	hajaparser "github.com/soumt-r/hana/parser/haja"
)

type position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type lspRange struct {
	Start position `json:"start"`
	End   position `json:"end"`
}

type diagnostic struct {
	Range    lspRange `json:"range"`
	Severity int      `json:"severity"`
	Source   string   `json:"source"`
	Message  string   `json:"message"`
}

const severityError = 1

// diagnosticsFor parses text as lang and converts the parser's diagnostics to
// LSP ones. It always returns a non-nil slice so an empty result clears the
// editor's previous squiggles.
func diagnosticsFor(lang *language, text string) []diagnostic {
	_, p := lang.parse(text)
	lines := splitLines(text)
	out := []diagnostic{}
	for _, d := range p.Diagnostics() {
		out = append(out, diagnostic{
			Range:    rangeOf(lines, d),
			Severity: severityError,
			Source:   "hana",
			Message:  errs.Message(lang.locale, d.Err()),
		})
	}
	return out
}

func splitLines(text string) []string {
	return strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
}

// rangeOf maps a parser diagnostic (1-based line, character column) onto an
// LSP range (0-based line, UTF-16 units). Diagnostics with no literal — the
// parser ran off the end of the input — are pinned to the last character of
// the last non-empty line so they stay visible.
func rangeOf(lines []string, d hajaparser.Diagnostic) lspRange {
	line := d.Line - 1
	col, length := d.Col, d.Length
	if line >= len(lines) || d.Literal == "" {
		line = len(lines) - 1
		for line > 0 && strings.TrimSpace(lines[line]) == "" {
			line--
		}
		runes := []rune(lines[line])
		col, length = max(len(runes)-1, 0), min(len(runes), 1)
	}
	if line < 0 {
		line = 0
	}
	runes := []rune(lines[line])
	col = min(col, len(runes))
	end := min(col+length, len(runes))
	return lspRange{
		Start: position{line, utf16Len(runes[:col])},
		End:   position{line, utf16Len(runes[:end])},
	}
}

func utf16Len(runes []rune) int {
	return len(utf16.Encode(runes))
}
