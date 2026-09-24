// Package pack turns a compiled program into one self-contained executable.
//
// A packed program is a copy of the runtime executable (cmd/hana-runtime) with a
// payload appended after it and a fixed-size trailer at the very end:
//
//	[runtime executable][payload: a zip archive][trailer]
//
// The payload holds the program as bytecode (program.hn) and libraries.json, the
// names of the packages whose native library the program loads. The libraries
// themselves are files named libraries/<package><extension>: written beside the
// executable by Install, or, when embedded, carried in the payload too. Package
// code written in Hari is already compiled into the program, so nothing else is
// needed. The trailer is
//
//	magic (8 bytes) | payload size (uint64, little endian) | payload id (8 bytes)
//
// where the id is the start of the payload's SHA-256; it names the folder embedded
// libraries are unpacked to, so a program that is run again finds them in place.
package pack

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/soumt-r/hana/bytecode"
	"github.com/soumt-r/hana/native"
	"github.com/soumt-r/hana/pkg"
)

const (
	trailerMagic  = "HJPACK\x00\x01"
	trailerSize   = len(trailerMagic) + 8 + 8
	programEntry  = "program.hn"
	librariesDir  = "libraries"
	librariesList = "libraries.json"
)

// LibraryFile is the name a package's library has inside the libraries folder:
// the package name (with / turned into _ for a git path like
// github.com/owner/repo) and the library extension of the operating system.
func LibraryFile(module, goos string) string {
	return strings.ReplaceAll(module, "/", "_") + pkg.LibraryExt(goos)
}

// Library is the native library of one package the program uses. File is its
// name inside the libraries folder: <package><extension of the target OS>.
type Library struct {
	Module string
	File   string
	Data   []byte
}

// Content is what goes into a packed program. Libraries listed here are
// embedded; every package the program loads a library of is named in Modules.
type Content struct {
	Program   []byte // a .hn file
	Modules   []string
	Libraries []Library
}

// Install writes libraries to <dir>/libraries/, where a packed program looks for
// them beside its executable. It returns the files it wrote.
func Install(dir string, libs []Library) ([]string, error) {
	var written []string
	folder := filepath.Join(dir, librariesDir)
	for _, lib := range libs {
		if err := os.MkdirAll(folder, 0o755); err != nil {
			return written, err
		}
		dest := filepath.Join(folder, lib.File)
		if err := os.WriteFile(dest, lib.Data, 0o755); err != nil {
			return written, err
		}
		written = append(written, dest)
	}
	return written, nil
}

// Write makes outPath: a copy of the runtime executable with c appended.
func Write(runtimePath, outPath string, c Content) error {
	if same, _ := sameFile(runtimePath, outPath); same {
		return errors.New("the output file is the runtime executable itself")
	}
	base, err := os.Open(runtimePath)
	if err != nil {
		return err
	}
	defer base.Close()
	out, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	if err := writeAll(out, base, c); err != nil {
		out.Close()
		os.Remove(outPath)
		return err
	}
	return out.Close()
}

func writeAll(out io.Writer, base io.Reader, c Content) error {
	if _, err := io.Copy(out, base); err != nil {
		return err
	}
	hash := sha256.New()
	counter := &countingWriter{w: io.MultiWriter(out, hash)}
	zw := zip.NewWriter(counter)
	add := func(name string, data []byte) error {
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	}
	if err := add(programEntry, c.Program); err != nil {
		return err
	}
	names, err := json.Marshal(c.Modules)
	if err != nil {
		return err
	}
	if err := add(librariesList, names); err != nil {
		return err
	}
	for _, lib := range c.Libraries {
		if err := add(path.Join(librariesDir, lib.File), lib.Data); err != nil {
			return err
		}
	}
	if err := zw.Close(); err != nil {
		return err
	}
	trailer := make([]byte, 0, trailerSize)
	trailer = append(trailer, trailerMagic...)
	trailer = binary.LittleEndian.AppendUint64(trailer, uint64(counter.n))
	trailer = append(trailer, hash.Sum(nil)[:8]...)
	_, err = out.Write(trailer)
	return err
}

type countingWriter struct {
	w io.Writer
	n int64
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.n += int64(n)
	return n, err
}

func sameFile(a, b string) (bool, error) {
	ia, err := os.Stat(a)
	if err != nil {
		return false, err
	}
	ib, err := os.Stat(b)
	if err != nil {
		return false, err
	}
	return os.SameFile(ia, ib), nil
}

// App is a packed program opened for running.
type App struct {
	file *os.File
	zip  *zip.Reader
	id   string
}

// ErrNotPacked is returned by Open for a file that carries no program.
var ErrNotPacked = errors.New("this executable does not contain a program")

// OpenSelf opens the running executable's own payload.
func OpenSelf() (*App, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	return Open(exe)
}

// Open reads the payload appended to the executable at exePath.
func Open(exePath string) (*App, error) {
	f, err := os.Open(exePath)
	if err != nil {
		return nil, err
	}
	app, err := openFile(f)
	if err != nil {
		f.Close()
		return nil, err
	}
	return app, nil
}

func openFile(f *os.File) (*App, error) {
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if info.Size() < int64(trailerSize) {
		return nil, ErrNotPacked
	}
	trailer := make([]byte, trailerSize)
	if _, err := f.ReadAt(trailer, info.Size()-int64(trailerSize)); err != nil {
		return nil, err
	}
	if string(trailer[:len(trailerMagic)]) != trailerMagic {
		return nil, ErrNotPacked
	}
	size := int64(binary.LittleEndian.Uint64(trailer[len(trailerMagic):]))
	start := info.Size() - int64(trailerSize) - size
	if size <= 0 || start < 0 {
		return nil, errors.New("the program stored in this executable is damaged")
	}
	zr, err := zip.NewReader(io.NewSectionReader(f, start, size), size)
	if err != nil {
		return nil, fmt.Errorf("the program stored in this executable is damaged: %v", err)
	}
	return &App{file: f, zip: zr, id: hex.EncodeToString(trailer[len(trailer)-8:])}, nil
}

func (a *App) Close() error { return a.file.Close() }

func (a *App) entry(name string) (*zip.File, bool) {
	for _, f := range a.zip.File {
		if f.Name == name {
			return f, true
		}
	}
	return nil, false
}

// Program decodes the packed program.
func (a *App) Program() (*bytecode.Program, bytecode.Lang, error) {
	f, ok := a.entry(programEntry)
	if !ok {
		return nil, 0, errors.New("the program stored in this executable is damaged")
	}
	r, err := f.Open()
	if err != nil {
		return nil, 0, err
	}
	defer r.Close()
	return bytecode.ReadProgram(r)
}

// Activate tells the native loader where the program's libraries are: in the
// libraries folder beside the executable or, if they are embedded, in a
// per-program folder of the user's cache they are unpacked to (once: a file that
// is already there is kept, which also matters on Windows, where a loaded library
// cannot be replaced).
func (a *App) Activate() error {
	list, ok := a.entry(librariesList)
	if !ok {
		return nil
	}
	r, err := list.Open()
	if err != nil {
		return err
	}
	defer r.Close()
	var modules []string
	if err := json.NewDecoder(r).Decode(&modules); err != nil {
		return errors.New("the program stored in this executable is damaged")
	}
	if len(modules) == 0 {
		return nil
	}

	var root string
	embedded := false
	for _, f := range a.zip.File {
		if strings.HasPrefix(f.Name, librariesDir+"/") && !f.FileInfo().IsDir() {
			embedded = true
			if root == "" {
				base, err := os.UserCacheDir()
				if err != nil {
					base = os.TempDir()
				}
				root = filepath.Join(base, "hj-packed", a.id)
			}
			if err := extract(root, f); err != nil {
				return fmt.Errorf("could not unpack %s: %v", f.Name, err)
			}
		}
	}
	if !embedded {
		exe, err := os.Executable()
		if err != nil {
			return err
		}
		root = filepath.Dir(exe)
	}
	native.Bundled = map[string]string{}
	for _, module := range modules {
		native.Bundled[module] = filepath.Join(root, librariesDir, LibraryFile(module, runtime.GOOS))
	}
	return nil
}

func extract(root string, f *zip.File) error {
	rel := path.Clean(f.Name)
	if path.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, "../") || strings.ContainsRune(rel, '\\') {
		return errors.New("unsafe file name")
	}
	dest := filepath.Join(root, filepath.FromSlash(rel))
	if info, err := os.Stat(dest); err == nil && info.Size() == int64(f.UncompressedSize64) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	src, err := f.Open()
	if err != nil {
		return err
	}
	defer src.Close()
	var data bytes.Buffer
	if _, err := io.Copy(&data, src); err != nil {
		return err
	}
	return os.WriteFile(dest, data.Bytes(), 0o755)
}
