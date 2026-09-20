package pkg

import "fmt"

// Code names what went wrong while installing packages. The command line turns a
// code and its arguments into a sentence in the user's language (cmd/i18n.go);
// Error() is the plain English fallback for logs and tests.
type Code string

const (
	GitMissing     Code = "GitMissing"     // -
	TagsFailed     Code = "TagsFailed"     // path, detail
	NoVersions     Code = "NoVersions"     // path
	VersionMissing Code = "VersionMissing" // path, version
	DownloadFailed Code = "DownloadFailed" // path, version, detail
	CommitMismatch Code = "CommitMismatch" // path, version, locked commit, actual commit
	NameMismatch   Code = "NameMismatch"   // path, name in the package's hana.pkg.json
	BadManifest    Code = "BadManifest"    // path, version, detail
	Circular       Code = "Circular"       // chain like a -> b -> a
	NotPath        Code = "NotPath"        // what was given

	NativeNoHash       Code = "NativeNoHash"       // path, platform
	NativeDownloadFail Code = "NativeDownloadFail" // path, url, detail
	NativeHashMismatch Code = "NativeHashMismatch" // path, platform
	ScriptChanged      Code = "ScriptChanged"      // path, version
	ScriptFailed       Code = "ScriptFailed"       // path, detail
)

// Error is a package-manager failure.
type Error struct {
	Code Code
	Args []interface{}
}

func newError(code Code, args ...interface{}) *Error { return &Error{Code: code, Args: args} }

var english = map[Code]string{
	GitMissing:     "git is not installed, or is not on the PATH",
	TagsFailed:     "could not list the versions of %s: %s",
	NoVersions:     "%s has no version tags (like v1.2.0)",
	VersionMissing: "%s has no version %s",
	DownloadFailed: "could not download %s %s: %s",
	CommitMismatch: "%s %s is locked at commit %s but its tag now points at %s",
	NameMismatch:   "the package at %s calls itself %q in its hana.pkg.json",
	BadManifest:    "the hana.pkg.json of %s %s is not valid: %s",
	Circular:       "circular dependency: %s",
	NotPath:        "%q is not a package path like github.com/owner/repo",

	NativeNoHash:       "%s declares a native library download for %s without a sha256",
	NativeDownloadFail: "could not download the native library of %s from %s: %s",
	NativeHashMismatch: "the native library of %s for %s does not match its sha256",
	ScriptChanged:      "the install script of %[1]s %[2]s is not the one that was approved; run hana add %[1]s --allow-scripts to approve the new one",
	ScriptFailed:       "the install script of %s failed: %s",
}

func (e *Error) Error() string { return fmt.Sprintf(english[e.Code], e.Args...) }
