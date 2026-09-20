package errs

// Label is a fixed CLI heading printed next to an error ("런타임 오류: ...").
// It lives here, not in cmd/, so the heading and the message under it are
// localized by the same Locale — a .knd script's runtime error should not
// read "런타임 오류: DivideByZeroError: 0で割ることはできません。".
type Label int

const (
	LabelFileRead Label = iota
	LabelParseFailed
	LabelCompileFailed
	LabelRuntime
	LabelBytecodeFile
	LabelOutputFile
	LabelBytecodeSave
)

var labelText = map[Locale]map[Label]string{
	English: {
		LabelFileRead:      "File read error",
		LabelParseFailed:   "Parsing failed",
		LabelCompileFailed: "Bytecode compilation failed",
		LabelRuntime:       "Runtime error",
		LabelBytecodeFile:  "Bytecode file error",
		LabelOutputFile:    "Output file creation error",
		LabelBytecodeSave:  "Bytecode save error",
	},
	Korean: {
		LabelFileRead:      "파일 읽기 오류",
		LabelParseFailed:   "구문 분석(Parsing) 중 오류가 발생했어요",
		LabelCompileFailed: "바이트코드 컴파일 중 오류가 발생했어요",
		LabelRuntime:       "런타임 오류",
		LabelBytecodeFile:  "바이트코드 파일 오류",
		LabelOutputFile:    "출력 파일 생성 오류",
		LabelBytecodeSave:  "바이트코드 저장 오류",
	},
	Japanese: {
		LabelFileRead:      "ファイル読み込みエラー",
		LabelParseFailed:   "構文解析中にエラーが発生しました",
		LabelCompileFailed: "バイトコードのコンパイル中にエラーが発生しました",
		LabelRuntime:       "ランタイムエラー",
		LabelBytecodeFile:  "バイトコードファイルエラー",
		LabelOutputFile:    "出力ファイルの作成エラー",
		LabelBytecodeSave:  "バイトコードの保存エラー",
	},
}

// LabelText returns l's heading in loc (English if loc lacks it).
func LabelText(loc Locale, l Label) string {
	if s, ok := labelText[loc][l]; ok {
		return s
	}
	return labelText[English][l]
}
