package vm

import (
	"github.com/soumt-r/hana/console"
	"github.com/soumt-r/hana/conv"
	"github.com/soumt-r/hana/errs"
	"github.com/soumt-r/hana/num"
	"github.com/soumt-r/hana/symbol"
	"github.com/soumt-r/hana/value"

	"github.com/soumt-r/hana/ast"
)

func (i *Interpreter) Execute(stmt ast.Statement, env *Environment) (interface{}, error) {
	switch s := stmt.(type) {
	case *ast.ThrowStatement:
		val, err := i.Evaluate(s.Value, env)
		if err != nil {
			return nil, err
		}
		return nil, &ThrownError{Value: val}
	case *ast.TryStatement:
		var err error
		for _, bs := range s.Block.Statements {
			_, err = i.Execute(bs, env)
			if err != nil {
				break
			}
		}

		if err != nil {
			if _, isRet := err.(*ReturnValue); isRet {
				// return은 catch로 잡지 않음 (그래도 finally는 실행 — 아래)
			} else if _, isBreak := err.(*BreakValue); isBreak {
				// break도 catch로 잡지 않음 (그래도 finally는 실행 — 아래)
			} else {
				// 에러 타입 매칭 (Runtime 스펙 4.2): 타입이 없는 핸들러는 무조건 매칭되고,
				// 타입이 있는 핸들러는 던져진 값이 그 타입의 인스턴스일 때만(업캐스팅 포함)
				// 매칭된다. 엔진이 직접 던진 에러(TypeError 등, HariObject가 아님)는 어떤
				// 하자 클래스의 인스턴스도 아니므로 타입 없는 핸들러로만 잡을 수 있다.
				var matchedHandler *ast.CatchClause
				errStr := errs.Localize(i.Config.Locale, err)
				for _, h := range s.Handlers {
					if h.Type == nil || i.thrownValueMatchesType(err, h.Type.Name) {
						matchedHandler = h
						break
					}
				}

				// matchedHandler가 nil이면(핸들러가 아예 없거나 타입이 하나도 안 맞으면)
				// err를 그대로 둔 채 아래로 흘려보낸다 — Runtime 스펙 4.2: "FinallyClause가
				// 끝난 직후 상위 호출 스택으로 해당 에러를 반드시 다시 던져야" 하므로, 여기서
				// 바로 return하면 안 되고 finally를 거친 뒤에 다시 던져져야 한다.
				if matchedHandler != nil {
					catchEnv := NewEnvironment(env)
					errObj := NewHariObject(i.Config.BuiltinErrorClass)
					if tErr, ok := err.(*ThrownError); ok {
						if obj, isObj := tErr.Value.(*HariObject); isObj {
							errObj = obj
						} else {
							errObj.Props[i.Config.BuiltinErrorMessage] = tErr.Error()
						}
					} else {
						errObj.Props[i.Config.BuiltinErrorMessage] = errStr
					}
					catchEnv.DeclareSym(matchedHandler.Param.Symbol(), errObj)

					err = nil // 에러 복구
					for _, bs := range matchedHandler.Body.Statements {
						_, err = i.Execute(bs, catchEnv)
						if err != nil {
							break
						}
					}
				}
			}
		}

		if rv, ok := err.(*ReturnValue); ok && rv == &i.retBuf {
			// the finalizer runs code that returns from functions too, which reuses retBuf
			err = &ReturnValue{Value: rv.Value}
		}
		if s.Finalizer != nil {
			for _, bs := range s.Finalizer.Statements {
				_, finErr := i.Execute(bs, env)
				if finErr != nil {
					// finally에서의 에러가 기존 에러를 덮어씀
					err = finErr
					break
				}
			}
		}

		return nil, err
	case *ast.ListPushStatement:
		targetVal, err := i.Evaluate(s.Target, env)
		if err != nil {
			return nil, err
		}
		list, ok := targetVal.(*value.List)
		if !ok {
			return nil, errs.New(errs.NotAList)
		}
		if err := i.requireMutable(s.Target, env); err != nil {
			return nil, err
		}
		pushVal, err := i.Evaluate(s.Value, env)
		if err != nil {
			return nil, err
		}
		front := s.Position == "front"
		list.Push(pushVal, front)
		if err := i.checkListPush(s.Target, list, env, front); err != nil {
			list.Unpush(front)
			return nil, err
		}
	case *ast.ListPopStatement:
		targetVal, err := i.Evaluate(s.Target, env)
		if err != nil {
			return nil, err
		}
		list, ok := targetVal.(*value.List)
		if !ok {
			return nil, errs.New(errs.NotAList)
		}
		if err := i.requireMutable(s.Target, env); err != nil {
			return nil, err
		}
		if len(list.Items) == 0 {
			return nil, errs.New(errs.ListEmpty)
		}
		list.Pop(s.Position)
	case *ast.ReturnStatement:
		var val interface{}
		if s.Value != nil {
			v, err := i.Evaluate(s.Value, env)
			if err != nil {
				return nil, err
			}
			val = v
		}
		i.retBuf.Value = val
		return nil, &i.retBuf
	case *ast.BreakStatement:
		if !env.inLoop() {
			return nil, errs.New(errs.IllegalBreak)
		}
		return nil, &BreakValue{}
	case *ast.ImportStatement:
		return i.executeImport(s, env)

	case *ast.VariableDeclaration:
		val, err := i.Evaluate(s.Value, env)
		if err != nil {
			return nil, err
		}
		if s.IsStatic {
			clsName, exists := env.GetSym(selfClassSym)
			if exists {
				if clsNameStr, ok := clsName.(string); ok {
					globalKey := clsNameStr + "." + s.Name.Value
					i.GlobalEnv.Declare(globalKey, val)
				}
			}
		} else {
			annotation := ""
			if s.TypeRef != nil {
				annotation = s.TypeRef.Name
			}
			if err := i.assignVariable(env, s.Name, val, annotation, s.IsConstant); err != nil {
				return nil, err
			}
		}
	case *ast.InputStatement:
		typeName := ""
		if s.TypeRef != nil {
			typeName = s.TypeRef.Name
		}
		text := ""
		if i.ReadLine != nil {
			text = i.ReadLine()
		}
		val, err := conv.ParseInput(typeName, i.Config.Types, text, i.Config.TrueString, i.Config.FalseString)
		if err != nil {
			return nil, err
		}
		if s.Target != nil {
			if err := i.assignVariable(env, s.Target, val, "", false); err != nil {
				return nil, err
			}
		}
	case *ast.PrintStatement:
		val, err := i.Evaluate(s.Value, env)
		if err != nil {
			return nil, err
		}
		strVal := i.FormatValue(val)
		sink := i.sink()
		sink.Output = append(sink.Output, strVal)
		if s.NewLine {
			console.Println(strVal)
		} else {
			console.Print(strVal)
		}
	case *ast.ForEachLoop:
		var listVal interface{}
		var itemName string
		var err error

		if memExpr, ok := s.List.(*ast.MemberExpression); ok {
			listVal, err = i.Evaluate(memExpr.Object, env)
			if id, ok := memExpr.Property.(*ast.Identifier); ok {
				itemName = id.Value
			} else {
				itemName = i.Config.DefaultItemName
			}
		} else {
			listVal, err = i.Evaluate(s.List, env)
			itemName = i.Config.DefaultItemName
		}

		if err != nil {
			return nil, err
		}
		var items []interface{}
		isList := false
		if text, ok := listVal.(string); ok {
			items = make([]interface{}, 0, len(text))
			for _, r := range text {
				items = append(items, string(r))
			}
			isList = true
		} else if list, ok := listVal.(*value.List); ok {
			// a loop walks the list as it was when the loop began
			items = append([]interface{}(nil), list.Items...)
			isList = true
		}
		if isList {
			itemSym := symbol.Intern(itemName)
			for _, item := range items {
				// 반복문 환경 생성 (옵션)
				loopEnv := i.newLoopScope(env)
				// 원래 Hari 스펙에서는 '꺼낸 값' 같은 특수 키워드나 명시적 순회 변수가 필요하지만
				// ForEachLoop의 순회 변수를 현재 AST가 지원하지 않으므로, 임시로 '아이템'이라고 지정
				loopEnv.DeclareSym(itemSym, item)
				for _, bs := range s.Body.Statements {
					_, err := i.Execute(bs, loopEnv)
					if err != nil {
						if _, isBreak := err.(*BreakValue); isBreak {
							return nil, nil
						}
						return nil, err
					}
				}
				i.freeScope(loopEnv)
			}
		} else {
			return nil, errs.New(errs.NotIterable)
		}
	case *ast.WhileLoop:
		for {
			condVal, err := i.Evaluate(s.Condition, env)
			if err != nil {
				return nil, err
			}
			isTrue, err := i.requireBool(condVal)
			if err != nil {
				return nil, err
			}
			if !isTrue {
				break
			}

			loopEnv := i.newLoopScope(env)
			for _, bs := range s.Body.Statements {
				_, err := i.Execute(bs, loopEnv)
				if err != nil {
					if _, isBreak := err.(*BreakValue); isBreak {
						return nil, nil
					}
					return nil, err
				}
			}
			i.freeScope(loopEnv)
		}
		return nil, nil
	case *ast.ForRangeStatement:
		startVal, err := i.Evaluate(s.Start, env)
		if err != nil {
			return nil, err
		}
		endVal, err := i.Evaluate(s.End, env)
		if err != nil {
			return nil, err
		}

		startNum, ok1 := startVal.(float64)
		endNum, ok2 := endVal.(float64)
		if !ok1 || !ok2 {
			return nil, errs.New(errs.RangeMustBeNumbers)
		}

		step := 1.0
		if startNum > endNum {
			step = -1.0
		}

		varName := s.LoopVar
		if varName == "" {
			varName = i.Config.DefaultIndexName
		}
		varSym := symbol.Intern(varName)
		for v := startNum; ; v += step {
			if (step > 0 && v > endNum) || (step < 0 && v < endNum) {
				break
			}
			loopEnv := i.newLoopScope(env)
			loopEnv.DeclareSym(varSym, num.Box(v))
			for _, bs := range s.Body.Statements {
				_, err := i.Execute(bs, loopEnv)
				if err != nil {
					if _, isBreak := err.(*BreakValue); isBreak {
						return nil, nil // 완전히 루프 탈출
					}
					return nil, err
				}
			}
			i.freeScope(loopEnv)
		}
	case *ast.ExpressionStatement:
		_, err := i.Evaluate(s.Expression, env)
		if err != nil {
			return nil, err
		}
	case *ast.SwitchStatement:
		val, err := i.Evaluate(s.Discriminant, env)
		if err != nil {
			return nil, err
		}

		matched := false
		fallthroughNext := false
		for _, c := range s.Cases {
			if !matched && !fallthroughNext {
				if c.IsDefault {
					matched = true
				} else {
					for _, t := range c.Tests {
						testVal, err := i.Evaluate(t, env)
						if err != nil {
							return nil, err
						}
						if value.Equal(val, testVal) {
							matched = true
							break
						}
					}
				}
			} else if fallthroughNext {
				matched = true
				fallthroughNext = false
			}

			if matched {
				for _, stmt := range c.Consequent.Statements {
					if _, ok := stmt.(*ast.FallthroughStatement); ok {
						fallthroughNext = true
						break
					}
					res, err := i.Execute(stmt, env)
					if err != nil {
						return nil, err
					}
					if res != nil {
						return res, nil
					}
				}
				if fallthroughNext {
					continue
				}
				break
			}
		}
	case *ast.FallthroughStatement:
		return nil, nil
	case *ast.IfStatement:
		cond, err := i.Evaluate(s.Condition, env)
		if err != nil {
			return nil, err
		}
		isTrue, err := i.requireBool(cond)
		if err != nil {
			return nil, err
		}
		if isTrue {
			for _, bs := range s.Consequent.Statements {
				_, err := i.Execute(bs, env)
				if err != nil {
					return nil, err
				}
			}
		} else if s.Alternate != nil {
			for _, bs := range s.Alternate.Statements {
				_, err := i.Execute(bs, env)
				if err != nil {
					return nil, err
				}
			}
		}
	case *ast.Assignment:
		val, err := i.Evaluate(s.Value, env)
		if err != nil {
			return nil, err
		}

		if id, ok := s.Target.(*ast.Identifier); ok {
			if err := i.checkDeclaredType(env, id, val); err != nil {
				return nil, err
			}
			if _, err := env.AssignSym(id.Symbol(), val); err != nil {
				return nil, err
			}
		} else if mem, ok := s.Target.(*ast.MemberExpression); ok {
			obj, err := i.Evaluate(mem.Object, env)
			if err != nil {
				return nil, err
			}

			if hariObj, ok := obj.(*HariObject); ok {
				if propId, ok := mem.Property.(*ast.Identifier); ok {
					// Check for setter
					setter := i.classMember(i.classOf(hariObj), propId.Symbol()).setter

					if setter != nil {
						setterEnv := NewEnvironment(i.globalOf(i.classOf(hariObj).Module))
						setterEnv.this = hariObj
						setterEnv.DeclareSym(thisSym, hariObj)
						setterEnv.DeclareSym(selfClassSym, hariObj.ClassName)
						if setter.Param != nil {
							setterEnv.DeclareSym(setter.Param.Symbol(), val)
						}
						prev := i.enterModule(i.classOf(hariObj).Module)
						for _, bs := range setter.Body {
							_, err := i.Execute(bs, setterEnv)
							if err != nil {
								i.scope = prev
								if ret, isRet := err.(*ReturnValue); isRet {
									return ret.Value, nil
								}
								return nil, err
							}
						}
						i.scope = prev
					} else {
						if err := i.checkField(hariObj, propId.Value, val); err != nil {
							return nil, err
						}
						hariObj.Props[propId.Value] = val
					}
				}

			} else if list, ok := obj.(*value.List); ok {
				var idx = -1
				if numLit, ok := mem.Property.(*ast.NumberLiteral); ok {
					idx = int(numLit.Value) - 1
				} else {
					// A key that cannot be computed reports its own error (as the
					// bytecode engine does); one that is not a number is ignored.
					propVal, err := i.Evaluate(mem.Property, env)
					if err != nil {
						return nil, err
					}
					if num, ok := propVal.(float64); ok {
						idx = int(num) - 1
					}
				}
				if idx >= 0 && idx < len(list.Items) {
					list.Items[idx] = val
				}
			} else if dict, ok := obj.(map[interface{}]interface{}); ok {
				propVal, err := i.Evaluate(mem.Property, env)
				if err == nil {
					dict[propVal] = val
				} else if id, ok := mem.Property.(*ast.Identifier); ok {
					dict[id.Value] = val
				}
			} else if _, ok := obj.(string); ok {
				return nil, errs.New(errs.StringIndexAssign)
			} else if clsRef, ok := obj.(*ClassReference); ok {
				var propName string
				if id, ok := mem.Property.(*ast.Identifier); ok {
					propName = id.Value
				}
				if propName != "" {
					globalKey := clsRef.ClassName + "." + propName
					i.GlobalEnv.Declare(globalKey, val)
				}
			}
		}
	default:
		// 지원 안 하는 문법 무시
	}
	return nil, nil
}
