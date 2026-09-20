// Package bytecode defines hana's bytecode instruction set and the
// AST → Chunk compiler.
package bytecode

// Opcode is a single bytecode instruction's operation.
type Opcode byte

const (
	PUSH_CONST Opcode = iota // Operand: int (index into Chunk.Constants)
	PUSH_NULL                // Operand: none
	PUSH_BOOL                // Operand: bool

	LOAD_VAR // Operand: int (index into Chunk.Names) — push the variable's value
	SET_VAR  // Operand: int (index into Chunk.Names) — pop top of stack; reassign
	// if the name is already bound anywhere in scope, else declare it in the
	// current (innermost) scope. Mirrors vm.Environment's VariableDeclaration
	// rule: assign-if-exists, else declare (Runtime spec 2.2).

	POP // discard top of stack

	ADD
	SUB
	MUL
	DIV
	MOD
	NEG // unary minus
	EQ
	NEQ
	LT
	LTE
	GT
	GTE
	NOT

	JUMP          // Operand: int (absolute instruction index)
	JUMP_IF_FALSE // Operand: int — pop top; jump if false
	JUMP_IF_TRUE  // Operand: int — pop top; jump if true

	CALL   // Operand: *CallOperand — pop Argc args, call the named function
	RETURN // pop top (or push null if nothing to pop) and stop the current chunk

	PRINT        // pop top, print with a trailing newline
	PRINT_INLINE // pop top, print without a trailing newline

	TO_DISPLAY_STRING // pop a value, push its Haja display string (참/거짓,
	// 비어있음, "[a, b, c]" for lists, ...) — same formatting PRINT uses.
	// Used to compile TemplateLiteral's {expr} segments so a chain of ADDs
	// over already-string operands does real concatenation.

	NEW_LIST  // Operand: int (element count) — pop that many values, push a list
	LIST_PUSH // Operand: string ("front"/"back") — pop a value, pop a list,
	// push the resulting (longer) list (Runtime spec 5.1). Purely a value
	// operation — the compiler is responsible for pushing the list to start
	// from and writing the result back to wherever it came from (a plain
	// variable or an object field), same as any other expression/assignment,
	// which is what lets this work on an object's list field, not just a
	// bare variable.
	LIST_POP // Operand: string ("front"/"back") — pop a list, push the
	// resulting (shorter) list, then push its removed front/back element on
	// top (IndexOutOfBoundsError if empty). Same "compiler owns read/write
	// back" split as LIST_PUSH.
	LIST_CLEAR // no operand — pop a value, push an empty list if it was a
	// list (TypeError otherwise). Compiles 하자/카나데's one mutating list
	// pseudo-method (비우기/空にする — see bcLang.listClearMethod), same
	// "compiler owns read/write back" split as LIST_PUSH/LIST_POP. Handled
	// as its own opcode (not through GET_MEMBER/CALL_METHOD's generic bound-
	// method dispatch) specifically because it's the one list operation that
	// needs to write its result back to wherever the list came from, which
	// only the compiler — not a runtime value with no memory of its own
	// origin — can do.
	GET_INDEX  // pop index, pop list; push list[index] (1-based, IndexOutOfBoundsError)
	GET_LENGTH // pop list; push float64(len(list))

	NEW_OBJECT // Operand: *NewObjectOperand — pop Argc args, construct+init+
	// construct an instance of the named class, push it

	GET_MEMBER // Operand: *MemberOperand — pop an object/list/string/class-ref/
	// super-ref; push the named field's value, or a bound (instance/static)
	// method value when the operand says the property was a <메서드> reference,
	// or (falling back to IndexChunk) a list/string index result — whichever
	// the popped value's runtime type calls for. One opcode because a
	// MemberExpression's meaning is only resolvable once you know what its
	// Object evaluated to (exactly how vm/eval_expr.go's MemberExpression
	// case works).
	SET_MEMBER // Operand: *MemberOperand — pop an object/list/class-ref, pop a
	// value below it; assign the named field (running a declared setter
	// instead, if any) or list index in place.

	CALL_METHOD // Operand: int (Argc) — pop Argc args, pop a bound (instance or
	// static) method value produced by GET_MEMBER, and call it. A separate
	// opcode from CALL because the compiler already knows statically whether
	// a CallExpression's callee is a plain <함수>(...) or an X의 <메서드>(...)
	// — resolving that at compile time instead of branching on it at runtime
	// on every call is strictly cheaper.

	LOAD_SELF // push the current frame's bound 나. Operand: none (a bare 나
	// keyword — ReferenceError if none is bound) or int (index into
	// Chunk.Names: a quoted '나' — if none is bound, look the name up as an
	// ordinary variable instead, like vm/eval_expr.go's Identifier case)
	LOAD_SUPER // push a super-reference wrapping the current frame's 나,
	// only meaningful as GET_MEMBER's object (for a 부모의 <메서드>(...) call)
	LOAD_STATIC_CLASS // push a class-ref for the current frame's 우리 class
	// (ReferenceError if not executing inside a class method/static method;
	// same optional int operand as LOAD_SELF for a quoted '우리')
	PUSH_CLASS_REF // Operand: int (index into Chunk.Names) — push a class-ref
	// for a compile-time-known class name (a bare [클래스] TypeReference)

	INSTANCE_OF // pop a class-ref, pop a value; push whether the value is an
	// instance of that class or (transitively, via BaseClass) a subclass of
	// it. Interface membership is intentionally NOT checked here: this
	// replicates classIsOrExtends exactly, including its existing gap (일종이다 against an interface name is always false).

	SET_STATIC_FIELD // Operand: *StaticFieldOperand — pop a value, store it as
	// the named class's static field. Compiled inline into Main at the
	// ClassDeclaration statement's own source position (not hoisted), so
	// static initializers run in program order exactly like
	// vm.Interpreter.Run()'s step 3 does for the tree-walker.

	THROW // pop a value and raise it: search the current exec() call's active
	// try handlers (innermost first) for a match — an untyped handler always
	// matches, a typed one only if the value is a class instance that is-or-
	// extends it — running each level's finally along the way. Unwinds to a
	// matched handler, or (if none matches) propagates as a Go error exactly
	// like a runtime error from any other opcode.

	TRY_PUSH // Operand: *TryOperand — push a handler frame (this instruction's
	// Catches, FinallyChunk, and the current operand stack depth) onto this
	// exec() call's try stack. Every runtime error from here until the
	// matching TRY_POP — not just THROW, any opcode's error — is subject to
	// this handler.
	TRY_POP // pop the try stack: the protected block finished without error,
	// so THROW/runtime errors past this point are no longer this handler's
	// concern. Also emitted (possibly several times, once per enclosing try)
	// ahead of a RETURN or a loop-exiting JUMP compiled inside a protected
	// block, since 부모의 ReturnStatement/BreakStatement bypass catch
	// matching entirely but must still leave the try stack accurate for
	// whatever runs after.
	RUN_FINALLY // Operand: *Chunk — run a 마무리는 항상 block (in its own
	// nested exec() call, sharing this frame's locals) inline. Compiled at
	// every place control can leave a try statement — normal completion, the
	// end of a matched catch handler, and ahead of a RETURN/break compiled
	// inside the try or its catch handlers — because Runtime spec 4.2
	// requires 마무리는 항상 to run no matter how the try statement is left,
	// with no way to duplicate that as a single fallthrough path once RETURN
	// is a raw Go return rather than a catchable signal.

	IMPORT_NATIVE // Operand: *ImportOperand — look up the named builtin
	// module's function (registered by the embedding program, e.g.
	// bcstdlib's [수학]) and make it callable by its target name from here
	// on (ImportError if the module or that name isn't registered). Only
	// [모듈]에서 <이름>을 가져오자 needs this — a local file import
	// ("파일"에서 ...) is resolved entirely at compile time by merging the
	// imported file's declaration into this Program, since unlike a native
	// module it's just more compiled Haja code (see compileLocalImport).

	HALT

	// Appended after HALT, not inserted before it: HALT is emitted into every
	// serialized program and encoded by value, so already-compiled .hn files
	// must keep meaning what they meant.

	NEW_DICT // Operand: int (pair count) — pop 2*N values (key, value, key,
	// value, ... in source order), push a dict. A later duplicate key wins,
	// like vm's DictLiteral.
	SET_CONST // Operand: int (index into Chunk.Names) — SET_VAR for a
	// `고정하자`/`固定しよう` declaration: same assign-if-bound-else-declare
	// rule, but a fresh declaration is marked constant, and assigning to any
	// name marked constant (by SET_VAR or SET_CONST) is a
	// ConstantAssignmentError — mirrors vm.Environment.DeclareConst/Assign.

	INPUT // Operand: string (the `[타입]으로` type name, "" for the default) —
	// read one line from the VM's input source, convert it (conv.ParseInput,
	// shared with the tree-walker) and push the value. Appended after SET_CONST
	// so already-compiled .hn files keep their meaning.

	SET_VAR_TYPED // Operand: *TypedSetOperand — SET_VAR/SET_CONST for a declaration
	// that wrote a [타입]: the value is checked against it (and against any type an
	// earlier declaration gave the name), and a new variable remembers the type so
	// later assignments keep honoring it (Runtime spec 2.2).

	PUSH_FUNC_REF // Operand: int (index into Chunk.Names) — push the function called
	// that name as a value, so it can be passed to a native library, which calls
	// it back later. Resolved by name at call time; appended after SET_VAR_TYPED so
	// already-compiled .hn files keep their meaning.

	CHECK_BOOL // No operand — the value on top of the stack must be true or false,
	// else ConditionNotBoolean; it stays on the stack. Ends the right side of
	// 그리고/또는, whose value is the result and so is not tested by a jump.

	PUSH_SCOPE // Operand: int (index into Chunk.Names) — open a scope: the hidden variable
	// of that name remembers how many variables the frame holds now, so POP_SCOPE
	// can drop everything declared after it. A loop body and a catch handler are
	// scopes, as in the tree-walker's per-iteration/handler Environment.
	POP_SCOPE // Operand: int (index into Chunk.Names) — drop every variable declared
	// since the PUSH_SCOPE of that name (the marker included). A POP_SCOPE of an
	// outer marker also drops the inner ones, so a `반복을 끝내자` needs only its own
	// loop's marker. Appended after CHECK_BOOL so .hn files keep their meaning.

	TO_ITERABLE // No operand — the value on top of the stack becomes what a
	// 마다 반복하자 walks: a list stays, a string becomes the list of its characters
	// (one-character strings), anything else raises NotIterable. Appended after POP_SCOPE.

	INIT_MODULE // Operand: *ModuleInit — run a module's top-level code, once: the first
	// INIT_MODULE of a name runs it in the module's own frame (where its variables
	// live and its functions look for theirs); later ones do nothing. Appended after TO_ITERABLE.
)

// ModuleInit is INIT_MODULE's operand: the module's name and its top-level code.
type ModuleInit struct {
	Name string
	Body *Chunk
}

// TypedSetOperand is SET_VAR_TYPED's operand.
type TypedSetOperand struct {
	NameIndex int
	Type      string
	Const     bool // a 고정하자/固定しよう declaration
}

// CallOperand is CALL's operand: which function (by name pool index) to
// invoke, and how many arguments the compiler pushed for it.
type CallOperand struct {
	NameIndex int
	Argc      int
}

// NewObjectOperand is NEW_OBJECT's operand.
type NewObjectOperand struct {
	ClassNameIndex int
	Argc           int
}

// MemberOperand is GET_MEMBER/SET_MEMBER's operand. PropNameIndex is always
// interned (possibly to ""), used when the runtime object turns out to be a
// class instance/class-ref (field or method name). IndexChunk, when
// non-nil, is a tiny self-contained chunk (ending in RETURN, like a
// parameter default) that evaluates the original Property expression —
// used when the runtime object turns out to be a list or string instead,
// exactly mirroring vm/eval_expr.go evaluating Property as an index
// expression in that branch. IsFunc marks a <메서드> property (GET_MEMBER
// only — SET_MEMBER targets are never method-shaped).
type MemberOperand struct {
	PropNameIndex int
	IsFunc        bool
	IndexChunk    *Chunk
}

// StaticFieldOperand is SET_STATIC_FIELD's operand.
type StaticFieldOperand struct {
	ClassNameIndex int
	FieldNameIndex int
}

// CatchInfo is one 오류가 발생했다면 clause compiled into TRY_PUSH's operand.
// TypeName is "" for an untyped (catch-all) handler. HandlerPc is where its
// body starts — the handler's own first instruction is a SET_VAR that binds
// whatever raise() pushed for it, so no separate "param name" needs to
// travel with CatchInfo.
type CatchInfo struct {
	TypeName  string
	HandlerPc int
}

// TryOperand is TRY_PUSH's operand.
type TryOperand struct {
	Catches      []CatchInfo
	FinallyChunk *Chunk // nil if the try statement has no 마무리는 항상
}

// ImportOperand is IMPORT_NATIVE's operand.
type ImportOperand struct {
	ModuleNameIndex int
	TargetNameIndex int // name to look up in the native module
	BindNameIndex   int // name to register the found function under — same
	// as TargetNameIndex unless the source used <이름>을 <별칭>으로 가져오자
	All bool // [모듈]에서 전부 가져오자: bind every function of the module
	// Library: the function comes from the package's native library
	// (packages/<모듈>/native/…), not from a module registered by the embedder.
	Library bool
}
