// The native half of the timezone package: the IANA time zone database behind
// hana's native library ABI v1 (see native/native.go). Build it for the current
// platform with build.sh or build.ps1 in this folder; it needs a C toolchain (cgo).
//
// It depends on nothing but the Go standard library, and carries its own copy of
// the time zone database (time/tzdata), so it does not need one on the machine.
package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unsafe"

	_ "time/tzdata"
)

//export HanaAlloc
func HanaAlloc(n C.size_t) *C.char { return (*C.char)(C.malloc(n)) }

//export HanaFree
func HanaFree(p *C.char) { C.free(unsafe.Pointer(p)) }

func reply(ok interface{}) *C.char {
	text, _ := json.Marshal(map[string]interface{}{"ok": ok})
	return C.CString(string(text))
}

func fail(format string, args ...interface{}) *C.char {
	text, _ := json.Marshal(map[string]interface{}{"error": fmt.Sprintf(format, args...)})
	return C.CString(string(text))
}

// arguments decodes the JSON array hana sends.
func arguments(raw *C.char) ([]json.RawMessage, error) {
	var args []json.RawMessage
	if err := json.Unmarshal([]byte(C.GoString(raw)), &args); err != nil {
		return nil, err
	}
	return args, nil
}

// A time is Unix seconds, like the [날짜] module's. A format is text where YYYY MM
// DD HH mm ss stand for year, month, day, hour, minute and second.
var tokens = []string{"YYYY", "MM", "DD", "HH", "mm", "ss"}

func tokenAt(layout string, i int) string {
	for _, t := range tokens {
		if strings.HasPrefix(layout[i:], t) {
			return t
		}
	}
	return ""
}

var fixedOffset = regexp.MustCompile(`^([+-])([0-9]{2}):?([0-9]{2})$`)

// zone reads a time zone: an IANA name ("Asia/Seoul"), "UTC", or a fixed offset
// ("+09:00", "-0530"). The computer's own zone is not an option, so a program means
// the same thing on every machine.
func zone(name string) (*time.Location, error) {
	switch name {
	case "UTC", "Z":
		return time.UTC, nil
	case "", "Local":
		return nil, fmt.Errorf("unknown time zone %q", name)
	}
	if m := fixedOffset.FindStringSubmatch(name); m != nil {
		hours, _ := strconv.Atoi(m[2])
		minutes, _ := strconv.Atoi(m[3])
		if hours > 23 || minutes > 59 {
			return nil, fmt.Errorf("unknown time zone %q", name)
		}
		seconds := hours*3600 + minutes*60
		if m[1] == "-" {
			seconds = -seconds
		}
		return time.FixedZone(name, seconds), nil
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return nil, fmt.Errorf("unknown time zone %q", name)
	}
	return loc, nil
}

func timeArg(raw json.RawMessage) (int64, error) {
	var seconds float64
	if json.Unmarshal(raw, &seconds) != nil || math.IsNaN(seconds) || math.IsInf(seconds, 0) || math.Abs(seconds) > 8.64e12 {
		return 0, fmt.Errorf("the time must be a number of seconds")
	}
	return int64(math.Floor(seconds)), nil
}

func stringArg(raw json.RawMessage, what string) (string, error) {
	var s string
	if json.Unmarshal(raw, &s) != nil {
		return "", fmt.Errorf("%s must be text", what)
	}
	return s, nil
}

func pad(n, width int) string {
	s := strconv.Itoa(n)
	for len(s) < width {
		s = "0" + s
	}
	return s
}

// Offset is [time, zone] -> the zone's offset from UTC at that time, in seconds
// (east is positive: 32400 for Asia/Seoul).
//
//export Offset
func Offset(raw *C.char) *C.char {
	args, err := arguments(raw)
	if err != nil || len(args) != 2 {
		return fail("Offset needs (time, zone)")
	}
	seconds, err := timeArg(args[0])
	if err != nil {
		return fail("%v", err)
	}
	name, err := stringArg(args[1], "the zone")
	if err != nil {
		return fail("%v", err)
	}
	loc, err := zone(name)
	if err != nil {
		return fail("%v", err)
	}
	_, offset := time.Unix(seconds, 0).In(loc).Zone()
	return reply(offset)
}

// Format is [time, format, zone] -> the time written in that zone's clock.
//
//export Format
func Format(raw *C.char) *C.char {
	args, err := arguments(raw)
	if err != nil || len(args) != 3 {
		return fail("Format needs (time, format, zone)")
	}
	seconds, err := timeArg(args[0])
	if err != nil {
		return fail("%v", err)
	}
	layout, err := stringArg(args[1], "the format")
	if err != nil {
		return fail("%v", err)
	}
	name, err := stringArg(args[2], "the zone")
	if err != nil {
		return fail("%v", err)
	}
	loc, err := zone(name)
	if err != nil {
		return fail("%v", err)
	}
	t := time.Unix(seconds, 0).In(loc)

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
	return reply(out.String())
}

// Parse is [text, format, zone] -> the Unix seconds of a time written on that
// zone's clock. Parts the format leaves out are 1970-01-01 00:00:00. A date that
// does not exist (February 30) and a clock time skipped by a clock change are
// errors; a clock time that happens twice is the earlier of the two.
//
//export Parse
func Parse(raw *C.char) *C.char {
	args, err := arguments(raw)
	if err != nil || len(args) != 3 {
		return fail("Parse needs (text, format, zone)")
	}
	text, err := stringArg(args[0], "the text")
	if err != nil {
		return fail("%v", err)
	}
	layout, err := stringArg(args[1], "the format")
	if err != nil {
		return fail("%v", err)
	}
	name, err := stringArg(args[2], "the zone")
	if err != nil {
		return fail("%v", err)
	}
	loc, err := zone(name)
	if err != nil {
		return fail("%v", err)
	}
	invalid := fmt.Sprintf("%q is not a date written as %q", text, layout)

	parts := map[string]int{"YYYY": 1970, "MM": 1, "DD": 1, "HH": 0, "mm": 0, "ss": 0}
	pos := 0
	for i := 0; i < len(layout); {
		token := tokenAt(layout, i)
		if token == "" {
			if pos >= len(text) || text[pos] != layout[i] {
				return fail("%s", invalid)
			}
			pos++
			i++
			continue
		}
		if pos+len(token) > len(text) {
			return fail("%s", invalid)
		}
		digits := text[pos : pos+len(token)]
		if strings.Trim(digits, "0123456789") != "" {
			return fail("%s", invalid)
		}
		parts[token], _ = strconv.Atoi(digits)
		pos += len(token)
		i += len(token)
	}
	if pos != len(text) {
		return fail("%s", invalid)
	}

	// The clock reading as if it were UTC, checked so that an impossible date
	// (February 30) is not quietly moved to the next month.
	clock := time.Date(parts["YYYY"], time.Month(parts["MM"]), parts["DD"], parts["HH"], parts["mm"], parts["ss"], 0, time.UTC)
	if clock.Year() != parts["YYYY"] || int(clock.Month()) != parts["MM"] || clock.Day() != parts["DD"] ||
		clock.Hour() != parts["HH"] || clock.Minute() != parts["mm"] || clock.Second() != parts["ss"] {
		return fail("%s", invalid)
	}
	local := clock.Unix()

	// The offsets a day before and a day after the reading bracket any clock change
	// near it; the reading is an instant only under an offset that is really in
	// force at that instant. Two such offsets: the reading happens twice (take the
	// earlier). None: a clock change skipped it.
	best, found := int64(0), false
	for _, probe := range []int64{local - 86400, local + 86400} {
		_, offset := time.Unix(probe, 0).In(loc).Zone()
		instant := local - int64(offset)
		if _, at := time.Unix(instant, 0).In(loc).Zone(); at == offset && (!found || instant < best) {
			best, found = instant, true
		}
	}
	if !found {
		return fail("%s (that clock time does not exist in %s)", invalid, name)
	}
	return reply(best)
}

// Weekday is [time, zone] -> the day of the week on that zone's calendar: 1 for
// Monday through 7 for Sunday, like [날짜].
//
//export Weekday
func Weekday(raw *C.char) *C.char {
	args, err := arguments(raw)
	if err != nil || len(args) != 2 {
		return fail("Weekday needs (time, zone)")
	}
	seconds, err := timeArg(args[0])
	if err != nil {
		return fail("%v", err)
	}
	name, err := stringArg(args[1], "the zone")
	if err != nil {
		return fail("%v", err)
	}
	loc, err := zone(name)
	if err != nil {
		return fail("%v", err)
	}
	day := int(time.Unix(seconds, 0).In(loc).Weekday())
	if day == 0 {
		day = 7
	}
	return reply(day)
}

func main() {}
