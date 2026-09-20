package pkg

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// fakeSource serves packages from memory: source[path][version] is the manifest
// (as JSON) of that version.
type fakeSource struct {
	source    map[string]map[string]string
	downloads []string
}

func (f *fakeSource) Tags(path string) ([]Tag, error) {
	var tags []Tag
	for v := range f.source[path] {
		ver, _ := ParseVersion(v)
		tags = append(tags, Tag{Version: ver, Name: "v" + v, Commit: fakeCommit(path, v)})
	}
	sortTags(tags)
	return tags, nil
}

func sortTags(tags []Tag) {
	for i := range tags {
		for j := i + 1; j < len(tags); j++ {
			if tags[j].Version.Compare(tags[i].Version) > 0 {
				tags[i], tags[j] = tags[j], tags[i]
			}
		}
	}
}

func fakeCommit(path, v string) string { return fmt.Sprintf("%x", len(path)*1000+len(v)) + v }

func (f *fakeSource) Download(path string, tag Tag, dest, wantCommit string) (string, error) {
	commit := tag.Commit
	if wantCommit != "" && wantCommit != commit {
		return "", newError(CommitMismatch, path, tag.Version.String(), wantCommit, commit)
	}
	f.downloads = append(f.downloads, path+"@"+tag.Version.String())
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return "", err
	}
	manifest := f.source[path][tag.Version.String()]
	if manifest != "" {
		if err := os.WriteFile(filepath.Join(dest, FileName), []byte(manifest), 0o644); err != nil {
			return "", err
		}
	}
	return commit, nil
}

func manifest(name string, deps map[string]string) string {
	data, _ := json.Marshal(map[string]interface{}{"name": name, "dependencies": deps})
	return string(data)
}

const (
	pkgA = "github.com/o/a"
	pkgB = "github.com/o/b"
	pkgC = "github.com/o/c"
)

func newProject(t *testing.T) *Project {
	t.Helper()
	t.Setenv(EnvHome, t.TempDir())
	return &Project{Dir: t.TempDir()}
}

func TestResolvePicksTheHighestRequestedVersion(t *testing.T) {
	src := &fakeSource{source: map[string]map[string]string{
		pkgA: {"1.0.0": manifest(pkgA, nil), "1.2.0": manifest(pkgA, nil), "1.5.0": manifest(pkgA, nil)},
		pkgB: {"1.0.0": manifest(pkgB, map[string]string{pkgA: "1.2.0"})},
	}}
	in := &Installer{Src: src, Proj: newProject(t)}
	lock, err := in.Resolve(map[string]string{pkgA: "1.0.0", pkgB: "1.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	if got := lock[pkgA].Version; got != "1.2.0" {
		t.Errorf("a = %s, want 1.2.0 (b needs 1.2.0)", got)
	}
	if got := lock[pkgB].Version; got != "1.0.0" {
		t.Errorf("b = %s", got)
	}
	if lock[pkgA].Commit == "" {
		t.Error("the lock has no commit for a")
	}
}

func TestResolveFollowsDependenciesOfDependencies(t *testing.T) {
	src := &fakeSource{source: map[string]map[string]string{
		pkgA: {"1.0.0": manifest(pkgA, map[string]string{pkgB: "2.0.0"})},
		pkgB: {"2.0.0": manifest(pkgB, map[string]string{pkgC: "0.1.0"})},
		pkgC: {"0.1.0": manifest(pkgC, nil)},
	}}
	in := &Installer{Src: src, Proj: newProject(t)}
	lock, err := in.Resolve(map[string]string{pkgA: "1.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(lock.Paths(), ","); got != strings.Join([]string{pkgA, pkgB, pkgC}, ",") {
		t.Errorf("lock = %s", got)
	}
}

func TestResolveRejectsACircle(t *testing.T) {
	src := &fakeSource{source: map[string]map[string]string{
		pkgA: {"1.0.0": manifest(pkgA, map[string]string{pkgB: "1.0.0"})},
		pkgB: {"1.0.0": manifest(pkgB, map[string]string{pkgA: "1.0.0"})},
	}}
	in := &Installer{Src: src, Proj: newProject(t)}
	_, err := in.Resolve(map[string]string{pkgA: "1.0.0"})
	var e *Error
	if !errors.As(err, &e) || e.Code != Circular {
		t.Fatalf("err = %v, want a Circular error", err)
	}
	if !strings.Contains(e.Error(), pkgA+" -> "+pkgB+" -> "+pkgA) {
		t.Errorf("the chain is missing from %q", e.Error())
	}
}

func TestResolveChecksTheNameInTheManifest(t *testing.T) {
	src := &fakeSource{source: map[string]map[string]string{
		pkgA: {"1.0.0": manifest("github.com/someone/else", nil)},
	}}
	in := &Installer{Src: src, Proj: newProject(t)}
	_, err := in.Resolve(map[string]string{pkgA: "1.0.0"})
	var e *Error
	if !errors.As(err, &e) || e.Code != NameMismatch {
		t.Fatalf("err = %v, want NameMismatch", err)
	}
}

func TestResolveReportsAMissingVersion(t *testing.T) {
	src := &fakeSource{source: map[string]map[string]string{pkgA: {"1.0.0": manifest(pkgA, nil)}}}
	in := &Installer{Src: src, Proj: newProject(t)}
	_, err := in.Resolve(map[string]string{pkgA: "1.1.0"})
	var e *Error
	if !errors.As(err, &e) || e.Code != VersionMissing {
		t.Fatalf("err = %v, want VersionMissing", err)
	}
}

func TestReplacedPackagesAreFollowedButNotLocked(t *testing.T) {
	proj := newProject(t)
	local := t.TempDir()
	if err := os.WriteFile(filepath.Join(local, FileName), []byte(manifest(pkgA, map[string]string{pkgB: "1.0.0"})), 0o644); err != nil {
		t.Fatal(err)
	}
	proj.Replace = map[string]string{pkgA: local}
	src := &fakeSource{source: map[string]map[string]string{pkgB: {"1.0.0": manifest(pkgB, nil)}}}
	in := &Installer{Src: src, Proj: proj}
	lock, err := in.Resolve(map[string]string{pkgA: "0.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := lock[pkgA]; ok {
		t.Error("a replaced package must not be locked")
	}
	if _, ok := lock[pkgB]; !ok {
		t.Error("what the replaced package needs must be locked")
	}
}

func TestAddWritesBothFilesAndInstallReusesTheCache(t *testing.T) {
	proj := newProject(t)
	src := &fakeSource{source: map[string]map[string]string{
		pkgA: {"1.0.0": manifest(pkgA, nil), "1.1.0": manifest(pkgA, nil)},
	}}
	in := &Installer{Src: src, Proj: proj}
	v, err := in.Add(pkgA, nil)
	if err != nil {
		t.Fatal(err)
	}
	if v.String() != "1.1.0" {
		t.Errorf("added %s, want the latest 1.1.0", v)
	}
	reread, err := LoadProject(proj.Dir)
	if err != nil || reread.Dependencies[pkgA] != "1.1.0" {
		t.Fatalf("hana.json = %+v, %v", reread, err)
	}
	lock, err := LoadLock(proj.Dir)
	if err != nil || lock[pkgA].Version != "1.1.0" {
		t.Fatalf("hana-lock.json = %+v, %v", lock, err)
	}

	before := len(src.downloads)
	if _, err := in.Install(); err != nil {
		t.Fatal(err)
	}
	if len(src.downloads) != before {
		t.Errorf("install downloaded again: %v", src.downloads[before:])
	}
}

func TestInstallRefusesATagThatMoved(t *testing.T) {
	proj := newProject(t)
	src := &fakeSource{source: map[string]map[string]string{pkgA: {"1.0.0": manifest(pkgA, nil)}}}
	in := &Installer{Src: src, Proj: proj}
	if _, err := in.Add(pkgA, nil); err != nil {
		t.Fatal(err)
	}
	// Someone else's machine: an empty cache and a lock file that names another commit.
	lock, _ := LoadLock(proj.Dir)
	e := lock[pkgA]
	e.Commit = "deadbeef"
	lock[pkgA] = e
	if err := SaveLock(proj.Dir, lock); err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvHome, t.TempDir())
	_, err := (&Installer{Src: src, Proj: proj}).Install()
	var pe *Error
	if !errors.As(err, &pe) || pe.Code != CommitMismatch {
		t.Fatalf("err = %v, want CommitMismatch", err)
	}
}

func TestInstallResolvesAgainWhenTheLockDoesNotCoverHanaJSON(t *testing.T) {
	proj := newProject(t)
	proj.Dependencies = map[string]string{pkgA: "1.0.0"}
	src := &fakeSource{source: map[string]map[string]string{pkgA: {"1.0.0": manifest(pkgA, nil)}}}
	lock, err := (&Installer{Src: src, Proj: proj}).Install()
	if err != nil {
		t.Fatal(err)
	}
	if lock[pkgA].Version != "1.0.0" {
		t.Fatalf("lock = %+v", lock)
	}
	if saved, _ := LoadLock(proj.Dir); saved[pkgA].Version != "1.0.0" {
		t.Errorf("the lock file was not written: %+v", saved)
	}
}

func TestIsPackagePath(t *testing.T) {
	for path, want := range map[string]bool{
		"github.com/owner/repo":      true,
		"gitlab.com/group/sub/repo":  true,
		"example.co.kr/o/my-pkg.v2":  true,
		"timezone":                   false,
		"http_server":                false,
		"github.com/owner":           false,
		"github.com/owner/repo.git":  false,
		"github.com/../repo":         false,
		"github.com/owner/../..":     false,
		"":                           false,
		"Github.com/owner/repo":      false,
		"github.com/owner/repo name": false,
		`github.com\owner\repo`:      false,
	} {
		if got := IsPackagePath(path); got != want {
			t.Errorf("IsPackagePath(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestVersions(t *testing.T) {
	v, err := ParseVersion("v1.10.2")
	if err != nil || v.String() != "1.10.2" {
		t.Fatalf("%v %v", v, err)
	}
	lower, _ := ParseVersion("1.9.9")
	if v.Compare(lower) <= 0 {
		t.Error("1.10.2 should be higher than 1.9.9")
	}
	for _, bad := range []string{"1.2", "1.2.3-rc1", "01.2.3", "latest", ""} {
		if _, err := ParseVersion(bad); err == nil {
			t.Errorf("%q parsed", bad)
		}
	}
}

func TestParseTagsPrefersThePeeledCommitOfAnAnnotatedTag(t *testing.T) {
	out := "aaa\trefs/tags/v1.0.0\nbbb\trefs/tags/v1.0.0^{}\nccc\trefs/tags/1.1.0\nddd\trefs/tags/nightly\n"
	tags := parseTags(out)
	if len(tags) != 2 || tags[0].Name != "1.1.0" || tags[1].Commit != "bbb" {
		t.Fatalf("tags = %+v", tags)
	}
}

// The real git: repositories on disk stand in for https://example.test/….
func TestGitDownloadsATagFromARepository(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	base := t.TempDir()
	repo := filepath.Join(base, "owner", "repo")
	run := func(dir string, args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false", "-c", "tag.gpgsign=false"}, args...)...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	run(repo, "init", "-q")
	write := func(name, content string) {
		if err := os.WriteFile(filepath.Join(repo, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(FileName, manifest("example.test/owner/repo", nil))
	write("hello.txt", "one")
	run(repo, "add", ".")
	run(repo, "commit", "-q", "-m", "one")
	run(repo, "tag", "v1.0.0")
	firstCommit := run(repo, "rev-parse", "HEAD")
	write("hello.txt", "two")
	run(repo, "commit", "-q", "-am", "two")
	run(repo, "tag", "-a", "v1.1.0", "-m", "release")
	run(repo, "tag", "nightly")

	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "url.file:///"+filepath.ToSlash(base)+"/.insteadOf")
	t.Setenv("GIT_CONFIG_VALUE_0", "https://example.test/")

	const path = "example.test/owner/repo"
	proj := newProject(t)
	in := &Installer{Src: Git{}, Proj: proj}
	v, err := in.Add(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if v.String() != "1.1.0" {
		t.Fatalf("latest = %s, want 1.1.0 (the tag nightly is not a version)", v)
	}
	dir, _ := CacheDir(path, "1.1.0")
	if data, err := os.ReadFile(filepath.Join(dir, "hello.txt")); err != nil || string(data) != "two" {
		t.Fatalf("hello.txt = %q, %v", data, err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
		t.Error("the download kept its .git folder")
	}
	lock, _ := LoadLock(proj.Dir)
	if lock[path].Commit != run(repo, "rev-parse", "v1.1.0^{commit}") {
		t.Errorf("lock commit = %s", lock[path].Commit)
	}

	// An older version, and a package folder that Dir now finds.
	if _, err := in.Add(path, &Version{1, 0, 0}); err != nil {
		t.Fatal(err)
	}
	lock, _ = LoadLock(proj.Dir)
	if lock[path].Version != "1.0.0" || lock[path].Commit != firstCommit {
		t.Errorf("lock = %+v, want 1.0.0 at %s", lock[path], firstCommit)
	}
	t.Chdir(proj.Dir)
	got := Dir(path)
	want, _ := CacheDir(path, "1.0.0")
	if got != want {
		t.Errorf("Dir = %s, want %s", got, want)
	}

	if _, err := in.Add(path, &Version{9, 9, 9}); err == nil {
		t.Error("a version that has no tag was accepted")
	}
}
