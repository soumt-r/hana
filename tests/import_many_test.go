package tests

// Importing several items at once (`<a>와 <b>를 가져오자`) and a whole module
// (`전부 가져오자`), for native modules and local files, on
// tree-walker/bytecode x hari/kanade.

import (
	"testing"

	"github.com/soumt-r/hana/bytecode"
	lexer "github.com/soumt-r/hana/lexer/hari"
	parser "github.com/soumt-r/hana/parser/hari"
)

func TestImportSeveralNativeItems(t *testing.T) {
	runTypeCases(t, []typeCase{
		{
			name:   "two items from a native module",
			hari:   "[수학]에서 <올림>과 <버림>을 가져오자\n<올림>(3.2)를 출력하자\n<버림>(3.8)을 출력하자\n",
			kanade: "【数学】から〈切り上げ〉と〈切り捨て〉を持ってこよう\n〈切り上げ〉(3.2)を出力しよう\n〈切り捨て〉(3.8)を出力しよう\n",
			want:   "4|3",
		},
		{
			name:   "three items from a native module",
			hari:   "[정규식]에서 <검사>와 <찾기>와 <분할>을 가져오자\n<검사>(\"a1\", \"[0-9]\")를 출력하자\n<찾기>(\"a1\", \"[0-9]\")를 출력하자\n<분할>(\"a,b\", \",\")를 출력하자\n",
			kanade: "【正規表現】から〈検査〉と〈検索〉と〈分割〉を持ってこよう\n〈検査〉(「a1」, 「[0-9]」)を出力しよう\n〈検索〉(「a1」, 「[0-9]」)を出力しよう\n〈分割〉(「a,b」, 「,」)を出力しよう\n",
			want:   "참|1|[a, b]", wantKanade: "真|1|[a, b]",
		},
		{
			name:    "a missing item in the list",
			hari:    "[수학]에서 <올림>과 <없는것>을 가져오자\n",
			kanade:  "【数学】から〈切り上げ〉と〈なし〉を持ってこよう\n",
			wantErr: "not found",
		},
		{
			name:   "a single item with an alias still works",
			hari:   "[수학]에서 <올림>을 <반올림>으로 가져오자\n<반올림>(3.2)를 출력하자\n",
			kanade: "【数学】から〈切り上げ〉を〈丸め〉に持ってこよう\n〈丸め〉(3.2)を出力しよう\n",
			want:   "4",
		},
	})
}

func TestImportWholeNativeModule(t *testing.T) {
	runTypeCases(t, []typeCase{
		{
			name:   "everything in a native module",
			hari:   "[수학]에서 전부 가져오자\n<올림>(3.2)를 출력하자\n<버림>(3.8)을 출력하자\n",
			kanade: "【数学】から全部持ってこよう\n〈切り上げ〉(3.2)を出力しよう\n〈切り捨て〉(3.8)を出力しよう\n",
			want:   "4|3",
		},
		{
			name:   "a bigger module",
			hari:   "[날짜]에서 전부 가져오자\n<서식>(<읽기>(\"2024-03-05\", \"YYYY-MM-DD\"), \"MM/DD\")를 출력하자\n",
			kanade: "【日時】から全部持ってこよう\n〈書式〉(〈読み取り〉(「2024-03-05」, 「YYYY-MM-DD」), 「MM/DD」)を出力しよう\n",
			want:   "03/05",
		},
		{
			name:    "a module that does not exist",
			hari:    "[없는모듈]에서 전부 가져오자\n",
			kanade:  "【なし】から全部持ってこよう\n",
			wantErr: "not found",
		},
		{
			name:    "nothing is imported before it is asked for",
			hari:    "<올림>(3.2)를 출력하자\n",
			kanade:  "〈切り上げ〉(3.2)を出力しよう\n",
			wantErr: "not found",
		},
	})
}

const libHari = `[숫자]를 돌려주는 <두배>를 만들자 ([숫자]인 '값'):
    ('값' * 2)를 돌려주자
[숫자]를 돌려주는 <세배>를 만들자 ([숫자]인 '값'):
    ('값' * 3)를 돌려주자
[상자]를 설계하자:
    '크기'를 [숫자]인 5로 정하자
`

const libKanade = `【数字】を返す〈二倍〉を作ろう(【数字】の『値』):
    (『値』 * 2)を返そう
【数字】を返す〈三倍〉を作ろう(【数字】の『値』):
    (『値』 * 3)を返そう
【箱】を設計しよう:
    『大きさ』を【数字】の5にしよう
`

func TestImportSeveralAndAllFromALocalFile(t *testing.T) {
	inTempDir(t, map[string]string{"lib.hr": libHari, "lib.knd": libKanade})
	runTypeCases(t, []typeCase{
		{
			name:   "two functions from a file",
			hari:   "\"lib.hr\"에서 <두배>와 <세배>를 가져오자\n<두배>(4)를 출력하자\n<세배>(4)를 출력하자\n",
			kanade: "「lib.knd」から〈二倍〉と〈三倍〉を持ってこよう\n〈二倍〉(4)を出力しよう\n〈三倍〉(4)を出力しよう\n",
			want:   "8|12",
		},
		{
			name:   "a function and a class in one list",
			hari:   "\"lib.hr\"에서 <두배>와 '상자'를 가져오자\n'통'을 [상자]인 새로운 [상자]()로 정하자\n'통'의 '크기'를 출력하자\n<두배>(4)를 출력하자\n",
			kanade: "「lib.knd」から〈二倍〉と『箱』を持ってこよう\n『入れ物』を【箱】の新しい【箱】()にしよう\n『入れ物』の『大きさ』を出力しよう\n〈二倍〉(4)を出力しよう\n",
			want:   "5|8",
		},
		{
			name:   "everything a file declares",
			hari:   "\"lib.hr\"에서 전부 가져오자\n<두배>(4)를 출력하자\n<세배>(4)를 출력하자\n'통'을 [상자]인 새로운 [상자]()로 정하자\n'통'의 '크기'를 출력하자\n",
			kanade: "「lib.knd」から全部持ってこよう\n〈二倍〉(4)を出力しよう\n〈三倍〉(4)を出力しよう\n『入れ物』を【箱】の新しい【箱】()にしよう\n『入れ物』の『大きさ』を出力しよう\n",
			want:   "8|12|5",
		},
	})
}

func TestMissingItemInAFileListIsAnError(t *testing.T) {
	inTempDir(t, map[string]string{"lib.hr": libHari})
	code := "\"lib.hr\"에서 <두배>와 <없는것>을 가져오자\n"
	if _, err := runHari(t, code); err == nil {
		t.Error("tree-walker: expected an error for the missing item")
	}
	compiler := bytecode.NewCompiler()
	compiler.Compile(parser.New(lexer.New(code)).ParseProgram())
	if len(compiler.Errors()) == 0 {
		t.Error("bytecode: expected a compile error for the missing item")
	}
}
