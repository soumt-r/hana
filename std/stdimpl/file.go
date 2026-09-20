package stdimpl

import (
	"errors"
	"io/fs"
	"os"
	"strings"

	"github.com/soumt-r/hana/errs"
	"github.com/soumt-r/hana/value"
)

// FileAccess is asked before every file operation. op names it ("read",
// "write", "list", ...); returning an error stops it. The default allows
// everything; `hana run --allow-file=false` swaps in DenyFiles, and a finer
// policy (a folder allow-list, say) can be plugged in here later without
// touching the functions below.
var FileAccess = func(op, path string) error { return nil }

// DenyFiles is the policy that turns file access off.
func DenyFiles(op, path string) error { return errs.New(errs.FileBlocked) }

// fileError turns what the operating system said into one of hana's own
// wordings (the OS text is in the OS's language, which is not the script's).
func fileError(path string, err error) error {
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return errs.New(errs.FileNotFound, path)
	case errors.Is(err, fs.ErrPermission):
		return errs.New(errs.FileAccessDenied, path)
	}
	return errs.New(errs.FileFailed, path)
}

// pathArg reads the path argument and asks the access policy about it.
func pathArg(args []interface{}, i int, op string) (string, error) {
	path, err := stringArg(args, i)
	if err != nil {
		return "", err
	}
	if err := FileAccess(op, path); err != nil {
		return "", err
	}
	return path, nil
}

// readFile is the file's text; a folder is reported as such.
func readFile(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", fileError(path, err)
	}
	if info.IsDir() {
		return "", errs.New(errs.FileIsDirectory, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fileError(path, err)
	}
	return string(data), nil
}

func fileRead(args []interface{}) (interface{}, error) {
	if err := exactly(args, 1); err != nil {
		return nil, err
	}
	path, err := pathArg(args, 0, "read")
	if err != nil {
		return nil, err
	}
	return readFile(path)
}

// fileLines is the file's lines as a list, without their line breaks (and
// without an empty last line when the file ends with one).
func fileLines(args []interface{}) (interface{}, error) {
	if err := exactly(args, 1); err != nil {
		return nil, err
	}
	path, err := pathArg(args, 0, "read")
	if err != nil {
		return nil, err
	}
	text, err := readFile(path)
	if err != nil {
		return nil, err
	}
	out := []interface{}{}
	if text == "" {
		return value.NewList(out), nil
	}
	parts := strings.Split(text, "\n")
	if parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	for _, line := range parts {
		out = append(out, strings.TrimSuffix(line, "\r"))
	}
	return value.NewList(out), nil
}

// writeText writes text to path, replacing the file or (append) adding to it.
func writeText(args []interface{}, op string, flags int) (interface{}, error) {
	if err := exactly(args, 2); err != nil {
		return nil, err
	}
	path, err := pathArg(args, 0, op)
	if err != nil {
		return nil, err
	}
	text, err := stringArg(args, 1)
	if err != nil {
		return nil, err
	}
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return nil, errs.New(errs.FileIsDirectory, path)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|flags, 0o644)
	if err != nil {
		return nil, fileError(path, err)
	}
	if _, err := f.WriteString(text); err != nil {
		f.Close()
		return nil, fileError(path, err)
	}
	if err := f.Close(); err != nil {
		return nil, fileError(path, err)
	}
	return nil, nil
}

func fileWrite(args []interface{}) (interface{}, error) { return writeText(args, "write", os.O_TRUNC) }
func fileAppend(args []interface{}) (interface{}, error) {
	return writeText(args, "write", os.O_APPEND)
}

func fileExists(args []interface{}) (interface{}, error) {
	if err := exactly(args, 1); err != nil {
		return nil, err
	}
	path, err := pathArg(args, 0, "stat")
	if err != nil {
		return nil, err
	}
	_, statErr := os.Stat(path)
	return statErr == nil, nil
}

func fileIsDir(args []interface{}) (interface{}, error) {
	if err := exactly(args, 1); err != nil {
		return nil, err
	}
	path, err := pathArg(args, 0, "stat")
	if err != nil {
		return nil, err
	}
	info, statErr := os.Stat(path)
	return statErr == nil && info.IsDir(), nil
}

// fileDelete removes a file or an empty folder.
func fileDelete(args []interface{}) (interface{}, error) {
	if err := exactly(args, 1); err != nil {
		return nil, err
	}
	path, err := pathArg(args, 0, "delete")
	if err != nil {
		return nil, err
	}
	if err := os.Remove(path); err != nil {
		return nil, fileError(path, err)
	}
	return nil, nil
}

// fileList is the names inside a folder, sorted.
func fileList(args []interface{}) (interface{}, error) {
	if err := exactly(args, 1); err != nil {
		return nil, err
	}
	path, err := pathArg(args, 0, "list")
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fileError(path, err)
	}
	out := make([]interface{}, len(entries))
	for i, e := range entries {
		out[i] = e.Name()
	}
	return value.NewList(out), nil
}

// fileMkdir makes a folder and any missing folders above it; an existing
// folder is fine.
func fileMkdir(args []interface{}) (interface{}, error) {
	if err := exactly(args, 1); err != nil {
		return nil, err
	}
	path, err := pathArg(args, 0, "write")
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return nil, fileError(path, err)
	}
	return nil, nil
}

// fileMove renames or moves a file or folder; an existing file at the
// destination is replaced.
func fileMove(args []interface{}) (interface{}, error) {
	if err := exactly(args, 2); err != nil {
		return nil, err
	}
	from, err := pathArg(args, 0, "write")
	if err != nil {
		return nil, err
	}
	to, err := pathArg(args, 1, "write")
	if err != nil {
		return nil, err
	}
	if err := os.Rename(from, to); err != nil {
		return nil, fileError(from, err)
	}
	return nil, nil
}
