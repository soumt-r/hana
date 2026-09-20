package pkg

import (
	"os"
	"path/filepath"
	"strings"
)

// EnvPackages names an environment variable holding extra package folders
// (separated like PATH), searched after the project's own and before the ones
// installed next to hana.
const EnvPackages = "HANA_PACKAGES"

// Roots are the folders that hold packages, in the order they are searched:
//
//  1. ./packages — the project's own; a package here hides a shipped one of the
//     same name, so a project can override or pin what it uses
//  2. the folders in $HANA_PACKAGES
//  3. packages/ next to the hana executable — the packages shipped with hana
//
// Because the last one does not depend on where hana is run from, a shipped
// package (like http_server) works in any folder.
func Roots() []string {
	roots := []string{"packages"}
	for _, r := range filepath.SplitList(os.Getenv(EnvPackages)) {
		if r != "" {
			roots = append(roots, r)
		}
	}
	if exe, err := os.Executable(); err == nil {
		roots = append(roots, filepath.Join(filepath.Dir(exe), "packages"))
	}
	return roots
}

// Dir is the folder of the package called module: the first root that has such
// a folder. When none does it is the project path (./packages/<module>), which
// simply does not exist, so callers report the package as not found. A name that
// could climb out of a root (a path, "..") never matches anything.
func Dir(module string) string {
	if module == "" || module == "." || module == ".." || strings.ContainsAny(module, `/\`) || strings.ContainsRune(module, 0) {
		return filepath.Join("packages", ".no-such-package")
	}
	for _, root := range Roots() {
		dir := filepath.Join(root, module)
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir
		}
	}
	return filepath.Join("packages", module)
}
