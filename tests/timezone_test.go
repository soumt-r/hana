package tests

// The timezone package end to end: the real package folder (manifest, Hari and
// Kanade entries) plus its native library built from packages/timezone/native (the
// IANA database behind ABI v1), run on both engines. Needs a C toolchain, like
// native_plugin_test.go, and is skipped without one.
//
// The offsets are checked against Go's own copy of the database, so what this shows
// is that the values cross the boundary intact: the zone name, the time, the
// offset, on both sides of every clock change.

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
	_ "time/tzdata"

	"github.com/soumt-r/hana/errs"
	"github.com/soumt-r/hana/pkg"
)

var (
	tzLibOnce sync.Once
	tzLibPath string
	tzLibErr  error
	tzRuns    int
)

func timezoneLibrary(t *testing.T) string {
	t.Helper()
	tzLibOnce.Do(func() {
		if _, err := exec.LookPath("gcc"); err != nil {
			if _, err := exec.LookPath("clang"); err != nil {
				tzLibErr = fmt.Errorf("no C compiler")
				return
			}
		}
		dir, err := filepath.Abs(filepath.Join("testdata", "tzbuild"))
		if err != nil {
			tzLibErr = err
			return
		}
		os.MkdirAll(dir, 0o755)
		tzLibPath = filepath.Join(dir, "timezone-"+pkg.Platform()+pkg.LibraryExt(runtime.GOOS))
		cmd := exec.Command("go", "build", "-buildmode=c-shared", "-o", tzLibPath, ".")
		cmd.Dir = filepath.Join("..", "packages", "timezone", "native")
		cmd.Env = append(os.Environ(), "CGO_ENABLED=1")
		if out, err := cmd.CombinedOutput(); err != nil {
			tzLibErr = fmt.Errorf("building the library: %v\n%s", err, out)
		}
	})
	if tzLibErr != nil {
		t.Skipf("timezone tests need a C toolchain: %v", tzLibErr)
	}
	return tzLibPath
}

// timezonePackage puts the real package, with the freshly built library, in a fresh
// working folder and makes it the current one.
func timezonePackage(t *testing.T) {
	t.Helper()
	lib, err := os.ReadFile(timezoneLibrary(t))
	if err != nil {
		t.Fatal(err)
	}
	src := filepath.Join("..", "packages", "timezone")
	tzRuns++
	dir := filepath.Join(filepath.Dir(tzLibPath), fmt.Sprintf("run-%d-%d", os.Getpid(), tzRuns))
	pkgDir := filepath.Join(dir, "packages", "timezone")
	for _, rel := range []string{"hana.pkg.json", filepath.Join("hari", "index.hr"), filepath.Join("kanade", "index.knd")} {
		data, err := os.ReadFile(filepath.Join(src, rel))
		if err != nil {
			t.Fatal(err)
		}
		dst := filepath.Join(pkgDir, rel)
		os.MkdirAll(filepath.Dir(dst), 0o755)
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	manifest, err := pkg.Load(pkgDir)
	if err != nil {
		t.Fatal(err)
	}
	native, ok := manifest.NativeFor(pkg.Platform())
	if !ok {
		t.Skipf("the manifest declares no library for %s", pkg.Platform())
	}
	dst := filepath.Join(pkgDir, filepath.FromSlash(native.File))
	os.MkdirAll(filepath.Dir(dst), 0o755)
	if err := os.WriteFile(dst, lib, 0o755); err != nil {
		t.Fatal(err)
	}
	old, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(old) })
}

const tzImport = "[timezone]에서 <시간대서식>과 <시간대읽기>와 <시간대요일>과 <시간대오프셋>을 가져오자\n"

func TestTimezoneOffsetsMatchTheDatabase(t *testing.T) {
	timezonePackage(t)
	zones := []string{"Asia/Seoul", "Asia/Tokyo", "Asia/Kolkata", "Asia/Kathmandu", "Europe/London", "Europe/Berlin",
		"America/New_York", "America/Los_Angeles", "America/Sao_Paulo", "Australia/Sydney", "Australia/Lord_Howe",
		"Pacific/Auckland", "Pacific/Apia", "UTC"}

	var code strings.Builder
	code.WriteString(tzImport)
	var want []string
	add := func(zone string, at int64) {
		loc, _ := time.LoadLocation(zone)
		_, offset := time.Unix(at, 0).In(loc).Zone()
		code.WriteString(fmt.Sprintf("<시간대오프셋>(%d, \"%s\")를 출력하자\n", at, zone))
		want = append(want, strconv.Itoa(offset))
	}
	for _, zone := range zones {
		loc, _ := time.LoadLocation(zone)
		// far apart moments across the years, before and after old rule changes
		for at := int64(315532800); at < 1924992000; at += 90 * 86400 {
			add(zone, at)
		}
		// both sides of every clock change from 2022 to 2026: walk the years six
		// hours at a time, and where the offset changes find the exact second
		offsetAt := func(at int64) int {
			_, offset := time.Unix(at, 0).In(loc).Zone()
			return offset
		}
		from := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
		to := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
		for at := from; at < to; at += 6 * 3600 {
			if offsetAt(at) == offsetAt(at+6*3600) {
				continue
			}
			low, high := at, at+6*3600 // the offset differs between them
			for high-low > 1 {
				middle := (low + high) / 2
				if offsetAt(middle) == offsetAt(low) {
					low = middle
				} else {
					high = middle
				}
			}
			add(zone, low)
			add(zone, high)
		}
	}
	for engine, r := range engines(t, code.String()) {
		if r.err != nil {
			t.Errorf("%s: %v", engine, r.err)
			continue
		}
		if len(r.out) != len(want) {
			t.Errorf("%s: %d answers for %d questions", engine, len(r.out), len(want))
			continue
		}
		for i, w := range want {
			if r.out[i] != w {
				t.Errorf("%s: answer %d is %s, want %s", engine, i+1, r.out[i], w)
				break
			}
		}
	}
}

func TestTimezoneFormatParseAndWeekday(t *testing.T) {
	timezonePackage(t)
	cases := []struct{ call, want string }{
		{`<시간대서식>(0, "YYYY-MM-DD HH:mm:ss", "Asia/Seoul")`, "1970-01-01 09:00:00"},
		{`<시간대서식>(1000000000, "YYYY-MM-DD HH:mm:ss", "America/New_York")`, "2001-09-08 21:46:40"},
		{`<시간대서식>(1000000000, "YYYY년 MM월 DD일", "Asia/Tokyo")`, "2001년 09월 09일"},
		{`<시간대서식>(1000000000, "HH:mm", "+05:30")`, "07:16"},
		{`<시간대서식>(1000000000, "HH:mm", "-0530")`, "20:16"},
		{`<시간대서식>(1000000000, "HH:mm", "UTC")`, "01:46"},
		{`<시간대읽기>("2001-09-09 10:46:40", "YYYY-MM-DD HH:mm:ss", "Asia/Seoul")`, "1000000000"},
		{`<시간대읽기>("2001-09-08 21:46:40", "YYYY-MM-DD HH:mm:ss", "America/New_York")`, "1000000000"},
		{`<시간대읽기>("2024-01-01", "YYYY-MM-DD", "UTC")`, "1704067200"},
		{`<시간대읽기>("2024-11-03 01:30:00", "YYYY-MM-DD HH:mm:ss", "America/New_York")`, "1730611800"}, // happens twice: the earlier
		{`<시간대요일>(1000000000, "Asia/Seoul")`, "7"},
		{`<시간대요일>(1000000000, "America/New_York")`, "6"}, // still Saturday there
		{`<시간대오프셋>(1000000000, "Asia/Seoul")`, "32400"},
		{`<시간대오프셋>(1000000000, "+09:00")`, "32400"},
		{`<시간대오프셋>(1000000000, "Asia/Kathmandu")`, "20700"},
	}
	var code strings.Builder
	code.WriteString(tzImport)
	for _, c := range cases {
		code.WriteString(c.call + "를 출력하자\n")
	}
	for engine, r := range engines(t, code.String()) {
		if r.err != nil {
			t.Errorf("%s: %v", engine, r.err)
			continue
		}
		for i, c := range cases {
			if i >= len(r.out) || r.out[i] != c.want {
				got := ""
				if i < len(r.out) {
					got = r.out[i]
				}
				t.Errorf("%s %s: got %q, want %q", engine, c.call, got, c.want)
			}
		}
	}
}

func TestTimezoneReportsProblems(t *testing.T) {
	timezonePackage(t)
	cases := []struct{ call, want string }{
		{`<시간대오프셋>(0, "Asia/Nowhere")`, `unknown time zone "Asia/Nowhere"`},
		{`<시간대오프셋>(0, "")`, `unknown time zone ""`},
		{`<시간대오프셋>(0, "Local")`, `unknown time zone "Local"`},
		{`<시간대오프셋>(0, "+25:00")`, `unknown time zone "+25:00"`},
		{`<시간대서식>("글", "YYYY", "UTC")`, "the time must be a number of seconds"},
		{`<시간대읽기>("2024-02-30", "YYYY-MM-DD", "UTC")`, "is not a date written as"},
		{`<시간대읽기>("2024/02/03", "YYYY-MM-DD", "UTC")`, "is not a date written as"},
		{`<시간대읽기>("2024-03-10 02:30:00", "YYYY-MM-DD HH:mm:ss", "America/New_York")`, "that clock time does not exist"},
		{`<시간대요일>(0)`, "MissingArgumentError"},
	}
	for _, c := range cases {
		code := tzImport + c.call + "를 출력하자\n"
		for engine, r := range engines(t, code) {
			if r.err == nil || !strings.Contains(errs.Localize(errs.Korean, r.err), c.want) {
				t.Errorf("%s %s: want an error containing %q, got %v", engine, c.call, c.want, r.err)
			}
		}
	}
}

func TestTimezoneInKanade(t *testing.T) {
	timezonePackage(t)
	code := "【timezone】から〈時間帯書式〉を持ってこよう\n【timezone】から〈時間帯読み取り〉を持ってこよう\n【timezone】から〈時間帯曜日〉を持ってこよう\n【timezone】から〈時間帯オフセット〉を持ってこよう\n" +
		"〈時間帯書式〉(1000000000, 「YYYY-MM-DD HH:mm:ss」, 「America/New_York」)を出力しよう\n" +
		"〈時間帯読み取り〉(「2001-09-09 10:46:40」, 「YYYY-MM-DD HH:mm:ss」, 「Asia/Seoul」)を出力しよう\n" +
		"〈時間帯曜日〉(1000000000, 「Asia/Seoul」)を出力しよう\n" +
		"〈時間帯オフセット〉(1000000000, 「Asia/Kathmandu」)を出力しよう\n"
	tw, err1 := runKanade(t, code)
	bc, err2 := runKanadeBytecode(t, code)
	for label, run := range map[string]struct {
		out []string
		err error
	}{"tree-walker": {tw.Output, err1}, "bytecode": {bc.Output, err2}} {
		if run.err != nil || strings.Join(run.out, "|") != "2001-09-08 21:46:40|1000000000|7|20700" {
			t.Errorf("kanade %s: got %v (%v)", label, run.out, run.err)
		}
	}
}
