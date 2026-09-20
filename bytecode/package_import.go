package bytecode

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/soumt-r/hana/ast"
	haja_lexer "github.com/soumt-r/hana/lexer/haja"
	kanade_lexer "github.com/soumt-r/hana/lexer/kanade"
	haja_parser "github.com/soumt-r/hana/parser/haja"
	kanade_parser "github.com/soumt-r/hana/parser/kanade"
	"github.com/soumt-r/hana/pkg"
)

// nativeBind is one name a `[모듈]에서 …` import binds while the program runs
// (IMPORT_NATIVE): a function of a native module registered by the embedding
// program, or of a package's native library. Everything else a package import
// brings in is compiled code, merged into the Program at compile time.
type nativeBind struct {
	module, target, bind string
	library              bool // target is in the package's native library (a <네이티브_…> import)
	all                  bool // bind every function of a core native module
}

// resolveBuiltinImport handles `[모듈]에서 …`: <네이티브_…> items come from the
// package's native library; the other items are compiled from the package's
// entry point when it has one for this language (its own native binds are
// hoisted to the import's position), else they are functions of a core native
// module like [수학].
func (c *Compiler) resolveBuiltinImport(s *ast.ImportStatement) {
	if c.pkgBinds == nil {
		c.pkgBinds = map[*ast.ImportStatement][]nativeBind{}
	}
	var rest []ast.ImportItem
	for _, item := range s.Items {
		if strings.HasPrefix(item.Name, c.lang.nativePrefix) {
			c.pkgBinds[s] = append(c.pkgBinds[s], nativeBind{
				module: s.Module, target: strings.TrimPrefix(item.Name, c.lang.nativePrefix),
				bind: item.BindName(), library: true,
			})
			continue
		}
		rest = append(rest, item)
	}
	if !s.All && len(rest) == 0 {
		return
	}
	if binds, ok := c.compilePackageImport(s, rest); ok {
		c.pkgBinds[s] = append(c.pkgBinds[s], binds...)
		return
	}
	if s.All {
		c.pkgBinds[s] = append(c.pkgBinds[s], nativeBind{module: s.Module, all: true})
	}
	for _, item := range rest {
		c.pkgBinds[s] = append(c.pkgBinds[s], nativeBind{module: s.Module, target: item.Name, bind: item.BindName()})
	}
}

// compilePackageImport compiles packages/<모듈>'s entry point for this
// language, like compileLocalImport does for a file, and merges the requested
// declarations. It returns the entry's own run-time binds (its native library
// functions and core modules), which must run before the importer uses the
// merged functions. ok is false when the package has no entry point here.
func (c *Compiler) compilePackageImport(s *ast.ImportStatement, items []ast.ImportItem) (binds []nativeBind, ok bool) {
	pkgDir := pkg.Dir(s.Module)
	manifest, err := pkg.Load(pkgDir)
	if err != nil {
		c.errorf("ImportError: The manifest of package '%s' is not valid: %v", s.Module, err)
		return nil, true
	}
	entry := filepath.Join(pkgDir, manifest.EntryPath(c.lang.name, c.lang.ext))
	content, err := os.ReadFile(entry)
	if err != nil {
		return nil, false
	}

	if c.importStack == nil {
		c.importStack = map[string]bool{}
	}
	key := "package:" + s.Module
	if c.importStack[key] {
		c.errorf("ImportError: circular import involving package '%s'.", s.Module)
		return nil, true
	}

	var prog *ast.Program
	var parseErrors []string
	if c.lang.name == "kanade" {
		p := kanade_parser.New(kanade_lexer.New(string(content)))
		prog, parseErrors = p.ParseProgram(), p.Errors()
	} else {
		p := haja_parser.New(haja_lexer.New(string(content)))
		prog, parseErrors = p.ParseProgram(), p.Errors()
	}
	if len(parseErrors) > 0 {
		c.errorf("ImportError: Syntax error in file '%s' of package '%s'.", entry, s.Module)
		return nil, true
	}

	c.importStack[key] = true
	defer delete(c.importStack, key)

	sub := newCompiler(c.lang)
	sub.importStack = c.importStack
	for _, stmt := range prog.Statements {
		switch st := stmt.(type) {
		case *ast.FunctionDeclaration:
			sub.compileFunction(st)
		case *ast.InterfaceDeclaration:
			sub.registerInterface(st)
		case *ast.ClassDeclaration:
			sub.registerClass(st)
		case *ast.ImportStatement:
			if st.IsBuiltin {
				sub.resolveBuiltinImport(st)
			} else {
				sub.compileLocalImport(st)
			}
		}
	}
	c.errors = append(c.errors, sub.errors...)
	binds = c.exportModule(s.Module, prog, sub, items)
	c.mergeImported(s.Module, s.All, items, prog, sub)
	return binds, true
}

// mergeImported copies what an import asked for out of a compiled module into
// this Program: with `전부`, every function/class/interface the module declares
// itself (not what it imported); then each named item under its bind name.
func (c *Compiler) mergeImported(source string, all bool, items []ast.ImportItem, prog *ast.Program, sub *Compiler) {
	if all {
		for _, stmt := range prog.Statements {
			switch d := stmt.(type) {
			case *ast.FunctionDeclaration:
				c.program.Functions[d.Name.Value] = sub.program.Functions[d.Name.Value]
			case *ast.ClassDeclaration:
				c.program.Classes[d.Name.Name] = sub.program.Classes[d.Name.Name]
			case *ast.InterfaceDeclaration:
				c.program.Interfaces[d.Name.Name] = sub.program.Interfaces[d.Name.Name]
			}
		}
	}
	for _, item := range items {
		bindName := item.BindName()
		if fn, ok := sub.program.Functions[item.Name]; ok {
			c.program.Functions[bindName] = fn
		} else if cls, ok := sub.program.Classes[item.Name]; ok {
			mine := sub.ownerOrElse(item.Name, source)
			if _, taken := c.program.Classes[bindName]; taken && item.As != "" && c.classOwner[bindName] != mine {
				c.conflict(source, bindName, c.classOwner[bindName])
			}
			c.program.Classes[bindName] = cls
			c.rememberOwner(bindName, mine)
		} else if iface, ok := sub.program.Interfaces[item.Name]; ok {
			c.program.Interfaces[bindName] = iface
		} else {
			c.errorf("ImportError: Target '%s' not found in '%s'.", item.Name, source)
		}
	}
}

// LibraryModules lists the packages whose native library the compiled program
// loads at run time. Everything else a package brings is compiled into the
// program, so these are the only package files a program still needs beside it.
func (c *Compiler) LibraryModules() []string { return c.libraries }

func (c *Compiler) noteLibrary(module string) {
	for _, m := range c.libraries {
		if m == module {
			return
		}
	}
	c.libraries = append(c.libraries, module)
}

// emitBuiltinImport compiles `[모듈]에서 …` into IMPORT_NATIVE instructions, at
// the import's position in Main.
func (c *Compiler) emitBuiltinImport(chunk *Chunk, s *ast.ImportStatement) {
	if _, resolved := c.pkgBinds[s]; !resolved {
		c.resolveBuiltinImport(s)
	}
	for _, b := range c.pkgBinds[s] {
		if b.library {
			c.noteLibrary(b.module)
		}
		op := &ImportOperand{ModuleNameIndex: chunk.addName(b.module), All: b.all, Library: b.library}
		if !b.all {
			op.TargetNameIndex = chunk.addName(b.target)
			op.BindNameIndex = chunk.addName(b.bind)
		}
		chunk.emit(IMPORT_NATIVE, op)
	}
}
