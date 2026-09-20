package vm

import (
	"fmt"
	"github.com/soumt-r/hana/errs"
	"github.com/soumt-r/hana/num"
	"github.com/soumt-r/hana/strcat"
	"github.com/soumt-r/hana/typecheck"
	"strings"
	"unicode/utf8"

	"github.com/soumt-r/hana/ast"
	"github.com/soumt-r/hana/conv"
	haja_lexer "github.com/soumt-r/hana/lexer/haja"
	haja_parser "github.com/soumt-r/hana/parser/haja"
	"github.com/soumt-r/hana/symbol"
)

func (i *Interpreter) Evaluate(expr ast.Expression, env *Environment) (interface{}, error) {
	switch expr.(type) {
	case *ast.CallExpression, *ast.NewExpression:
		i.callDepth++
		if i.callDepth > MaxCallDepth {
			i.callDepth--
			return nil, errs.New(errs.CallTooDeep, MaxCallDepth)
		}
		val, err := i.evaluate(expr, env)
		i.callDepth--
		return val, err
	}
	return i.evaluate(expr, env)
}

func (i *Interpreter) evaluate(expr ast.Expression, env *Environment) (interface{}, error) {
	if expr == nil {
		return nil, nil
	}

	switch e := expr.(type) {
	case *ast.TypeReference:
		clsObj, exists := env.Get(e.Name)
		if exists {
			return clsObj, nil
		}
		if _, exists := i.Classes[e.Name]; exists {
			return &ClassReference{ClassName: e.Name}, nil
		}
		return e.Name, nil
	case *ast.StringLiteral:
		val := e.Value
		val = strings.ReplaceAll(val, "\\n", "\n")
		val = strings.ReplaceAll(val, "\\\"", "\"")
		val = strings.ReplaceAll(val, "\\t", "\t")
		val = strings.ReplaceAll(val, "\\\\", "\\")
		return val, nil
	case *ast.BooleanLiteral:
		return e.Value, nil
	case *ast.TemplateLiteral:
		// {} 보간 구현: {expr} 부분을 파싱 후 평가
		raw := e.Value
		raw = strings.ReplaceAll(raw, "\\n", "\n")
		raw = strings.ReplaceAll(raw, "\\\"", "\"")
		raw = strings.ReplaceAll(raw, "\\t", "\t")
		raw = strings.ReplaceAll(raw, "\\\\", "\\")
		var result strings.Builder
		for len(raw) > 0 {
			open := strings.Index(raw, "{")
			if open == -1 {
				result.WriteString(raw)
				break
			}
			result.WriteString(raw[:open])
			raw = raw[open+1:]
			close := strings.Index(raw, "}")
			if close == -1 {
				result.WriteString("{")
				result.WriteString(raw)
				break
			}
			innerCode := raw[:close]
			raw = raw[close+1:]
			// innerCode를 파싱 후 평가
			var innerExpr ast.Expression
			if i.Config.ParseEmbeddedExpr != nil {
				innerExpr = i.Config.ParseEmbeddedExpr(innerCode)
			} else {
				innerLexer := haja_lexer.New(innerCode)
				innerParser := haja_parser.New(innerLexer)
				innerExpr = innerParser.ParseExpression()
			}
			if innerExpr != nil {
				val, err := i.Evaluate(innerExpr, env)
				if err != nil {
					return nil, err
				}
				result.WriteString(i.FormatValue(val))
			}
		}
		return result.String(), nil
	case *ast.NullLiteral:
		return nil, nil
	case *ast.NumberLiteral:
		return e.Value, nil
	case *ast.SelfReference:
		cur := env
		for cur != nil {
			if cur.this != nil {
				return cur.this, nil
			}
			cur = cur.parent
		}
		return nil, errs.New(errs.ThisNotBound)
	case *ast.SuperReference:
		cur := env
		for cur != nil {
			if cur.this != nil {
				return &SuperReference{Object: cur.this}, nil
			}
			cur = cur.parent
		}
		return nil, errs.New(errs.SuperOutsideMethod)
	case *ast.StaticReference:
		cur := env
		for cur != nil {
			if val, ok := cur.GetSym(selfClassSym); ok {
				if clsName, ok := val.(string); ok {
					return &ClassReference{ClassName: clsName}, nil
				}
			}
			cur = cur.parent
		}
		return nil, errs.New(errs.StaticOutsideMethod)
	case *ast.Identifier:
		sym := e.Symbol()
		if i.Config.IsSelfSym(sym) {
			cur := env
			for cur != nil {
				if cur.this != nil {
					return cur.this, nil
				}
				cur = cur.parent
			}
		} else if i.Config.IsPluralSelfSym(sym) {
			if clsName, exists := env.GetSym(selfClassSym); exists {
				if clsNameStr, ok := clsName.(string); ok {
					return &ClassReference{ClassName: clsNameStr}, nil
				}
			}
		}

		val, ok := env.GetSym(sym)
		if !ok {
			// 클래스 이름인지 확인
			if _, exists := i.Classes[e.Value]; exists {
				return &ClassReference{ClassName: e.Value}, nil // 클래스 이름 반환
			}
			return nil, errs.New(errs.VariableNotFound, e.Value)
		}
		return val, nil
	case *ast.FunctionReference:
		if strings.HasPrefix(e.Name, i.Config.VarQuoteOpen) && strings.HasSuffix(e.Name, i.Config.VarQuoteClose) {
			idName := e.Name[len(i.Config.VarQuoteOpen) : len(e.Name)-len(i.Config.VarQuoteClose)]
			if val, ok := env.Get(idName); ok {
				if strVal, ok := val.(string); ok {
					e.Name = strVal
				}
			}
		}
		if strings.Contains(e.Name, ".") {
			parts := strings.SplitN(e.Name, ".", 2)
			objName, methodName := parts[0], parts[1]
			if val, ok := env.Get(objName); ok {
				if obj, isObj := val.(*HajaObject); isObj {
					return &BoundMethod{Object: obj, FuncName: methodName, Sym: symbol.Intern(methodName)}, nil
				}
			}
		}

		if v, ok := i.scopedFunction(e.Name); ok {
			return v, nil
		}

		var fnDecl *ast.FunctionDeclaration
		for _, stmt := range i.ast.Statements {
			if f, ok := stmt.(*ast.FunctionDeclaration); ok && f.Name.Value == e.Name {
				fnDecl = f
				break
			}
		}
		if fnDecl == nil {
			if val, ok := env.Get(e.Name); ok {
				if f, ok := val.(*ast.FunctionDeclaration); ok {
					fnDecl = f
				}
			}
		}
		if fnDecl != nil {
			return fnDecl, nil
		}

		return e.Name, nil
	case *ast.NewExpression:
		clsName := e.Class.Name
		cls, ok := i.Classes[clsName]
		if !ok {
			if _, isIface := i.Interfaces[clsName]; isIface {
				return nil, errs.New(errs.InstantiateInterface, clsName)
			}
			return nil, errs.New(errs.ClassNotFound, clsName)
		}
		if cls.IsAbstract {
			return nil, errs.New(errs.InstantiateAbstract, clsName)
		}
		obj := NewHajaObject(clsName)

		if err := i.initFields(cls, obj, env); err != nil {
			return nil, err
		}

		// 생성자 호출. 자식 클래스에 생성자가 없으면 부모 체인에서 찾아 자동 호출한다
		// (Runtime 스펙 3.1.4: "자식 클래스에 생성자가 선언되어 있지 않다면, 런타임은
		// 부모 클래스의 생성자를 자동으로 찾아서 호출해야 합니다").
		var ctor *ast.ConstructorDeclaration
		i.findInClassChain(cls, func(body []ast.Statement) bool {
			for _, stmt := range body {
				if c, ok := stmt.(*ast.ConstructorDeclaration); ok {
					ctor = c
					return true
				}
			}
			return false
		})

		if ctor != nil {
			args := make([]interface{}, len(e.Arguments))
			for idx, argExpr := range e.Arguments {
				argVal, err := i.Evaluate(argExpr, env)
				if err != nil {
					return nil, err
				}
				args[idx] = argVal
			}
			if err := i.runConstructor(ctor, obj, clsName, args); err != nil {
				return nil, err
			}
		}

		return obj, nil
	case *ast.CallExpression:
		callCallee := e.Callee
		// "TYPE의 〈함수〉()": parser/haja's TYPE-then-TYPE_IN branch can't
		// tell at parse time whether TYPE names a real class (a static
		// method call, e.g. "【データベース】の〈取得する〉()") or a
		// built-in type name used as decorative packaging around a plain
		// builtin call (e.g. kanade-docs' "【文字列】の〈文字列に〉(123)")
		// — 문자열/숫자 aren't registered classes, so TypeReference
		// evaluation falls back to returning the name as a bare string,
		// which then gets misread as a BoundStringMethod on that literal
		// name. Only a real registered class means "static method call";
		// anything else means "ignore the TYPE, call the function plainly".
		if mem, ok := callCallee.(*ast.MemberExpression); ok {
			if typeRef, ok := mem.Object.(*ast.TypeReference); ok {
				if _, isClass := i.Classes[typeRef.Name]; !isClass {
					if fr, ok := mem.Property.(*ast.FunctionReference); ok {
						callCallee = fr
					}
				}
			}
		}
		callee, err := i.Evaluate(callCallee, env)
		if err != nil {
			return nil, err
		}

		args := make([]interface{}, len(e.Arguments))
		for idx, argExpr := range e.Arguments {
			argVal, err := i.Evaluate(argExpr, env)
			if err != nil {
				return nil, err
			}
			args[idx] = argVal
		}

		// 전역 함수 호출
		if funcName, ok := callee.(string); ok {
			// 먼저 env에서 BuiltinFunction 확인 (임포트된 외부 함수 등)
			if fnVal, ok := env.Get(funcName); ok {
				if builtIn, ok := fnVal.(*BuiltinFunction); ok {
					return builtIn.Fn(i, env, args...)
				}
			}

			var funcDecl *ast.FunctionDeclaration

			// env에 FunctionDeclaration이 있으면 사용 (import된 함수)
			if fnVal, ok := env.Get(funcName); ok {
				if f, ok := fnVal.(*ast.FunctionDeclaration); ok {
					funcDecl = f
				}
			}

			// 없으면 현재 AST에서 찾기
			if funcDecl == nil {
				for _, stmt := range i.ast.Statements {
					if f, ok := stmt.(*ast.FunctionDeclaration); ok && f.Name.Value == funcName {
						funcDecl = f
						break
					}
				}
			}
			if funcDecl == nil {
				return nil, errs.New(errs.GlobalFunctionNotFound, funcName)
			}
			funcEnv := i.newScope(i.globalOf(funcDecl.Module))
			res, err := i.runFunctionBody(funcDecl, args, funcEnv)
			i.freeScope(funcEnv)
			return res, err
		}

		if builtIn, ok := callee.(*BuiltinFunction); ok {
			return builtIn.Fn(i, env, args...)
		}

		if fnDecl, ok := callee.(*ast.FunctionDeclaration); ok {
			callEnv := i.newScope(i.globalOf(fnDecl.Module))
			res, err := i.runFunctionBody(fnDecl, args, callEnv)
			i.freeScope(callEnv)
			return res, err
		}

		if bm, ok := callee.(*BoundStaticMethod); ok {
			cls := i.Classes[bm.ClassName]
			var funcDecl *ast.FunctionDeclaration
			for _, stmt := range cls.Body {
				if f, ok := stmt.(*ast.FunctionDeclaration); ok && f.IsStatic && f.Name.Value == bm.FuncName {
					funcDecl = f
					break
				}
			}
			if funcDecl == nil {
				return nil, errs.New(errs.StaticMethodNotFound, bm.FuncName)
			}
			funcEnv := i.newScope(i.globalOf(funcDecl.Module))
			funcEnv.DeclareSym(selfClassSym, bm.ClassName)
			res, err := i.runFunctionBody(funcDecl, args, funcEnv)
			i.freeScope(funcEnv)
			return res, err
		} else if bsm, ok := callee.(*BoundStringMethod); ok {
			// 인자 개수/타입을 먼저 확인한다 — 예전엔 자르기만 개수를 체크하고
			// 나머지 셋은 곧장 args[0].(string) 같은 타입 단언을 했는데,
			// 인자가 없거나 타입이 틀리면 Haja 에러가 아니라 Go 런타임 패닉으로
			// 프로세스 전체가 죽었다(bcvm.callStringMethod 구현하며 발견,
			// 거기 먼저 안전하게 고치고 여기도 같은 체크로 맞춤).
			if bsm.FuncName == i.Config.StringSliceMethod {
				if len(args) != 2 {
					return nil, errs.New(errs.ArgCountExact, 2)
				}
				startNum, ok1 := args[0].(float64)
				endNum, ok2 := args[1].(float64)
				if !ok1 || !ok2 {
					return nil, errs.New(errs.MethodArgMustBeNumber, bsm.FuncName)
				}
				runes := []rune(bsm.Value)
				start := int(startNum) - 1
				end := int(endNum)
				if start < 0 {
					start = 0
				}
				if end > len(runes) {
					end = len(runes)
				}
				if start > end {
					start = end
				}
				return string(runes[start:end]), nil
			} else if bsm.FuncName == i.Config.StringReplaceMethod {
				if len(args) != 2 {
					return nil, errs.New(errs.ArgCountExact, 2)
				}
				oldStr, ok1 := args[0].(string)
				newStr, ok2 := args[1].(string)
				if !ok1 || !ok2 {
					return nil, errs.New(errs.MethodArgMustBeString, bsm.FuncName)
				}
				return strings.ReplaceAll(bsm.Value, oldStr, newStr), nil
			} else if bsm.FuncName == i.Config.StringSplitMethod {
				if len(args) != 1 {
					return nil, errs.New(errs.ArgCountExact, 1)
				}
				sep, ok := args[0].(string)
				if !ok {
					return nil, errs.New(errs.MethodArgMustBeString, bsm.FuncName)
				}
				parts := strings.Split(bsm.Value, sep)
				res := make([]interface{}, len(parts))
				for i, p := range parts {
					res[i] = p
				}
				return res, nil
			} else if bsm.FuncName == i.Config.StringContainsMethod {
				if len(args) != 1 {
					return nil, errs.New(errs.ArgCountExact, 1)
				}
				sub, ok := args[0].(string)
				if !ok {
					return nil, errs.New(errs.MethodArgMustBeString, bsm.FuncName)
				}
				return strings.Contains(bsm.Value, sub), nil
			}
			return nil, errs.New(errs.MethodNotFound, bsm.FuncName)
		} else if blm, ok := callee.(*BoundListMethod); ok {
			// 목록은 문자열과 달리 "언어 네이티브 구문"(추가하자/꺼내자 등)으로
			// 조작하는 게 기본 설계라(스펙 2.6), 메서드 형태로 남은 건 비우기
			// 하나뿐 — TS 참조 구현(haja-docs)도 딱 이것만 지원한다.
			if blm.FuncName == i.Config.ListClearMethod {
				if len(args) != 0 {
					return nil, errs.New(errs.ArgCountExact, 0)
				}
				if blm.Target != nil {
					if err := i.assignListBack(blm.Target, []interface{}{}, env, listShrunk); err != nil {
						return nil, err
					}
				}
				return nil, nil
			}
			return nil, errs.New(errs.MethodNotFound, blm.FuncName)
		} else if bm, ok := callee.(*BoundMethod); ok {
			cls := i.Classes[bm.Object.ClassName]
			var funcDecl *ast.FunctionDeclaration
			var ctorDecl *ast.ConstructorDeclaration

			// super 호출이면 바로 부모부터 탐색을 시작해, 오버라이딩되기 전의 원본을 찾는다.
			startCls := cls
			if bm.IsSuper && startCls != nil && startCls.BaseClass != nil {
				startCls = i.Classes[startCls.BaseClass.Name]
			}
			if bm.FuncName == "__init__" {
				i.findInClassChain(startCls, func(body []ast.Statement) bool {
					for _, stmt := range body {
						if f, ok := stmt.(*ast.FunctionDeclaration); ok && f.Name.Value == bm.FuncName {
							funcDecl = f
							return true
						} else if c, ok := stmt.(*ast.ConstructorDeclaration); ok {
							ctorDecl = c
							return true
						}
					}
					return false
				})
			} else {
				funcDecl = i.classMember(startCls, bm.Sym).method
			}
			if funcDecl == nil && ctorDecl == nil {
				return nil, errs.New(errs.MethodNotFound, bm.FuncName)
			}
			var module interface{}
			if funcDecl != nil {
				module = funcDecl.Module
			} else {
				module = ctorDecl.Module
			}
			funcEnv := i.newScope(i.globalOf(module))
			funcEnv.this = bm.Object
			funcEnv.DeclareSym(thisSym, bm.Object)
			funcEnv.DeclareSym(selfClassSym, bm.Object.ClassName)

			if funcDecl != nil {
				res, err := i.runFunctionBody(funcDecl, args, funcEnv)
				i.freeScope(funcEnv)
				return res, err
			} else if ctorDecl != nil {
				if err := i.bindParams(ctorDecl.Params, args, funcEnv); err != nil {
					return nil, err
				}
				for _, stmt := range ctorDecl.Body {
					_, err := i.Execute(stmt, funcEnv)
					if err != nil {
						if ret, isRet := err.(*ReturnValue); isRet {
							return ret.Value, nil
						}
						return nil, err
					}
				}
			}
			return nil, nil
		}
		return nil, errs.New(errs.NotCallable)
	case *ast.ListLiteral:
		elements := []interface{}{}
		for _, el := range e.Elements {
			val, err := i.Evaluate(el, env)
			if err != nil {
				return nil, err
			}
			elements = append(elements, val)
		}
		return elements, nil
	case *ast.ListPopExpression:
		targetVal, err := i.Evaluate(e.Target, env)
		if err != nil {
			return nil, err
		}
		list, ok := targetVal.([]interface{})
		if !ok {
			return nil, errs.New(errs.NotAList)
		}
		if len(list) == 0 {
			return nil, errs.New(errs.ListEmpty)
		}
		popped, newList := popFromList(list, e.Position)
		if err := i.assignListBack(e.Target, newList, env, listShrunk); err != nil {
			return nil, err
		}
		return popped, nil
	case *ast.DictLiteral:
		dict := make(map[interface{}]interface{})
		for _, prop := range e.Properties {
			key, err := i.Evaluate(prop.Key, env)
			if err != nil {
				return nil, err
			}
			val, err := i.Evaluate(prop.Value, env)
			if err != nil {
				return nil, err
			}
			dict[key] = val
		}
		return dict, nil
	case *ast.MemberExpression:
		if fr, ok := e.Property.(*ast.FunctionReference); ok {
			if strings.HasPrefix(fr.Name, i.Config.VarQuoteOpen) && strings.HasSuffix(fr.Name, i.Config.VarQuoteClose) {
				idName := fr.Name[len(i.Config.VarQuoteOpen) : len(fr.Name)-len(i.Config.VarQuoteClose)]
				if val, ok := env.Get(idName); ok {
					if strVal, ok := val.(string); ok {
						fr.Name = strVal
					}
				}
			}
		}
		obj, err := i.Evaluate(e.Object, env)
		if err != nil {
			return nil, err
		}
		if super, ok := obj.(*SuperReference); ok {
			propId, ok := e.Property.(*ast.FunctionReference)
			if !ok {
				return nil, errs.New(errs.SuperMemberMustBeMethod)
			}
			return &BoundMethod{Object: super.Object, FuncName: propId.Name, Sym: propId.Symbol(), IsSuper: true}, nil
		}
		if hajaObj, ok := obj.(*HajaObject); ok {
			propName := ""
			var propSym symbol.Symbol
			isFunc := false
			if fr, ok := e.Property.(*ast.FunctionReference); ok {
				propName, propSym = fr.Name, fr.Symbol()
				isFunc = true
			} else if id, ok := e.Property.(*ast.Identifier); ok {
				propName, propSym = id.Value, id.Symbol()
			}

			// 접근 제한자 검사
			member := i.classMember(i.classOf(hajaObj), propSym)
			access := member.fieldAccess
			if isFunc {
				access = member.methodAccess
			}

			if access != "public" {
				thisObj, hasThis := env.GetSym(thisSym)
				if !hasThis {
					return nil, errs.AccessViolation(access, isFunc, propName)
				}
				if access == "private" && thisObj != hajaObj {
					return nil, errs.AccessViolation(access, isFunc, propName)
				}
			}

			if isFunc {
				return &BoundMethod{Object: hajaObj, FuncName: propName, Sym: propSym}, nil
			}

			// Check for getter
			getterBody := member.getter
			if getterBody != nil {
				getterEnv := NewEnvironment(i.globalOf(i.classOf(hajaObj).Module))
				getterEnv.this = hajaObj
				getterEnv.DeclareSym(thisSym, hajaObj)
				getterEnv.DeclareSym(selfClassSym, hajaObj.ClassName)
				prev := i.enterModule(i.classOf(hajaObj).Module)
				for _, bs := range getterBody {
					_, err := i.Execute(bs, getterEnv)
					if err != nil {
						i.scope = prev
						if ret, isRet := err.(*ReturnValue); isRet {
							return ret.Value, nil
						}
						return nil, err
					}
				}
				i.scope = prev
				return nil, nil // Or throw error if no return?
			}
			return hajaObj.Props[propName], nil
		} else if list, ok := obj.([]interface{}); ok {
			propName := ""
			isFunc := false
			if fr, ok := e.Property.(*ast.FunctionReference); ok {
				propName = fr.Name
				isFunc = true
			} else if id, ok := e.Property.(*ast.Identifier); ok {
				propName = id.Value
			}

			if isFunc {
				return &BoundListMethod{List: list, FuncName: propName, Target: e.Object}, nil
			}
			if propName == i.Config.LengthWord {
				return float64(len(list)), nil
			}

			idxObj, err := i.Evaluate(e.Property, env)
			if err == nil {
				if numVal, isNum := idxObj.(float64); isNum {
					idx := int(numVal) - 1 // 1-based to 0-based
					if idx < 0 || idx >= len(list) {
						return nil, errs.New(errs.ListIndexOutOfRange)
					}
					return list[idx], nil
				}
			}
			return nil, errs.New(errs.ListIndexMustBeNumber)
		} else if dict, ok := obj.(map[interface{}]interface{}); ok {
			key, err := i.Evaluate(e.Property, env)
			if err != nil {
				return nil, err
			}
			if val, exists := dict[key]; exists {
				return val, nil
			}
			return nil, errs.New(errs.DictKeyNotFound, key)
		} else if clsRef, ok := obj.(*ClassReference); ok {
			clsName := clsRef.ClassName
			propName := ""
			isFunc := false
			if fr, ok := e.Property.(*ast.FunctionReference); ok {
				propName = fr.Name
				isFunc = true
			} else if id, ok := e.Property.(*ast.Identifier); ok {
				propName = id.Value
			} else if numVal, ok := e.Property.(*ast.NumberLiteral); ok {
				propName = fmt.Sprintf("%g", numVal.Value)
			}

			cls := i.Classes[clsName]
			if cls == nil {
				return nil, errs.New(errs.ClassNotFound, clsName)
			}

			if isFunc {
				for _, stmt := range cls.Body {
					if f, ok := stmt.(*ast.FunctionDeclaration); ok && f.IsStatic && f.Name.Value == propName {
						return &BoundStaticMethod{ClassName: clsName, FuncName: propName}, nil
					}
				}
			} else {
				globalKey := clsName + "." + propName
				if val, ok := i.GlobalEnv.Get(globalKey); ok {
					return val, nil
				}
			}
			return nil, errs.New(errs.StaticMemberNotFound, propName)
		} else if strVal, ok := obj.(string); ok {
			// Actual String Object Access
			propName := ""
			isFunc := false
			if fr, ok := e.Property.(*ast.FunctionReference); ok {
				propName = fr.Name
				isFunc = true
			} else if id, ok := e.Property.(*ast.Identifier); ok {
				propName = id.Value
			} else if numVal, ok := e.Property.(*ast.NumberLiteral); ok {
				propName = fmt.Sprintf("%g", numVal.Value)
			}

			if isFunc {
				return &BoundStringMethod{Value: strVal, FuncName: propName}, nil
			}
			if propName == i.Config.LengthWord {
				return float64(utf8.RuneCountInString(strVal)), nil
			}

			idxObj, err := i.Evaluate(e.Property, env)
			if err == nil {
				if numVal, isNum := idxObj.(float64); isNum {
					char, ok := conv.RuneAt(strVal, int(numVal)-1)
					if !ok {
						return nil, errs.New(errs.StringIndexOutOfRange)
					}
					return char, nil
				}
				return nil, errs.New(errs.MemberAccessUnsupported, errs.TypeNameOf(obj))
			}
			return nil, errs.New(errs.MemberAccessOnString)
		}
		return nil, errs.New(errs.MemberAccessUnsupported, errs.TypeNameOf(obj))
	case *ast.LogicalExpression:
		left, err := i.Evaluate(e.Left, env)
		if err != nil {
			return nil, err
		}
		leftBool, err := i.requireBool(left)
		if err != nil {
			return nil, err
		}
		if e.Operator == "그리고" {
			if !leftBool {
				return false, nil
			}
		} else if leftBool {
			return true, nil
		}
		right, err := i.Evaluate(e.Right, env)
		if err != nil {
			return nil, err
		}
		return i.requireBool(right)
	case *ast.BinaryExpression:
		left, err := i.Evaluate(e.Left, env)
		if err != nil {
			return nil, err
		}
		right, err := i.Evaluate(e.Right, env)
		if err != nil {
			return nil, err
		}

		if e.Operator == "instanceof" {
			if leftObj, ok := left.(*HajaObject); ok {
				if clsRef, ok := right.(*ClassReference); ok {
					return i.classIsOrExtends(leftObj.ClassName, clsRef.ClassName), nil
				}
			}
			return false, nil
		}

		if e.Operator == "==" || e.Operator == "!=" {
			if leftObj, ok := left.(*HajaObject); ok {
				cls := i.Classes[leftObj.ClassName]
				var funcDecl *ast.FunctionDeclaration
				for _, stmt := range cls.Body {
					if f, ok := stmt.(*ast.FunctionDeclaration); ok && f.Name.Value == i.Config.EqualsMethodName {
						funcDecl = f
						break
					}
				}
				if funcDecl != nil {
					funcEnv := NewEnvironment(i.globalOf(funcDecl.Module))
					funcEnv.this = leftObj
					funcEnv.DeclareSym(selfClassSym, leftObj.ClassName)
					if len(funcDecl.Params) > 0 {
						funcEnv.DeclareSym(funcDecl.Params[0].Name.Symbol(), right)
					}
					for _, bs := range funcDecl.Body.Statements {
						_, err := i.Execute(bs, funcEnv)
						if err != nil {
							if retErr, isRet := err.(*ReturnValue); isRet {
								if e.Operator == "!=" {
									if retBool, ok := retErr.Value.(bool); ok {
										return !retBool, nil
									}
								}
								return retErr.Value, nil
							}
							return nil, err
						}
					}
					if e.Operator == "!=" {
						return true, nil
					}
					return false, nil
				}
			}
			if e.Operator == "==" {
				return left == right, nil
			} else {
				return left != right, nil
			}
		}

		switch e.Operator {
		case "+", "-", "*", "/", "%", ">", "<", ">=", "<=":
		default:
			return nil, errs.New(errs.UnknownOperator, e.Operator)
		}

		// Null-safe (Runtime spec 2.4): only the equality operators may see 비어있음.
		if left == nil || right == nil {
			return nil, errs.New(errs.NullOperand, e.Operator)
		}

		// 숫자 연산
		leftNum, leftIsNum := left.(float64)
		rightNum, rightIsNum := right.(float64)

		if leftIsNum && rightIsNum {
			switch e.Operator {
			case "+":
				return num.Box(leftNum + rightNum), nil
			case "-":
				return num.Box(leftNum - rightNum), nil
			case "*":
				return num.Box(leftNum * rightNum), nil
			case "/":
				if rightNum == 0 {
					return nil, errs.New(errs.DivideByZero)
				}
				return num.Box(leftNum / rightNum), nil
			case "%":
				if int64(rightNum) == 0 {
					return nil, errs.New(errs.DivideByZero)
				}
				return num.Box(float64(int64(leftNum) % int64(rightNum))), nil
			case ">":
				return leftNum > rightNum, nil
			case "<":
				return leftNum < rightNum, nil
			case ">=":
				return leftNum >= rightNum, nil
			case "<=":
				return leftNum <= rightNum, nil
			}
		}

		// 문자열 덧셈: 문자열끼리만. 묵시적 형변환은 없다 (Runtime spec 2.2).
		if e.Operator == "+" {
			if ls, ok := left.(string); ok {
				if rs, ok := right.(string); ok {
					return strcat.Join(ls, rs), nil
				}
			}
		}
		return nil, errs.New(errs.OperandTypeMismatch, e.Operator,
			typecheck.Describe(i.Config.Types, left, i.host()), typecheck.Describe(i.Config.Types, right, i.host()))
	}
	return nil, nil
}
