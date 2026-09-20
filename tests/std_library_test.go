package tests

// Behavior of the json / random / datetime / regex native modules, on
// tree-walker/bytecode x haja/kanade (they share one implementation in
// std/stdimpl; the docs sites' browser engines mirror it and compare_tests.ts
// checks them).

import (
	"testing"
	"time"
)

const (
	hajaJSON     = "[JSON]에서 <파싱>을 가져오자\n[JSON]에서 <문자열화>를 가져오자\n"
	kanadeJSON   = "【JSON】から〈パース〉を持ってこよう\n【JSON】から〈文字列化〉を持ってこよう\n"
	hajaRandom   = "[무작위]에서 <실수>를 가져오자\n[무작위]에서 <정수>를 가져오자\n[무작위]에서 <고르기>를 가져오자\n[무작위]에서 <섞기>를 가져오자\n"
	kanadeRandom = "【乱数】から〈実数〉を持ってこよう\n【乱数】から〈整数〉を持ってこよう\n【乱数】から〈選ぶ〉を持ってこよう\n【乱数】から〈シャッフル〉を持ってこよう\n"
	hajaDate     = "[날짜]에서 <서식>을 가져오자\n[날짜]에서 <읽기>를 가져오자\n"
	kanadeDate   = "【日時】から〈書式〉を持ってこよう\n【日時】から〈読み取り〉を持ってこよう\n"
	hajaRegex    = "[정규식]에서 <검사>를 가져오자\n[정규식]에서 <찾기>를 가져오자\n[정규식]에서 <그룹>을 가져오자\n[정규식]에서 <모두찾기>를 가져오자\n[정규식]에서 <치환>을 가져오자\n[정규식]에서 <분할>을 가져오자\n"
	kanadeRegex  = "【正規表現】から〈検査〉を持ってこよう\n【正規表現】から〈検索〉を持ってこよう\n【正規表現】から〈グループ〉を持ってこよう\n【正規表現】から〈全検索〉を持ってこよう\n【正規表現】から〈置換〉を持ってこよう\n【正規表現】から〈分割〉を持ってこよう\n"
)

func TestJSONModule(t *testing.T) {
	runTypeCases(t, []typeCase{
		{
			name:       "parse an object with nesting, booleans and null",
			haja:       hajaJSON + "'값'을 <파싱>(\"{\\\"a\\\": [1, {\\\"b\\\": true}], \\\"c\\\": null}\")로 정하자\n'값'을 출력하자\n",
			kanade:     kanadeJSON + "『値』を〈パース〉(「{\"a\": [1, {\"b\": true}], \"c\": null}」)にしよう\n『値』を出力しよう\n",
			want:       "{a: [1, {b: 참}], c: 비어있음}",
			wantKanade: "{a: [1, {b: 真}], c: 空っぽ}",
		},
		{
			name:   "stringify sorts dictionary keys",
			haja:   hajaJSON + "<문자열화>({\"b\": 1, \"a\": [참, 비어있음, \"x\"]})를 출력하자\n",
			kanade: kanadeJSON + "〈文字列化〉({「b」: 1, 「a」: 【真, 空っぽ, 「x」】})を出力しよう\n",
			want:   `{"a":[true,null,"x"],"b":1}`,
		},
		{
			name:   "stringify with indentation",
			haja:   hajaJSON + "<문자열화>({\"a\": [1, 2], \"b\": {}}, 2)를 출력하자\n",
			kanade: kanadeJSON + "〈文字列化〉({「a」: 【1, 2】, 「b」: {}}, 2)を出力しよう\n",
			want:   "{\n  \"a\": [\n    1,\n    2\n  ],\n  \"b\": {}\n}",
		},
		{
			name:   "numbers keep their shortest form",
			haja:   hajaJSON + "<문자열화>([0.5, 100, 2.25])를 출력하자\n",
			kanade: kanadeJSON + "〈文字列化〉(【0.5, 100, 2.25】)を出力しよう\n",
			want:   "[0.5,100,2.25]",
		},
		{
			name:   "text survives a round trip",
			haja:   hajaJSON + "<파싱>(<문자열화>({\"이름\": \"하나\"}))를 출력하자\n",
			kanade: kanadeJSON + "〈パース〉(〈文字列化〉({「名前」: 「はな」}))を出力しよう\n",
			want:   "{이름: 하나}", wantKanade: "{名前: はな}",
		},
		{
			name:    "invalid JSON",
			haja:    hajaJSON + "<파싱>(\"{\")를 출력하자\n",
			kanade:  kanadeJSON + "〈パース〉(「{」)を出力しよう\n",
			wantErr: "not valid JSON",
		},
		{
			name:    "dictionary keys must be strings",
			haja:    hajaJSON + "<문자열화>({1: 2})를 출력하자\n",
			kanade:  kanadeJSON + "〈文字列化〉({1: 2})を出力しよう\n",
			wantErr: "cannot represent",
		},
		{
			name:    "parse needs a string",
			haja:    hajaJSON + "<파싱>(3)을 출력하자\n",
			kanade:  kanadeJSON + "〈パース〉(3)を出力しよう\n",
			wantErr: "Argument 1 must be a string",
		},
	})
}

func TestRandomModule(t *testing.T) {
	runTypeCases(t, []typeCase{
		{
			name:       "a real number below one",
			haja:       hajaRandom + "'값'을 <실수>()로 정하자\n(('값' >= 0) 그리고 ('값' < 1))을 출력하자\n",
			kanade:     kanadeRandom + "『値』を〈実数〉()にしよう\n(『値』 >= 0)を出力しよう\n",
			want:       "참",
			wantKanade: "真",
		},
		{
			name:   "an integer inside the range",
			haja:   hajaRandom + "'값'을 <정수>(3, 5)로 정하자\n(('값' >= 3) 그리고 ('값' <= 5))를 출력하자\n",
			kanade: kanadeRandom + "『値』を〈整数〉(3, 5)にしよう\n(『値』 >= 3)を出力しよう\n",
			want:   "참", wantKanade: "真",
		},
		{
			name:   "choice comes from the list",
			haja:   hajaRandom + "<고르기>([\"가\"])를 출력하자\n",
			kanade: kanadeRandom + "〈選ぶ〉(【「あ」】)を出力しよう\n",
			want:   "가", wantKanade: "あ",
		},
		{
			name:   "shuffle keeps every element and the original",
			haja:   hajaRandom + "'원본'을 [1, 2, 3, 4]로 정하자\n'섞음'을 <섞기>('원본')으로 정하자\n'섞음'의 '길이'를 출력하자\n'원본'을 출력하자\n",
			kanade: kanadeRandom + "『元』を【1, 2, 3, 4】にしよう\n『混ぜた』を〈シャッフル〉(『元』)にしよう\n『混ぜた』の『長さ』を出力しよう\n『元』を出力しよう\n",
			want:   "4|[1, 2, 3, 4]",
		},
		{
			name:    "minimum above maximum",
			haja:    hajaRandom + "<정수>(3, 1)을 출력하자\n",
			kanade:  kanadeRandom + "〈整数〉(3, 1)を出力しよう\n",
			wantErr: "minimum 3 is greater than the maximum 1",
		},
		{
			name:    "fractions are not whole numbers",
			haja:    hajaRandom + "<정수>(1.5, 3)을 출력하자\n",
			kanade:  kanadeRandom + "〈整数〉(1.5, 3)を出力しよう\n",
			wantErr: "Argument 1 must be a whole number",
		},
		{
			name:    "choosing from an empty list",
			haja:    hajaRandom + "'빈'을 [(아무거나)목록]인 []로 정하자\n<고르기>('빈')을 출력하자\n",
			kanade:  kanadeRandom + "『空』を【(何でも)リスト】の【】にしよう\n〈選ぶ〉(『空』)を出力しよう\n",
			wantErr: "empty list",
		},
	})
}

func TestDatetimeModule(t *testing.T) {
	runTypeCases(t, []typeCase{
		{
			name:   "format reads back what parse wrote",
			haja:   hajaDate + "<서식>(<읽기>(\"2024-03-05 07:08:09\", \"YYYY-MM-DD HH:mm:ss\"), \"YYYY/MM/DD HH:mm:ss\")를 출력하자\n",
			kanade: kanadeDate + "〈書式〉(〈読み取り〉(「2024-03-05 07:08:09」, 「YYYY-MM-DD HH:mm:ss」), 「YYYY/MM/DD HH:mm:ss」)を出力しよう\n",
			want:   "2024/03/05 07:08:09",
		},
		{
			name:   "other characters in a format are kept",
			haja:   hajaDate + "<서식>(<읽기>(\"2024-03-05\", \"YYYY-MM-DD\"), \"YYYY년 MM월 DD일\")을 출력하자\n",
			kanade: kanadeDate + "〈書式〉(〈読み取り〉(「2024-03-05」, 「YYYY-MM-DD」), 「YYYY年MM月DD日」)を出力しよう\n",
			want:   "2024년 03월 05일", wantKanade: "2024年03月05日",
		},
		{
			name:   "weekday is 1 for Monday and 7 for Sunday",
			haja:   hajaDate + "[날짜]에서 <요일>을 가져오자\n<요일>(<읽기>(\"2024-03-04\", \"YYYY-MM-DD\"))를 출력하자\n<요일>(<읽기>(\"2024-03-10\", \"YYYY-MM-DD\"))를 출력하자\n",
			kanade: kanadeDate + "【日時】から〈曜日〉を持ってこよう\n〈曜日〉(〈読み取り〉(「2024-03-04」, 「YYYY-MM-DD」))を出力しよう\n〈曜日〉(〈読み取り〉(「2024-03-10」, 「YYYY-MM-DD」))を出力しよう\n",
			want:   "1|7",
		},
		{
			name:    "a date that does not exist",
			haja:    hajaDate + "<읽기>(\"2024-02-30\", \"YYYY-MM-DD\")를 출력하자\n",
			kanade:  kanadeDate + "〈読み取り〉(「2024-02-30」, 「YYYY-MM-DD」)を出力しよう\n",
			wantErr: "not a date that matches",
		},
		{
			name:    "text that does not fit the format",
			haja:    hajaDate + "<읽기>(\"2024/03/05\", \"YYYY-MM-DD\")를 출력하자\n",
			kanade:  kanadeDate + "〈読み取り〉(「2024/03/05」, 「YYYY-MM-DD」)を出力しよう\n",
			wantErr: "not a date that matches",
		},
		{
			name:    "signs are not digits",
			haja:    hajaDate + "<읽기>(\"2024-+3-05\", \"YYYY-MM-DD\")를 출력하자\n",
			kanade:  kanadeDate + "〈読み取り〉(「2024-+3-05」, 「YYYY-MM-DD」)を出力しよう\n",
			wantErr: "not a date that matches",
		},
	})
}

func TestRegexModule(t *testing.T) {
	runTypeCases(t, []typeCase{
		{
			name:   "hangul is matched as text",
			haja:   hajaRegex + "<찾기>(\"abc한글def\", \"[가-힣]+\")을 출력하자\n",
			kanade: kanadeRegex + "〈検索〉(「abcひらがなdef」, 「[ぁ-ん]+」)を出力しよう\n",
			want:   "한글", wantKanade: "ひらがな",
		},
		{
			name:   "no match is null",
			haja:   hajaRegex + "<찾기>(\"abc\", \"[0-9]+\")를 출력하자\n",
			kanade: kanadeRegex + "〈検索〉(「abc」, 「[0-9]+」)を出力しよう\n",
			want:   "비어있음", wantKanade: "空っぽ",
		},
		{
			name:   "groups that did not take part are null",
			haja:   hajaRegex + "<그룹>(\"b\", \"(a)|(b)\")을 출력하자\n",
			kanade: kanadeRegex + "〈グループ〉(「b」, 「(a)|(b)」)を出力しよう\n",
			want:   "[b, 비어있음, b]", wantKanade: "[b, 空っぽ, b]",
		},
		{
			name:   "replace with groups",
			haja:   hajaRegex + "<치환>(\"홍길동 김철수\", \"([^ ]+) ([^ ]+)\", \"$2 $1\")을 출력하자\n",
			kanade: kanadeRegex + "〈置換〉(「山田 鈴木」, 「([^ ]+) ([^ ]+)」, 「$2 $1」)を出力しよう\n",
			want:   "김철수 홍길동", wantKanade: "鈴木 山田",
		},
		{
			name:   "a dollar sign followed by a letter stays",
			haja:   hajaRegex + "<치환>(\"a1\", \"[0-9]\", \"$x$$\")를 출력하자\n",
			kanade: kanadeRegex + "〈置換〉(「a1」, 「[0-9]」, 「$x$$」)を出力しよう\n",
			want:   "a$x$",
		},
		{
			name:   "a group that does not exist stays as written",
			haja:   hajaRegex + "<치환>(\"a1\", \"[0-9]\", \"<$3>\")을 출력하자\n",
			kanade: kanadeRegex + "〈置換〉(「a1」, 「[0-9]」, 「<$3>」)を出力しよう\n",
			want:   "a<$3>",
		},
		{
			name:   "split drops the separators",
			haja:   hajaRegex + "<분할>(\"a, b,c\", \", ?\")을 출력하자\n",
			kanade: kanadeRegex + "〈分割〉(「a, b,c」, 「, ?」)を出力しよう\n",
			want:   "[a, b, c]",
		},
		{
			name:   "split keeps captured text out of the result",
			haja:   hajaRegex + "<분할>(\"a1b2c\", \"([0-9])\")를 출력하자\n",
			kanade: kanadeRegex + "〈分割〉(「a1b2c」, 「([0-9])」)を出力しよう\n",
			want:   "[a, b, c]",
		},
		{
			name:   "find all with none is an empty list",
			haja:   hajaRegex + "<모두찾기>(\"abc\", \"[0-9]\")를 출력하자\n",
			kanade: kanadeRegex + "〈全検索〉(「abc」, 「[0-9]」)を出力しよう\n",
			want:   "[]",
		},
		{
			name:    "an invalid pattern",
			haja:    hajaRegex + "<검사>(\"a\", \"(\")를 출력하자\n",
			kanade:  kanadeRegex + "〈検査〉(「a」, 「(」)を出力しよう\n",
			wantErr: "not a valid regular expression",
		},
		{
			name:    "the text must be a string",
			haja:    hajaRegex + "<검사>(1, \"a\")를 출력하자\n",
			kanade:  kanadeRegex + "〈検査〉(1, 「a」)を出力しよう\n",
			wantErr: "Argument 1 must be a string",
		},
	})
}

func TestSleepWaitsAndChecksItsRange(t *testing.T) {
	h := hajaImports("날짜", "기다리기")
	k := kanadeImports("日時", "待つ")
	start := time.Now()
	runTypeCases(t, []typeCase{
		{
			name:   "waiting continues afterwards",
			haja:   h + "<기다리기>(0.05)를 실행하자\n\"끝\"을 출력하자\n",
			kanade: k + "〈待つ〉(0.05)を実行しよう\n「終」を出力しよう\n",
			want:   "끝", wantKanade: "終",
		},
		{
			name:    "a negative time",
			haja:    h + "<기다리기>((0 - 1))을 실행하자\n",
			kanade:  k + "〈待つ〉((0 - 1))を実行しよう\n",
			wantErr: "between 0 and 3600 seconds",
		},
		{
			name:    "more than an hour",
			haja:    h + "<기다리기>(3601)을 실행하자\n",
			kanade:  k + "〈待つ〉(3601)を実行しよう\n",
			wantErr: "between 0 and 3600 seconds",
		},
		{
			name:    "the time must be a number",
			haja:    h + "<기다리기>(\"잠깐\")을 실행하자\n",
			kanade:  k + "〈待つ〉(「少し」)を実行しよう\n",
			wantErr: "Argument 1 must be a number",
		},
	})
	// the first case alone waits 0.05s on each of the four engines
	if elapsed := time.Since(start); elapsed < 200*time.Millisecond {
		t.Errorf("four engines waited only %v in total, want at least 200ms", elapsed)
	}
}
