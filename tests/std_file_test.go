package tests

// The [파일] module (native only): every case starts by writing what it needs,
// because the same folder is used by all four engines in turn.

import (
	"os"
	"testing"

	"github.com/soumt-r/hana/std/stdimpl"
)

func TestFileModule(t *testing.T) {
	inTempDir(t, map[string]string{
		"seed.txt":  "첫째\r\n둘째\n\n넷째\n",
		"empty.txt": "",
		"dir/x.txt": "x",
	})
	h := hajaImports("파일", "읽기", "줄읽기", "쓰기", "덧붙이기", "있는지", "폴더인지", "지우기", "목록", "폴더만들기", "옮기기")
	k := kanadeImports("ファイル", "読む", "行で読む", "書く", "追記", "あるか", "フォルダか", "削除", "一覧", "フォルダ作成", "移動")
	runTypeCases(t, []typeCase{
		{
			name:   "write, read back, and append",
			haja:   h + "<쓰기>(\"a.txt\", \"안녕\")을 실행하자\n<덧붙이기>(\"a.txt\", \" 하자\")를 실행하자\n<읽기>(\"a.txt\")를 출력하자\n",
			kanade: k + "〈書く〉(「a.txt」, 「こんにちは」)を実行しよう\n〈追記〉(「a.txt」, 「 カナデ」)を実行しよう\n〈読む〉(「a.txt」)を出力しよう\n",
			want:   "안녕 하자", wantKanade: "こんにちは カナデ",
		},
		{
			name:   "writing again replaces the file",
			haja:   h + "<쓰기>(\"b.txt\", \"긴 내용이에요\")를 실행하자\n<쓰기>(\"b.txt\", \"짧게\")를 실행하자\n<읽기>(\"b.txt\")를 출력하자\n",
			kanade: k + "〈書く〉(「b.txt」, 「長い内容です」)を実行しよう\n〈書く〉(「b.txt」, 「短く」)を実行しよう\n〈読む〉(「b.txt」)を出力しよう\n",
			want:   "짧게", wantKanade: "短く",
		},
		{
			name:   "lines drop line breaks and an empty last line",
			haja:   h + "<줄읽기>(\"seed.txt\")를 출력하자\n<줄읽기>(\"empty.txt\")를 출력하자\n",
			kanade: k + "〈行で読む〉(「seed.txt」)を出力しよう\n〈行で読む〉(「empty.txt」)を出力しよう\n",
			want:   "[첫째, 둘째, , 넷째]|[]",
		},
		{
			name:   "exists and is-a-folder",
			haja:   h + "<있는지>(\"seed.txt\")를 출력하자\n<있는지>(\"nope.txt\")를 출력하자\n<폴더인지>(\"dir\")을 출력하자\n<폴더인지>(\"seed.txt\")를 출력하자\n",
			kanade: k + "〈あるか〉(「seed.txt」)を出力しよう\n〈あるか〉(「nope.txt」)を出力しよう\n〈フォルダか〉(「dir」)を出力しよう\n〈フォルダか〉(「seed.txt」)を出力しよう\n",
			want:   "참|거짓|참|거짓", wantKanade: "真|偽|真|偽",
		},
		{
			name:   "delete removes the file",
			haja:   h + "<쓰기>(\"c.txt\", \"c\")를 실행하자\n<지우기>(\"c.txt\")를 실행하자\n<있는지>(\"c.txt\")를 출력하자\n",
			kanade: k + "〈書く〉(「c.txt」, 「c」)を実行しよう\n〈削除〉(「c.txt」)を実行しよう\n〈あるか〉(「c.txt」)を出力しよう\n",
			want:   "거짓", wantKanade: "偽",
		},
		{
			name:   "list is sorted, mkdir makes the path, move replaces",
			haja:   h + "<폴더만들기>(\"m/n\")을 실행하자\n<쓰기>(\"m/n/b.txt\", \"b\")를 실행하자\n<쓰기>(\"m/n/a.txt\", \"a\")를 실행하자\n<쓰기>(\"m/n/c.txt\", \"c\")를 실행하자\n<옮기기>(\"m/n/a.txt\", \"m/n/c.txt\")를 실행하자\n<목록>(\"m/n\")을 출력하자\n<읽기>(\"m/n/c.txt\")를 출력하자\n",
			kanade: k + "〈フォルダ作成〉(「m/n」)を実行しよう\n〈書く〉(「m/n/b.txt」, 「b」)を実行しよう\n〈書く〉(「m/n/a.txt」, 「a」)を実行しよう\n〈書く〉(「m/n/c.txt」, 「c」)を実行しよう\n〈移動〉(「m/n/a.txt」, 「m/n/c.txt」)を実行しよう\n〈一覧〉(「m/n」)を出力しよう\n〈読む〉(「m/n/c.txt」)を出力しよう\n",
			want:   "[b.txt, c.txt]|a",
		},
		{
			name:    "reading a file that is not there",
			haja:    h + "<읽기>(\"nope.txt\")를 출력하자\n",
			kanade:  k + "〈読む〉(「nope.txt」)を出力しよう\n",
			wantErr: "'nope.txt' not found",
		},
		{
			name:    "reading a folder",
			haja:    h + "<읽기>(\"dir\")을 출력하자\n",
			kanade:  k + "〈読む〉(「dir」)を出力しよう\n",
			wantErr: "'dir' is a folder",
		},
		{
			name:    "writing into a folder that does not exist",
			haja:    h + "<쓰기>(\"none/x.txt\", \"x\")를 실행하자\n",
			kanade:  k + "〈書く〉(「none/x.txt」, 「x」)を実行しよう\n",
			wantErr: "'none/x.txt' not found",
		},
		{
			name:    "writing over a folder",
			haja:    h + "<쓰기>(\"dir\", \"x\")를 실행하자\n",
			kanade:  k + "〈書く〉(「dir」, 「x」)を実行しよう\n",
			wantErr: "'dir' is a folder",
		},
		{
			name:    "deleting a folder that is not empty",
			haja:    h + "<지우기>(\"dir\")을 실행하자\n",
			kanade:  k + "〈削除〉(「dir」)を実行しよう\n",
			wantErr: "Could not complete",
		},
		{
			name:    "the path must be a string",
			haja:    h + "<읽기>(3)을 출력하자\n",
			kanade:  k + "〈読む〉(3)を出力しよう\n",
			wantErr: "Argument 1 must be a string",
		},
	})
}

func TestFileAccessPolicy(t *testing.T) {
	inTempDir(t, map[string]string{"a.txt": "a"})
	old := stdimpl.FileAccess
	t.Cleanup(func() { stdimpl.FileAccess = old })
	stdimpl.FileAccess = stdimpl.DenyFiles

	h := hajaImports("파일", "읽기", "쓰기", "있는지", "목록")
	k := kanadeImports("ファイル", "読む", "書く", "あるか", "一覧")
	var cases []typeCase
	for _, call := range []struct{ h, k string }{
		{"<읽기>(\"a.txt\")를 출력하자\n", "〈読む〉(「a.txt」)を出力しよう\n"},
		{"<쓰기>(\"b.txt\", \"b\")를 실행하자\n", "〈書く〉(「b.txt」, 「b」)を実行しよう\n"},
		{"<있는지>(\"a.txt\")를 출력하자\n", "〈あるか〉(「a.txt」)を出力しよう\n"},
		{"<목록>(\".\")을 출력하자\n", "〈一覧〉(「.」)を出力しよう\n"},
	} {
		cases = append(cases, typeCase{name: call.h, haja: h + call.h, kanade: k + call.k, wantErr: "File access is turned off"})
	}
	runTypeCases(t, cases)
	if _, err := os.Stat("b.txt"); err == nil {
		t.Error("a blocked write must not create the file")
	}
}

// Opening the policy again restores normal behavior, so a stray test cannot
// leave every later test without files.
func TestFileAccessIsAllowedByDefault(t *testing.T) {
	if err := stdimpl.FileAccess("read", "anything"); err != nil {
		t.Errorf("the default policy must allow everything, got %v", err)
	}
}
