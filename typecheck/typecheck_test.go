package typecheck

import "testing"

var ko = Names{Number: "숫자", String: "문자열", Boolean: "논리", Any: "아무거나", List: "목록", Dict: "사전", Null: "비어있음"}

type fakeHost map[string][]string // class -> its supertypes (classes and interfaces)

type obj struct{ class string }

func (h fakeHost) ClassOf(v interface{}) (string, bool) {
	if o, ok := v.(obj); ok {
		return o.class, true
	}
	return "", false
}

func (h fakeHost) IsSubtype(class, target string) bool {
	for _, s := range h[class] {
		if s == target || h.IsSubtype(s, target) {
			return true
		}
	}
	return false
}

func TestParseGenerics(t *testing.T) {
	cases := map[string]Spec{
		"숫자":          {Name: "숫자"},
		"":            {},
		"(숫자)목록":      {Name: "목록", Args: []string{"숫자"}},
		"(문자열, 숫자)사전": {Name: "사전", Args: []string{"문자열", "숫자"}},
		"(文字列,何でも)辞書": {Name: "辞書", Args: []string{"文字列", "何でも"}},
	}
	for in, want := range cases {
		got := Parse(in)
		if got.Name != want.Name || len(got.Args) != len(want.Args) {
			t.Errorf("Parse(%q) = %+v, want %+v", in, got, want)
			continue
		}
		for i := range want.Args {
			if got.Args[i] != want.Args[i] {
				t.Errorf("Parse(%q) args = %v, want %v", in, got.Args, want.Args)
			}
		}
	}
}

func TestPrimitivesNullAndAny(t *testing.T) {
	type c struct {
		typ  string
		v    interface{}
		want bool
	}
	for _, x := range []c{
		{"숫자", 3.0, true}, {"숫자", "3", false}, {"숫자", true, false},
		{"문자열", "a", true}, {"문자열", 3.0, false},
		{"논리", false, true}, {"논리", "참", false},
		{"숫자", nil, true}, {"문자열", nil, true}, {"자동차", nil, true}, // Nullable by default
		{"아무거나", "무엇이든", true}, {"", 3.0, true},
		{"비어있음", nil, true}, {"비어있음", 1.0, false},
		{"목록", []interface{}{}, true}, {"목록", "x", false},
		{"사전", map[interface{}]interface{}{}, true}, {"사전", []interface{}{}, false},
	} {
		if got := Parse(x.typ).Accepts(&ko, x.v, nil); got != x.want {
			t.Errorf("[%s] accepts %#v = %v, want %v", x.typ, x.v, got, x.want)
		}
	}
}

func TestGenericsAreCheckedElementByElement(t *testing.T) {
	nums := []interface{}{1.0, 2.0}
	mixed := []interface{}{1.0, "둘"}
	if !Parse("(숫자)목록").Accepts(&ko, nums, nil) || Parse("(숫자)목록").Accepts(&ko, mixed, nil) {
		t.Error("[(숫자)목록] must accept only all-number lists")
	}
	if !Parse("(숫자)목록").Accepts(&ko, []interface{}{1.0, nil}, nil) {
		t.Error("a null element is allowed")
	}
	d := map[interface{}]interface{}{"a": 1.0}
	bad := map[interface{}]interface{}{"a": "x"}
	badKey := map[interface{}]interface{}{1.0: 1.0}
	spec := Parse("(문자열, 숫자)사전")
	if !spec.Accepts(&ko, d, nil) || spec.Accepts(&ko, bad, nil) || spec.Accepts(&ko, badKey, nil) {
		t.Error("[(문자열, 숫자)사전] must check keys and values")
	}
	if !Parse("(아무거나)목록").Accepts(&ko, mixed, nil) {
		t.Error("[(아무거나)목록] accepts mixed elements")
	}
}

func TestClassesUpcastingAndInterfaces(t *testing.T) {
	h := fakeHost{"강아지": {"동물", "짖을수있는"}, "동물": nil}
	dog := obj{"강아지"}
	animal := obj{"동물"}
	if !Parse("강아지").Accepts(&ko, dog, h) || !Parse("동물").Accepts(&ko, dog, h) || !Parse("짖을수있는").Accepts(&ko, dog, h) {
		t.Error("an object satisfies its own class, its parent and its interfaces (upcasting)")
	}
	if Parse("강아지").Accepts(&ko, animal, h) {
		t.Error("a parent is not a child")
	}
	if Parse("동물").Accepts(&ko, "문자", h) || Parse("동물").Accepts(&ko, 3.0, h) {
		t.Error("non-objects are not instances of a class")
	}
	if Parse("없는클래스").Accepts(&ko, dog, h) {
		t.Error("an unknown type name matches nothing")
	}
}

func TestDescribeAndErrors(t *testing.T) {
	if Describe(&ko, 3.0, nil) != "숫자" || Describe(&ko, nil, nil) != "비어있음" || Describe(&ko, []interface{}{}, nil) != "목록" {
		t.Error("Describe should use the language's names")
	}
	if err := Check(ko, "숫자", "나이", "스물", nil); err == nil {
		t.Error("expected a mismatch error")
	}
	if err := Check(ko, "숫자", "나이", nil, nil); err != nil {
		t.Errorf("null passes: %v", err)
	}
	if err := CheckArgument(ko, "문자열", "이름", 3.0, nil); err == nil {
		t.Error("expected an argument mismatch error")
	}
}
