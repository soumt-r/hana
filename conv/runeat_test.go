package conv

import "testing"

func TestRuneAt(t *testing.T) {
	cases := []struct {
		s    string
		idx  int
		want string
		ok   bool
	}{
		{"안녕하세요", 0, "안", true},
		{"안녕하세요", 4, "요", true},
		{"안녕하세요", 5, "", false},
		{"안녕하세요", -1, "", false},
		{"", 0, "", false},
		{"a😀b", 1, "😀", true},
		{"a😀b", 2, "b", true},
		{"a\xffb", 1, "\uFFFD", true}, // an invalid byte is one character, as in []rune(s)
		{"a\xffb", 2, "b", true},
	}
	for _, c := range cases {
		got, ok := RuneAt(c.s, c.idx)
		if got != c.want || ok != c.ok {
			t.Errorf("RuneAt(%q, %d) = %q, %v; want %q, %v", c.s, c.idx, got, ok, c.want, c.ok)
		}
		if runes := []rune(c.s); c.idx >= 0 && c.idx < len(runes) && string(runes[c.idx]) != got {
			t.Errorf("RuneAt(%q, %d) disagrees with []rune", c.s, c.idx)
		}
	}
}
