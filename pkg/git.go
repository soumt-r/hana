package pkg

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// Tag is a release of a package: a git tag that reads as a version.
type Tag struct {
	Version Version
	// Name is the tag as git has it ("v1.2.0" or "1.2.0").
	Name string
	// Commit is what the tag points at.
	Commit string
}

// Source is where packages come from. Git is the real one.
type Source interface {
	// Tags lists the versions a package has, highest first.
	Tags(path string) ([]Tag, error)
	// Download puts the files of a tag into dest (without any git data) and
	// returns its commit. When wantCommit is not empty and the tag points
	// elsewhere, nothing is written and the error is CommitMismatch.
	Download(path string, tag Tag, dest, wantCommit string) (commit string, err error)
}

// Git fetches packages with the git command: the package github.com/owner/repo
// is the repository https://github.com/owner/repo.
type Git struct{}

// EnvGitProtocol set to "ssh" makes hana clone over SSH (git@host:owner/repo) instead
// of https. What git itself is configured with (credential helper, ssh keys,
// url.<base>.insteadOf) works too: hana runs the git command.
const EnvGitProtocol = "HANA_GIT_PROTOCOL"

// RepoURL is the address hana clones a package path from.
func RepoURL(path string) string {
	if os.Getenv(EnvGitProtocol) == "ssh" {
		if host, rest, ok := strings.Cut(path, "/"); ok {
			return "git@" + host + ":" + rest
		}
	}
	return "https://" + path
}

// needsLogin reports whether git's complaint is about credentials (or a repository
// that a stranger cannot see, which hosts report as "not found").
func needsLogin(detail string) bool {
	for _, marker := range []string{
		"Authentication failed", "could not read Username", "could not read Password",
		"terminal prompts disabled", "Permission denied (publickey)", "Repository not found",
		"Host key verification failed", "access denied",
	} {
		if strings.Contains(detail, marker) {
			return true
		}
	}
	return false
}

// interactive reports whether git may ask the user for a login: standard input is a
// terminal and the user did not decide otherwise with GIT_TERMINAL_PROMPT.
func interactive() bool {
	if os.Getenv("GIT_TERMINAL_PROMPT") != "" {
		return false
	}
	fi, err := os.Stdin.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

func git(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if interactive() {
		cmd.Stdin = os.Stdin
	} else if os.Getenv("GIT_TERMINAL_PROMPT") == "" {
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	}
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	if err := cmd.Run(); err != nil {
		if _, lookErr := exec.LookPath("git"); lookErr != nil {
			return "", newError(GitMissing)
		}
		detail := strings.TrimSpace(errOut.String())
		if detail == "" {
			detail = err.Error()
		}
		return "", fmt.Errorf("%s", detail)
	}
	return out.String(), nil
}

// Tags asks the repository for its tags.
func (Git) Tags(path string) ([]Tag, error) {
	out, err := git("", "ls-remote", "--tags", RepoURL(path))
	if err != nil {
		if e, ok := err.(*Error); ok {
			return nil, e
		}
		if needsLogin(err.Error()) {
			return nil, newError(AuthFailed, path, firstLine(err.Error()))
		}
		return nil, newError(TagsFailed, path, err.Error())
	}
	return parseTags(out), nil
}

// parseTags reads `git ls-remote --tags` output. An annotated tag is listed twice;
// the peeled line (refs/tags/x^{}) has the commit.
func parseTags(out string) []Tag {
	commits := map[string]string{}
	peeled := map[string]bool{}
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 || !strings.HasPrefix(fields[1], "refs/tags/") {
			continue
		}
		name := strings.TrimPrefix(fields[1], "refs/tags/")
		if strings.HasSuffix(name, "^{}") {
			name = strings.TrimSuffix(name, "^{}")
			commits[name] = fields[0]
			peeled[name] = true
		} else if !peeled[name] {
			commits[name] = fields[0]
		}
	}
	var tags []Tag
	for name, commit := range commits {
		if v, err := ParseVersion(name); err == nil {
			tags = append(tags, Tag{Version: v, Name: name, Commit: commit})
		}
	}
	sort.Slice(tags, func(i, j int) bool { return tags[i].Version.Compare(tags[j].Version) > 0 })
	return tags
}

// Download clones one tag, checks the commit and moves the files (without .git)
// to dest.
func (Git) Download(path string, tag Tag, dest, wantCommit string) (string, error) {
	parent := filepath.Dir(dest)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return "", err
	}
	tmp, err := os.MkdirTemp(parent, ".download-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)
	work := filepath.Join(tmp, "src")
	if _, err := git("", "clone", "--quiet", "--depth", "1", "--branch", tag.Name, RepoURL(path), work); err != nil {
		if e, ok := err.(*Error); ok {
			return "", e
		}
		if needsLogin(err.Error()) {
			return "", newError(AuthFailed, path, firstLine(err.Error()))
		}
		return "", newError(DownloadFailed, path, tag.Version.String(), err.Error())
	}
	out, err := git(work, "rev-parse", "HEAD")
	if err != nil {
		return "", newError(DownloadFailed, path, tag.Version.String(), err.Error())
	}
	commit := strings.TrimSpace(out)
	if wantCommit != "" && commit != wantCommit {
		return "", newError(CommitMismatch, path, tag.Version.String(), shortCommit(wantCommit), shortCommit(commit))
	}
	if err := os.RemoveAll(filepath.Join(work, ".git")); err != nil {
		return "", err
	}
	if err := os.RemoveAll(dest); err != nil {
		return "", err
	}
	if err := os.Rename(work, dest); err != nil {
		return "", err
	}
	return commit, nil
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(s), "\n")
	return line
}

func shortCommit(c string) string {
	if len(c) > 7 {
		return c[:7]
	}
	return c
}
