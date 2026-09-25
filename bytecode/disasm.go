package bytecode

import (
	"fmt"
	"strings"
)

var opcodeNames = map[Opcode]string{
	PUSH_CONST:        "PUSH_CONST",
	PUSH_NULL:         "PUSH_NULL",
	PUSH_BOOL:         "PUSH_BOOL",
	LOAD_VAR:          "LOAD_VAR",
	SET_VAR:           "SET_VAR",
	POP:               "POP",
	ADD:               "ADD",
	SUB:               "SUB",
	MUL:               "MUL",
	DIV:               "DIV",
	MOD:               "MOD",
	NEG:               "NEG",
	EQ:                "EQ",
	NEQ:               "NEQ",
	LT:                "LT",
	LTE:               "LTE",
	GT:                "GT",
	GTE:               "GTE",
	NOT:               "NOT",
	JUMP:              "JUMP",
	JUMP_IF_FALSE:     "JUMP_IF_FALSE",
	JUMP_IF_TRUE:      "JUMP_IF_TRUE",
	CALL:              "CALL",
	RETURN:            "RETURN",
	PRINT:             "PRINT",
	PRINT_INLINE:      "PRINT_INLINE",
	TO_DISPLAY_STRING: "TO_DISPLAY_STRING",
	NEW_LIST:          "NEW_LIST",
	LIST_PUSH:         "LIST_PUSH",
	LIST_POP:          "LIST_POP",
	LIST_CLEAR:        "LIST_CLEAR",
	GET_INDEX:         "GET_INDEX",
	GET_LENGTH:        "GET_LENGTH",
	NEW_OBJECT:        "NEW_OBJECT",
	GET_MEMBER:        "GET_MEMBER",
	SET_MEMBER:        "SET_MEMBER",
	CALL_METHOD:       "CALL_METHOD",
	LOAD_SELF:         "LOAD_SELF",
	LOAD_SUPER:        "LOAD_SUPER",
	LOAD_STATIC_CLASS: "LOAD_STATIC_CLASS",
	PUSH_CLASS_REF:    "PUSH_CLASS_REF",
	INSTANCE_OF:       "INSTANCE_OF",
	SET_STATIC_FIELD:  "SET_STATIC_FIELD",
	THROW:             "THROW",
	TRY_PUSH:          "TRY_PUSH",
	TRY_POP:           "TRY_POP",
	RUN_FINALLY:       "RUN_FINALLY",
	IMPORT_NATIVE:     "IMPORT_NATIVE",
	HALT:              "HALT",
	NEW_DICT:          "NEW_DICT",
	SET_CONST:         "SET_CONST",
	INPUT:             "INPUT",
	SET_VAR_TYPED:     "SET_VAR_TYPED",
	PUSH_FUNC_REF:     "PUSH_FUNC_REF",
	CHECK_BOOL:        "CHECK_BOOL",
	PUSH_SCOPE:        "PUSH_SCOPE",
	POP_SCOPE:         "POP_SCOPE",
	TO_ITERABLE:       "TO_ITERABLE",
	INIT_MODULE:       "INIT_MODULE",
	SET_LIST_VAR:      "SET_LIST_VAR",
	CHECK_LIST_FIELD:  "CHECK_LIST_FIELD",
	CHECK_CONST_VAR:   "CHECK_CONST_VAR",
	BIN:               "BIN",
	FOR_STEP:          "FOR_STEP",
	DECLARE_VAR:       "DECLARE_VAR",
}

// binText renders a BIN's operand (also the two halves of a FOR_STEP's).
func binText(chunk *Chunk, b *BinOperand) string {
	arg := func(a BinArg) string {
		switch a.Kind {
		case ArgVar:
			return "var " + chunk.Names[a.Index]
		case ArgConst:
			return fmt.Sprintf("const %#v", chunk.Constants[a.Index])
		}
		return "stack"
	}
	out := fmt.Sprintf("%s %s, %s", opcodeNames[b.Op], arg(b.L), arg(b.R))
	if b.Set >= 0 {
		out += " -> " + chunk.Names[b.Set]
	}
	if b.Jump >= 0 {
		out += fmt.Sprintf(" ; if false -> %d", b.Jump)
	}
	return out
}

// Disassemble renders chunk as human-readable text: one line per
// instruction, its index, opcode name, and operand (resolved against the
// constant/name pools where that makes the output more readable).
func Disassemble(name string, chunk *Chunk) string {
	var b strings.Builder
	fmt.Fprintf(&b, "== %s ==\n", name)
	for i, instr := range chunk.Instructions {
		fmt.Fprintf(&b, "%4d  %-18s %s\n", i, opcodeNames[instr.Op], operandString(chunk, instr))
	}
	return b.String()
}

// DisassembleProgram disassembles every function plus the top-level Main
// chunk, in a stable (name-sorted) order for functions.
func DisassembleProgram(prog *Program) string {
	var b strings.Builder
	names := make([]string, 0, len(prog.Functions))
	for name := range prog.Functions {
		names = append(names, name)
	}
	for i := 0; i < len(names); i++ {
		for j := i + 1; j < len(names); j++ {
			if names[j] < names[i] {
				names[i], names[j] = names[j], names[i]
			}
		}
	}
	for _, name := range names {
		fn := prog.Functions[name]
		b.WriteString(Disassemble("함수 "+name, fn.BodyChunk))
		for _, p := range fn.Params {
			if p.Default != nil {
				b.WriteString(Disassemble("함수 "+name+" 기본값("+p.Name+")", p.Default))
			}
		}
	}

	clsNames := make([]string, 0, len(prog.Classes))
	for name := range prog.Classes {
		clsNames = append(clsNames, name)
	}
	for i := 0; i < len(clsNames); i++ {
		for j := i + 1; j < len(clsNames); j++ {
			if clsNames[j] < clsNames[i] {
				clsNames[i], clsNames[j] = clsNames[j], clsNames[i]
			}
		}
	}
	for _, name := range clsNames {
		cls := prog.Classes[name]
		if cls.Constructor != nil {
			b.WriteString(Disassemble("클래스 "+name+" 생성자", cls.Constructor.BodyChunk))
		}
		for _, f := range cls.Fields {
			if f.Default != nil {
				b.WriteString(Disassemble("클래스 "+name+" 필드 기본값("+f.Name+")", f.Default))
			}
			if f.Setter != nil {
				b.WriteString(Disassemble("클래스 "+name+" 세터("+f.Name+")", f.Setter.Body))
			}
		}
		for mName, m := range cls.Methods {
			b.WriteString(Disassemble("클래스 "+name+" 메서드 "+mName, m.BodyChunk))
		}
		for mName, m := range cls.StaticMethods {
			b.WriteString(Disassemble("클래스 "+name+" 정적메서드 "+mName, m.BodyChunk))
		}
	}

	b.WriteString(Disassemble("main", prog.Main))
	return b.String()
}

func operandString(chunk *Chunk, instr Instruction) string {
	switch instr.Op {
	case PUSH_CONST:
		idx := instr.Operand.(int)
		return fmt.Sprintf("%d ; %#v", idx, chunk.Constants[idx])
	case LOAD_SELF, LOAD_STATIC_CLASS:
		if idx, ok := instr.Operand.(int); ok {
			return fmt.Sprintf("%d ; %s", idx, chunk.Names[idx])
		}
		return ""
	case LOAD_VAR, SET_VAR, SET_CONST, PUSH_FUNC_REF, PUSH_SCOPE, POP_SCOPE, DECLARE_VAR:
		idx := instr.Operand.(int)
		return fmt.Sprintf("%d ; %s", idx, chunk.Names[idx])
	case SET_VAR_TYPED:
		op := instr.Operand.(*TypedSetOperand)
		kind := ""
		if op.Const {
			kind = " const"
		}
		return fmt.Sprintf("%s : [%s]%s", chunk.Names[op.NameIndex], op.Type, kind)
	case BIN:
		return binText(chunk, instr.Operand.(*BinOperand))
	case FOR_STEP:
		op := instr.Operand.(*ForStepOperand)
		if op.Span {
			return fmt.Sprintf("%s ; then (end - counter) * step >= 0 ; if false -> %d, if true -> %d", binText(chunk, op.Inc), op.Test.Jump, op.Body)
		}
		return fmt.Sprintf("%s ; then %s, if true -> %d", binText(chunk, op.Inc), binText(chunk, op.Test), op.Body)
	case SET_LIST_VAR, CHECK_LIST_FIELD:
		op := instr.Operand.(*ListSetOperand)
		return fmt.Sprintf("%s ; change %d", chunk.Names[op.NameIndex], op.Change)
	case INIT_MODULE:
		return instr.Operand.(*ModuleInit).Name
	case PUSH_BOOL:
		return fmt.Sprintf("%v", instr.Operand)
	case JUMP, JUMP_IF_FALSE, JUMP_IF_TRUE:
		return fmt.Sprintf("-> %d", instr.Operand)
	case CALL:
		op := instr.Operand.(*CallOperand)
		return fmt.Sprintf("%s argc=%d", chunk.Names[op.NameIndex], op.Argc)
	case NEW_LIST, NEW_DICT, CALL_METHOD:
		return fmt.Sprintf("%d", instr.Operand)
	case PUSH_CLASS_REF:
		idx := instr.Operand.(int)
		return fmt.Sprintf("%d ; [%s]", idx, chunk.Names[idx])
	case NEW_OBJECT:
		op := instr.Operand.(*NewObjectOperand)
		return fmt.Sprintf("[%s] argc=%d", chunk.Names[op.ClassNameIndex], op.Argc)
	case GET_MEMBER, SET_MEMBER:
		op := instr.Operand.(*MemberOperand)
		name := chunk.Names[op.PropNameIndex]
		if op.IsFunc {
			return fmt.Sprintf("<%s>", name)
		}
		return name
	case SET_STATIC_FIELD:
		op := instr.Operand.(*StaticFieldOperand)
		return fmt.Sprintf("[%s].%s", chunk.Names[op.ClassNameIndex], chunk.Names[op.FieldNameIndex])
	case TRY_PUSH:
		op := instr.Operand.(*TryOperand)
		parts := make([]string, len(op.Catches))
		for i, c := range op.Catches {
			if c.TypeName == "" {
				parts[i] = fmt.Sprintf("* -> %d", c.HandlerPc)
			} else {
				parts[i] = fmt.Sprintf("[%s] -> %d", c.TypeName, c.HandlerPc)
			}
		}
		finally := "no"
		if op.FinallyChunk != nil {
			finally = "yes"
		}
		return fmt.Sprintf("catches=[%s] finally=%s", strings.Join(parts, ", "), finally)
	case RUN_FINALLY:
		return "(마무리는 항상 청크)"
	case IMPORT_NATIVE:
		op := instr.Operand.(*ImportOperand)
		if op.All {
			return fmt.Sprintf("[%s].*", chunk.Names[op.ModuleNameIndex])
		}
		target := chunk.Names[op.TargetNameIndex]
		bind := chunk.Names[op.BindNameIndex]
		if bind != target {
			return fmt.Sprintf("[%s].%s as %s", chunk.Names[op.ModuleNameIndex], target, bind)
		}
		return fmt.Sprintf("[%s].%s", chunk.Names[op.ModuleNameIndex], target)
	default:
		if instr.Operand == nil {
			return ""
		}
		return fmt.Sprintf("%v", instr.Operand)
	}
}
