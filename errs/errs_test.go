package errs

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

var verbRe = regexp.MustCompile(`%(\[\d+\])?[a-z]`)

func verbCount(tmpl string) int { return len(verbRe.FindAllString(tmpl, -1)) }

var catalogs = map[string]Catalog{"en": enCatalog, "ko": koCatalog, "ja": jaCatalog}

// Every Code must exist in every catalog, and no catalog may carry a Code
// that isn't declared — so adding a Code without translating it (or leaving
// a stale entry behind) fails here instead of degrading silently at runtime.
func TestCatalogsCoverExactlyAllCodes(t *testing.T) {
	declared := map[Code]bool{}
	for _, c := range allCodes {
		if declared[c] {
			t.Errorf("Code %q listed twice in allCodes", c)
		}
		declared[c] = true
	}
	for lang, cat := range catalogs {
		for c := range declared {
			if _, ok := cat[c]; !ok {
				t.Errorf("%s catalog is missing %q", lang, c)
			}
		}
		for c := range cat {
			if !declared[c] {
				t.Errorf("%s catalog has %q, which is not in allCodes", lang, c)
			}
		}
	}
}

// One Code carries one argument list, so every language's template must
// interpolate the same number of values — otherwise throwing it renders
// "%!s(MISSING)" or "%!(EXTRA ...)" in exactly one language.
func TestTemplatesAgreeOnArity(t *testing.T) {
	for _, c := range allCodes {
		want := verbCount(enCatalog[c])
		for lang, cat := range catalogs {
			if got := verbCount(cat[c]); got != want {
				t.Errorf("%q: %s template has %d verbs, en has %d", c, lang, got, want)
			}
		}
	}
}

var indexRe = regexp.MustCompile(`%\[(\d+)\]`)

func TestTemplatesRenderCleanly(t *testing.T) {
	for _, c := range allCodes {
		// %d needs a number; every other verb in the catalogs takes text.
		// Verbs may be indexed (%[2]s), so place each argument by its index.
		args := make([]interface{}, 0)
		next := 0
		for _, v := range verbRe.FindAllString(enCatalog[c], -1) {
			idx := next
			if m := indexRe.FindStringSubmatch(v); m != nil {
				idx, _ = strconv.Atoi(m[1])
				idx--
			}
			next = idx + 1
			for len(args) <= idx {
				args = append(args, "x")
			}
			if strings.HasSuffix(v, "d") {
				args[idx] = 1
			}
		}
		for _, loc := range []Locale{English, Korean, Japanese} {
			out := Localize(loc, New(c, args...))
			if strings.Contains(out, "%!") {
				t.Errorf("%q (locale %d) rendered badly: %s", c, loc, out)
			}
		}
	}
}

// Korean is 해요체 (ends in 요), Japanese is です・ます調. This turns the
// project's stated tone goal into a check instead of a convention.
func TestTone(t *testing.T) {
	jaEnd := []string{"ます。", "ません。", "です。", "ました。"}
	for _, c := range allCodes {
		ko := koCatalog[c]
		if !strings.HasSuffix(ko, "요.") {
			t.Errorf("ko %q must end in 요. (해요체): %s", c, ko)
		}
		ja := jaCatalog[c]
		ok := false
		for _, e := range jaEnd {
			if strings.HasSuffix(ja, e) {
				ok = true
			}
		}
		if !ok {
			t.Errorf("ja %q must end in です/ます調: %s", c, ja)
		}
	}
}

func TestKindPrefixStaysEnglish(t *testing.T) {
	for _, c := range allCodes {
		for _, loc := range []Locale{English, Korean, Japanese} {
			out := Localize(loc, New(c, make([]interface{}, verbCount(enCatalog[c]))...))
			if !strings.HasPrefix(out, c.Kind()+": ") {
				t.Errorf("%q (locale %d) lost its %q prefix: %s", c, loc, c.Kind(), out)
			}
		}
	}
}

func TestLocalizeDocQuotedMessages(t *testing.T) {
	cases := []struct {
		loc  Locale
		err  error
		want string
	}{
		{Korean, New(ListIndexOutOfRange), "IndexOutOfBoundsError: 목록의 길이를 벗어난 위치(인덱스)예요."},
		{Japanese, New(ListIndexOutOfRange), "IndexOutOfBoundsError: リストの長さを超えた位置（インデックス）です。"},
		{Korean, New(DictKeyNotFound, "직업"), "KeyError: 사전에서 '직업' 이름을 찾을 수 없어요."},
		{Japanese, New(DictKeyNotFound, "職業"), "KeyError: 辞書から '職業' という名前を見つけることができません。"},
		{Korean, New(PrivateMethodAccess, "일기읽기"), "AccessViolationError: '일기읽기' 기능은 내부 전용(private)이라 외부에서 부를 수 없어요."},
		{Korean, New(ConstantAssignment, "생일"), "ConstantAssignmentError: 상수 '생일'의 값은 변경할 수 없어요."},
		{Japanese, New(ConstantAssignment, "誕生日"), "ConstantAssignmentError: 定数 '誕生日' の値は変更できません。"},
		{Korean, New(InterfaceNotImplemented, "고장난로봇", "움직이는것", "멈추기"), "InterfaceImplementationError: 클래스 '고장난로봇'은(는) 인터페이스 '움직이는것'의 '멈추기' 메서드를 구현해야 해요."},
		{English, New(InterfaceNotImplemented, "Robot", "Mover", "stop"), "InterfaceImplementationError: Class 'Robot' must implement method 'stop' of interface 'Mover'."},
	}
	for _, tc := range cases {
		if got := Localize(tc.loc, tc.err); got != tc.want {
			t.Errorf("got %q, want %q", got, tc.want)
		}
	}
}

func TestLocalizePassesThroughOtherErrors(t *testing.T) {
	plain := errors.New("boom")
	if got := Localize(Korean, plain); got != "boom" {
		t.Errorf("plain error changed: %q", got)
	}
	wrapped := fmt.Errorf("ctx: %w", New(DivideByZero))
	if got := Localize(Korean, wrapped); got != "DivideByZeroError: 0으로 나눌 수 없어요." {
		t.Errorf("wrapped *Error not localized: %q", got)
	}
}

func TestErrorMethodIsEnglish(t *testing.T) {
	if got := New(NotCallable).Error(); got != "TypeError: Not callable." {
		t.Errorf("Error() = %q", got)
	}
}

func TestLabelsCoverEveryLocale(t *testing.T) {
	for l := LabelFileRead; l <= LabelBytecodeSave; l++ {
		for _, loc := range []Locale{English, Korean, Japanese} {
			if labelText[loc][l] == "" {
				t.Errorf("label %d has no text for locale %d", l, loc)
			}
		}
	}
}

// The docs sites' browser engines import a TypeScript catalog generated from
// these Go catalogs (cmd/errsgen). When those repos sit next to hana, fail if
// their copy is stale — otherwise editing a message here would quietly leave
// the Playground printing the old wording.
func TestGeneratedTypeScriptCatalogsAreCurrent(t *testing.T) {
	want := TypeScriptCatalog()
	for _, rel := range []string{
		"../../haja-docs/src/utils/haja/errCatalog.ts",
		"../../kanade-docs/src/utils/kanade/errCatalog.ts",
	} {
		got, err := os.ReadFile(rel)
		if err != nil {
			t.Logf("skipping %s: %v", rel, err)
			continue
		}
		if string(got) != want {
			t.Errorf("%s is stale — run: go run ./cmd/errsgen <both errCatalog.ts paths>", rel)
		}
	}
}
