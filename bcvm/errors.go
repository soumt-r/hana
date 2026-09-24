package bcvm

import (
	"fmt"

	"github.com/soumt-r/hana/bytecode"
	"github.com/soumt-r/hana/errs"
)

// tryHandler is one active TRY_PUSH frame on the current exec() call's try
// stack: which catches it declared, its finally chunk (if any), and the
// operand stack depth to restore to before binding the caught value —
// arbitrary pushes/pops may have happened between TRY_PUSH and wherever the
// error actually occurred, and the matched handler's compiled code expects
// the stack exactly as TRY_PUSH left it, plus the caught value on top.
type tryHandler struct {
	catches    []bytecode.CatchInfo
	finally    *bytecode.Chunk
	stackDepth int
}

// dispatchError searches tryStack (innermost/last first) for a handler
// whose declared catches match err, running each level's finally along the
// way whether or not it ends up matching — Runtime spec 4.2: 마무리는 항상
// runs "에러가 발생했든" before the error is re-thrown further up. A
// matching handler's ok is true, with the stack already unwound and the
// caught value pushed so the handler's own leading SET_VAR can bind it. No
// match anywhere returns ok=false — err (possibly overwritten by a
// finally's own error, exactly like vm/exec_stmt.go's `err = finErr`)
// propagates as an ordinary Go error from there.
func (vm *VM) dispatchError(err error, tryStack *[]*tryHandler, stack *[]interface{}, push func(interface{}), locals *frame) (matchedPc int, ok bool, resultErr error) {
	for len(*tryStack) > 0 {
		h := (*tryStack)[len(*tryStack)-1]
		*tryStack = (*tryStack)[:len(*tryStack)-1]

		typeName, matchable := catchTypeName(err)
		for _, cb := range h.catches {
			if cb.TypeName == "" || (matchable && vm.classIsOrExtends(typeName, cb.TypeName)) {
				*stack = (*stack)[:h.stackDepth]
				push(vm.errorObjectFor(err))
				return cb.HandlerPc, true, nil
			}
		}

		if h.finally != nil {
			if _, finErr := vm.exec(h.finally, locals); finErr != nil {
				err = finErr
			}
		}
	}
	return 0, false, err
}

// thrownValue wraps whatever a THROW statement raised (any Hari value, not
// just an *Object — 던지자 "문자열" is legal, just only catchable by an
// untyped handler). Mirrors vm.ThrownError.
type thrownValue struct {
	Value interface{}
}

// Error is what the CLI prints for an uncaught throw: the message of a thrown
// error object (its 메시지/メッセージ property), else the value as text.
func (t *thrownValue) Error() string {
	if obj, ok := t.Value.(*Object); ok {
		for _, key := range []string{"메시지", "メッセージ"} {
			if msg, ok := obj.Props[key].(string); ok {
				return msg
			}
		}
	}
	return fmt.Sprintf("%v", t.Value)
}

// catchTypeName reports the class name a typed catch clause could match err
// against, and whether err is even eligible for typed matching at all — only
// a thrown *Object (a real class instance) has a class; an engine-raised
// error (ReferenceError, TypeError, ...) or a thrown non-object value never
// matches a specific type and is only reachable through an untyped handler.
// Mirrors vm.thrownValueMatchesType's own eligibility check.
func catchTypeName(err error) (string, bool) {
	tv, ok := err.(*thrownValue)
	if !ok {
		return "", false
	}
	obj, ok := tv.Value.(*Object)
	if !ok {
		return "", false
	}
	return obj.ClassName, true
}

// errorObjectFor builds the value a catch clause's param binds to, mirroring
// vm/exec_stmt.go's TryStatement handler: a thrown *Object is used as-is
// (so a user-defined exception class keeps all its own fields/methods);
// anything else — a thrown non-object value, or an engine-raised error — is
// wrapped in a fresh 오류 instance with its string form as '메시지'.
func (vm *VM) errorObjectFor(err error) interface{} {
	if tv, ok := err.(*thrownValue); ok {
		if obj, ok := tv.Value.(*Object); ok {
			return obj
		}
	}
	obj := newObject(vm.ErrorClass)
	obj.Props[vm.ErrorMessage] = errs.Localize(vm.Locale, err)
	return obj
}
