package tests

// [경로] is pure text work, so its edge cases are checked on the shared
// implementation directly; the browser engines are compared against the same
// table in compare_tests.ts.

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/soumt-r/hana/std"
	"github.com/soumt-r/hana/std/stdimpl"
	"github.com/soumt-r/hana/value"
)

func TestPathFunctionsEdgeCases(t *testing.T) {
	list := func(items ...string) []interface{} {
		out := []interface{}{}
		for _, s := range items {
			out = append(out, s)
		}
		return out
	}
	str := func(args ...string) []interface{} { return list(args...) }
	vl := func(items ...string) *value.List { return value.NewList(list(items...)) }
	cases := []struct {
		fn   string
		args []interface{}
		want interface{}
	}{
		{std.PathJoin, str("a", "b"), "a/b"},
		{std.PathJoin, str("a/", "b"), "a/b"},
		{std.PathJoin, str("a", "", "b"), "a/b"},
		{std.PathJoin, str("a", "/b", "c"), "/b/c"},
		{std.PathJoin, str("a\\b", "c"), "a/b/c"},
		{std.PathJoin, str("C:\\data", "x.txt"), "C:/data/x.txt"},
		{std.PathJoin, str(""), ""},

		{std.PathDirname, str("a/b/c.txt"), "a/b"},
		{std.PathDirname, str("c.txt"), ""},
		{std.PathDirname, str("/c"), "/"},
		{std.PathDirname, str("a/b/"), "a/b"},
		{std.PathDirname, str("/"), "/"},
		{std.PathDirname, str("C:\\a\\b"), "C:/a"},
		{std.PathDirname, str("C:x"), "C:"},

		{std.PathBasename, str("a/b/c.txt"), "c.txt"},
		{std.PathBasename, str("a/b/"), ""},
		{std.PathBasename, str("C:\\a\\b.txt"), "b.txt"},
		{std.PathBasename, str("한글/파일.txt"), "파일.txt"},

		{std.PathExt, str("a/b.tar.gz"), ".gz"},
		{std.PathExt, str("a/.bashrc"), ""},
		{std.PathExt, str("a/b."), "."},
		{std.PathExt, str("a/..x"), ""},
		{std.PathExt, str("noext"), ""},
		{std.PathExt, str("dir.d/file"), ""},

		{std.PathStem, str("a/b.tar.gz"), "b.tar"},
		{std.PathStem, str("a/.bashrc"), ".bashrc"},
		{std.PathStem, str("a/b."), "b"},

		{std.PathWithExt, str("a/b.txt", ".md"), "a/b.md"},
		{std.PathWithExt, str("a/b.txt", "md"), "a/b.md"},
		{std.PathWithExt, str("a/b.txt", ""), "a/b"},
		{std.PathWithExt, str("a/b", ".md"), "a/b.md"},
		{std.PathWithExt, str("a/.bashrc", ".x"), "a/.bashrc.x"},
		{std.PathWithExt, str("a/b/", ".md"), "a/b/"},
		{std.PathWithExt, str("C:\\a\\b.txt", ".md"), "C:/a/b.md"},

		{std.PathNormalize, str("a//b/./c/../d"), "a/b/d"},
		{std.PathNormalize, str("../a/../.."), "../.."},
		{std.PathNormalize, str("/../a"), "/a"},
		{std.PathNormalize, str(""), "."},
		{std.PathNormalize, str("./"), "."},
		{std.PathNormalize, str("/"), "/"},
		{std.PathNormalize, str("a/.."), "."},
		{std.PathNormalize, str("C:\\a\\..\\b"), "C:/b"},
		{std.PathNormalize, str("C:"), "C:"},

		{std.PathIsAbs, str("/a"), true},
		{std.PathIsAbs, str("a/b"), false},
		{std.PathIsAbs, str("C:\\a"), true},
		{std.PathIsAbs, str("C:a"), false},
		{std.PathIsAbs, str(""), false},

		{std.PathParts, str("/a/b"), vl("/", "a", "b")},
		{std.PathParts, str("a/./b//c/.."), vl("a", "b", "c", "..")},
		{std.PathParts, str("C:\\a\\b"), vl("C:/", "a", "b")},
		{std.PathParts, str("C:a"), vl("C:", "a")},
		{std.PathParts, str(""), vl()},
	}
	for _, c := range cases {
		got, err := stdimpl.Impls[c.fn](c.args)
		if err != nil || !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s%q = %#v (%v), want %#v", c.fn, c.args, got, err, c.want)
		}
	}
}

func TestPathFunctionsCheckTheirArguments(t *testing.T) {
	bad := []struct {
		fn   string
		args []interface{}
	}{
		{std.PathJoin, nil},
		{std.PathJoin, []interface{}{"a", 1.0}},
		{std.PathDirname, nil},
		{std.PathBasename, []interface{}{1.0}},
		{std.PathWithExt, []interface{}{"a"}},
		{std.PathNormalize, []interface{}{"a", "b"}},
	}
	for _, c := range bad {
		if _, err := stdimpl.Impls[c.fn](c.args); err == nil {
			t.Errorf("%s(%s) should be refused", c.fn, fmt.Sprint(c.args...))
		}
	}
}
