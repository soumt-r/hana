package tests

// Behavior of the math / statistics / text / list / encoding / hash modules
// (and random's UUID), on tree-walker/bytecode x haja/kanade. The docs sites'
// browser engines mirror them; compare_tests.ts checks they agree with hana.

import "testing"

// imports builds the `[모듈]에서 <이름>을 가져오자` lines for a module.
func hajaImports(module string, names ...string) string {
	out := ""
	for _, n := range names {
		out += "[" + module + "]에서 <" + n + ">을 가져오자\n"
	}
	return out
}

func kanadeImports(module string, names ...string) string {
	out := ""
	for _, n := range names {
		out += "【" + module + "】から〈" + n + "〉を持ってこよう\n"
	}
	return out
}

func TestMathExtras(t *testing.T) {
	h := hajaImports("수학", "제곱근", "거듭제곱", "절댓값", "반올림", "로그", "최대공약수", "팩토리얼")
	k := kanadeImports("数学", "平方根", "べき乗", "絶対値", "四捨五入", "対数", "最大公約数", "階乗")
	runTypeCases(t, []typeCase{
		{
			name:   "round half away from zero, also for negatives",
			haja:   h + "<반올림>(2.5)를 출력하자\n<반올림>((0 - 2.5))를 출력하자\n<반올림>(1234.5678, 2)를 출력하자\n<반올림>((0 - 0.4))를 출력하자\n",
			kanade: k + "〈四捨五入〉(2.5)を出力しよう\n〈四捨五入〉((0 - 2.5))を出力しよう\n〈四捨五入〉(1234.5678, 2)を出力しよう\n〈四捨五入〉((0 - 0.4))を出力しよう\n",
			want:   "3|-3|1234.57|0",
		},
		{
			name:   "gcd, factorial and abs",
			haja:   h + "<최대공약수>(0, 5)를 출력하자\n<최대공약수>((0 - 12), 18)을 출력하자\n<팩토리얼>(0)을 출력하자\n<팩토리얼>(10)을 출력하자\n<절댓값>((0 - 7))을 출력하자\n",
			kanade: k + "〈最大公約数〉(0, 5)を出力しよう\n〈最大公約数〉((0 - 12), 18)を出力しよう\n〈階乗〉(0)を出力しよう\n〈階乗〉(10)を出力しよう\n〈絶対値〉((0 - 7))を出力しよう\n",
			want:   "5|6|1|3628800|7",
		},
		{
			name:    "square root of a negative",
			haja:    h + "<제곱근>((0 - 1))을 출력하자\n",
			kanade:  k + "〈平方根〉((0 - 1))を出力しよう\n",
			wantErr: "cannot be used in this calculation",
		},
		{
			name:    "logarithm of zero",
			haja:    h + "<로그>(0)을 출력하자\n",
			kanade:  k + "〈対数〉(0)を出力しよう\n",
			wantErr: "cannot be used in this calculation",
		},
		{
			name:    "a power that overflows",
			haja:    h + "<거듭제곱>(10, 400)을 출력하자\n",
			kanade:  k + "〈べき乗〉(10, 400)を出力しよう\n",
			wantErr: "cannot be used in this calculation",
		},
		{
			name:    "factorial too big for a number",
			haja:    h + "<팩토리얼>(171)을 출력하자\n",
			kanade:  k + "〈階乗〉(171)を出力しよう\n",
			wantErr: "cannot be used in this calculation",
		},
		{
			name:    "factorial needs a whole number",
			haja:    h + "<팩토리얼>(2.5)를 출력하자\n",
			kanade:  k + "〈階乗〉(2.5)を出力しよう\n",
			wantErr: "must be a whole number",
		},
	})
}

func TestStatsModule(t *testing.T) {
	h := hajaImports("통계", "합계", "최솟값", "최댓값", "평균", "중앙값", "표준편차")
	k := kanadeImports("統計", "合計", "最小値", "最大値", "平均", "中央値", "標準偏差")
	runTypeCases(t, []typeCase{
		{
			name:   "an even number of values has a middle pair",
			haja:   h + "<중앙값>([4, 1, 3, 2])를 출력하자\n<평균>([1, 2, 3, 4])를 출력하자\n<합계>([])을 출력하자\n",
			kanade: k + "〈中央値〉(【4, 1, 3, 2】)を出力しよう\n〈平均〉(【1, 2, 3, 4】)を出力しよう\n〈合計〉(【】)を出力しよう\n",
			want:   "2.5|2.5|0",
		},
		{
			name:    "the mean of nothing",
			haja:    h + "<평균>([])를 출력하자\n",
			kanade:  k + "〈平均〉(【】)を出力しよう\n",
			wantErr: "not enough values",
		},
		{
			name:    "a deviation needs two values",
			haja:    h + "<표준편차>([5])를 출력하자\n",
			kanade:  k + "〈標準偏差〉(【5】)を出力しよう\n",
			wantErr: "not enough values",
		},
		{
			name:    "only numbers",
			haja:    h + "<최댓값>([1, \"둘\"])을 출력하자\n",
			kanade:  k + "〈最大値〉(【1, 「二」】)を出力しよう\n",
			wantErr: "list of numbers",
		},
	})
}

func TestTextModule(t *testing.T) {
	h := hajaImports("텍스트", "대문자", "다듬기", "왼쪽채우기", "반복", "거꾸로", "시작하는지", "잇기", "세기", "위치")
	k := kanadeImports("テキスト", "大文字", "トリム", "左埋め", "繰り返し", "逆さ", "始まるか", "連結", "数える", "位置")
	runTypeCases(t, []typeCase{
		{
			name:   "upper case leaves other scripts alone",
			haja:   h + "<대문자>(\"hello 한글\")를 출력하자\n",
			kanade: k + "〈大文字〉(「hello ひらがな」)を出力しよう\n",
			want:   "HELLO 한글", wantKanade: "HELLO ひらがな",
		},
		{
			name:   "strip removes the ideographic space too",
			haja:   h + "틀\"[{<다듬기>(\"　 가 나 　\")}]\"를 출력하자\n",
			kanade: k + "枠「[{〈トリム〉(「　 あ い 　」)}]」を出力しよう\n",
			want:   "[가 나]", wantKanade: "[あ い]",
		},
		{
			name:   "padding counts characters and never cuts",
			haja:   h + "<왼쪽채우기>(\"가\", 3, \"0\")을 출력하자\n<왼쪽채우기>(\"가나다라\", 3, \"0\")을 출력하자\n",
			kanade: k + "〈左埋め〉(「あ」, 3, 「0」)を出力しよう\n〈左埋め〉(「あいうえ」, 3, 「0」)を出力しよう\n",
			want:   "00가|가나다라", wantKanade: "00あ|あいうえ",
		},
		{
			name:    "the fill must be one character",
			haja:    h + "<왼쪽채우기>(\"7\", 3, \"ab\")를 출력하자\n",
			kanade:  k + "〈左埋め〉(「7」, 3, 「ab」)を出力しよう\n",
			wantErr: "exactly one character",
		},
		{
			name:   "repeat zero times is empty",
			haja:   h + "틀\"[{<반복>(\"ab\", 0)}]\"를 출력하자\n",
			kanade: k + "枠「[{〈繰り返し〉(「ab」, 0)}]」を出力しよう\n",
			want:   "[]",
		},
		{
			name:    "repeat a negative number of times",
			haja:    h + "<반복>(\"ab\", (0 - 1))을 출력하자\n",
			kanade:  k + "〈繰り返し〉(「ab」, (0 - 1))を出力しよう\n",
			wantErr: "must be a whole number",
		},
		{
			name:    "repeat that would be huge",
			haja:    h + "<반복>(\"abcdefghij\", 200000)을 출력하자\n",
			kanade:  k + "〈繰り返し〉(「abcdefghij」, 200000)を出力しよう\n",
			wantErr: "too large",
		},
		{
			name:   "reverse works on characters",
			haja:   h + "<거꾸로>(\"가나다\")를 출력하자\n",
			kanade: k + "〈逆さ〉(「あいう」)を出力しよう\n",
			want:   "다나가", wantKanade: "ういあ",
		},
		{
			name:   "an empty prefix always matches",
			haja:   h + "<시작하는지>(\"abc\", \"\")를 출력하자\n",
			kanade: k + "〈始まるか〉(「abc」, 「」)を出力しよう\n",
			want:   "참", wantKanade: "真",
		},
		{
			name:   "join strings",
			haja:   h + "<잇기>([\"가\", \"나\", \"다\"], \", \")를 출력하자\n<잇기>([], \", \")를 출력하자\n",
			kanade: k + "〈連結〉(【「あ」, 「い」, 「う」】, 「、」)を出力しよう\n",
			want:   "가, 나, 다|", wantKanade: "あ、い、う",
		},
		{
			name:    "join needs strings",
			haja:    h + "<잇기>([1, 2], \",\")를 출력하자\n",
			kanade:  k + "〈連結〉(【1, 2】, 「,」)を出力しよう\n",
			wantErr: "list of strings",
		},
		{
			name:   "count does not overlap",
			haja:   h + "<세기>(\"aaaa\", \"aa\")를 출력하자\n",
			kanade: k + "〈数える〉(「aaaa」, 「aa」)を出力しよう\n",
			want:   "2",
		},
		{
			name:    "counting nothing",
			haja:    h + "<세기>(\"abc\", \"\")를 출력하자\n",
			kanade:  k + "〈数える〉(「abc」, 「」)を出力しよう\n",
			wantErr: "cannot be empty",
		},
		{
			name:   "position counts characters, not bytes",
			haja:   h + "<위치>(\"가나다\", \"다\")를 출력하자\n<위치>(\"가나다\", \"라\")를 출력하자\n",
			kanade: k + "〈位置〉(「あいう」, 「う」)を出力しよう\n〈位置〉(「あいう」, 「え」)を出力しよう\n",
			want:   "3|비어있음", wantKanade: "3|空っぽ",
		},
	})
}

func TestListModule(t *testing.T) {
	h := hajaImports("목록", "정렬", "뒤집기", "중복제거", "범위", "평탄화", "조각내기", "짝짓기")
	k := kanadeImports("リスト", "並べ替え", "反転", "重複除去", "範囲", "平坦化", "小分け", "ペア")
	runTypeCases(t, []typeCase{
		{
			name:   "sort strings and leave the original",
			haja:   h + "'원본'을 [\"나\", \"가\", \"다\"]로 정하자\n<정렬>('원본')을 출력하자\n'원본'을 출력하자\n",
			kanade: k + "『元』を【「う」, 「あ」, 「い」】にしよう\n〈並べ替え〉(『元』)を出力しよう\n『元』を出力しよう\n",
			want:   "[가, 나, 다]|[나, 가, 다]", wantKanade: "[あ, い, う]|[う, あ, い]",
		},
		{
			name:   "sorting nothing",
			haja:   h + "<정렬>([])을 출력하자\n",
			kanade: k + "〈並べ替え〉(【】)を出力しよう\n",
			want:   "[]",
		},
		{
			name:    "sorting mixed types",
			haja:    h + "<정렬>([1, \"둘\"])를 출력하자\n",
			kanade:  k + "〈並べ替え〉(【1, 「二」】)を出力しよう\n",
			wantErr: "all numbers or all strings",
		},
		{
			name:   "unique keeps the first and tells 1 from \"1\"",
			haja:   h + "<중복제거>([3, 1, 3, \"1\", 1, \"1\"])를 출력하자\n",
			kanade: k + "〈重複除去〉(【3, 1, 3, 「1」, 1, 「1」】)を出力しよう\n",
			want:   "[3, 1, 1]",
		},
		{
			name:    "unique cannot compare lists",
			haja:    h + "<중복제거>([[1], [1]])을 출력하자\n",
			kanade:  k + "〈重複除去〉(【【1】, 【1】】)を出力しよう\n",
			wantErr: "can be compared here",
		},
		{
			name:   "range counts down by itself and takes a step",
			haja:   h + "<범위>(5, 1)을 출력하자\n<범위>(1, 10, 4)를 출력하자\n<범위>(1, 5, (0 - 1))을 출력하자\n<범위>(3, 3)을 출력하자\n",
			kanade: k + "〈範囲〉(5, 1)を出力しよう\n〈範囲〉(1, 10, 4)を出力しよう\n〈範囲〉(1, 5, (0 - 1))を出力しよう\n〈範囲〉(3, 3)を出力しよう\n",
			want:   "[5, 4, 3, 2, 1]|[1, 5, 9]|[]|[3]",
		},
		{
			name:    "a zero step",
			haja:    h + "<범위>(1, 5, 0)을 출력하자\n",
			kanade:  k + "〈範囲〉(1, 5, 0)を出力しよう\n",
			wantErr: "step cannot be zero",
		},
		{
			name:    "a range that is too long",
			haja:    h + "<범위>(1, 2000000)을 출력하자\n",
			kanade:  k + "〈範囲〉(1, 2000000)を出力しよう\n",
			wantErr: "too large",
		},
		{
			name:   "flatten opens one level only",
			haja:   h + "<평탄화>([1, [2, [3]], 4])를 출력하자\n",
			kanade: k + "〈平坦化〉(【1, 【2, 【3】】, 4】)を出力しよう\n",
			want:   "[1, 2, [3], 4]",
		},
		{
			name:   "chunk and zip",
			haja:   h + "<조각내기>([1, 2], 5)를 출력하자\n<짝짓기>([1, 2, 3], [\"가\", \"나\"])를 출력하자\n<뒤집기>([1, 2, 3])을 출력하자\n",
			kanade: k + "〈小分け〉(【1, 2】, 5)を出力しよう\n〈ペア〉(【1, 2, 3】, 【「あ」, 「い」】)を出力しよう\n〈反転〉(【1, 2, 3】)を出力しよう\n",
			want:   "[[1, 2]]|[[1, 가], [2, 나]]|[3, 2, 1]", wantKanade: "[[1, 2]]|[[1, あ], [2, い]]|[3, 2, 1]",
		},
		{
			name:    "chunks of zero",
			haja:    h + "<조각내기>([1, 2], 0)을 출력하자\n",
			kanade:  k + "〈小分け〉(【1, 2】, 0)を出力しよう\n",
			wantErr: "must be a whole number",
		},
	})
}

func TestEncodingAndHashModules(t *testing.T) {
	h := hajaImports("인코딩", "베이스64인코딩", "베이스64디코딩", "주소인코딩", "주소디코딩") + hajaImports("해시", "SHA256")
	k := kanadeImports("エンコード", "Base64エンコード", "Base64デコード", "URLエンコード", "URLデコード") + kanadeImports("ハッシュ", "SHA256")
	runTypeCases(t, []typeCase{
		{
			name:   "base64 of Korean and back",
			haja:   h + "<베이스64인코딩>(\"한글\")을 출력하자\n<베이스64디코딩>(\"7ZWc6riA\")를 출력하자\n",
			kanade: k + "〈Base64エンコード〉(「日本」)を出力しよう\n〈Base64デコード〉(「5pel5pys」)を出力しよう\n",
			want:   "7ZWc6riA|한글", wantKanade: "5pel5pys|日本",
		},
		{
			name:    "base64 without padding is rejected",
			haja:    h + "<베이스64디코딩>(\"aGk\")를 출력하자\n",
			kanade:  k + "〈Base64デコード〉(「aGk」)を出力しよう\n",
			wantErr: "not valid base64",
		},
		{
			name:    "base64 of bytes that are not text",
			haja:    h + "<베이스64디코딩>(\"/w==\")를 출력하자\n",
			kanade:  k + "〈Base64デコード〉(「/w==」)を出力しよう\n",
			wantErr: "not valid base64",
		},
		{
			name:   "url escaping of Korean and back",
			haja:   h + "<주소인코딩>(\"가 a+b\")를 출력하자\n<주소디코딩>(\"%EA%B0%80%20a%2Bb\")를 출력하자\n",
			kanade: k + "〈URLエンコード〉(「あ a+b」)を出力しよう\n〈URLデコード〉(「%E3%81%82%20a%2Bb」)を出力しよう\n",
			want:   "%EA%B0%80%20a%2Bb|가 a+b", wantKanade: "%E3%81%82%20a%2Bb|あ a+b",
		},
		{
			name:    "a broken percent escape",
			haja:    h + "<주소디코딩>(\"%2\")를 출력하자\n",
			kanade:  k + "〈URLデコード〉(「%2」)を出力しよう\n",
			wantErr: "not valid URL encoding",
		},
		{
			name:    "a percent escape that is not hex",
			haja:    h + "<주소디코딩>(\"%zz\")를 출력하자\n",
			kanade:  k + "〈URLデコード〉(「%zz」)を出力しよう\n",
			wantErr: "not valid URL encoding",
		},
		{
			name:       "sha256 of nothing and of Korean text",
			haja:       h + "<SHA256>(\"\")를 출력하자\n<SHA256>(\"한글\")을 출력하자\n",
			kanade:     k + "〈SHA256〉(「」)を出力しよう\n〈SHA256〉(「日本」)を出力しよう\n",
			want:       "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855|bd87f9bb68b67d2fa1cb82b6751820e946d5b1316d25d5fd96512fb4be44a2a8",
			wantKanade: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855|cf2abf0c5be326cb922a70f8163f91079c4d9aa8655c60ead89ad545c9de2e92",
		},
	})
}
