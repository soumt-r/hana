package pack

import "testing"

func TestLibraryFileFlattensAGitPath(t *testing.T) {
	for _, tc := range []struct{ module, goos, want string }{
		{"http_server", "windows", "http_server.dll"},
		{"github.com/owner/repo", "linux", "github.com_owner_repo.so"},
		{"github.com/owner/repo", "darwin", "github.com_owner_repo.dylib"},
	} {
		if got := LibraryFile(tc.module, tc.goos); got != tc.want {
			t.Errorf("LibraryFile(%q, %q) = %q, want %q", tc.module, tc.goos, got, tc.want)
		}
	}
}
