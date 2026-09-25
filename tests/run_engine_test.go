package tests

// `hana run` uses the bytecode VM by default and the tree-walker for a program the
// bytecode compiler cannot handle; --bc insists on bytecode, --tree on the tree-walker.
// `-t` names the engine that ran. These run the real binary.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunPicksTheEngine(t *testing.T) {
	dir := t.TempDir()
	plain := filepath.Join(dir, "plain.hr")
	os.WriteFile(plain, []byte("1부터 3까지 반복하자 ('수'):\n    '수'를 이어출력하자\n\"\"를 출력하자\n"), 0o644)
	// a function declared inside a block: only the tree-walker takes it
	nested := filepath.Join(dir, "nested.hr")
	os.WriteFile(nested, []byte("\"앞\"을 출력하자\n만약 (1 == 1) 라면:\n    <안>을 만들자 ():\n        \"안\"을 출력하자\n"), 0o644)
	kanade := filepath.Join(dir, "plain.knd")
	os.WriteFile(kanade, []byte("1から2まで繰り返そう(『数』):\n    『数』を出力しよう\n"), 0o644)

	cases := []struct {
		name  string
		args  []string
		wants []string
	}{
		{"bytecode by default", []string{"run", "-t", plain}, []string{"123", "엔진:  바이트코드", "파싱+컴파일"}},
		{"Kanade too", []string{"run", "-t", kanade}, []string{"1\n2", "エンジン:  バイトコード"}},
		{"--tree uses the tree-walker", []string{"run", "-t", "--tree", plain}, []string{"123", "엔진:  트리워커", "파싱:"}},
		{"what bytecode cannot compile runs on the tree-walker", []string{"run", "-t", nested}, []string{"앞", "엔진:  트리워커"}},
		{"--bc does not fall back", []string{"run", "--bc", nested}, []string{"바이트코드 컴파일 중 오류"}},
		{"--bc and --tree together", []string{"run", "--bc", "--tree", plain}, []string{"--bc와 --tree는 함께 쓸 수 없어요."}},
	}
	for _, c := range cases {
		out := hanaSays(t, []string{"HANA_LANG=" + map[bool]string{true: "ja", false: "ko"}[strings.HasSuffix(c.args[len(c.args)-1], ".knd")]}, c.args...)
		for _, want := range c.wants {
			if !strings.Contains(out, want) {
				t.Errorf("%s: want %q in\n%s", c.name, want, out)
			}
		}
	}
}
