// Package stdimpl holds the behavior of hana's native standard-library
// functions, once, for both engines. A native function here sees only plain
// values (float64, string, bool, []interface{}, map[interface{}]interface{},
// nil), which the tree-walker and the bytecode VM share, so stdlib and
// bcstdlib just wrap these functions under the names std gives them. The docs
// sites' browser engines mirror them by hand (stdlib.ts) and compare_tests.ts
// checks the two agree.
package stdimpl

import (
	"math"

	"github.com/soumt-r/hana/errs"
	"github.com/soumt-r/hana/std"
)

// Func is one native function's behavior.
type Func func(args []interface{}) (interface{}, error)

// Impls maps every function ID in std.Modules to its implementation.
var Impls = map[string]Func{
	std.MathCeil:  numberFunc(math.Ceil),
	std.MathFloor: numberFunc(math.Floor),

	std.MathSqrt:      mathSqrt,
	std.MathPow:       mathPow,
	std.MathAbs:       mathAbs,
	std.MathRound:     mathRound,
	std.MathSin:       mathSin,
	std.MathCos:       mathCos,
	std.MathTan:       mathTan,
	std.MathLog:       mathLog,
	std.MathPi:        mathPi,
	std.MathGcd:       mathGcd,
	std.MathFactorial: mathFactorial,

	std.StatsSum:    statsSum,
	std.StatsMin:    statsMin,
	std.StatsMax:    statsMax,
	std.StatsMean:   statsMean,
	std.StatsMedian: statsMedian,
	std.StatsStdev:  statsStdev,

	std.TextUpper:      textUpper,
	std.TextLower:      textLower,
	std.TextStrip:      textStrip,
	std.TextPadLeft:    textPadLeft,
	std.TextPadRight:   textPadRight,
	std.TextRepeat:     textRepeat,
	std.TextReverse:    textReverse,
	std.TextStartsWith: textStartsWith,
	std.TextEndsWith:   textEndsWith,
	std.TextJoin:       textJoin,
	std.TextCount:      textCount,
	std.TextFind:       textFind,

	std.ListSort:    listSort,
	std.ListReverse: listReverse,
	std.ListUnique:  listUnique,
	std.ListRange:   listRange,
	std.ListFlatten: listFlatten,
	std.ListChunk:   listChunk,
	std.ListZip:     listZip,

	std.EncodingBase64Encode: encodingBase64Encode,
	std.EncodingBase64Decode: encodingBase64Decode,
	std.EncodingURLEncode:    encodingURLEncode,
	std.EncodingURLDecode:    encodingURLDecode,

	std.FileRead:   fileRead,
	std.FileLines:  fileLines,
	std.FileWrite:  fileWrite,
	std.FileAppend: fileAppend,
	std.FileExists: fileExists,
	std.FileIsDir:  fileIsDir,
	std.FileDelete: fileDelete,
	std.FileList:   fileList,
	std.FileMkdir:  fileMkdir,
	std.FileMove:   fileMove,

	std.SocketConnect:     socketConnect,
	std.SocketListen:      socketListen,
	std.SocketAccept:      socketAccept,
	std.SocketSend:        socketSend,
	std.SocketReceive:     socketReceive,
	std.SocketReceiveLine: socketReceiveLine,
	std.SocketTimeout:     socketTimeout,
	std.SocketAddress:     socketAddress,
	std.SocketClose:       socketClose,

	std.HTTPGet:     httpGet,
	std.HTTPPost:    httpPost,
	std.HTTPRequest: httpRequest,

	std.PathJoin:      pathJoin,
	std.PathDirname:   pathDirname,
	std.PathBasename:  pathBasename,
	std.PathExt:       pathExt,
	std.PathStem:      pathStem,
	std.PathWithExt:   pathWithExt,
	std.PathNormalize: pathNormalize,
	std.PathIsAbs:     pathIsAbs,
	std.PathParts:     pathParts,

	std.HashSHA256: hashSHA256,
	std.RandomUUID: randomUUID,

	std.JSONParse:     jsonParse,
	std.JSONStringify: jsonStringify,
	std.CSVParse:      csvParse,
	std.CSVStringify:  csvStringify,

	std.RandomFloat:   randomFloat,
	std.RandomInt:     randomInt,
	std.RandomChoice:  randomChoice,
	std.RandomShuffle: randomShuffle,

	std.DatetimeNow:     datetimeNow,
	std.DatetimeFormat:  datetimeFormat,
	std.DatetimeParse:   datetimeParse,
	std.DatetimeWeekday: datetimeWeekday,
	std.DatetimeSleep:   datetimeSleep,

	std.RegexTest:    regexTest,
	std.RegexFind:    regexFind,
	std.RegexGroups:  regexGroups,
	std.RegexFindAll: regexFindAll,
	std.RegexReplace: regexReplace,
	std.RegexSplit:   regexSplit,
}

func numberFunc(f func(float64) float64) Func {
	return func(args []interface{}) (interface{}, error) {
		if err := exactly(args, 1); err != nil {
			return nil, err
		}
		if n, ok := args[0].(float64); ok {
			return f(n), nil
		}
		return nil, errs.New(errs.NotANumber)
	}
}

func exactly(args []interface{}, n int) error {
	if len(args) != n {
		return errs.New(errs.ArgCountExact, n)
	}
	return nil
}

func between(args []interface{}, min, max int) error {
	if len(args) < min || len(args) > max {
		return errs.New(errs.ArgCountRange, min, max)
	}
	return nil
}

// stringArg/numberArg/integerArg/listArg read args[i] (0-based) or fail naming
// its 1-based position.
func stringArg(args []interface{}, i int) (string, error) {
	if s, ok := args[i].(string); ok {
		return s, nil
	}
	return "", errs.New(errs.NativeArgString, i+1)
}

func numberArg(args []interface{}, i int) (float64, error) {
	if n, ok := args[i].(float64); ok {
		return n, nil
	}
	return 0, errs.New(errs.NativeArgNumber, i+1)
}

// maxSafeInteger is 2^53: past it a float64 stops holding every whole number.
const maxSafeInteger = 9007199254740992

func integerArg(args []interface{}, i int) (int64, error) {
	n, ok := args[i].(float64)
	if !ok || n != math.Trunc(n) || math.Abs(n) > maxSafeInteger {
		return 0, errs.New(errs.NativeArgInteger, i+1)
	}
	return int64(n), nil
}

func listArg(args []interface{}, i int) ([]interface{}, error) {
	if l, ok := args[i].([]interface{}); ok {
		return l, nil
	}
	return nil, errs.New(errs.NativeArgList, i+1)
}
