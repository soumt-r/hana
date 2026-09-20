// Package std is the language-neutral catalog of hana's native standard-library
// modules. A native function is identified by a stable ID ("math.ceil") that
// says nothing about any language; each language then gives modules and
// functions their own names (수학 / 数学, 올림 / 切り上げ). Implementations in the
// tree-walker (stdlib), the bytecode VM (bcstdlib) and the docs sites' browser
// engines all look their names up here, so a name is written down exactly once.
//
// The docs engines get their copy generated from this table (cmd/stdgen).
package std

// Language keys match vm.LangConfig.Name.
const (
	Haja   = "haja"
	Kanade = "kanade"
)

// Languages lists every language that has a name table.
var Languages = []string{Haja, Kanade}

// Native function IDs.
const (
	MathCeil  = "math.ceil"
	MathFloor = "math.floor"

	JSONParse     = "json.parse"
	JSONStringify = "json.stringify"

	CSVParse     = "csv.parse"
	CSVStringify = "csv.stringify"

	RandomFloat   = "random.float"
	RandomInt     = "random.int"
	RandomChoice  = "random.choice"
	RandomShuffle = "random.shuffle"

	DatetimeNow     = "datetime.now"
	DatetimeFormat  = "datetime.format"
	DatetimeParse   = "datetime.parse"
	DatetimeWeekday = "datetime.weekday"
	DatetimeSleep   = "datetime.sleep"

	RegexTest    = "regex.test"
	RegexFind    = "regex.find"
	RegexGroups  = "regex.groups"
	RegexFindAll = "regex.findall"
	RegexReplace = "regex.replace"
	RegexSplit   = "regex.split"

	MathSqrt      = "math.sqrt"
	MathPow       = "math.pow"
	MathAbs       = "math.abs"
	MathRound     = "math.round"
	MathSin       = "math.sin"
	MathCos       = "math.cos"
	MathTan       = "math.tan"
	MathLog       = "math.log"
	MathPi        = "math.pi"
	MathGcd       = "math.gcd"
	MathFactorial = "math.factorial"

	StatsSum    = "stats.sum"
	StatsMin    = "stats.min"
	StatsMax    = "stats.max"
	StatsMean   = "stats.mean"
	StatsMedian = "stats.median"
	StatsStdev  = "stats.stdev"

	TextUpper      = "text.upper"
	TextLower      = "text.lower"
	TextStrip      = "text.strip"
	TextPadLeft    = "text.padleft"
	TextPadRight   = "text.padright"
	TextRepeat     = "text.repeat"
	TextReverse    = "text.reverse"
	TextStartsWith = "text.startswith"
	TextEndsWith   = "text.endswith"
	TextJoin       = "text.join"
	TextCount      = "text.count"
	TextFind       = "text.find"

	ListSort    = "list.sort"
	ListReverse = "list.reverse"
	ListUnique  = "list.unique"
	ListRange   = "list.range"
	ListFlatten = "list.flatten"
	ListChunk   = "list.chunk"
	ListZip     = "list.zip"

	// Functions that take a function of the program (see stdimpl.HostImpls).
	ListMap    = "list.map"
	ListFilter = "list.filter"
	ListReduce = "list.reduce"
	ListFind   = "list.find"
	ListAny    = "list.any"
	ListAll    = "list.all"
	ListSortBy = "list.sortby"

	EncodingBase64Encode = "encoding.base64encode"
	EncodingBase64Decode = "encoding.base64decode"
	EncodingURLEncode    = "encoding.urlencode"
	EncodingURLDecode    = "encoding.urldecode"

	HashSHA256 = "hash.sha256"

	RandomUUID = "random.uuid"

	FileRead   = "file.read"
	FileLines  = "file.lines"
	FileWrite  = "file.write"
	FileAppend = "file.append"
	FileExists = "file.exists"
	FileIsDir  = "file.isdir"
	FileDelete = "file.delete"
	FileList   = "file.list"
	FileMkdir  = "file.mkdir"
	FileMove   = "file.move"

	SocketConnect     = "socket.connect"
	SocketListen      = "socket.listen"
	SocketAccept      = "socket.accept"
	SocketSend        = "socket.send"
	SocketReceive     = "socket.receive"
	SocketReceiveLine = "socket.receiveline"
	SocketTimeout     = "socket.timeout"
	SocketAddress     = "socket.address"
	SocketClose       = "socket.close"

	HTTPGet     = "http.get"
	HTTPPost    = "http.post"
	HTTPRequest = "http.request"

	PathJoin      = "path.join"
	PathDirname   = "path.dirname"
	PathBasename  = "path.basename"
	PathExt       = "path.ext"
	PathStem      = "path.stem"
	PathWithExt   = "path.withext"
	PathNormalize = "path.normalize"
	PathIsAbs     = "path.isabs"
	PathParts     = "path.parts"
)

// Module is one native module: its ID and the IDs of its functions, in the
// order they are exported.
type Module struct {
	ID        string
	Functions []string
	// NativeOnly marks a module that needs the operating system (files, sockets)
	// and so exists only in hana itself. The browser engines have no
	// implementation and refuse to import it (errs.ImportNativeOnly).
	NativeOnly bool
}

// Modules is every native module hana ships, in declaration order.
var Modules = []Module{
	{ID: "math", Functions: []string{MathCeil, MathFloor, MathSqrt, MathPow, MathAbs, MathRound, MathSin, MathCos, MathTan, MathLog, MathPi, MathGcd, MathFactorial}},
	{ID: "json", Functions: []string{JSONParse, JSONStringify}},
	{ID: "csv", Functions: []string{CSVParse, CSVStringify}},
	{ID: "random", Functions: []string{RandomFloat, RandomInt, RandomChoice, RandomShuffle, RandomUUID}},
	{ID: "datetime", Functions: []string{DatetimeNow, DatetimeFormat, DatetimeParse, DatetimeWeekday, DatetimeSleep}},
	{ID: "stats", Functions: []string{StatsSum, StatsMin, StatsMax, StatsMean, StatsMedian, StatsStdev}},
	{ID: "text", Functions: []string{TextUpper, TextLower, TextStrip, TextPadLeft, TextPadRight, TextRepeat, TextReverse, TextStartsWith, TextEndsWith, TextJoin, TextCount, TextFind}},
	{ID: "list", Functions: []string{ListSort, ListReverse, ListUnique, ListRange, ListFlatten, ListChunk, ListZip, ListMap, ListFilter, ListReduce, ListFind, ListAny, ListAll, ListSortBy}},
	{ID: "encoding", Functions: []string{EncodingBase64Encode, EncodingBase64Decode, EncodingURLEncode, EncodingURLDecode}},
	{ID: "hash", Functions: []string{HashSHA256}},
	{ID: "file", NativeOnly: true, Functions: []string{FileRead, FileLines, FileWrite, FileAppend, FileExists, FileIsDir, FileDelete, FileList, FileMkdir, FileMove}},
	{ID: "socket", NativeOnly: true, Functions: []string{SocketConnect, SocketListen, SocketAccept, SocketSend, SocketReceive, SocketReceiveLine, SocketTimeout, SocketAddress, SocketClose}},
	{ID: "http", NativeOnly: true, Functions: []string{HTTPGet, HTTPPost, HTTPRequest}},
	{ID: "path", Functions: []string{PathJoin, PathDirname, PathBasename, PathExt, PathStem, PathWithExt, PathNormalize, PathIsAbs, PathParts}},
	{ID: "regex", Functions: []string{RegexTest, RegexFind, RegexGroups, RegexFindAll, RegexReplace, RegexSplit}},
}

// names maps a language to the localized name of every module and function ID.
var names = map[string]map[string]string{
	Haja: {
		"math":    "수학",
		MathCeil:  "올림",
		MathFloor: "버림",

		"json":        "JSON",
		JSONParse:     "파싱",
		JSONStringify: "문자열화",

		"csv":        "CSV",
		CSVParse:     "파싱",
		CSVStringify: "문자열화",

		"random":      "무작위",
		RandomFloat:   "실수",
		RandomInt:     "정수",
		RandomChoice:  "고르기",
		RandomShuffle: "섞기",

		"datetime":      "날짜",
		DatetimeNow:     "지금",
		DatetimeFormat:  "서식",
		DatetimeParse:   "읽기",
		DatetimeWeekday: "요일",
		DatetimeSleep:   "기다리기",

		"regex":      "정규식",
		RegexTest:    "검사",
		RegexFind:    "찾기",
		RegexGroups:  "그룹",
		RegexFindAll: "모두찾기",
		RegexReplace: "치환",
		RegexSplit:   "분할",

		"file":     "파일",
		FileRead:   "읽기",
		FileLines:  "줄읽기",
		FileWrite:  "쓰기",
		FileAppend: "덧붙이기",
		FileExists: "있는지",
		FileIsDir:  "폴더인지",
		FileDelete: "지우기",
		FileList:   "목록",
		FileMkdir:  "폴더만들기",
		FileMove:   "옮기기",

		MathSqrt:      "제곱근",
		MathPow:       "거듭제곱",
		MathAbs:       "절댓값",
		MathRound:     "반올림",
		MathSin:       "사인",
		MathCos:       "코사인",
		MathTan:       "탄젠트",
		MathLog:       "로그",
		MathPi:        "파이",
		MathGcd:       "최대공약수",
		MathFactorial: "팩토리얼",
		RandomUUID:    "고유번호",

		"stats":     "통계",
		StatsSum:    "합계",
		StatsMin:    "최솟값",
		StatsMax:    "최댓값",
		StatsMean:   "평균",
		StatsMedian: "중앙값",
		StatsStdev:  "표준편차",

		"text":         "텍스트",
		TextUpper:      "대문자",
		TextLower:      "소문자",
		TextStrip:      "다듬기",
		TextPadLeft:    "왼쪽채우기",
		TextPadRight:   "오른쪽채우기",
		TextRepeat:     "반복",
		TextReverse:    "거꾸로",
		TextStartsWith: "시작하는지",
		TextEndsWith:   "끝나는지",
		TextJoin:       "잇기",
		TextCount:      "세기",
		TextFind:       "위치",

		"list":      "목록",
		ListSort:    "정렬",
		ListReverse: "뒤집기",
		ListUnique:  "중복제거",
		ListRange:   "범위",
		ListFlatten: "평탄화",
		ListChunk:   "조각내기",
		ListZip:     "짝짓기",
		ListMap:     "변환하기",
		ListFilter:  "걸러내기",
		ListReduce:  "접기",
		ListFind:    "찾기",
		ListAny:     "하나라도",
		ListAll:     "모두",
		ListSortBy:  "기준정렬",

		"encoding":           "인코딩",
		EncodingBase64Encode: "베이스64인코딩",
		EncodingBase64Decode: "베이스64디코딩",
		EncodingURLEncode:    "주소인코딩",
		EncodingURLDecode:    "주소디코딩",

		"hash":     "해시",
		HashSHA256: "SHA256",

		"socket":          "소켓",
		SocketConnect:     "연결하기",
		SocketListen:      "듣기",
		SocketAccept:      "받아들이기",
		SocketSend:        "보내기",
		SocketReceive:     "받기",
		SocketReceiveLine: "줄받기",
		SocketTimeout:     "시간제한",
		SocketAddress:     "주소",
		SocketClose:       "닫기",

		"http":      "HTTP",
		HTTPGet:     "가져오기",
		HTTPPost:    "보내기",
		HTTPRequest: "요청",

		"path":        "경로",
		PathJoin:      "합치기",
		PathDirname:   "폴더이름",
		PathBasename:  "파일이름",
		PathExt:       "확장자",
		PathStem:      "확장자뺀이름",
		PathWithExt:   "확장자바꾸기",
		PathNormalize: "정리하기",
		PathIsAbs:     "절대경로인지",
		PathParts:     "부분나누기",
	},
	Kanade: {
		"math":    "数学",
		MathCeil:  "切り上げ",
		MathFloor: "切り捨て",

		"json":        "JSON",
		JSONParse:     "パース",
		JSONStringify: "文字列化",

		"csv":        "CSV",
		CSVParse:     "パース",
		CSVStringify: "文字列化",

		"random":      "乱数",
		RandomFloat:   "実数",
		RandomInt:     "整数",
		RandomChoice:  "選ぶ",
		RandomShuffle: "シャッフル",

		"datetime":      "日時",
		DatetimeNow:     "今",
		DatetimeFormat:  "書式",
		DatetimeParse:   "読み取り",
		DatetimeWeekday: "曜日",
		DatetimeSleep:   "待つ",

		"regex":      "正規表現",
		RegexTest:    "検査",
		RegexFind:    "検索",
		RegexGroups:  "グループ",
		RegexFindAll: "全検索",
		RegexReplace: "置換",
		RegexSplit:   "分割",

		"file":     "ファイル",
		FileRead:   "読む",
		FileLines:  "行で読む",
		FileWrite:  "書く",
		FileAppend: "追記",
		FileExists: "あるか",
		FileIsDir:  "フォルダか",
		FileDelete: "削除",
		FileList:   "一覧",
		FileMkdir:  "フォルダ作成",
		FileMove:   "移動",

		MathSqrt:      "平方根",
		MathPow:       "べき乗",
		MathAbs:       "絶対値",
		MathRound:     "四捨五入",
		MathSin:       "サイン",
		MathCos:       "コサイン",
		MathTan:       "タンジェント",
		MathLog:       "対数",
		MathPi:        "円周率",
		MathGcd:       "最大公約数",
		MathFactorial: "階乗",
		RandomUUID:    "UUID",

		"stats":     "統計",
		StatsSum:    "合計",
		StatsMin:    "最小値",
		StatsMax:    "最大値",
		StatsMean:   "平均",
		StatsMedian: "中央値",
		StatsStdev:  "標準偏差",

		"text":         "テキスト",
		TextUpper:      "大文字",
		TextLower:      "小文字",
		TextStrip:      "トリム",
		TextPadLeft:    "左埋め",
		TextPadRight:   "右埋め",
		TextRepeat:     "繰り返し",
		TextReverse:    "逆さ",
		TextStartsWith: "始まるか",
		TextEndsWith:   "終わるか",
		TextJoin:       "連結",
		TextCount:      "数える",
		TextFind:       "位置",

		"list":      "リスト",
		ListSort:    "並べ替え",
		ListReverse: "反転",
		ListUnique:  "重複除去",
		ListRange:   "範囲",
		ListFlatten: "平坦化",
		ListChunk:   "小分け",
		ListZip:     "ペア",
		ListMap:     "変換",
		ListFilter:  "絞り込み",
		ListReduce:  "畳み込み",
		ListFind:    "探す",
		ListAny:     "どれか",
		ListAll:     "すべて",
		ListSortBy:  "基準並べ替え",

		"encoding":           "エンコード",
		EncodingBase64Encode: "Base64エンコード",
		EncodingBase64Decode: "Base64デコード",
		EncodingURLEncode:    "URLエンコード",
		EncodingURLDecode:    "URLデコード",

		"hash":     "ハッシュ",
		HashSHA256: "SHA256",

		"socket":          "ソケット",
		SocketConnect:     "接続",
		SocketListen:      "待ち受け",
		SocketAccept:      "受け入れ",
		SocketSend:        "送信",
		SocketReceive:     "受信",
		SocketReceiveLine: "行受信",
		SocketTimeout:     "タイムアウト",
		SocketAddress:     "アドレス",
		SocketClose:       "閉じる",

		"http":      "HTTP",
		HTTPGet:     "取得",
		HTTPPost:    "送信",
		HTTPRequest: "リクエスト",

		"path":        "パス",
		PathJoin:      "結合",
		PathDirname:   "フォルダ名",
		PathBasename:  "ファイル名",
		PathExt:       "拡張子",
		PathStem:      "拡張子なし名",
		PathWithExt:   "拡張子変更",
		PathNormalize: "正規化",
		PathIsAbs:     "絶対パスか",
		PathParts:     "構成要素",
	},
}

// ModuleName is the name a module is imported by in lang (`[수학]`, `【数学】`).
func ModuleName(lang, moduleID string) string { return names[lang][moduleID] }

// FunctionName is the name a native function is called by in lang.
func FunctionName(lang, functionID string) string { return names[lang][functionID] }
