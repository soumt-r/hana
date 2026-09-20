package stdimpl

import (
	"encoding/json"
	"math"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/soumt-r/hana/errs"
)

// jsonParse turns JSON text into hana values: objects become dictionaries,
// arrays lists, numbers numbers, null 비어있음.
func jsonParse(args []interface{}) (interface{}, error) {
	if err := exactly(args, 1); err != nil {
		return nil, err
	}
	text, err := stringArg(args, 0)
	if err != nil {
		return nil, err
	}
	var raw interface{}
	if err := json.Unmarshal([]byte(text), &raw); err != nil {
		return nil, errs.New(errs.JSONInvalid)
	}
	return fromJSON(raw), nil
}

func fromJSON(v interface{}) interface{} {
	switch x := v.(type) {
	case map[string]interface{}:
		dict := make(map[interface{}]interface{}, len(x))
		for k, el := range x {
			dict[k] = fromJSON(el)
		}
		return dict
	case []interface{}:
		list := make([]interface{}, len(x))
		for i, el := range x {
			list[i] = fromJSON(el)
		}
		return list
	}
	return v
}

// jsonStringify writes a value as JSON text. Dictionary keys must be strings and
// are written sorted (a dictionary has no order); the optional second argument
// is how many spaces to indent each level by (0 for a single line).
func jsonStringify(args []interface{}) (interface{}, error) {
	if err := between(args, 1, 2); err != nil {
		return nil, err
	}
	indent := 0
	if len(args) == 2 {
		n, err := integerArg(args, 1)
		if err != nil {
			return nil, err
		}
		if n < 0 || n > 10 {
			return nil, errs.New(errs.NativeArgInteger, 2)
		}
		indent = int(n)
	}
	var out strings.Builder
	if err := writeJSON(&out, args[0], indent, 0); err != nil {
		return nil, err
	}
	return out.String(), nil
}

func writeJSON(out *strings.Builder, v interface{}, indent, depth int) error {
	switch x := v.(type) {
	case nil:
		out.WriteString("null")
	case bool:
		out.WriteString(strconv.FormatBool(x))
	case float64:
		if math.IsNaN(x) || math.IsInf(x, 0) {
			return errs.New(errs.JSONUnsupported)
		}
		out.WriteString(jsonNumber(x))
	case string:
		out.WriteString(jsonString(x))
	case []interface{}:
		if len(x) == 0 {
			out.WriteString("[]")
			return nil
		}
		out.WriteByte('[')
		for i, el := range x {
			if i > 0 {
				out.WriteByte(',')
			}
			newline(out, indent, depth+1)
			if err := writeJSON(out, el, indent, depth+1); err != nil {
				return err
			}
		}
		newline(out, indent, depth)
		out.WriteByte(']')
	case map[interface{}]interface{}:
		if len(x) == 0 {
			out.WriteString("{}")
			return nil
		}
		keys := make([]string, 0, len(x))
		for k := range x {
			key, ok := k.(string)
			if !ok {
				return errs.New(errs.JSONUnsupported)
			}
			keys = append(keys, key)
		}
		sort.Strings(keys)
		out.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				out.WriteByte(',')
			}
			newline(out, indent, depth+1)
			out.WriteString(jsonString(k))
			out.WriteByte(':')
			if indent > 0 {
				out.WriteByte(' ')
			}
			if err := writeJSON(out, x[k], indent, depth+1); err != nil {
				return err
			}
		}
		newline(out, indent, depth)
		out.WriteByte('}')
	default:
		return errs.New(errs.JSONUnsupported)
	}
	return nil
}

func newline(out *strings.Builder, indent, depth int) {
	if indent == 0 {
		return
	}
	out.WriteByte('\n')
	out.WriteString(strings.Repeat(" ", indent*depth))
}

// jsonNumber matches JavaScript's Number-to-string, which the browser engines
// use: plain digits between 1e-6 and 1e21, an exponent outside, "0" for -0.
func jsonNumber(f float64) string {
	if f == 0 {
		return "0"
	}
	abs := math.Abs(f)
	if abs >= 1e21 || abs < 1e-6 {
		s := strconv.FormatFloat(f, 'e', -1, 64)
		mantissa, exp, _ := strings.Cut(s, "e")
		sign := exp[:1]
		digits := strings.TrimLeft(exp[1:], "0")
		return mantissa + "e" + sign + digits
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// jsonString quotes s the way JSON.stringify does: only ", \ and control
// characters are escaped.
func jsonString(s string) string {
	var out strings.Builder
	out.WriteByte('"')
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		switch {
		case r == utf8.RuneError && size == 1:
			out.WriteString("�")
		case r == '"':
			out.WriteString(`\"`)
		case r == '\\':
			out.WriteString(`\\`)
		case r == '\b':
			out.WriteString(`\b`)
		case r == '\f':
			out.WriteString(`\f`)
		case r == '\n':
			out.WriteString(`\n`)
		case r == '\r':
			out.WriteString(`\r`)
		case r == '\t':
			out.WriteString(`\t`)
		case r < 0x20:
			out.WriteString(`\u00`)
			out.WriteByte("0123456789abcdef"[r>>4])
			out.WriteByte("0123456789abcdef"[r&0xf])
		default:
			out.WriteRune(r)
		}
		i += size
	}
	out.WriteByte('"')
	return out.String()
}

// ToJSON writes a hana value as compact JSON, for the native-library bridge.
func ToJSON(v interface{}) (string, error) {
	var out strings.Builder
	if err := writeJSON(&out, v, 0, 0); err != nil {
		return "", err
	}
	return out.String(), nil
}

// FromJSON reads JSON text into hana values (objects become dictionaries).
func FromJSON(text string) (interface{}, error) {
	var raw interface{}
	if err := json.Unmarshal([]byte(text), &raw); err != nil {
		return nil, errs.New(errs.JSONInvalid)
	}
	return fromJSON(raw), nil
}
