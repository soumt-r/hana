package pkg

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
)

// The two files of a project that uses installed packages. hana.json is what the
// project asks for (the lowest version of each package it accepts); hana-lock.json
// is what was chosen (the exact version and commit of every package, including
// the ones those packages need). Both belong in git.
const (
	ProjectFile = "hana.json"
	LockFile    = "hana-lock.json"
)

// EnvHome names the folder hana keeps downloaded packages in. Without it the
// folder is .hana inside the user's home.
const EnvHome = "HANA_HOME"

// Project is a parsed hana.json.
type Project struct {
	// Dir is the folder that holds hana.json (not part of the file).
	Dir string `json:"-"`
	// Dependencies maps a package path (github.com/owner/repo) to the lowest
	// version this project accepts.
	Dependencies map[string]string `json:"dependencies,omitempty"`
	// Replace points a package path at a local folder for now: the folder is used
	// as it is, and nothing is downloaded or locked for it. It applies to this
	// project only, never to the packages it depends on.
	Replace map[string]string `json:"replace,omitempty"`
	// TrustedScripts lists the packages whose install script may run.
	TrustedScripts []string `json:"trustedScripts,omitempty"`
}

// LockEntry is one package as chosen: its exact version, and the git commit that
// version's tag pointed at when it was chosen.
type LockEntry struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	// Scripts is the SHA-256 of the install script that was approved, if the
	// package has one.
	Scripts string `json:"scripts,omitempty"`
}

// Lock maps a package path to its LockEntry.
type Lock map[string]LockEntry

var packagePathRe = regexp.MustCompile(`^[a-z0-9-]+(\.[a-z0-9-]+)+(/[A-Za-z0-9][A-Za-z0-9._-]*){2,}$`)

// IsPackagePath reports whether a name is a package's git path like
// github.com/owner/repo: a host with a dot, then at least an owner and a repo.
// Names of the packages shipped with hana (timezone) are not paths.
func IsPackagePath(name string) bool {
	return packagePathRe.MatchString(name) && !hasDotGit(name)
}

func hasDotGit(name string) bool { return len(name) > 4 && name[len(name)-4:] == ".git" }

// FindProject looks for hana.json in dir and the folders above it. It returns
// (nil, nil) when there is none.
func FindProject(dir string) (*Project, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	for {
		p, err := LoadProject(dir)
		if err != nil || p != nil {
			return p, err
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil, nil
		}
		dir = parent
	}
}

// LoadProject reads dir/hana.json, or returns (nil, nil) when there is none.
func LoadProject(dir string) (*Project, error) {
	data, err := os.ReadFile(filepath.Join(dir, ProjectFile))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var p Project
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&p); err != nil {
		return nil, fmt.Errorf("%s: invalid JSON: %v", ProjectFile, err)
	}
	p.Dir = dir
	if err := p.validate(); err != nil {
		return nil, fmt.Errorf("%s: %v", ProjectFile, err)
	}
	return &p, nil
}

func (p *Project) validate() error {
	for path, v := range p.Dependencies {
		if !IsPackagePath(path) {
			return fmt.Errorf("dependencies: %q is not a package path like github.com/owner/repo", path)
		}
		if _, err := ParseVersion(v); err != nil {
			return fmt.Errorf("dependencies.%s: %v", path, err)
		}
	}
	for path, dir := range p.Replace {
		if !IsPackagePath(path) {
			return fmt.Errorf("replace: %q is not a package path like github.com/owner/repo", path)
		}
		if dir == "" {
			return fmt.Errorf("replace.%s is empty", path)
		}
	}
	for _, path := range p.TrustedScripts {
		if !IsPackagePath(path) {
			return fmt.Errorf("trustedScripts: %q is not a package path like github.com/owner/repo", path)
		}
	}
	return nil
}

// Save writes hana.json into p.Dir.
func (p *Project) Save() error {
	return writeJSON(filepath.Join(p.Dir, ProjectFile), p)
}

// ReplaceDir is the local folder a package path is pointed at, made absolute
// against the project's folder.
func (p *Project) ReplaceDir(path string) (string, bool) {
	dir, ok := p.Replace[path]
	if !ok {
		return "", false
	}
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(p.Dir, dir)
	}
	return dir, true
}

// LoadLock reads dir/hana-lock.json. A missing file is an empty lock.
func LoadLock(dir string) (Lock, error) {
	data, err := os.ReadFile(filepath.Join(dir, LockFile))
	if errors.Is(err, fs.ErrNotExist) {
		return Lock{}, nil
	}
	if err != nil {
		return nil, err
	}
	lock := Lock{}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&lock); err != nil {
		return nil, fmt.Errorf("%s: invalid JSON: %v", LockFile, err)
	}
	for path, e := range lock {
		if !IsPackagePath(path) {
			return nil, fmt.Errorf("%s: %q is not a package path", LockFile, path)
		}
		if _, err := ParseVersion(e.Version); err != nil {
			return nil, fmt.Errorf("%s: %s: %v", LockFile, path, err)
		}
	}
	return lock, nil
}

// SaveLock writes hana-lock.json into dir.
func SaveLock(dir string, lock Lock) error {
	return writeJSON(filepath.Join(dir, LockFile), lock)
}

// Paths are the package paths of a lock, sorted.
func (l Lock) Paths() []string {
	paths := make([]string, 0, len(l))
	for p := range l {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return paths
}

func writeJSON(file string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(file, append(data, '\n'), 0o644)
}

// CacheRoot is the folder downloaded packages live in: $HANA_HOME/pkg, or
// ~/.hana/pkg.
func CacheRoot() (string, error) {
	if home := os.Getenv(EnvHome); home != "" {
		return filepath.Join(home, "pkg"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".hana", "pkg"), nil
}

// CacheDir is where one version of a package lives once downloaded:
// <cache>/<host>/<owner>/<repo>@<version>.
func CacheDir(path, version string) (string, error) {
	root, err := CacheRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, filepath.FromSlash(path)+"@"+version), nil
}

// installedDir is the folder of a package path for the project the running hana
// is in: the replacement folder, or the locked version in the cache. ok is false
// when the project does not have that package.
func installedDir(path string) (dir string, ok bool) {
	proj, err := FindProject(".")
	if err != nil || proj == nil {
		return "", false
	}
	if dir, ok := proj.ReplaceDir(path); ok {
		return dir, true
	}
	lock, err := LoadLock(proj.Dir)
	if err != nil {
		return "", false
	}
	e, ok := lock[path]
	if !ok {
		return "", false
	}
	dir, err = CacheDir(path, e.Version)
	if err != nil {
		return "", false
	}
	return dir, true
}
