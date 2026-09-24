package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:  "init",
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dir := "."
		if len(args) == 1 {
			dir = args[0]
		}
		lang, _ := cmd.Flags().GetString("lang")
		created, err := scaffold(dir, lang)
		if err != nil {
			fatalf("%v", err)
		}
		for _, path := range created {
			fmt.Printf("✅ %s\n", path)
		}
		fmt.Print(T("init.tryRun", filepath.Join(dir, mainFileFor(lang))))
	},
}

const hariTemplate = `(참고: 시작 파일이에요. 터미널에서 hana run main.hr 로 실행해요.)

'이름'을 [문자열]인 "하리"로 정하자
틀"안녕, {'이름'}!"을 출력하자
`

const kanadeTemplate = `(参考: スタートファイルです。ターミナルで hana run main.knd と実行します。)

『名前』を【文字列】の「カナデ」にしよう
枠「こんにちは、{『名前』}！」を出力しよう
`

// The files a build makes are not source; a project usually keeps them out of git.
const gitignoreTemplate = `*.hn
*.exe
libraries/
`

func mainFileFor(lang string) string {
	if lang == "kanade" {
		return "main.knd"
	}
	return "main.hr"
}

// scaffold writes the files of a new project into dir (created if missing) and
// returns their paths. It never overwrites a file that is already there.
func scaffold(dir, lang string) ([]string, error) {
	template := hariTemplate
	switch lang {
	case "hari":
	case "kanade":
		template = kanadeTemplate
	default:
		return nil, fmt.Errorf("%s", T("init.badLang", lang))
	}
	files := []struct{ name, content string }{
		{mainFileFor(lang), template},
		{".gitignore", gitignoreTemplate},
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	for _, f := range files {
		if _, err := os.Stat(filepath.Join(dir, f.name)); err == nil && f.name != ".gitignore" {
			return nil, fmt.Errorf("%s", T("init.exists", filepath.Join(dir, f.name)))
		}
	}
	var created []string
	for _, f := range files {
		path := filepath.Join(dir, f.name)
		if _, err := os.Stat(path); err == nil {
			continue // a .gitignore that exists is the user's
		}
		if err := os.WriteFile(path, []byte(f.content), 0o644); err != nil {
			return created, err
		}
		created = append(created, path)
	}
	return created, nil
}

func init() {
	initCmd.Flags().String("lang", "hari", "")
	rootCmd.AddCommand(initCmd)
}
