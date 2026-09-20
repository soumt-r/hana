package pkg

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// commitFile, inside a downloaded package folder, remembers the commit it came
// from, so a later install can tell it is the one the lock file names.
const commitFile = ".hana-commit"

// Installer downloads packages into the cache and decides which versions a
// project gets.
type Installer struct {
	Src  Source
	Proj *Project
	// Log, when set, is told about progress: "download" with (path, version),
	// "native" with (path, platform), "script" with (path, command line) just
	// before an install script runs, and "skipScript" with (path, command line)
	// for one that was not run because the project does not trust the package.
	Log func(event string, args ...interface{})
	// Fetch opens a web address (the prebuilt native libraries a package declares).
	// It is a field so that this package does not need net/http, which would
	// make the small hana-runtime much bigger.
	Fetch func(url string) (io.ReadCloser, error)
	// Out is where an install script's output goes (default: standard error).
	Out io.Writer
	// Approve names packages whose changed install script the user approves now
	// (hana add <path> --allow-scripts); the lock file then records the new one.
	Approve map[string]bool

	tags    map[string][]Tag
	locked  Lock
	scripts map[string]string // path -> hash of the script that was run or approved
}

func (in *Installer) log(event string, args ...interface{}) {
	if in.Log != nil {
		in.Log(event, args...)
	}
}

func (in *Installer) tagsOf(path string) ([]Tag, error) {
	if tags, ok := in.tags[path]; ok {
		return tags, nil
	}
	tags, err := in.Src.Tags(path)
	if err != nil {
		return nil, err
	}
	if in.tags == nil {
		in.tags = map[string][]Tag{}
	}
	in.tags[path] = tags
	return tags, nil
}

// Latest is the highest version a package has.
func (in *Installer) Latest(path string) (Version, error) {
	tags, err := in.tagsOf(path)
	if err != nil {
		return Version{}, err
	}
	if len(tags) == 0 {
		return Version{}, newError(NoVersions, path)
	}
	return tags[0].Version, nil
}

// ensure makes sure the folder of path@v is in the cache and returns it and the
// commit it holds. A non-empty wantCommit (from the lock file) must match.
func (in *Installer) ensure(path string, v Version, wantCommit string) (dir, commit string, err error) {
	dir, err = CacheDir(path, v.String())
	if err != nil {
		return "", "", err
	}
	if data, err := os.ReadFile(filepath.Join(dir, commitFile)); err == nil {
		commit = strings.TrimSpace(string(data))
		if wantCommit != "" && commit != wantCommit {
			return "", "", newError(CommitMismatch, path, v.String(), shortCommit(wantCommit), shortCommit(commit))
		}
		return dir, commit, in.prepare(path, v, dir)
	}
	tags, err := in.tagsOf(path)
	if err != nil {
		return "", "", err
	}
	var tag *Tag
	for i := range tags {
		if tags[i].Version.Compare(v) == 0 {
			tag = &tags[i]
			break
		}
	}
	if tag == nil {
		return "", "", newError(VersionMissing, path, v.String())
	}
	in.log("download", path, v.String())
	commit, err = in.Src.Download(path, *tag, dir, wantCommit)
	if err != nil {
		return "", "", err
	}
	if err := os.WriteFile(filepath.Join(dir, commitFile), []byte(commit+"\n"), 0o644); err != nil {
		return "", "", err
	}
	return dir, commit, in.prepare(path, v, dir)
}

// scriptFile, inside a package folder, holds the hash of the install script that
// has run there.
const scriptFile = ".hana-script"

// prepare finishes a downloaded package: it fetches the prebuilt native library
// the manifest declares for this platform, and runs the install script when the
// project trusts the package.
func (in *Installer) prepare(path string, v Version, dir string) error {
	m, err := manifestOf(path, v.String(), dir)
	if err != nil || m == nil {
		return err
	}
	if err := in.fetchNative(path, m, dir); err != nil {
		return err
	}
	cmd := m.Scripts["install"]
	if len(cmd) == 0 {
		return nil
	}
	line := strings.Join(cmd, " ")
	hash := hashScript(cmd)
	if !in.trusts(path) {
		in.log("skipScript", path, line)
		return nil
	}
	if want := in.locked[path].Scripts; want != "" && want != hash && !in.Approve[path] {
		return newError(ScriptChanged, path, v.String())
	}
	if in.scripts == nil {
		in.scripts = map[string]string{}
	}
	in.scripts[path] = hash
	if done, err := os.ReadFile(filepath.Join(dir, scriptFile)); err == nil && strings.TrimSpace(string(done)) == hash {
		return nil
	}
	in.log("script", path, line)
	c := exec.Command(cmd[0], cmd[1:]...)
	c.Dir = dir
	c.Stdout, c.Stderr = in.out(), in.out()
	if err := c.Run(); err != nil {
		return newError(ScriptFailed, path, err.Error())
	}
	return os.WriteFile(filepath.Join(dir, scriptFile), []byte(hash+"\n"), 0o644)
}

func (in *Installer) out() io.Writer {
	if in.Out != nil {
		return in.Out
	}
	return os.Stderr
}

func (in *Installer) trusts(path string) bool {
	return in.Approve[path] || in.trustedInProject(path)
}

func hashScript(cmd []string) string {
	sum := sha256.Sum256([]byte(strings.Join(cmd, "\x00")))
	return hex.EncodeToString(sum[:])
}

// fetchNative downloads the native library the manifest lists for this platform
// when it is not in the package folder already, and checks its sha256.
func (in *Installer) fetchNative(path string, m *Manifest, dir string) error {
	platform := Platform()
	n, ok := m.NativeFor(platform)
	if !ok || n.URL == "" {
		return nil
	}
	file := filepath.Join(dir, filepath.FromSlash(n.File))
	if _, err := os.Stat(file); err == nil {
		return nil
	}
	if n.SHA256 == "" {
		return newError(NativeNoHash, path, platform)
	}
	if in.Fetch == nil {
		return newError(NativeDownloadFail, path, n.URL, "no way to download")
	}
	in.log("native", path, platform)
	body, err := in.Fetch(n.URL)
	if err != nil {
		return newError(NativeDownloadFail, path, n.URL, err.Error())
	}
	defer body.Close()
	data, err := io.ReadAll(body)
	if err != nil {
		return newError(NativeDownloadFail, path, n.URL, err.Error())
	}
	sum := sha256.Sum256(data)
	if !strings.EqualFold(hex.EncodeToString(sum[:]), n.SHA256) {
		return newError(NativeHashMismatch, path, platform)
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return err
	}
	return os.WriteFile(file, data, 0o755)
}

// manifestOf reads a package folder's manifest and checks it is the package it
// should be.
func manifestOf(path, version, dir string) (*Manifest, error) {
	m, err := Load(dir)
	if err != nil {
		return nil, newError(BadManifest, path, version, err.Error())
	}
	if m != nil && m.Name != "" && m.Name != path {
		return nil, newError(NameMismatch, path, m.Name)
	}
	return m, nil
}

// Resolve picks a version for every package the roots need, directly or through
// other packages: for each package, the highest version anyone asks for (minimal
// version selection — versions are minimums, so there is never a conflict to
// solve). It downloads what it looks at, and returns the lock for it. Packages the
// project replaces with a local folder are followed but not locked.
func (in *Installer) Resolve(roots map[string]string) (Lock, error) {
	if in.locked == nil {
		in.locked, _ = LoadLock(in.Proj.Dir)
	}
	type node struct{ path, version string }
	visited := map[node]bool{}
	best := map[string]Version{}
	commits := map[node]string{}

	var visit func(path string, v Version, stack []string) error
	visit = func(path string, v Version, stack []string) error {
		for _, p := range stack {
			if p == path {
				return newError(Circular, strings.Join(append(append([]string{}, stack...), path), " -> "))
			}
		}
		replaced, isReplaced := "", false
		if in.Proj != nil {
			replaced, isReplaced = in.Proj.ReplaceDir(path)
		}
		n := node{path, v.String()}
		if isReplaced {
			n.version = ""
		}
		if visited[n] {
			return nil
		}
		visited[n] = true

		var dir, commit string
		var err error
		if isReplaced {
			dir = replaced
		} else {
			dir, commit, err = in.ensure(path, v, "")
			if err != nil {
				return err
			}
			commits[n] = commit
			if cur, ok := best[path]; !ok || v.Compare(cur) > 0 {
				best[path] = v
			}
		}
		m, err := manifestOf(path, v.String(), dir)
		if err != nil {
			return err
		}
		if m == nil {
			return nil
		}
		deps := make([]string, 0, len(m.Dependencies))
		for dep := range m.Dependencies {
			deps = append(deps, dep)
		}
		sort.Strings(deps)
		for _, dep := range deps {
			dv, err := ParseVersion(m.Dependencies[dep])
			if err != nil {
				return newError(BadManifest, path, v.String(), err.Error())
			}
			if err := visit(dep, dv, append(stack, path)); err != nil {
				return err
			}
		}
		return nil
	}

	paths := make([]string, 0, len(roots))
	for p := range roots {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		v, err := ParseVersion(roots[p])
		if err != nil {
			return nil, newError(BadManifest, p, roots[p], err.Error())
		}
		if err := visit(p, v, nil); err != nil {
			return nil, err
		}
	}

	lock := Lock{}
	for p, v := range best {
		lock[p] = LockEntry{Version: v.String(), Commit: commits[node{p, v.String()}], Scripts: in.scripts[p]}
	}
	return lock, nil
}

// Add records a package in the project's hana.json (at version, or at its latest
// version when version is nil), chooses versions for the whole project, and
// writes hana.json and hana-lock.json. It returns the version recorded.
func (in *Installer) Add(path string, version *Version) (Version, error) {
	if !IsPackagePath(path) {
		return Version{}, newError(NotPath, path)
	}
	var v Version
	if version != nil {
		v = *version
	} else {
		latest, err := in.Latest(path)
		if err != nil {
			return Version{}, err
		}
		v = latest
	}
	if in.Proj.Dependencies == nil {
		in.Proj.Dependencies = map[string]string{}
	}
	in.Proj.Dependencies[path] = v.String()
	if in.Approve[path] && !in.trustedInProject(path) {
		in.Proj.TrustedScripts = append(in.Proj.TrustedScripts, path)
		sort.Strings(in.Proj.TrustedScripts)
	}
	lock, err := in.Resolve(in.Proj.Dependencies)
	if err != nil {
		return Version{}, err
	}
	if err := in.Proj.Save(); err != nil {
		return Version{}, err
	}
	return v, SaveLock(in.Proj.Dir, lock)
}

// Install makes the cache hold everything the lock file names. When the lock file
// does not cover hana.json (a package missing, or locked below the minimum),
// it first chooses versions again and rewrites the lock file.
func (in *Installer) Install() (Lock, error) {
	lock, err := LoadLock(in.Proj.Dir)
	if err != nil {
		return nil, err
	}
	in.locked = lock
	if !in.covers(lock) {
		lock, err = in.Resolve(in.Proj.Dependencies)
		if err != nil {
			return nil, err
		}
		if err := SaveLock(in.Proj.Dir, lock); err != nil {
			return nil, err
		}
	}
	for _, path := range lock.Paths() {
		e := lock[path]
		v, err := ParseVersion(e.Version)
		if err != nil {
			return nil, err
		}
		if _, _, err := in.ensure(path, v, e.Commit); err != nil {
			return nil, err
		}
	}
	changed := false
	for path, hash := range in.scripts {
		if e, ok := lock[path]; ok && e.Scripts != hash {
			e.Scripts = hash
			lock[path] = e
			changed = true
		}
	}
	if changed {
		if err := SaveLock(in.Proj.Dir, lock); err != nil {
			return nil, err
		}
	}
	return lock, nil
}

func (in *Installer) trustedInProject(path string) bool {
	for _, p := range in.Proj.TrustedScripts {
		if p == path {
			return true
		}
	}
	return false
}

func (in *Installer) covers(lock Lock) bool {
	for path, min := range in.Proj.Dependencies {
		if _, replaced := in.Proj.Replace[path]; replaced {
			continue
		}
		e, ok := lock[path]
		if !ok {
			return false
		}
		have, err1 := ParseVersion(e.Version)
		want, err2 := ParseVersion(min)
		if err1 != nil || err2 != nil || have.Compare(want) < 0 {
			return false
		}
	}
	return true
}

// Remove takes a package out of hana.json and chooses versions again, so what only
// it needed leaves the lock file too. It reports whether the package was there.
func (in *Installer) Remove(path string) (bool, error) {
	if _, ok := in.Proj.Dependencies[path]; !ok {
		return false, nil
	}
	delete(in.Proj.Dependencies, path)
	delete(in.Proj.Replace, path)
	lock, err := in.Resolve(in.Proj.Dependencies)
	if err != nil {
		return false, err
	}
	if err := in.Proj.Save(); err != nil {
		return false, err
	}
	return true, SaveLock(in.Proj.Dir, lock)
}
