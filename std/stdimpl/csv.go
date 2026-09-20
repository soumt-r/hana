package stdimpl

import (
	"strings"
	"unicode/utf8"

	"github.com/soumt-r/hana/errs"
	"github.com/soumt-r/hana/value"
)

// [CSV] reads and writes comma-separated text (RFC 4180): a value is a list of
// rows and a row is a list of texts. Reading never turns a cell into a number;
// that is for the script to do with <숫자로>.

// csvDelimiter reads the optional delimiter argument: one character that is
// not a quote or a line break. The default is a comma.
func csvDelimiter(args []interface{}, i int) (rune, error) {
	if i >= len(args) {
		return ',', nil
	}
	s, err := stringArg(args, i)
	if err != nil {
		return 0, err
	}
	r, size := utf8.DecodeRuneInString(s)
	if size == 0 || size != len(s) || r == '"' || r == '\n' || r == '\r' || r == utf8.RuneError {
		return 0, errs.New(errs.CSVDelimiter)
	}
	return r, nil
}

// csvParse(글[, 구분자]) → list of rows.
func csvParse(args []interface{}) (interface{}, error) {
	if err := between(args, 1, 2); err != nil {
		return nil, err
	}
	text, err := stringArg(args, 0)
	if err != nil {
		return nil, err
	}
	delim, err := csvDelimiter(args, 1)
	if err != nil {
		return nil, err
	}
	return parseCSV(strings.TrimPrefix(text, string(rune(0xFEFF))), delim)
}

// parseCSV reads the text row by row. A line break inside quotes belongs to the
// cell; a quote inside a cell that did not start with one is an ordinary
// character; anything but a delimiter or a line break right after a closing
// quote is an error. A blank line is a row with one empty cell, and a line break
// at the very end of the text does not start another row.
func parseCSV(text string, delim rune) (interface{}, error) {
	runes := []rune(text)
	rows := []interface{}{}
	row := []interface{}{}
	var field strings.Builder
	started, quoted, inQuotes, afterQuote := false, false, false, false

	endField := func() {
		row = append(row, field.String())
		field.Reset()
		started, quoted, afterQuote = false, false, false
	}
	endRow := func() {
		endField()
		rows = append(rows, value.NewList(row))
		row = []interface{}{}
	}

	for i := 0; i < len(runes); i++ {
		c := runes[i]
		switch {
		case inQuotes:
			if c != '"' {
				field.WriteRune(c)
			} else if i+1 < len(runes) && runes[i+1] == '"' {
				field.WriteRune('"')
				i++
			} else {
				inQuotes, afterQuote = false, true
			}
		case c == delim:
			endField()
		case c == '\n' || c == '\r':
			if c == '\r' && i+1 < len(runes) && runes[i+1] == '\n' {
				i++
			}
			endRow()
		case afterQuote:
			return nil, errs.New(errs.CSVInvalid)
		case c == '"' && !started:
			inQuotes, quoted, started = true, true, true
		default:
			field.WriteRune(c)
			started = true
		}
	}
	if inQuotes {
		return nil, errs.New(errs.CSVInvalid)
	}
	if started || quoted || len(row) > 0 {
		endRow()
	}
	return value.NewList(rows), nil
}

// csvStringify(행들[, 구분자]) → text. Rows are separated by a line break (none
// after the last one). A cell is quoted only when it has to be: it holds the
// delimiter, a quote or a line break, or it is the empty only cell of its row.
func csvStringify(args []interface{}) (interface{}, error) {
	if err := between(args, 1, 2); err != nil {
		return nil, err
	}
	rows, err := listArg(args, 0)
	if err != nil {
		return nil, err
	}
	delim, err := csvDelimiter(args, 1)
	if err != nil {
		return nil, err
	}
	var out strings.Builder
	for n, r := range rows {
		row, ok := r.(*value.List)
		if !ok || len(row.Items) == 0 {
			return nil, errs.New(errs.CSVUnsupported)
		}
		if n > 0 {
			out.WriteByte('\n')
		}
		for k, cell := range row.Items {
			if k > 0 {
				out.WriteRune(delim)
			}
			text, err := csvCell(cell)
			if err != nil {
				return nil, err
			}
			needsQuotes := strings.ContainsAny(text, "\"\r\n") || strings.ContainsRune(text, delim) || (text == "" && len(row.Items) == 1)
			if needsQuotes {
				out.WriteByte('"')
				out.WriteString(strings.ReplaceAll(text, "\"", "\"\""))
				out.WriteByte('"')
			} else {
				out.WriteString(text)
			}
		}
	}
	return out.String(), nil
}

// csvCell is a cell's text: a text as it is, a number in JSON's number form,
// nothing (비어있음) as an empty text. Anything else has no CSV form.
func csvCell(v interface{}) (string, error) {
	switch x := v.(type) {
	case string:
		return x, nil
	case nil:
		return "", nil
	case float64:
		if x != x || x > 1.7976931348623157e308 || x < -1.7976931348623157e308 {
			return "", errs.New(errs.CSVUnsupported)
		}
		return jsonNumber(x), nil
	}
	return "", errs.New(errs.CSVUnsupported)
}
