package vm

import (
	"github.com/soumt-r/hana/ast"
	"github.com/soumt-r/hana/symbol"
)

// findInClassChain은 cls부터 시작해 BaseClass 체인을 따라 부모 쪽으로 올라가며 visit을
// 호출합니다. visit은 현재 클래스의 Body를 받아, 찾고 있던 멤버(필드/메서드/생성자 등)를
// 그 클래스 안에서 발견하면 true를 반환해야 합니다 — 그러면 탐색을 즉시 멈춥니다.
//
// 어떤 클래스에서 이름은 있는데 원하는 조건(예: getter가 실제로 있는지)은 아닐 때도
// visit이 true를 반환하면, 그 시점에 부모 탐색을 멈춥니다 — 이름이 한 번 그 클래스에서
// "발견"되면(섀도잉되면) 더 위 조상까지 올라가 같은 이름을 찾지 않는다는 하자의 규칙을
// 그대로 따릅니다. 이 판단은 visit 클로저가 맡습니다.
// classMembers are the answers found so far for one class, indexed by Symbol.
type classMembers struct {
	cls   *ast.ClassDeclaration
	bySym []*classMember
}

// classMember는 한 클래스에서 시작해 이름 하나를 BaseClass 체인에서 찾은 결과입니다.
// 필드(접근 제한자/getter/setter)와 메서드는 각각 체인에서 가장 먼저 그 이름을
// 선언한 클래스가 정합니다. 클래스 선언은 실행 중에 바뀌지 않으므로(가져오기로
// 등록될 때만 registerClass가 비웁니다) 결과를 기억해 두고 다시 씁니다.
type classMember struct {
	fieldAccess  string
	getter       []ast.Statement
	setter       *ast.SetterInfo
	method       *ast.FunctionDeclaration
	methodAccess string
}

func (i *Interpreter) classMember(cls *ast.ClassDeclaration, sym symbol.Symbol) *classMember {
	cache := i.lastMembers
	if cache == nil || cache.cls != cls {
		cache = i.memberCache[cls]
		if cache == nil {
			if i.memberCache == nil {
				i.memberCache = make(map[*ast.ClassDeclaration]*classMembers)
			}
			cache = &classMembers{cls: cls}
			i.memberCache[cls] = cache
		}
		i.lastMembers = cache
	}
	if int(sym) < len(cache.bySym) {
		if m := cache.bySym[sym]; m != nil {
			return m
		}
	}
	name := sym.String()
	m := &classMember{fieldAccess: "public", methodAccess: "public"}
	fieldDone := false
	i.findInClassChain(cls, func(body []ast.Statement) bool {
		for _, stmt := range body {
			switch d := stmt.(type) {
			case *ast.VariableDeclaration:
				if !fieldDone && d.Name.Value == name {
					m.fieldAccess, m.getter, m.setter = d.AccessModifier, d.Getter, d.Setter
					fieldDone = true
				}
			case *ast.FunctionDeclaration:
				if m.method == nil && d.Name.Value == name {
					m.method, m.methodAccess = d, d.AccessModifier
				}
			}
		}
		return fieldDone && m.method != nil
	})
	if int(sym) >= len(cache.bySym) {
		grown := make([]*classMember, symbol.Count()+16)
		copy(grown, cache.bySym)
		cache.bySym = grown
	}
	cache.bySym[sym] = m
	return m
}

// classOf는 객체의 클래스 선언입니다. 처음 찾은 것을 객체가 기억해 두어, 멤버에 접근할
// 때마다 이름으로 클래스를 다시 찾지 않습니다.
func (i *Interpreter) classOf(obj *HajaObject) *ast.ClassDeclaration {
	if obj.class == nil {
		obj.class = i.Classes[obj.ClassName]
	}
	return obj.class
}

// registerClass는 이름에 클래스를 등록하고, 이전에 기억한 멤버 조회를 비웁니다.
func (i *Interpreter) registerClass(name string, cls *ast.ClassDeclaration) {
	i.Classes[name] = cls
	i.memberCache = nil
	i.lastMembers = nil
}

func (i *Interpreter) findInClassChain(cls *ast.ClassDeclaration, visit func(body []ast.Statement) (found bool)) {
	for cls != nil {
		if visit(cls.Body) {
			return
		}
		if cls.BaseClass == nil {
			return
		}
		cls = i.Classes[cls.BaseClass.Name]
	}
}

// classIsOrExtends reports whether the class named className is targetName
// itself, or inherits from it directly or transitively via BaseClass
// (upcasting). Shared by the `instanceof` operator and TryStatement's
// catch-type matching (Runtime spec 4.2), which are both the same
// "is A a B" question over the class hierarchy.
func (i *Interpreter) classIsOrExtends(className, targetName string) bool {
	for className != "" {
		if className == targetName {
			return true
		}
		cls := i.Classes[className]
		if cls == nil || cls.BaseClass == nil {
			return false
		}
		className = cls.BaseClass.Name
	}
	return false
}

// thrownValueMatchesType checks whether the value a TryStatement's block
// threw matches a CatchClause's declared type. Only a thrown *HajaObject (an
// actual class instance, typically of [오류] or a subclass) has a class to
// match against — an engine-raised error (TypeError, IndexOutOfBoundsError,
// etc., a plain Go error with no Haja class) can never match a specific
// type and is only reachable through an untyped `오류가 발생했다면` handler.
func (i *Interpreter) thrownValueMatchesType(err error, typeName string) bool {
	tErr, ok := err.(*ThrownError)
	if !ok {
		return false
	}
	obj, ok := tErr.Value.(*HajaObject)
	if !ok {
		return false
	}
	return i.classIsOrExtends(obj.ClassName, typeName)
}
