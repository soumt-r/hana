// Package errs is hana's shared runtime-error vocabulary. Both VMs (vm/ and
// bcvm/) and both native stdlibs throw *errs.Error values that say only
// *what* went wrong (a Code plus its arguments) — never a pre-rendered
// sentence. A per-language Catalog (ko.go/ja.go/en.go) holds the actual
// wording, and Localize turns an error into text at the few places a user
// actually sees it: the value a `발생했다면` handler catches, and the CLI's
// top-level error print. Throwing code therefore never needs to know which
// language the running script is written in.
//
// Wording rules (enforced by errs_test.go's tone lint):
//   - Korean is 해요체 (~해요/~예요/~어요), Japanese is です・ます調.
//   - The "<Kind>: " prefix (TypeError, ArgumentError, ...) stays English in
//     every language: it is the stable, greppable identifier of the error.
package errs

import (
	"errors"
	"fmt"
	"strings"
)

// Code identifies one kind of runtime error. Its text is "<Kind>.<Name>";
// Kind (the part before the dot) is the English prefix Localize prepends.
type Code string

// Kind is the language-neutral prefix shown before every message.
func (c Code) Kind() string {
	if i := strings.IndexByte(string(c), '.'); i >= 0 {
		return string(c[:i])
	}
	return string(c)
}

// Error is what runtime code throws: a Code and the values its catalog
// template interpolates.
type Error struct {
	Code Code
	Args []interface{}
}

// New builds a runtime error. Args must match the verbs in the Code's
// catalog templates (errs_test.go checks the counts agree across languages).
func New(code Code, args ...interface{}) *Error {
	return &Error{Code: code, Args: args}
}

// Error renders the English wording — a language-neutral fallback for logs
// and Go-level callers that never localize. The Kind prefix is identical in
// every language, so substring checks like Contains(err.Error(),
// "IndexOutOfBoundsError") keep working regardless of locale.
func (e *Error) Error() string {
	return Localize(English, e)
}

// Locale selects which Catalog Localize renders with.
type Locale int

const (
	English Locale = iota
	Korean
	Japanese
)

func (l Locale) catalog() Catalog {
	switch l {
	case Korean:
		return koCatalog
	case Japanese:
		return jaCatalog
	}
	return enCatalog
}

// Catalog maps a Code to its message template (without the Kind prefix).
type Catalog map[Code]string

// Localize renders err in loc. A *Error is looked up in loc's catalog (falling
// back to English, then to the bare Code, so a missing entry degrades instead
// of panicking — errs_test.go makes a missing entry a test failure anyway).
// Anything else (a user-thrown value's text, a standard error) passes through
// unchanged.
func Localize(loc Locale, err error) string {
	var e *Error
	if !errors.As(err, &e) {
		return err.Error()
	}
	tmpl, ok := loc.catalog()[e.Code]
	if !ok {
		tmpl, ok = enCatalog[e.Code]
	}
	if !ok {
		return string(e.Code)
	}
	return e.Code.Kind() + ": " + fmt.Sprintf(tmpl, e.Args...)
}

// Message is Localize without the "<Kind>: " prefix, for places (an editor's
// squiggle) where the kind is noise.
func Message(loc Locale, err error) string {
	var e *Error
	if !errors.As(err, &e) {
		return err.Error()
	}
	text := Localize(loc, e)
	return strings.TrimPrefix(text, e.Code.Kind()+": ")
}

// AccessViolation picks the Code for a member access refused by its
// modifier ("private" or "protected") and whether the member is a method or
// a field — four wordings, one call for the throwing VM.
func AccessViolation(access string, isMethod bool, name string) *Error {
	switch {
	case access == "protected" && isMethod:
		return New(ProtectedMethodAccess, name)
	case access == "protected":
		return New(ProtectedFieldAccess, name)
	case isMethod:
		return New(PrivateMethodAccess, name)
	}
	return New(PrivateFieldAccess, name)
}

// TypeNameOf names a runtime value's type in plain English for messages that
// need to mention it (e.g. MemberAccessUnsupported), without leaking Go's
// internal type names.
func TypeNameOf(v interface{}) string {
	switch v.(type) {
	case nil:
		return "null"
	case float64:
		return "number"
	case string:
		return "string"
	case bool:
		return "boolean"
	case []interface{}:
		return "list"
	case map[interface{}]interface{}:
		return "dict"
	}
	return "object"
}

// Every Code, one per distinct message. The tone/arity tests iterate
// allCodes, so a new Code must be added here and to all three catalogs.
const (
	// ReferenceError
	ThisNotBound        Code = "ReferenceError.ThisNotBound"
	SuperOutsideMethod  Code = "ReferenceError.SuperOutsideMethod"
	StaticOutsideMethod Code = "ReferenceError.StaticOutsideMethod"
	VariableNotFound    Code = "ReferenceError.VariableNotFound" // name
	ClassNotFound       Code = "ReferenceError.ClassNotFound"    // name

	// InstantiationError / ImmutableAssignmentError (Runtime spec 3.1, 5.4)
	InstantiateInterface Code = "InstantiationError.Interface"     // name
	InstantiateAbstract  Code = "InstantiationError.AbstractClass" // name
	StringIndexAssign    Code = "ImmutableAssignmentError.StringIndex"

	// MethodNotFoundError
	GlobalFunctionNotFound Code = "MethodNotFoundError.GlobalFunctionNotFound" // name
	StaticMethodNotFound   Code = "MethodNotFoundError.StaticMethodNotFound"   // name
	MethodNotFound         Code = "MethodNotFoundError.MethodNotFound"         // name

	// TypeError (declared types, Runtime spec 2.2)
	VariableTypeMismatch Code = "TypeError.VariableTypeMismatch" // name, declared type, actual type
	ArgumentTypeMismatch Code = "TypeError.ArgumentTypeMismatch" // parameter, declared type, actual type
	ReturnTypeMismatch   Code = "TypeError.ReturnTypeMismatch"   // function, declared type, actual type

	// Operators (Runtime spec 2.2 no implicit casting, 2.4 null safety)
	OperandTypeMismatch Code = "TypeError.OperandTypeMismatch"  // operator, left type, right type
	NullOperand         Code = "NullReferenceError.NullOperand" // operator
	UnknownOperator     Code = "TypeError.UnknownOperator"      // operator

	// Native standard library (json / random / datetime / regex)
	NativeArgString   Code = "TypeError.NativeArgString"   // position
	NativeArgNumber   Code = "TypeError.NativeArgNumber"   // position
	NativeArgInteger  Code = "TypeError.NativeArgInteger"  // position
	NativeArgList     Code = "TypeError.NativeArgList"     // position
	NativeArgDict     Code = "TypeError.NativeArgDict"     // position
	NativeDictStrings Code = "TypeError.NativeDictStrings" // position
	ArgCountRange     Code = "ArgumentError.ArgCountRange" // min, max
	JSONInvalid       Code = "ValueError.JSONInvalid"
	JSONUnsupported   Code = "ValueError.JSONUnsupported"
	CSVInvalid        Code = "ValueError.CSVInvalid"
	CSVUnsupported    Code = "ValueError.CSVUnsupported"
	CSVDelimiter      Code = "ValueError.CSVDelimiter"
	RegexInvalid      Code = "ValueError.RegexInvalid" // pattern
	RandomRange       Code = "ValueError.RandomRange"  // min, max
	RandomEmpty       Code = "ValueError.RandomEmpty"
	DateInvalid       Code = "ValueError.DateInvalid" // text

	// Native standard library, second batch (math / statistics / text / list / encoding)
	NativeListNumbers    Code = "TypeError.NativeListNumbers" // position
	NativeListStrings    Code = "TypeError.NativeListStrings" // position
	ListNotSortable      Code = "TypeError.ListNotSortable"
	CallbackNotBoolean   Code = "TypeError.CallbackNotBoolean"
	ConditionNotBoolean  Code = "TypeError.ConditionNotBoolean" // type name
	ListValueUnsupported Code = "TypeError.ListValueUnsupported"
	MathDomain           Code = "ValueError.MathDomain"
	StatsNotEnough       Code = "ValueError.StatsNotEnough"
	RangeStepZero        Code = "ValueError.RangeStepZero"
	ResultTooLarge       Code = "ValueError.ResultTooLarge"
	TextEmptyPart        Code = "ValueError.TextEmptyPart"
	PadFillLength        Code = "ValueError.PadFillLength"
	Base64Invalid        Code = "ValueError.Base64Invalid"
	URLDecodeInvalid     Code = "ValueError.URLDecodeInvalid"
	SleepRange           Code = "ValueError.SleepRange"
	FileNotFound         Code = "FileError.FileNotFound"     // path
	FileIsDirectory      Code = "FileError.FileIsDirectory"  // path
	FileAccessDenied     Code = "FileError.FileAccessDenied" // path
	FileFailed           Code = "FileError.FileFailed"       // path
	FileBlocked          Code = "FileError.FileBlocked"

	// NetworkError
	NetBlocked          Code = "NetworkError.NetBlocked"
	SocketBadHandle     Code = "NetworkError.SocketBadHandle"     // number
	SocketNotConnection Code = "NetworkError.SocketNotConnection" // (a listening socket)
	SocketNotListener   Code = "NetworkError.SocketNotListener"   // (a connected socket)
	SocketConnectFailed Code = "NetworkError.SocketConnectFailed" // address
	SocketListenFailed  Code = "NetworkError.SocketListenFailed"  // address
	SocketTimeout       Code = "NetworkError.SocketTimeout"
	SocketFailed        Code = "NetworkError.SocketFailed"
	SocketPort          Code = "NetworkError.SocketPort"
	HTTPBadURL          Code = "NetworkError.HTTPBadURL"    // url
	HTTPFailed          Code = "NetworkError.HTTPFailed"    // url
	HTTPTooLarge        Code = "NetworkError.HTTPTooLarge"  // url
	HTTPBadMethod       Code = "NetworkError.HTTPBadMethod" // method

	// RecursionError
	CallTooDeep Code = "RecursionError.CallTooDeep" // limit

	// SyntaxError (reported by the parser; see parser/hari Diagnostic)
	SyntaxUnexpectedToken Code = "SyntaxError.UnexpectedToken" // line, column, token
	SyntaxUnexpectedEnd   Code = "SyntaxError.UnexpectedEnd"   // line

	// TypeError
	NotCallable             Code = "TypeError.NotCallable"
	NotAList                Code = "TypeError.NotAList"
	NotIterable             Code = "TypeError.NotIterable"
	RangeMustBeNumbers      Code = "TypeError.RangeMustBeNumbers"
	ListIndexMustBeNumber   Code = "TypeError.ListIndexMustBeNumber"
	SuperMemberMustBeMethod Code = "TypeError.SuperMemberMustBeMethod"
	MethodArgMustBeNumber   Code = "TypeError.MethodArgMustBeNumber" // method name
	MethodArgMustBeString   Code = "TypeError.MethodArgMustBeString" // method name
	NotANumber              Code = "TypeError.NotANumber"
	InvalidOperands         Code = "TypeError.InvalidOperands"
	ListOnlyMethod          Code = "TypeError.ListOnlyMethod"

	// MemberAccessError
	MemberAccessOnString    Code = "MemberAccessError.MemberAccessOnString"
	MemberAccessUnsupported Code = "MemberAccessError.MemberAccessUnsupported" // type name

	// IndexOutOfBoundsError
	ListEmpty             Code = "IndexOutOfBoundsError.ListEmpty"
	ListIndexOutOfRange   Code = "IndexOutOfBoundsError.ListIndexOutOfRange"
	StringIndexOutOfRange Code = "IndexOutOfBoundsError.StringIndexOutOfRange"

	// KeyError
	DictKeyNotFound      Code = "KeyError.DictKeyNotFound"      // key
	StaticMemberNotFound Code = "KeyError.StaticMemberNotFound" // name

	// ArgumentError / MissingArgumentError
	ArgCountExact    Code = "ArgumentError.ArgCountExact"          // count
	TooManyArguments Code = "ArgumentError.TooManyArguments"       // want, got
	MissingArgument  Code = "MissingArgumentError.MissingArgument" // name

	// ConstantAssignmentError
	ConstantAssignment Code = "ConstantAssignmentError.ConstantAssignment" // name

	// AccessViolationError
	PrivateMethodAccess   Code = "AccessViolationError.PrivateMethodAccess"   // name
	PrivateFieldAccess    Code = "AccessViolationError.PrivateFieldAccess"    // name
	ProtectedMethodAccess Code = "AccessViolationError.ProtectedMethodAccess" // name
	ProtectedFieldAccess  Code = "AccessViolationError.ProtectedFieldAccess"  // name

	// InterfaceImplementationError
	InterfaceNotImplemented Code = "InterfaceImplementationError.InterfaceNotImplemented" // class, interface, method

	// DivideByZeroError
	DivideByZero Code = "DivideByZeroError.DivideByZero"

	// ConversionError
	ConvertToNumberFailed     Code = "ConversionError.ConvertToNumberFailed" // text
	ConvertToNumberInvalid    Code = "ConversionError.ConvertToNumberInvalid"
	ConvertToCodeNeedsOneChar Code = "ConversionError.ConvertToCodeNeedsOneChar" // builtin name
	ConvertToTextNeedsNumber  Code = "ConversionError.ConvertToTextNeedsNumber"

	// ImportError
	ImportNativeFunctionNotFound Code = "ImportError.ImportNativeFunctionNotFound" // package, function
	ImportDLLNotFound            Code = "ImportError.ImportDLLNotFound"            // package
	ImportPluginUnsupported      Code = "ImportError.ImportPluginUnsupported"      // GOOS
	ImportManifestInvalid        Code = "ImportError.ImportManifestInvalid"        // package, problem
	ImportNativeMissing          Code = "ImportError.ImportNativeMissing"          // package, platform
	ImportNativeLoadFailed       Code = "ImportError.ImportNativeLoadFailed"       // file
	NativeCallFailed             Code = "ImportError.NativeCallFailed"             // function, message
	ImportUnsupportedLocale      Code = "ImportError.ImportUnsupportedLocale"      // package, language
	ImportPackageSyntax          Code = "ImportError.ImportPackageSyntax"          // package, file
	ImportPackageNotFound        Code = "ImportError.ImportPackageNotFound"        // package
	ImportPackageNotInstalled    Code = "ImportError.ImportPackageNotInstalled"    // package
	ImportClassConflict          Code = "ImportError.ImportClassConflict"          // module, class, other module
	ImportClassConflictOwn       Code = "ImportError.ImportClassConflictOwn"       // module, class
	ImportFileNotFound           Code = "ImportError.ImportFileNotFound"           // file
	ImportFileSyntax             Code = "ImportError.ImportFileSyntax"             // file
	ImportTargetNotFound         Code = "ImportError.ImportTargetNotFound"         // source, target
	ImportNativeOnly             Code = "ImportError.ImportNativeOnly"             // package (raised by the browser engines only)

	// ImportUnsupported is raised only by the browser (TypeScript) engines, which
	// have no file system to import files from. The input errors come from
	// 입력받자's conversion (conv.ParseInput on the Go engines, mirrored in the
	// browser engines).
	ImportUnsupported    Code = "ImportError.ImportUnsupported"
	InputTypeUnsupported Code = "UnsupportedInputTypeError.InputTypeUnsupported" // type
	InputToNumberFailed  Code = "InputConversionError.InputToNumberFailed"       // text
	InputToBooleanFailed Code = "InputConversionError.InputToBooleanFailed"      // text
)

var allCodes = []Code{
	UnknownOperator, NativeListNumbers, NativeListStrings, ListNotSortable, CallbackNotBoolean, ConditionNotBoolean, ListValueUnsupported, MathDomain, StatsNotEnough,
	RangeStepZero, ResultTooLarge, TextEmptyPart, PadFillLength, Base64Invalid, URLDecodeInvalid, SleepRange,
	FileNotFound, FileIsDirectory, FileAccessDenied, FileFailed, FileBlocked,
	NetBlocked, SocketBadHandle, SocketNotConnection, SocketNotListener, SocketConnectFailed, SocketListenFailed, SocketTimeout, SocketFailed, SocketPort, HTTPBadURL, HTTPFailed, HTTPTooLarge, HTTPBadMethod,
	NativeArgString, NativeArgNumber, NativeArgInteger, NativeArgList, NativeArgDict, NativeDictStrings, ArgCountRange,
	JSONInvalid, JSONUnsupported, CSVInvalid, CSVUnsupported, CSVDelimiter, RegexInvalid, RandomRange, RandomEmpty, DateInvalid,
	InstantiateInterface, InstantiateAbstract, StringIndexAssign,
	OperandTypeMismatch, NullOperand,
	VariableTypeMismatch, ArgumentTypeMismatch, ReturnTypeMismatch,
	CallTooDeep,
	SyntaxUnexpectedToken, SyntaxUnexpectedEnd,
	ThisNotBound, SuperOutsideMethod, StaticOutsideMethod, VariableNotFound, ClassNotFound,
	GlobalFunctionNotFound, StaticMethodNotFound, MethodNotFound,
	NotCallable, NotAList, NotIterable, RangeMustBeNumbers, ListIndexMustBeNumber,
	SuperMemberMustBeMethod, MethodArgMustBeNumber, MethodArgMustBeString, NotANumber,
	InvalidOperands, ListOnlyMethod,
	MemberAccessOnString, MemberAccessUnsupported,
	ListEmpty, ListIndexOutOfRange, StringIndexOutOfRange,
	DictKeyNotFound, StaticMemberNotFound,
	ArgCountExact, TooManyArguments, MissingArgument,
	ConstantAssignment,
	PrivateMethodAccess, PrivateFieldAccess, ProtectedMethodAccess, ProtectedFieldAccess,
	InterfaceNotImplemented,
	DivideByZero,
	ConvertToNumberFailed, ConvertToNumberInvalid, ConvertToCodeNeedsOneChar, ConvertToTextNeedsNumber,
	ImportManifestInvalid, ImportNativeMissing, ImportNativeLoadFailed, NativeCallFailed,
	ImportNativeFunctionNotFound, ImportDLLNotFound, ImportPluginUnsupported, ImportUnsupportedLocale, ImportPackageSyntax, ImportPackageNotFound, ImportPackageNotInstalled, ImportClassConflict, ImportClassConflictOwn,
	ImportFileNotFound, ImportFileSyntax, ImportTargetNotFound, ImportNativeOnly,
	ImportUnsupported, InputTypeUnsupported, InputToNumberFailed, InputToBooleanFailed,
}
