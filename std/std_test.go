package std

import (
	"os"
	"strings"
	"testing"
)

func TestEveryModuleAndFunctionHasANameInEveryLanguage(t *testing.T) {
	for _, lang := range Languages {
		seen := map[string]string{}
		for _, m := range Modules {
			if ModuleName(lang, m.ID) == "" {
				t.Errorf("%s: module %q has no name", lang, m.ID)
			}
			seenInModule := map[string]string{}
			for _, f := range m.Functions {
				name := FunctionName(lang, f)
				if name == "" {
					t.Errorf("%s: function %q has no name", lang, f)
				}
				if !strings.HasPrefix(f, m.ID+".") {
					t.Errorf("function ID %q should be prefixed with its module %q", f, m.ID)
				}
				if other, dup := seenInModule[name]; dup {
					t.Errorf("%s: %q names both %q and %q in module %q", lang, name, other, f, m.ID)
				}
				seenInModule[name] = f
			}
			if other, dup := seen[ModuleName(lang, m.ID)]; dup {
				t.Errorf("%s: module name %q used by %q and %q", lang, ModuleName(lang, m.ID), other, m.ID)
			}
			seen[ModuleName(lang, m.ID)] = m.ID
		}
	}
}

func TestNameTablesHaveNoStrayEntries(t *testing.T) {
	known := map[string]bool{}
	for _, m := range Modules {
		known[m.ID] = true
		for _, f := range m.Functions {
			known[f] = true
		}
	}
	for lang, table := range names {
		for id := range table {
			if !known[id] {
				t.Errorf("%s names %q, which is not a declared module or function", lang, id)
			}
		}
	}
}

// The docs sites keep a generated copy of these names; a stale copy would let
// the browser Playground disagree with hana about what `[수학]` contains.
func TestGeneratedTypeScriptNamesAreCurrent(t *testing.T) {
	for lang, rel := range map[string]string{
		Hari:   "../../hari-docs/src/utils/hari/stdNames.ts",
		Kanade: "../../kanade-docs/src/utils/kanade/stdNames.ts",
	} {
		got, err := os.ReadFile(rel)
		if err != nil {
			t.Logf("skipping %s: %v", rel, err)
			continue
		}
		if strings.ReplaceAll(string(got), "\r\n", "\n") != TypeScriptNames(lang) {
			t.Errorf("%s is stale — run: go run ./cmd/stdgen -hari <file> -kanade <file>", rel)
		}
	}
}

// A NativeOnly module is emitted with its flag; the others carry none, so
// adding the flag never changes the output for ordinary modules.
func TestTypeScriptNamesMarkNativeOnlyModules(t *testing.T) {
	names[Hari]["fake"] = "가짜"
	names[Hari]["fake.read"] = "읽기"
	defer func() {
		delete(names[Hari], "fake")
		delete(names[Hari], "fake.read")
	}()
	out := typeScriptNames(Hari, []Module{
		{ID: "math", Functions: []string{MathCeil}},
		{ID: "fake", Functions: []string{"fake.read"}, NativeOnly: true},
	})
	if strings.Count(out, "nativeOnly: true") != 1 {
		t.Errorf("want exactly one nativeOnly marker, got:\n%s", out)
	}
	want := "  {\n    id: \"fake\",\n    name: \"가짜\",\n    nativeOnly: true,\n"
	if !strings.Contains(out, want) {
		t.Errorf("the native-only module should carry the flag right after its name:\n%s", out)
	}
	if strings.Contains(out, "id: \"math\",\n    name: \"수학\",\n    nativeOnly") {
		t.Errorf("an ordinary module must not carry the flag:\n%s", out)
	}
}
