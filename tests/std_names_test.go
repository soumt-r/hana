package tests

// Every native function in hana/std must be reachable, under its localized
// name, in both languages and on both engines — and compute the same thing.

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/soumt-r/hana/std"
)

// stdCase is one call of a native function, written in each language's syntax.
// wrap, when set, turns the call's %s into a deterministic expression for
// functions whose raw result varies (random numbers, the clock, time zones).
type stdCase struct {
	haja, kanade string // the call's arguments
	wrap         string
	want         string
	wantKanade   string            // when the printed text is language-specific (참/真)
	matches      string            // when set, a regexp the output must match instead of want
	files        map[string]string // when non-nil, the case runs in a fresh folder holding these files

	prelude       string // Haja functions the call passes in, defined before it
	preludeKanade string // the same in Kanade

	// elsewhere names the test that covers a function one call cannot show (it needs
	// a peer to talk to); the case is then not run here.
	elsewhere string
}

// stdCases has an entry for every ID in std.Modules; adding a function without
// one fails TestStdFunctionsWorkEverywhere.
var stdCases = map[string]stdCase{
	std.MathCeil:  {haja: "3.5", kanade: "3.5", want: "4"},
	std.MathFloor: {haja: "3.5", kanade: "3.5", want: "3"},

	std.JSONParse:     {haja: `"[1, 2, 3]"`, kanade: `「[1, 2, 3]」`, want: "[1, 2, 3]"},
	std.JSONStringify: {haja: `[1, 2]`, kanade: `【1, 2】`, want: "[1,2]"},

	std.CSVParse:     {haja: `"a,b"`, kanade: `「a,b」`, want: "[[a, b]]"},
	std.CSVStringify: {haja: `[["a", 1]]`, kanade: `【【「a」, 1】】`, want: "a,1"},

	std.RandomFloat:   {haja: ``, kanade: ``, wrap: "(%s < 1)", want: "참", wantKanade: "真"},
	std.RandomInt:     {haja: `5, 5`, kanade: `5, 5`, want: "5"},
	std.RandomChoice:  {haja: `[7]`, kanade: `【7】`, want: "7"},
	std.RandomShuffle: {haja: `[1]`, kanade: `【1】`, want: "[1]"},

	std.DatetimeNow:    {haja: ``, kanade: ``, wrap: "(%s > 1600000000)", want: "참", wantKanade: "真"},
	std.DatetimeFormat: {haja: `1000000000, "YYYY"`, kanade: `1000000000, 「YYYY」`, want: "2001"},
	std.DatetimeParse:  {haja: `"2001-09-09", "YYYY-MM-DD"`, kanade: `「2001-09-09」, 「YYYY-MM-DD」`, wrap: "(%s > 900000000)", want: "참", wantKanade: "真"},

	std.DatetimeSleep:   {haja: `0`, kanade: `0`, want: "비어있음", wantKanade: "空っぽ"},
	std.DatetimeWeekday: {haja: `1000000000`, kanade: `1000000000`, wrap: "(%s >= 1)", want: "참", wantKanade: "真"},

	std.MathSqrt:      {haja: `2`, kanade: `2`, want: "1.4142135623730951"},
	std.MathPow:       {haja: `2, 10`, kanade: `2, 10`, want: "1024"},
	std.MathAbs:       {haja: `(0 - 3.5)`, kanade: `(0 - 3.5)`, want: "3.5"},
	std.MathRound:     {haja: `2.567, 2`, kanade: `2.567, 2`, want: "2.57"},
	std.MathSin:       {haja: `0`, kanade: `0`, want: "0"},
	std.MathCos:       {haja: `0`, kanade: `0`, want: "1"},
	std.MathTan:       {haja: `0`, kanade: `0`, want: "0"},
	std.MathLog:       {haja: `1`, kanade: `1`, want: "0"},
	std.MathPi:        {haja: ``, kanade: ``, wrap: "(%s > 3.14)", want: "참", wantKanade: "真"},
	std.MathGcd:       {haja: `12, 18`, kanade: `12, 18`, want: "6"},
	std.MathFactorial: {haja: `5`, kanade: `5`, want: "120"},

	std.StatsSum:    {haja: `[1, 2, 3.5]`, kanade: `【1, 2, 3.5】`, want: "6.5"},
	std.StatsMin:    {haja: `[3, 1, 2]`, kanade: `【3, 1, 2】`, want: "1"},
	std.StatsMax:    {haja: `[3, 1, 2]`, kanade: `【3, 1, 2】`, want: "3"},
	std.StatsMean:   {haja: `[1, 2, 3, 4]`, kanade: `【1, 2, 3, 4】`, want: "2.5"},
	std.StatsMedian: {haja: `[5, 1, 3]`, kanade: `【5, 1, 3】`, want: "3"},
	std.StatsStdev:  {haja: `[2, 4, 4, 4, 5, 5, 7, 9]`, kanade: `【2, 4, 4, 4, 5, 5, 7, 9】`, want: "2.138089935299395"},

	std.TextUpper:      {haja: `"abc"`, kanade: `「abc」`, want: "ABC"},
	std.TextLower:      {haja: `"ABC"`, kanade: `「ABC」`, want: "abc"},
	std.TextStrip:      {haja: `"  a b  "`, kanade: `「  a b  」`, want: "a b"},
	std.TextPadLeft:    {haja: `"7", 3, "0"`, kanade: `「7」, 3, 「0」`, want: "007"},
	std.TextPadRight:   {haja: `"7", 3, "0"`, kanade: `「7」, 3, 「0」`, want: "700"},
	std.TextRepeat:     {haja: `"ab", 3`, kanade: `「ab」, 3`, want: "ababab"},
	std.TextReverse:    {haja: `"abc"`, kanade: `「abc」`, want: "cba"},
	std.TextStartsWith: {haja: `"hello", "he"`, kanade: `「hello」, 「he」`, want: "참", wantKanade: "真"},
	std.TextEndsWith:   {haja: `"hello", "lo"`, kanade: `「hello」, 「lo」`, want: "참", wantKanade: "真"},
	std.TextJoin:       {haja: `["a", "b"], "-"`, kanade: `【「a」, 「b」】, 「-」`, want: "a-b"},
	std.TextCount:      {haja: `"banana", "an"`, kanade: `「banana」, 「an」`, want: "2"},
	std.TextFind:       {haja: `"banana", "nan"`, kanade: `「banana」, 「nan」`, want: "3"},

	std.ListSort:    {haja: `[3, 1, 2]`, kanade: `【3, 1, 2】`, want: "[1, 2, 3]"},
	std.ListReverse: {haja: `[1, 2, 3]`, kanade: `【1, 2, 3】`, want: "[3, 2, 1]"},
	std.ListUnique:  {haja: `[1, 1, 2]`, kanade: `【1, 1, 2】`, want: "[1, 2]"},
	std.ListRange:   {haja: `1, 5`, kanade: `1, 5`, want: "[1, 2, 3, 4, 5]"},
	std.ListFlatten: {haja: `[[1, 2], [3]]`, kanade: `【【1, 2】, 【3】】`, want: "[1, 2, 3]"},
	std.ListChunk:   {haja: `[1, 2, 3], 2`, kanade: `【1, 2, 3】, 2`, want: "[[1, 2], [3]]"},
	std.ListZip:     {haja: `[1, 2], ["a", "b"]`, kanade: `【1, 2】, 【「a」, 「b」】`, want: "[[1, a], [2, b]]"},

	std.EncodingBase64Encode: {haja: `"hi"`, kanade: `「hi」`, want: "aGk="},
	std.EncodingBase64Decode: {haja: `"aGk="`, kanade: `「aGk=」`, want: "hi"},
	std.EncodingURLEncode:    {haja: `"a b"`, kanade: `「a b」`, want: "a%20b"},
	std.EncodingURLDecode:    {haja: `"a%20b"`, kanade: `「a%20b」`, want: "a b"},

	std.HashSHA256: {haja: `"abc"`, kanade: `「abc」`, want: "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"},
	std.RandomUUID: {haja: ``, kanade: ``, matches: "^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$"},

	std.FileRead:   {haja: `"a.txt"`, kanade: `「a.txt」`, files: map[string]string{"a.txt": "안녕"}, want: "안녕"},
	std.FileLines:  {haja: `"a.txt"`, kanade: `「a.txt」`, files: map[string]string{"a.txt": "가\r\n나\n"}, want: "[가, 나]"},
	std.FileWrite:  {haja: `"o.txt", "x"`, kanade: `「o.txt」, 「x」`, files: map[string]string{}, want: "비어있음", wantKanade: "空っぽ"},
	std.FileAppend: {haja: `"a.txt", "x"`, kanade: `「a.txt」, 「x」`, files: map[string]string{"a.txt": "a"}, want: "비어있음", wantKanade: "空っぽ"},
	std.FileExists: {haja: `"a.txt"`, kanade: `「a.txt」`, files: map[string]string{"a.txt": "a"}, want: "참", wantKanade: "真"},
	std.FileIsDir:  {haja: `"."`, kanade: `「.」`, files: map[string]string{}, want: "참", wantKanade: "真"},
	std.FileDelete: {haja: `"a.txt"`, kanade: `「a.txt」`, files: map[string]string{"a.txt": "a"}, want: "비어있음", wantKanade: "空っぽ"},
	std.FileList:   {haja: `"."`, kanade: `「.」`, files: map[string]string{"b.txt": "", "a.txt": ""}, want: "[a.txt, b.txt]"},
	std.FileMkdir:  {haja: `"d/e"`, kanade: `「d/e」`, files: map[string]string{}, want: "비어있음", wantKanade: "空っぽ"},
	std.FileMove:   {haja: `"a.txt", "b.txt"`, kanade: `「a.txt」, 「b.txt」`, files: map[string]string{"a.txt": "a"}, want: "비어있음", wantKanade: "空っぽ"},

	std.ListMap:    {prelude: hajaDouble, preludeKanade: kanadeDouble, haja: `[1, 2, 3], <두배>`, kanade: `【1, 2, 3】, 〈二倍〉`, want: "[2, 4, 6]"},
	std.ListFilter: {prelude: hajaEven, preludeKanade: kanadeEven, haja: `[1, 2, 3, 4], <짝수>`, kanade: `【1, 2, 3, 4】, 〈偶数〉`, want: "[2, 4]"},
	std.ListReduce: {prelude: hajaAdd, preludeKanade: kanadeAdd, haja: `[1, 2, 3, 4], <더하기>, 0`, kanade: `【1, 2, 3, 4】, 〈足す〉, 0`, want: "10"},
	std.ListFind:   {prelude: hajaEven, preludeKanade: kanadeEven, haja: `[1, 3, 4, 6], <짝수>`, kanade: `【1, 3, 4, 6】, 〈偶数〉`, want: "4"},
	std.ListAny:    {prelude: hajaEven, preludeKanade: kanadeEven, haja: `[1, 3, 4], <짝수>`, kanade: `【1, 3, 4】, 〈偶数〉`, want: "참", wantKanade: "真"},
	std.ListAll:    {prelude: hajaEven, preludeKanade: kanadeEven, haja: `[2, 4], <짝수>`, kanade: `【2, 4】, 〈偶数〉`, want: "참", wantKanade: "真"},
	std.ListSortBy: {prelude: hajaNegate, preludeKanade: kanadeNegate, haja: `[1, 3, 2], <음수>`, kanade: `【1, 3, 2】, 〈負〉`, want: "[3, 2, 1]"},

	std.SocketConnect:     {elsewhere: "net_test.go"},
	std.SocketListen:      {elsewhere: "net_test.go"},
	std.SocketAccept:      {elsewhere: "net_test.go"},
	std.SocketSend:        {elsewhere: "net_test.go"},
	std.SocketReceive:     {elsewhere: "net_test.go"},
	std.SocketReceiveLine: {elsewhere: "net_test.go"},
	std.SocketTimeout:     {elsewhere: "net_test.go"},
	std.SocketAddress:     {elsewhere: "net_test.go"},
	std.SocketClose:       {elsewhere: "net_test.go"},
	std.HTTPGet:           {elsewhere: "net_test.go"},
	std.HTTPPost:          {elsewhere: "net_test.go"},
	std.HTTPRequest:       {elsewhere: "net_test.go"},

	std.PathJoin:      {haja: `"a", "b", "c.txt"`, kanade: `「a」, 「b」, 「c.txt」`, want: "a/b/c.txt"},
	std.PathDirname:   {haja: `"a/b/c.txt"`, kanade: `「a/b/c.txt」`, want: "a/b"},
	std.PathBasename:  {haja: `"a/b/c.txt"`, kanade: `「a/b/c.txt」`, want: "c.txt"},
	std.PathExt:       {haja: `"a/b/c.tar.gz"`, kanade: `「a/b/c.tar.gz」`, want: ".gz"},
	std.PathStem:      {haja: `"a/b/c.txt"`, kanade: `「a/b/c.txt」`, want: "c"},
	std.PathWithExt:   {haja: `"a/c.txt", "md"`, kanade: `「a/c.txt」, 「md」`, want: "a/c.md"},
	std.PathNormalize: {haja: `"a//b/./c/../d"`, kanade: `「a//b/./c/../d」`, want: "a/b/d"},
	std.PathIsAbs:     {haja: `"C:/data"`, kanade: `「C:/data」`, want: "참", wantKanade: "真"},
	std.PathParts:     {haja: `"/a/b"`, kanade: `「/a/b」`, want: "[/, a, b]"},

	std.RegexTest:    {haja: `"abc123", "[0-9]+"`, kanade: `「abc123」, 「[0-9]+」`, want: "참", wantKanade: "真"},
	std.RegexFind:    {haja: `"abc123", "[0-9]+"`, kanade: `「abc123」, 「[0-9]+」`, want: "123"},
	std.RegexGroups:  {haja: `"a1b2", "([a-z])([0-9])"`, kanade: `「a1b2」, 「([a-z])([0-9])」`, want: "[a1, a, 1]"},
	std.RegexFindAll: {haja: `"a1b22", "[0-9]+"`, kanade: `「a1b22」, 「[0-9]+」`, want: "[1, 22]"},
	std.RegexReplace: {haja: `"a1b2", "[0-9]", "#"`, kanade: `「a1b2」, 「[0-9]」, 「#」`, want: "a#b#"},
	std.RegexSplit:   {haja: `"a, b,c", ", ?"`, kanade: `「a, b,c」, 「, ?」`, want: "[a, b, c]"},
}

// The functions the callback cases pass in.
const (
	hajaDouble   = "<두배>를 만들자 ('x'):\n    ('x' * 2)를 돌려주자\n"
	hajaEven     = "<짝수>를 만들자 ('x'):\n    만약 ('x' % 2 == 0) 라면:\n        참을 돌려주자\n    거짓을 돌려주자\n"
	hajaAdd      = "<더하기>를 만들자 ('합', 'x'):\n    ('합' + 'x')를 돌려주자\n"
	hajaNegate   = "<음수>를 만들자 ('x'):\n    (0 - 'x')를 돌려주자\n"
	kanadeDouble = "〈二倍〉を作ろう(『x』):\n    (『x』 * 2)を返そう\n"
	kanadeEven   = "〈偶数〉を作ろう(『x』):\n    もし(『x』 % 2 == 0)なら:\n        真を返そう\n    偽を返そう\n"
	kanadeAdd    = "〈足す〉を作ろう(『合計』, 『x』):\n    (『合計』 + 『x』)を返そう\n"
	kanadeNegate = "〈負〉を作ろう(『x』):\n    (0 - 『x』)を返そう\n"
)

func TestStdFunctionsWorkEverywhere(t *testing.T) {
	for _, m := range std.Modules {
		for _, id := range m.Functions {
			c, ok := stdCases[id]
			if !ok {
				t.Errorf("no test case for %s in stdCases", id)
				continue
			}
			if c.elsewhere != "" {
				continue
			}
			reset := func() {}
			if c.files != nil {
				dir := inTempDir(t, c.files)
				reset = func() { resetFolder(t, dir, c.files) }
			}
			hajaName, kanadeName := std.FunctionName(std.Haja, id), std.FunctionName(std.Kanade, id)
			wrap := c.wrap
			if wrap == "" {
				wrap = "%s"
			}
			hajaCode := fmt.Sprintf("[%s]에서 <%s>를 가져오자\n%s틀\"{%s}\"를 출력하자\n",
				std.ModuleName(std.Haja, m.ID), hajaName, c.prelude, fmt.Sprintf(wrap, fmt.Sprintf("<%s>(%s)", hajaName, c.haja)))
			kanadeCode := fmt.Sprintf("【%s】から〈%s〉を持ってこよう\n%s枠「{%s}」を出力しよう\n",
				std.ModuleName(std.Kanade, m.ID), kanadeName, c.preludeKanade, fmt.Sprintf(wrap, fmt.Sprintf("〈%s〉(%s)", kanadeName, c.kanade)))

			reset()
			tw, err := runHaja(t, hajaCode)
			if err != nil {
				t.Fatalf("haja tree-walker %s: %v", id, err)
			}
			reset()
			bw, err := runBytecode(t, hajaCode)
			if err != nil {
				t.Fatalf("haja bytecode %s: %v", id, err)
			}
			reset()
			tk, err := runKanade(t, kanadeCode)
			if err != nil {
				t.Fatalf("kanade tree-walker %s: %v", id, err)
			}
			reset()
			bk, err := runKanadeBytecode(t, kanadeCode)
			if err != nil {
				t.Fatalf("kanade bytecode %s: %v", id, err)
			}
			wantKanade := c.wantKanade
			if wantKanade == "" {
				wantKanade = c.want
			}
			if c.matches != "" {
				re := regexp.MustCompile(c.matches)
				for label, out := range map[string][]string{"haja/tree": tw.Output, "haja/bytecode": bw.Output, "kanade/tree": tk.Output, "kanade/bytecode": bk.Output} {
					if !re.MatchString(strings.Join(out, "|")) {
						t.Errorf("%s %s: got %v, want a match for %s", id, label, out, c.matches)
					}
				}
				continue
			}
			for label, got := range map[string]struct {
				out  []string
				want string
			}{
				"haja/tree":       {tw.Output, c.want},
				"haja/bytecode":   {bw.Output, c.want},
				"kanade/tree":     {tk.Output, wantKanade},
				"kanade/bytecode": {bk.Output, wantKanade},
			} {
				if strings.Join(got.out, "|") != got.want {
					t.Errorf("%s %s: got %v, want [%s]", id, label, got.out, got.want)
				}
			}
		}
	}
}

// resetFolder empties dir and writes files into it again, so a case that
// deletes or moves a file can run once per engine.
func resetFolder(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		os.RemoveAll(filepath.Join(dir, e.Name()))
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
