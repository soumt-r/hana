// Package pkg reads a package's hana.pkg.json — the package's own metadata:
// its name and version, the entry point for each language, and the prebuilt
// native library for each platform.
//
// A package folder needs no manifest: without one, the conventional layout
// (haja/index.hj, kanade/index.knd) is assumed and there is no native library
// beyond the legacy <name>.dll next to it.
package pkg

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// FileName is the manifest's name inside a package folder.
const FileName = "hana.pkg.json"

// Manifest is a parsed hana.pkg.json.
type Manifest struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	// Entry maps a language key ("haja", "kanade") to the entry point's path,
	// relative to the package folder. A missing key means the conventional path.
	Entry map[string]string `json:"entry"`
	// Native maps a platform ("windows-amd64", "linux-arm64", ...) to the
	// prebuilt native library for it.
	Native       map[string]NativeFile `json:"native"`
	Dependencies map[string]string     `json:"dependencies"`
}

// NativeFile is one platform's native library: File is its path inside the
// package folder; URL and SHA256 say where the package manager downloads it
// from and what it must hash to (used by the installer, not by `hana run`).
type NativeFile struct {
	File   string `json:"file"`
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
}

var (
	languages  = map[string]bool{"haja": true, "kanade": true}
	platformRe = regexp.MustCompile(`^[a-z0-9]+-[a-z0-9]+$`)
	sha256Re   = regexp.MustCompile(`^[0-9a-fA-F]{64}$`)
)

// Load reads dir/hana.pkg.json. A package without a manifest is fine: Load
// returns (nil, nil). A manifest that is unreadable or invalid is an error
// whose text names the problem.
func Load(dir string) (*Manifest, error) {
	data, err := os.ReadFile(filepath.Join(dir, FileName))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return Parse(data)
}

// Parse reads and validates manifest text.
func Parse(data []byte) (*Manifest, error) {
	var m Manifest
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&m); err != nil {
		return nil, fmt.Errorf("invalid JSON: %v", err)
	}
	if err := m.validate(); err != nil {
		return nil, err
	}
	return &m, nil
}

func (m *Manifest) validate() error {
	for lang, entry := range m.Entry {
		if !languages[lang] {
			return fmt.Errorf("entry: unknown language %q (use haja or kanade)", lang)
		}
		if err := relativePath("entry."+lang, entry); err != nil {
			return err
		}
	}
	for platform, n := range m.Native {
		if !platformRe.MatchString(platform) {
			return fmt.Errorf("native: %q is not a platform like windows-amd64", platform)
		}
		if err := relativePath("native."+platform+".file", n.File); err != nil {
			return err
		}
		if n.SHA256 != "" && !sha256Re.MatchString(n.SHA256) {
			return fmt.Errorf("native.%s.sha256 must be 64 hex digits", platform)
		}
	}
	return nil
}

// relativePath requires a non-empty path that stays inside the package folder.
func relativePath(field, p string) error {
	if p == "" {
		return fmt.Errorf("%s is empty", field)
	}
	clean := path.Clean(strings.ReplaceAll(p, "\\", "/"))
	if path.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, "../") || filepath.IsAbs(p) {
		return fmt.Errorf("%s must stay inside the package folder", field)
	}
	return nil
}

// EntryPath is the entry point for a language: the manifest's, else the
// conventional <lang>/index<ext>. It works on a nil Manifest (no manifest).
func (m *Manifest) EntryPath(lang, ext string) string {
	if m != nil {
		if e, ok := m.Entry[lang]; ok {
			return filepath.FromSlash(strings.ReplaceAll(e, "\\", "/"))
		}
	}
	return filepath.Join(lang, "index"+ext)
}

// Platform is the running platform's key in a manifest's native table.
func Platform() string { return runtime.GOOS + "-" + runtime.GOARCH }

// NativeFor is the native library declared for a platform, if any.
func (m *Manifest) NativeFor(platform string) (NativeFile, bool) {
	if m == nil {
		return NativeFile{}, false
	}
	n, ok := m.Native[platform]
	return n, ok
}

// HasNative reports whether the manifest declares any native library at all.
func (m *Manifest) HasNative() bool { return m != nil && len(m.Native) > 0 }

// LibraryExt is the shared-library extension of an operating system.
func LibraryExt(goos string) string {
	switch goos {
	case "windows":
		return ".dll"
	case "darwin":
		return ".dylib"
	}
	return ".so"
}
