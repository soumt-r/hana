package pkg

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestHelperScript is the install script of these tests: the test binary run
// again with HANA_TEST_SCRIPT set writes built.txt into the folder it runs in.
func TestHelperScript(t *testing.T) {
	if os.Getenv("HANA_TEST_SCRIPT") != "1" {
		return
	}
	if err := os.WriteFile("built.txt", []byte("built"), 0o644); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}

func scriptManifest(t *testing.T, name string) string {
	t.Helper()
	t.Setenv("HANA_TEST_SCRIPT", "1")
	cmd, _ := json.Marshal([]string{os.Args[0], "-test.run=TestHelperScript"})
	return `{"name": "` + name + `", "scripts": {"install": ` + string(cmd) + `}}`
}

func TestAnInstallScriptOnlyRunsForATrustedPackage(t *testing.T) {
	proj := newProject(t)
	src := &fakeSource{source: map[string]map[string]string{pkgA: {"1.0.0": scriptManifest(t, pkgA)}}}
	var events []string
	in := &Installer{Src: src, Proj: proj, Out: io.Discard, Log: func(e string, a ...interface{}) { events = append(events, e) }}
	if _, err := in.Add(pkgA, nil); err != nil {
		t.Fatal(err)
	}
	dir, _ := CacheDir(pkgA, "1.0.0")
	if _, err := os.Stat(filepath.Join(dir, "built.txt")); err == nil {
		t.Fatal("the script ran although the project does not trust the package")
	}
	if !contains(events, "skipScript") {
		t.Errorf("the skipped script was not reported: %v", events)
	}
	lock, _ := LoadLock(proj.Dir)
	if lock[pkgA].Scripts != "" {
		t.Error("a script that did not run must not be recorded")
	}

	proj.TrustedScripts = []string{pkgA}
	events = nil
	lock, err := (&Installer{Src: src, Proj: proj, Out: io.Discard, Log: in.Log}).Install()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "built.txt")); err != nil {
		t.Fatalf("the trusted script did not run: %v", err)
	}
	if lock[pkgA].Scripts == "" {
		t.Error("the lock does not record the approved script")
	}
	if saved, _ := LoadLock(proj.Dir); saved[pkgA].Scripts != lock[pkgA].Scripts {
		t.Error("the lock file was not rewritten with the script hash")
	}

	events = nil
	if _, err := (&Installer{Src: src, Proj: proj, Out: io.Discard, Log: in.Log}).Install(); err != nil {
		t.Fatal(err)
	}
	if contains(events, "script") {
		t.Error("the script ran a second time")
	}
}

func TestAChangedScriptNeedsANewApproval(t *testing.T) {
	proj := newProject(t)
	proj.TrustedScripts = []string{pkgA}
	src := &fakeSource{source: map[string]map[string]string{pkgA: {"1.0.0": scriptManifest(t, pkgA)}}}
	if _, err := (&Installer{Src: src, Proj: proj, Out: io.Discard}).Add(pkgA, nil); err != nil {
		t.Fatal(err)
	}
	lock, _ := LoadLock(proj.Dir)
	e := lock[pkgA]
	e.Scripts = "0000"
	lock[pkgA] = e
	if err := SaveLock(proj.Dir, lock); err != nil {
		t.Fatal(err)
	}

	t.Setenv(EnvHome, t.TempDir())
	_, err := (&Installer{Src: src, Proj: proj, Out: io.Discard}).Install()
	var pe *Error
	if !errors.As(err, &pe) || pe.Code != ScriptChanged {
		t.Fatalf("err = %v, want ScriptChanged", err)
	}

	t.Setenv(EnvHome, t.TempDir())
	if _, err := (&Installer{Src: src, Proj: proj, Out: io.Discard, Approve: map[string]bool{pkgA: true}}).Add(pkgA, nil); err != nil {
		t.Fatalf("approving again: %v", err)
	}
	if saved, _ := LoadLock(proj.Dir); saved[pkgA].Scripts == "0000" || saved[pkgA].Scripts == "" {
		t.Errorf("the new script hash was not recorded: %+v", saved[pkgA])
	}
}

func TestAllowScriptsIsRememberedInHanaJSON(t *testing.T) {
	proj := newProject(t)
	src := &fakeSource{source: map[string]map[string]string{pkgA: {"1.0.0": scriptManifest(t, pkgA)}}}
	in := &Installer{Src: src, Proj: proj, Out: io.Discard, Approve: map[string]bool{pkgA: true}}
	if _, err := in.Add(pkgA, nil); err != nil {
		t.Fatal(err)
	}
	reread, err := LoadProject(proj.Dir)
	if err != nil || len(reread.TrustedScripts) != 1 || reread.TrustedScripts[0] != pkgA {
		t.Fatalf("hana.json = %+v, %v", reread, err)
	}
}

func nativeManifest(name, url, hash string) string {
	return `{"name": "` + name + `", "native": {"` + Platform() + `": {"file": "native/lib.bin", "url": "` + url + `", "sha256": "` + hash + `"}}}`
}

func TestANativeLibraryIsDownloadedAndChecked(t *testing.T) {
	content := []byte("library bytes")
	sum := sha256.Sum256(content)
	good := hex.EncodeToString(sum[:])
	fetch := func(url string) (io.ReadCloser, error) { return io.NopCloser(strings.NewReader(string(content))), nil }

	proj := newProject(t)
	src := &fakeSource{source: map[string]map[string]string{pkgA: {"1.0.0": nativeManifest(pkgA, "https://example.test/lib.bin", good)}}}
	if _, err := (&Installer{Src: src, Proj: proj, Fetch: fetch}).Add(pkgA, nil); err != nil {
		t.Fatal(err)
	}
	dir, _ := CacheDir(pkgA, "1.0.0")
	if data, err := os.ReadFile(filepath.Join(dir, "native", "lib.bin")); err != nil || string(data) != string(content) {
		t.Fatalf("native/lib.bin = %q, %v", data, err)
	}

	for name, tc := range map[string]struct {
		hash string
		code Code
	}{
		"wrong hash": {strings.Repeat("0", 64), NativeHashMismatch},
		"no hash":    {"", NativeNoHash},
	} {
		t.Run(name, func(t *testing.T) {
			proj := newProject(t)
			src := &fakeSource{source: map[string]map[string]string{pkgA: {"1.0.0": nativeManifest(pkgA, "https://example.test/lib.bin", tc.hash)}}}
			_, err := (&Installer{Src: src, Proj: proj, Fetch: fetch}).Add(pkgA, nil)
			var pe *Error
			if !errors.As(err, &pe) || pe.Code != tc.code {
				t.Fatalf("err = %v, want %s", err, tc.code)
			}
			dir, _ := CacheDir(pkgA, "1.0.0")
			if _, err := os.Stat(filepath.Join(dir, "native", "lib.bin")); err == nil {
				t.Error("a library that failed the check was kept")
			}
		})
	}
}

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}
