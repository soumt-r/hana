package pkg

import "testing"

func TestRepoURLCanBeSSH(t *testing.T) {
	if got := RepoURL("github.com/o/r"); got != "https://github.com/o/r" {
		t.Errorf("default = %s", got)
	}
	t.Setenv(EnvGitProtocol, "ssh")
	if got := RepoURL("github.com/o/r"); got != "git@github.com:o/r" {
		t.Errorf("ssh = %s", got)
	}
	if got := RepoURL("gitlab.com/group/sub/r"); got != "git@gitlab.com:group/sub/r" {
		t.Errorf("ssh with a subgroup = %s", got)
	}
}

func TestLoginProblemsAreRecognised(t *testing.T) {
	for detail, want := range map[string]bool{
		"fatal: could not read Username for 'https://github.com': terminal prompts disabled": true,
		"remote: Repository not found.":                                                      true,
		"git@github.com: Permission denied (publickey).":                                     true,
		"fatal: unable to access: Could not resolve host: example.test":                      false,
		"fatal: Remote branch v9.9.9 not found in upstream origin":                           false,
	} {
		if got := needsLogin(detail); got != want {
			t.Errorf("needsLogin(%q) = %v, want %v", detail, got, want)
		}
	}
}
