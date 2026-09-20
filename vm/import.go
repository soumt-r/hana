package vm

import (
	"github.com/soumt-r/hana/errs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/soumt-r/hana/ast"
	"github.com/soumt-r/hana/native"
	"github.com/soumt-r/hana/pkg"
)

// executeImport implements ImportStatement: it loads a builtin package
// (`[모듈]에서 ... 가져오자`) or a local user file (`"파일"에서 ... 가져오자`)
// once and binds each requested item (or, for `전부`, everything the module
// declares) into env.
func (i *Interpreter) executeImport(s *ast.ImportStatement, env *Environment) (interface{}, error) {
	if s.IsBuiltin {
		return i.importBuiltin(s, env)
	}
	return i.importLocalFile(s, env)
}

// importBuiltin resolves `[모듈]에서 <타겟>을 가져오자`. It tries, in order:
//  1. a native library of the package (`<네이티브_이름>` targets, or plain names
//     when the package has no source entry point)
//  2. a source package with an entry point for the current language
//     (packages/<모듈>/<haja|kanade>/index.<hj|knd>, or what its hana.pkg.json says)
//  3. a core engine native module (예: [수학])
//
// A native library's functions can't be listed, so `전부` has nothing to bind
// from one itself; it works through the package's source entry point or a core
// module.
func (i *Interpreter) importBuiltin(s *ast.ImportStatement, env *Environment) (interface{}, error) {
	pkgDir := pkg.Dir(s.Module)
	if pkg.IsPackagePath(s.Module) {
		if _, err := os.Stat(pkgDir); err != nil {
			return nil, errs.New(errs.ImportPackageNotInstalled, s.Module)
		}
	}
	manifest, err := pkg.Load(pkgDir)
	if err != nil {
		return nil, errs.New(errs.ImportManifestInvalid, s.Module, err.Error())
	}

	var lib *native.Library
	loadLib := func() (*native.Library, error) {
		if lib != nil {
			return lib, nil
		}
		var err error
		lib, err = native.OpenPackage(pkgDir, s.Module, manifest)
		return lib, err
	}

	// 1. 네이티브 직접 바인딩 요청(소스 진입점 무시)은 여기서 끝내고, 나머지는
	// 소스 진입점이나 코어 모듈이 풀 항목으로 남긴다.
	var remaining []ast.ImportItem
	for _, item := range s.Items {
		if strings.HasPrefix(item.Name, i.Config.NativePrefix) {
			target := item.Name[len(i.Config.NativePrefix):]
			l, err := loadLib()
			if err != nil {
				return nil, err
			}
			fn, ok := l.Lookup(target)
			if !ok {
				return nil, errs.New(errs.ImportNativeFunctionNotFound, s.Module, target)
			}
			env.Declare(item.BindName(), nativeBuiltin(item.Name, fn))
			continue
		}
		remaining = append(remaining, item)
	}
	if !s.All && len(remaining) == 0 {
		return nil, nil
	}

	// 2. 언어별 진입점이 있는 소스 패키지
	entryPath := i.entryPath(pkgDir, manifest, i.Config)
	if _, err := os.Stat(entryPath); err == nil {
		content, err := ioutil.ReadFile(entryPath)
		if err != nil {
			return nil, errs.New(errs.ImportFileNotFound, entryPath)
		}
		prog, parseErrs := i.Config.ParseProgram(string(content))
		if len(parseErrs) > 0 {
			return nil, errs.New(errs.ImportPackageSyntax, s.Module, entryPath)
		}
		sub := i.newSubInterpreter(prog, i.Config)
		if err := sub.Run(); err != nil {
			return nil, err
		}
		return nil, i.bindImports(sub, prog, s, remaining, s.Module, env)
	}
	// 패키지 폴더는 있는데 이 언어의 진입점만 없는 경우: 다른 언어용 진입점이 있는지 확인한다.
	for _, other := range allConfigs {
		if other.Name == i.Config.Name {
			continue
		}
		if _, err := os.Stat(i.entryPath(pkgDir, manifest, other)); err == nil {
			return nil, errs.New(errs.ImportUnsupportedLocale, s.Module, i.Config.Name)
		}
	}

	// 3. 소스 진입점이 없으면 네이티브 라이브러리가 그대로 내보낸 이름과 코어 엔진 내장 패키지에서 찾는다.
	mod, hasCore := i.NativeModules[s.Module]
	if s.All && hasCore {
		for name, fn := range mod {
			env.Declare(name, fn)
		}
	}
	for _, item := range remaining {
		if l, err := loadLib(); err == nil {
			if fn, ok := l.Lookup(item.Name); ok {
				env.Declare(item.BindName(), nativeBuiltin(item.Name, fn))
				continue
			}
		}
		if fn, ok := mod[item.Name]; hasCore && ok {
			env.Declare(item.BindName(), fn)
			continue
		}
		return nil, errs.New(errs.ImportPackageNotFound, s.Module)
	}
	if s.All && !hasCore {
		return nil, errs.New(errs.ImportPackageNotFound, s.Module)
	}
	return nil, nil
}

// nativeBuiltin wraps a native library function as a callable value.
func nativeBuiltin(name string, fn native.Func) *BuiltinFunction {
	return &BuiltinFunction{
		Name: name,
		Fn: func(interp *Interpreter, env *Environment, args ...interface{}) (interface{}, error) {
			return fn(interp, args)
		},
	}
}

// importLocalFile resolves `"파일"에서 <타겟>을 가져오자`: parses and runs the
// local source file as its own sub-interpreter — in the file's own language,
// chosen by its extension (.hj / .knd) — then binds the requested items.
func (i *Interpreter) importLocalFile(s *ast.ImportStatement, env *Environment) (interface{}, error) {
	filename := s.Module
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, errs.New(errs.ImportFileNotFound, filename)
	}

	cfg := ConfigForFile(filename)
	prog, parseErrs := cfg.ParseProgram(string(content))
	if len(parseErrs) > 0 {
		return nil, errs.New(errs.ImportFileSyntax, filename)
	}

	sub := i.newSubInterpreter(prog, cfg)
	if err := sub.Run(); err != nil {
		return nil, err
	}
	return nil, i.bindImports(sub, prog, s, s.Items, filename, env)
}

// bindImports binds what the import asked for out of a module that has run:
// every item, or with `전부` everything the module declares.
func (i *Interpreter) bindImports(sub *Interpreter, prog *ast.Program, s *ast.ImportStatement, items []ast.ImportItem, source string, env *Environment) error {
	i.injectNatives(sub, env)
	if s.All {
		i.bindAllDeclarations(sub, prog, env)
	}
	for _, item := range items {
		if err := i.bindImportTarget(sub, prog, item.Name, item.BindName(), source, env); err != nil {
			return err
		}
	}
	return nil
}

// injectNatives copies the native functions a package loaded into the caller's
// environment (dynamic scoping).
func (i *Interpreter) injectNatives(sub *Interpreter, env *Environment) {
	for k, v := range sub.GlobalEnv.GetAll() {
		if strings.HasPrefix(k, i.Config.NativePrefix) {
			env.Declare(k, v)
		}
	}
}

// bindAllDeclarations is `[모듈]에서 전부 가져오자`: every function, class and
// interface the module's own source declares (its imports and builtins are not
// re-exported), each under its own name.
func (i *Interpreter) bindAllDeclarations(sub *Interpreter, prog *ast.Program, env *Environment) {
	for _, stmt := range prog.Statements {
		switch d := stmt.(type) {
		case *ast.FunctionDeclaration:
			env.Declare(d.Name.Value, d)
		case *ast.ClassDeclaration:
			if cls, ok := sub.Classes[d.Name.Name]; ok {
				i.registerClass(d.Name.Name, cls)
			}
		case *ast.InterfaceDeclaration:
			if iface, ok := sub.Interfaces[d.Name.Name]; ok {
				i.Interfaces[d.Name.Name] = iface
			}
		}
	}
}

// entryPath is where a package keeps its entry point for one language: what
// its manifest says, else the conventional <언어>/index.<확장자>.
func (i *Interpreter) entryPath(pkgDir string, manifest *pkg.Manifest, cfg LangConfig) string {
	return filepath.Join(pkgDir, manifest.EntryPath(cfg.Name, cfg.SourceExt))
}

// bindImportTarget resolves target after running an imported package/file's
// sub-interpreter: first as a declared global (variable, builtin, or
// function value), then as a class, then an interface, then a bare top-level
// function declaration — and binds it under bindName (== target, unless the
// import used `<이름>을 <별칭>으로 가져오자`). source is used only for the
// error message (the module name for builtin packages, or the file path for
// local imports).
func (i *Interpreter) bindImportTarget(sub *Interpreter, prog *ast.Program, target, bindName, source string, env *Environment) error {
	if val, exists := sub.GlobalEnv.Get(target); exists {
		env.Declare(bindName, val)
		return nil
	}
	if cls, exists := sub.Classes[target]; exists {
		i.registerClass(bindName, cls)
		return nil
	}
	if iface, exists := sub.Interfaces[target]; exists {
		i.Interfaces[bindName] = iface
		return nil
	}
	for _, stmt := range prog.Statements {
		if f, ok := stmt.(*ast.FunctionDeclaration); ok && f.Name.Value == target {
			env.Declare(bindName, f)
			return nil
		}
	}
	return errs.New(errs.ImportTargetNotFound, source, target)
}
