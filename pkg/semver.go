package pkg

import (
	"fmt"
	"regexp"
	"strconv"
)

// Version is a release number like 1.2.3 (an optional leading "v", as git tags
// have it, is accepted and dropped). Pre-release and build suffixes are not
// supported: a tag either is a plain release number or it is not a version.
type Version struct{ Major, Minor, Patch int }

var versionRe = regexp.MustCompile(`^v?(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

// ParseVersion reads "1.2.3" or "v1.2.3".
func ParseVersion(s string) (Version, error) {
	m := versionRe.FindStringSubmatch(s)
	if m == nil {
		return Version{}, fmt.Errorf("%q is not a version like 1.2.3", s)
	}
	n := func(i int) int { v, _ := strconv.Atoi(m[i]); return v }
	return Version{n(1), n(2), n(3)}, nil
}

func (v Version) String() string { return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch) }

// Compare is -1, 0 or 1 as v is lower than, equal to or higher than w.
func (v Version) Compare(w Version) int {
	switch {
	case v.Major != w.Major:
		return sign(v.Major - w.Major)
	case v.Minor != w.Minor:
		return sign(v.Minor - w.Minor)
	}
	return sign(v.Patch - w.Patch)
}

func sign(n int) int {
	switch {
	case n < 0:
		return -1
	case n > 0:
		return 1
	}
	return 0
}
