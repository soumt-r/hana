package pkg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const goodManifest = `{
  "name": "example/mypkg",
  "version": "1.2.0",
  "entry": {"haja": "haja/index.hj", "kanade": "kanade/index.knd"},
  "native": {
    "windows-amd64": {"file": "native/mypkg-windows-amd64.dll", "sha256": "0000000000000000000000000000000000000000000000000000000000000000"},
    "linux-arm64": {"file": "native/libmypkg.so", "url": "https://example.com/libmypkg.so"}
  },
  "dependencies": {"example/utils": "2.0.0"}
}`

func TestParseAGoodManifest(t *testing.T) {
	m, err := Parse([]byte(goodManifest))
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != "example/mypkg" || m.Version != "1.2.0" || m.Dependencies["example/utils"] != "2.0.0" {
		t.Errorf("unexpected manifest: %+v", m)
	}
	if n, ok := m.NativeFor("linux-arm64"); !ok || n.File != "native/libmypkg.so" || n.URL == "" {
		t.Errorf("native lookup wrong: %+v %v", n, ok)
	}
	if _, ok := m.NativeFor("darwin-arm64"); ok {
		t.Error("a platform that is not declared must not be found")
	}
	if !m.HasNative() {
		t.Error("HasNative should be true")
	}
}

func TestEntryPathFallsBackToTheConvention(t *testing.T) {
	var none *Manifest
	if got := none.EntryPath("haja", ".hj"); got != filepath.Join("haja", "index.hj") {
		t.Errorf("no manifest: %q", got)
	}
	m, _ := Parse([]byte(`{"entry": {"kanade": "src/main.knd"}}`))
	if got := m.EntryPath("kanade", ".knd"); got != filepath.Join("src", "main.knd") {
		t.Errorf("declared entry: %q", got)
	}
	if got := m.EntryPath("haja", ".hj"); got != filepath.Join("haja", "index.hj") {
		t.Errorf("undeclared language keeps the convention: %q", got)
	}
}

func TestInvalidManifests(t *testing.T) {
	cases := map[string]string{
		"not json":            `{`,
		"unknown field":       `{"nome": "x"}`,
		"unknown language":    `{"entry": {"ruby": "a.rb"}}`,
		"absolute entry":      `{"entry": {"haja": "/etc/x.hj"}}`,
		"entry leaves folder": `{"entry": {"haja": "../x.hj"}}`,
		"empty entry":         `{"entry": {"haja": ""}}`,
		"bad platform":        `{"native": {"windows": {"file": "a.dll"}}}`,
		"native leaves":       `{"native": {"linux-amd64": {"file": "../a.so"}}}`,
		"bad hash":            `{"native": {"linux-amd64": {"file": "a.so", "sha256": "abc"}}}`,
	}
	for name, text := range cases {
		if _, err := Parse([]byte(text)); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestLoadReadsTheFileAndAllowsNone(t *testing.T) {
	dir := t.TempDir()
	if m, err := Load(dir); m != nil || err != nil {
		t.Errorf("no manifest should be (nil, nil), got %v %v", m, err)
	}
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(goodManifest), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := Load(dir)
	if err != nil || m == nil || m.Name != "example/mypkg" {
		t.Fatalf("Load = %v, %v", m, err)
	}
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(`{"name": 3}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir); err == nil || !strings.Contains(err.Error(), "invalid JSON") {
		t.Errorf("a broken manifest should be reported, got %v", err)
	}
}

func TestLibraryExtension(t *testing.T) {
	for goos, want := range map[string]string{"windows": ".dll", "darwin": ".dylib", "linux": ".so", "freebsd": ".so"} {
		if got := LibraryExt(goos); got != want {
			t.Errorf("%s: %q, want %q", goos, got, want)
		}
	}
}

func TestRootsAndDirSearchOrder(t *testing.T) {
	shared := t.TempDir()
	if err := os.MkdirAll(filepath.Join(shared, "tools"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvPackages, shared)

	roots := Roots()
	if roots[0] != "packages" || roots[1] != shared {
		t.Fatalf("the project folder comes first, then $%s: %v", EnvPackages, roots)
	}
	if last := roots[len(roots)-1]; !strings.HasSuffix(last, "packages") || last == "packages" {
		t.Errorf("the folder next to the executable comes last: %q", last)
	}

	// Run from an empty folder: only the shared root has "tools".
	work := t.TempDir()
	old, _ := os.Getwd()
	if err := os.Chdir(work); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)
	if got := Dir("tools"); got != filepath.Join(shared, "tools") {
		t.Errorf("a package in a shared root is found from anywhere: %q", got)
	}
	if got := Dir("missing"); got != filepath.Join("packages", "missing") {
		t.Errorf("a package nobody has falls back to the project path: %q", got)
	}

	// A project's own package hides the shared one of the same name.
	if err := os.MkdirAll(filepath.Join("packages", "tools"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := Dir("tools"); got != filepath.Join("packages", "tools") {
		t.Errorf("the project's package should win: %q", got)
	}
}

func TestDirRefusesNamesThatClimbOut(t *testing.T) {
	for _, name := range []string{"", ".", "..", "../x", `a\b`, "a/b", "x\x00y"} {
		if got := Dir(name); filepath.Base(got) != ".no-such-package" {
			t.Errorf("Dir(%q) = %q, want the never-matching path", name, got)
		}
	}
}
