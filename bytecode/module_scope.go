package bytecode

import (
	"strings"

	"github.com/soumt-r/hana/ast"
)

// Module scope: imported code keeps the scope of the module it came from.
//
// The Program has one flat table of functions, so the functions of a module go
// into it under a private name, <module>::<name>, and the calls inside the module's
// own code are rewritten to use it. The module's helper functions, and what the
// module imported itself, can then be called by the code that was imported without
// the importer bringing them in, and two modules may declare helpers of the same
// name. What the importer asked for is bound under its own name as well (an alias
// for the same function), which is how the importer calls it.
//
// Classes and interfaces stay under their own names (they are visible in messages
// and in type annotations); those the importer does not already have are added, so
// the module's functions can create them.

// scopeSep separates a module's name from a name in it; a name that has it is
// already private to some module.
const scopeSep = "::"

func isScoped(name string) bool { return strings.Contains(name, scopeSep) }

// exportModule moves a compiled module (sub, compiled from prog and imported as
// module) into this Program under private names. It returns the run-time binds of
// the module's own package imports (the native functions it brought in), renamed
// the same way, for the importer to run at the import's position.
func (c *Compiler) exportModule(module string, prog *ast.Program, sub *Compiler) (binds []nativeBind) {
	for _, stmt := range prog.Statements {
		if st, ok := stmt.(*ast.ImportStatement); ok && st.IsBuiltin {
			binds = append(binds, sub.pkgBinds[st]...)
		}
	}

	// The names that mean something in the module: what it declares, what it
	// imported, and the native functions its imports bind.
	scope := map[string]string{}
	for name := range sub.program.Functions {
		if !isScoped(name) {
			scope[name] = module + scopeSep + name
		}
	}
	for i := range binds {
		if !binds[i].all && !isScoped(binds[i].bind) {
			private := module + scopeSep + binds[i].bind
			scope[binds[i].bind] = private
			binds[i].bind = private
		}
	}

	// Rewrite the calls in the module's own code (what it imported was rewritten
	// when that module was exported).
	for _, stmt := range prog.Statements {
		switch d := stmt.(type) {
		case *ast.FunctionDeclaration:
			rewriteFunction(sub.program.Functions[d.Name.Value], scope)
		case *ast.ClassDeclaration:
			rewriteClass(sub.program.Classes[d.Name.Name], scope)
		}
	}

	for name, fn := range sub.program.Functions {
		if !isScoped(name) {
			name = module + scopeSep + name
		}
		c.program.Functions[name] = fn
	}
	for name, cls := range sub.program.Classes {
		if _, exists := c.program.Classes[name]; !exists {
			c.program.Classes[name] = cls
		}
	}
	for name, iface := range sub.program.Interfaces {
		if _, exists := c.program.Interfaces[name]; !exists {
			c.program.Interfaces[name] = iface
		}
	}
	return binds
}

func rewriteFunction(fn *Function, scope map[string]string) {
	if fn == nil {
		return
	}
	rewriteChunk(fn.BodyChunk, scope)
	for _, p := range fn.Params {
		rewriteChunk(p.Default, scope)
	}
}

func rewriteClass(cls *ClassInfo, scope map[string]string) {
	if cls == nil {
		return
	}
	for _, fields := range [][]FieldInfo{cls.Fields, cls.StaticFields} {
		for _, f := range fields {
			rewriteChunk(f.Default, scope)
			rewriteChunk(f.Getter, scope)
			if f.Setter != nil {
				rewriteChunk(f.Setter.Body, scope)
			}
		}
	}
	for _, m := range cls.Methods {
		rewriteFunction(m, scope)
	}
	for _, m := range cls.StaticMethods {
		rewriteFunction(m, scope)
	}
	rewriteFunction(cls.Constructor, scope)
}

// rewriteChunk points the calls and function references in a chunk that name
// something in scope at its private name.
func rewriteChunk(chunk *Chunk, scope map[string]string) {
	if chunk == nil {
		return
	}
	for i := range chunk.Instructions {
		in := &chunk.Instructions[i]
		switch in.Op {
		case CALL:
			op := in.Operand.(*CallOperand)
			if private, ok := scope[chunk.Names[op.NameIndex]]; ok {
				op.NameIndex = chunk.addName(private)
			}
		case PUSH_FUNC_REF:
			if private, ok := scope[chunk.Names[in.Operand.(int)]]; ok {
				in.Operand = chunk.addName(private)
			}
		}
	}
}
