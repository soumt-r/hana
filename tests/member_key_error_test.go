package tests

import (
	"strings"
	"testing"
)

// A member whose key cannot be computed (a variable that does not exist, a
// division by zero) reports that error, on strings and lists alike, for reads
// and list writes; a string's key that is not a number is
// MemberAccessOnString. The bytecode engine always did this; the
// tree-walker used to replace the key's error with its own.
func TestMemberKeyErrorsAreTheKeys(t *testing.T) {
	cases := []struct{ code, want string }{
		{"'x'를 \"가나\"로 정하자\n('x'의 없는이름)을 출력하자\n", "ReferenceError"},
		{"'x'를 \"가나\"로 정하자\n('x'의 (1 / 0))을 출력하자\n", "DivideByZero"},
		{"'x'를 \"가나\"로 정하자\n('x'의 \"a\")을 출력하자\n", "members on a string"},
		{"'x'를 [1, 2]로 정하자\n('x'의 '없는이름')을 출력하자\n", "ReferenceError"},
		{"'x'를 [1, 2]로 정하자\n('x'의 \"a\")을 출력하자\n", "list index must be a number"},
		{"'x'를 [1, 2]로 정하자\n'x'의 '없는이름'을 9로 정하자\n", "ReferenceError"},
		{"'x'를 [1, 2]로 정하자\n'x'의 (1 / 0)을 9로 정하자\n", "DivideByZero"},
	}
	for _, c := range cases {
		_, treeErr := runHari(t, c.code)
		_, bcErr := runBytecode(t, c.code)
		if treeErr == nil || bcErr == nil {
			t.Errorf("%q: tree %v, bytecode %v; want errors", c.code, treeErr, bcErr)
			continue
		}
		if treeErr.Error() != bcErr.Error() || !strings.Contains(treeErr.Error(), c.want) {
			t.Errorf("%q: tree %q, bytecode %q; want %s from both", c.code, treeErr.Error(), bcErr.Error(), c.want)
		}
	}
}
