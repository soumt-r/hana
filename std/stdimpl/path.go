package stdimpl

import (
	"strings"

	"github.com/soumt-r/hana/value"
)

// [경로] works on the text of a path only — it never looks at the disk, so the
// browser engines can do the same. Paths are written with "/" (a "\" in the
// input counts as "/"), and a letter and colon at the start ("C:") is a drive.
// Results always use "/".

// splitPath separates a path into its drive ("" or "C:") and the rest, after
// turning "\" into "/".
func splitPath(p string) (drive, rest string) {
	p = strings.ReplaceAll(p, "\\", "/")
	if len(p) >= 2 && p[1] == ':' && (p[0] >= 'a' && p[0] <= 'z' || p[0] >= 'A' && p[0] <= 'Z') {
		return p[:2], p[2:]
	}
	return "", p
}

func pathFunc1(f func(string) interface{}) Func {
	return func(args []interface{}) (interface{}, error) {
		if err := exactly(args, 1); err != nil {
			return nil, err
		}
		p, err := stringArg(args, 0)
		if err != nil {
			return nil, err
		}
		return f(p), nil
	}
}

const maxJoinParts = 100

// pathJoin glues parts with "/", skipping empty ones; a part that is an
// absolute path starts the result over.
func pathJoin(args []interface{}) (interface{}, error) {
	if err := between(args, 1, maxJoinParts); err != nil {
		return nil, err
	}
	result := ""
	for i := range args {
		part, err := stringArg(args, i)
		if err != nil {
			return nil, err
		}
		drive, rest := splitPath(part)
		part = drive + rest
		switch {
		case part == "":
		case strings.HasPrefix(rest, "/"):
			result = part
		case result == "":
			result = part
		case strings.HasSuffix(result, "/"):
			result += part
		default:
			result += "/" + part
		}
	}
	return result, nil
}

var pathDirname = pathFunc1(func(p string) interface{} {
	drive, rest := splitPath(p)
	i := strings.LastIndex(rest, "/")
	if i < 0 {
		return drive
	}
	head := strings.TrimRight(rest[:i+1], "/")
	if head == "" {
		head = "/"
	}
	return drive + head
})

func baseName(p string) string {
	_, rest := splitPath(p)
	return rest[strings.LastIndex(rest, "/")+1:]
}

// splitExt splits a file name at its last dot, ignoring dots the name starts
// with (".bashrc" has no extension).
func splitExt(name string) (stem, ext string) {
	lead := 0
	for lead < len(name) && name[lead] == '.' {
		lead++
	}
	if j := strings.LastIndex(name, "."); j >= lead {
		return name[:j], name[j:]
	}
	return name, ""
}

var pathBasename = pathFunc1(func(p string) interface{} { return baseName(p) })

var pathExt = pathFunc1(func(p string) interface{} {
	_, ext := splitExt(baseName(p))
	return ext
})

var pathStem = pathFunc1(func(p string) interface{} {
	stem, _ := splitExt(baseName(p))
	return stem
})

// pathWithExt replaces the extension of the last part ("" removes it). A path
// that ends in "/" has no file name and comes back unchanged.
func pathWithExt(args []interface{}) (interface{}, error) {
	if err := exactly(args, 2); err != nil {
		return nil, err
	}
	p, err := stringArg(args, 0)
	if err != nil {
		return nil, err
	}
	ext, err := stringArg(args, 1)
	if err != nil {
		return nil, err
	}
	drive, rest := splitPath(p)
	dir := rest[:strings.LastIndex(rest, "/")+1]
	base := rest[len(dir):]
	if base == "" {
		return drive + rest, nil
	}
	stem, _ := splitExt(base)
	if ext != "" && !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return drive + dir + stem + ext, nil
}

// pathNormalize folds "//", "." and "x/.." without touching the disk. Leading
// ".." stay in a relative path and vanish at the root of an absolute one.
var pathNormalize = pathFunc1(func(p string) interface{} {
	drive, rest := splitPath(p)
	absolute := strings.HasPrefix(rest, "/")
	var stack []string
	for _, seg := range strings.Split(rest, "/") {
		switch {
		case seg == "" || seg == ".":
		case seg == "..":
			if len(stack) > 0 && stack[len(stack)-1] != ".." {
				stack = stack[:len(stack)-1]
			} else if !absolute {
				stack = append(stack, "..")
			}
		default:
			stack = append(stack, seg)
		}
	}
	result := strings.Join(stack, "/")
	if absolute {
		result = "/" + result
	}
	if result == "" {
		if drive != "" {
			return drive
		}
		return "."
	}
	return drive + result
})

var pathIsAbs = pathFunc1(func(p string) interface{} {
	_, rest := splitPath(p)
	return strings.HasPrefix(rest, "/")
})

// pathParts lists the parts of a path: the root first ("/" or "C:/") when it is
// absolute, then every name; empty parts and "." are dropped, ".." stay.
var pathParts = pathFunc1(func(p string) interface{} {
	drive, rest := splitPath(p)
	parts := []interface{}{}
	if strings.HasPrefix(rest, "/") {
		parts = append(parts, drive+"/")
	} else if drive != "" {
		parts = append(parts, drive)
	}
	for _, seg := range strings.Split(rest, "/") {
		if seg != "" && seg != "." {
			parts = append(parts, seg)
		}
	}
	return value.NewList(parts)
})
