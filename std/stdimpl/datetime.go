package stdimpl

import (
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/soumt-r/hana/errs"
)

// Dates are Unix seconds (a number) in the computer's local time zone. A format
// is text where YYYY MM DD HH mm ss stand for year, month, day, hour, minute
// and second; every other character is itself.
var dateTokens = []string{"YYYY", "MM", "DD", "HH", "mm", "ss"}

func datetimeNow(args []interface{}) (interface{}, error) {
	if err := exactly(args, 0); err != nil {
		return nil, err
	}
	return float64(time.Now().UnixMilli()) / 1000, nil
}

func datetimeFormat(args []interface{}) (interface{}, error) {
	if err := exactly(args, 2); err != nil {
		return nil, err
	}
	seconds, err := numberArg(args, 0)
	if err != nil {
		return nil, err
	}
	layout, err := stringArg(args, 1)
	if err != nil {
		return nil, err
	}
	if math.IsNaN(seconds) || math.IsInf(seconds, 0) || math.Abs(seconds) > 8.64e12 {
		return nil, errs.New(errs.NativeArgNumber, 1)
	}
	t := time.Unix(int64(math.Floor(seconds)), 0)

	var out strings.Builder
	for i := 0; i < len(layout); {
		token := tokenAt(layout, i)
		switch token {
		case "YYYY":
			out.WriteString(pad(t.Year(), 4))
		case "MM":
			out.WriteString(pad(int(t.Month()), 2))
		case "DD":
			out.WriteString(pad(t.Day(), 2))
		case "HH":
			out.WriteString(pad(t.Hour(), 2))
		case "mm":
			out.WriteString(pad(t.Minute(), 2))
		case "ss":
			out.WriteString(pad(t.Second(), 2))
		default:
			out.WriteByte(layout[i])
			i++
			continue
		}
		i += len(token)
	}
	return out.String(), nil
}

// datetimeParse reads text written in the given format back into Unix seconds.
// Parts the format leaves out default to 1970-01-01 00:00:00; a date that does
// not exist (like February 30) is an error.
func datetimeParse(args []interface{}) (interface{}, error) {
	if err := exactly(args, 2); err != nil {
		return nil, err
	}
	text, err := stringArg(args, 0)
	if err != nil {
		return nil, err
	}
	layout, err := stringArg(args, 1)
	if err != nil {
		return nil, err
	}
	invalid := errs.New(errs.DateInvalid, text)

	parts := map[string]int{"YYYY": 1970, "MM": 1, "DD": 1, "HH": 0, "mm": 0, "ss": 0}
	pos := 0
	for i := 0; i < len(layout); {
		token := tokenAt(layout, i)
		if token == "" {
			if pos >= len(text) || text[pos] != layout[i] {
				return nil, invalid
			}
			pos++
			i++
			continue
		}
		if pos+len(token) > len(text) {
			return nil, invalid
		}
		digits := text[pos : pos+len(token)]
		if strings.Trim(digits, "0123456789") != "" {
			return nil, invalid
		}
		n, _ := strconv.Atoi(digits)
		parts[token] = n
		pos += len(token)
		i += len(token)
	}
	if pos != len(text) {
		return nil, invalid
	}

	t := time.Date(parts["YYYY"], time.Month(parts["MM"]), parts["DD"], parts["HH"], parts["mm"], parts["ss"], 0, time.Local)
	if t.Year() != parts["YYYY"] || int(t.Month()) != parts["MM"] || t.Day() != parts["DD"] ||
		t.Hour() != parts["HH"] || t.Minute() != parts["mm"] || t.Second() != parts["ss"] {
		return nil, invalid
	}
	return float64(t.Unix()), nil
}

// datetimeWeekday is the day of the week of a time: 1 for Monday through 7 for Sunday.
func datetimeWeekday(args []interface{}) (interface{}, error) {
	if err := exactly(args, 1); err != nil {
		return nil, err
	}
	seconds, err := numberArg(args, 0)
	if err != nil {
		return nil, err
	}
	if math.IsNaN(seconds) || math.IsInf(seconds, 0) || math.Abs(seconds) > 8.64e12 {
		return nil, errs.New(errs.NativeArgNumber, 1)
	}
	day := int(time.Unix(int64(math.Floor(seconds)), 0).Weekday())
	if day == 0 {
		day = 7
	}
	return float64(day), nil
}

// datetimeSleep waits for that many seconds (0 to 3600) and returns 비어있음.
func datetimeSleep(args []interface{}) (interface{}, error) {
	if err := exactly(args, 1); err != nil {
		return nil, err
	}
	seconds, err := numberArg(args, 0)
	if err != nil {
		return nil, err
	}
	if math.IsNaN(seconds) || seconds < 0 || seconds > 3600 {
		return nil, errs.New(errs.SleepRange)
	}
	time.Sleep(time.Duration(seconds * float64(time.Second)))
	return nil, nil
}

func tokenAt(layout string, i int) string {
	for _, token := range dateTokens {
		if strings.HasPrefix(layout[i:], token) {
			return token
		}
	}
	return ""
}

func pad(n, width int) string {
	s := strconv.Itoa(n)
	for len(s) < width {
		s = "0" + s
	}
	return s
}
