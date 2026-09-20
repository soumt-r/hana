package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/soumt-r/hana/errs"
	"github.com/spf13/cobra"
)

// What the command line itself says — help texts, flag descriptions, progress and
// problem messages — comes in Korean and Japanese. The language is, first match
// wins:
//
//  1. --locale ko|ja
//  2. the HANA_LANG environment variable (ko or ja)
//  3. the script it works on: a .knd file (or `init --lang kanade`) means Japanese
//  4. the system's language, from LC_ALL, LC_MESSAGES, LANG, LANGUAGE — or, on
//     Windows, the user's locale
//
// and Korean when none of them says. Errors a script raises have their own
// language (the script's, see localeFor) and are not part of this.

type message struct{ ko, ja string }

var catalog = map[string]message{
	"root.short": {"Hana는 하자(Haja)와 카나데(Kanade)를 위한 통합 도구예요.", "Hanaは、ハジャ（Haja）とカナデ（Kanade）のための統合ツールです。"},
	"root.long": {"하자(한국어)와 카나데(일본어) 프로그래밍 언어를 위한 빠르고 유연한 명령줄 도구이자 가상 머신이에요.\n프로그램 실행, 바이트코드 컴파일, 실행 파일로 묶기, 언어 서버까지 한 곳에서 해요.",
		"ハジャ（韓国語）とカナデ（日本語）のプログラミング言語のための、高速で柔軟なコマンドラインツール兼仮想マシンです。\nプログラムの実行、バイトコードへのコンパイル、実行ファイルへのパッケージ化、言語サーバーまで、これ一つでできます。"},
	"flag.locale": {"화면에 나오는 안내 문구의 언어: ko 또는 ja (기본값: 환경 변수 HANA_LANG, 시스템 언어)", "画面に表示する案内文の言語：koまたはja（既定：環境変数HANA_LANG、システムの言語）"},
	"flag.help":   {"%s 도움말", "%sのヘルプ"},

	"run.use":            {"run <파일명.hj|파일명.knd|파일명.hn>", "run <ファイル名.hj|ファイル名.knd|ファイル名.hn>"},
	"run.short":          {"Haja(.hj) 또는 Kanade(.knd) 스크립트를 실행합니다 (.hn이면 이미 컴파일된 바이트코드를 바로 불러와 실행)", "Haja（.hj）またはKanade（.knd）のスクリプトを実行します（.hnなら、コンパイル済みのバイトコードをそのまま読み込んで実行）"},
	"run.flag.allowFile": {"[파일] 모듈로 파일과 폴더에 접근하는 것을 허용합니다 (기본 허용, --allow-file=false로 막기)", "【ファイル】モジュールによるファイルとフォルダへのアクセスを許可します（既定は許可、--allow-file=falseで禁止）"},
	"run.flag.allowNet":  {"[소켓]과 [HTTP] 모듈로 네트워크에 접근하는 것을 허용합니다 (기본 허용, --allow-net=false로 막기)", "【ソケット】と【HTTP】モジュールによるネットワークへのアクセスを許可します（既定は許可、--allow-net=falseで禁止）"},
	"run.flag.timing":    {"파싱/실행 시간을 표시합니다", "パース／実行にかかった時間を表示します"},
	"run.flag.bc":        {"트리워킹 대신 바이트코드 컴파일러+VM으로 실행합니다 (.knd도 지원). .hn 파일에는 영향 없음(항상 바이트코드)", "ツリーウォークの代わりにバイトコードコンパイラ＋VMで実行します（.kndにも対応）。.hnファイルには影響しません（常にバイトコード）"},

	"timing.title":        {"⏱  실행 시간", "⏱  実行時間"},
	"timing.parse":        {"파싱", "パース"},
	"timing.parseCompile": {"파싱+컴파일", "パース＋コンパイル"},
	"timing.load":         {"불러오기", "読み込み"},
	"timing.run":          {"실행", "実行"},
	"timing.total":        {"합계", "合計"},

	"build.use":         {"build <파일명.hj|파일명.knd>", "build <ファイル名.hj|ファイル名.knd>"},
	"build.short":       {"Haja/Kanade 스크립트를 바이트코드(.hn)로 컴파일해 파일로 저장합니다", "Haja/Kanadeのスクリプトをバイトコード（.hn）にコンパイルしてファイルに保存します"},
	"build.flag.output": {"출력 파일 경로 (기본값: 입력 파일명 + .hn)", "出力ファイルのパス（既定：入力ファイル名＋.hn）"},
	"build.createFail":  {"출력 파일 생성 오류: %v", "出力ファイルを作成できませんでした: %v"},
	"build.saveFail":    {"바이트코드 저장 오류: %v", "バイトコードを保存できませんでした: %v"},

	"disasm.use":   {"disasm <파일명.hj|파일명.knd|파일명.hn>", "disasm <ファイル名.hj|ファイル名.knd|ファイル名.hn>"},
	"disasm.short": {"Haja/Kanade 스크립트(또는 이미 컴파일된 .hn 파일)를 디스어셈블 결과로 출력합니다", "Haja/Kanadeのスクリプト（またはコンパイル済みの.hnファイル）を逆アセンブルした結果を出力します"},

	"lsp.short": {"에디터용 언어 서버(LSP)를 표준 입출력으로 실행합니다 (Haja .hj, Kanade .knd)", "エディター用の言語サーバー（LSP）を標準入出力で実行します（Haja .hj、Kanade .knd）"},

	"init.use":       {"init [폴더]", "init [フォルダ]"},
	"init.short":     {"새 하자/카나데 프로젝트를 만들어요 (폴더를 안 주면 지금 폴더에 만들어요)", "新しいHaja/Kanadeプロジェクトを作ります（フォルダを指定しないと、今のフォルダに作ります）"},
	"init.flag.lang": {"프로젝트의 언어: haja 또는 kanade", "プロジェクトの言語：hajaまたはkanade"},
	"init.tryRun":    {"\n실행해 보세요: hana run %s\n", "\n次のコマンドで実行してみましょう: hana run %s\n"},
	"init.badLang":   {"--lang은 haja나 kanade여야 해요: %s", "--langはhajaかkanadeにしてください: %s"},
	"init.exists":    {"%s이(가) 이미 있어요. 덮어쓰지 않아요", "%sはすでにあります。上書きしません"},

	"pack.use":          {"pack <파일명.hj|파일명.knd>", "pack <ファイル名.hj|ファイル名.knd>"},
	"pack.short":        {"프로그램을 실행 파일 하나로 묶습니다 (작은 런타임에 바이트코드와 필요한 네이티브 라이브러리를 붙여요)", "プログラムを1つの実行ファイルにまとめます（小さなランタイムにバイトコードと必要なネイティブライブラリを付けます）"},
	"pack.flag.output":  {"만들 실행 파일 경로 (기본값: 입력 파일명, 윈도우용이면 .exe)", "作る実行ファイルのパス（既定：入力ファイル名、Windows用なら.exe）"},
	"pack.flag.runtime": {"바탕이 될 hana-runtime 실행 파일 (기본값: hana 옆의 hana-runtime)", "ベースにするhana-runtime実行ファイル（既定：hanaの隣のhana-runtime）"},
	"pack.flag.embed":   {"네이티브 라이브러리도 실행 파일 안에 넣어 파일 하나로 만듭니다 (기본값: 실행 파일 옆 libraries/ 폴더로 내보냄)", "ネイティブライブラリも実行ファイルの中に入れて1つのファイルにします（既定：実行ファイルの隣のlibraries/フォルダに書き出す）"},
	"pack.flag.target":  {"만들 플랫폼, 예: linux-amd64 (기본값: 지금 플랫폼). 그 플랫폼용 hana-runtime-<플랫폼>이 hana 옆에 있거나 --runtime으로 주어야 해요", "作るプラットフォーム。例：linux-amd64（既定：今のプラットフォーム）。そのプラットフォーム用のhana-runtime-<プラットフォーム>をhanaの隣に置くか、--runtimeで指定してください"},
	"pack.badTarget":    {"--target은 windows-amd64처럼 <운영체제>-<구조> 꼴이어야 해요: %s", "--targetはwindows-amd64のように<OS>-<アーキテクチャ>の形にしてください: %s"},
	"pack.saveFail":     {"바이트코드 저장 오류: %v", "バイトコードを保存できませんでした: %v"},
	"pack.writeFail":    {"실행 파일을 만들지 못했어요: %v", "実行ファイルを作れませんでした: %v"},
	"pack.exportFail":   {"네이티브 라이브러리를 내보내지 못했어요: %v", "ネイティブライブラリを書き出せませんでした: %v"},
	"pack.badManifest":  {"패키지 %s의 설정 파일이 올바르지 않아요: %v", "パッケージ%sの設定ファイルが正しくありません: %v"},
	"pack.noNative":     {"패키지 %s에는 %s용 네이티브 라이브러리가 없어요", "パッケージ%sには%s用のネイティブライブラリがありません"},
	"pack.readFail":     {"패키지 %s의 네이티브 라이브러리를 읽지 못했어요: %v", "パッケージ%sのネイティブライブラリを読み込めませんでした: %v"},
	"pack.noRuntime":    {"%s을(를) hana 옆에서 찾지 못했어요. --runtime으로 위치를 알려 주세요.", "%sがhanaの隣に見つかりません。--runtimeで場所を指定してください。"},
}

// help lists what cobra's own usage template says, with the words that replace it.
var help = []struct {
	english string
	message
}{
	{"Usage:", message{"사용법:", "使い方:"}},
	{"Aliases:", message{"별칭:", "別名:"}},
	{"Examples:", message{"예시:", "例:"}},
	{"Available Commands:", message{"사용할 수 있는 명령:", "使えるコマンド:"}},
	{"Additional Commands:", message{"그 밖의 명령:", "その他のコマンド:"}},
	{"Global Flags:", message{"전역 플래그:", "グローバルフラグ:"}},
	{"Flags:", message{"플래그:", "フラグ:"}},
	{"Additional help topics:", message{"추가 도움말:", "追加のヘルプ:"}},
	{`Use "{{.CommandPath}} [command] --help" for more information about a command.`,
		message{`"{{.CommandPath}} [명령] --help"로 명령의 더 자세한 도움말을 볼 수 있어요.`, `「{{.CommandPath}} [コマンド] --help」でコマンドの詳しいヘルプを見られます。`}},
	// after the sentence above, which contains the same words
	{"{{.CommandPath}} [command]", message{"{{.CommandPath}} [명령]", "{{.CommandPath}} [コマンド]"}},
}

var uiLocale = errs.Korean

// T is the text of a message in the language the command line speaks.
func T(key string, args ...interface{}) string {
	m, ok := catalog[key]
	if !ok {
		return key
	}
	text := m.ko
	if uiLocale == errs.Japanese {
		text = m.ja
	}
	if len(args) == 0 {
		return text
	}
	return fmt.Sprintf(text, args...)
}

func parseLocale(name string) (errs.Locale, bool) {
	name = strings.ToLower(strings.TrimSpace(name))
	switch {
	case strings.HasPrefix(name, "ja"):
		return errs.Japanese, true
	case strings.HasPrefix(name, "ko"):
		return errs.Korean, true
	}
	return 0, false
}

// detectUILocale picks the language from the command line arguments (without the
// program name), the environment and the system, in the order described above.
func detectUILocale(args []string) errs.Locale {
	if loc, ok := parseLocale(flagValue(args, "locale")); ok {
		return loc
	}
	if loc, ok := parseLocale(os.Getenv("HANA_LANG")); ok {
		return loc
	}
	for _, a := range args {
		if strings.HasSuffix(a, ".knd") {
			return errs.Japanese
		}
	}
	if flagValue(args, "lang") == "kanade" {
		return errs.Japanese
	}
	for _, name := range []string{"LC_ALL", "LC_MESSAGES", "LANG", "LANGUAGE"} {
		if loc, ok := parseLocale(os.Getenv(name)); ok {
			return loc
		}
	}
	if loc, ok := parseLocale(userLocaleName()); ok {
		return loc
	}
	return errs.Korean
}

// flagValue is the value of --name given as `--name value` or `--name=value`, or "".
func flagValue(args []string, name string) string {
	for i, a := range args {
		if a == "--"+name && i+1 < len(args) {
			return args[i+1]
		}
		if value, ok := strings.CutPrefix(a, "--"+name+"="); ok {
			return value
		}
	}
	return ""
}

// applyLocale sets every command's and flag's wording in the chosen language. It
// runs once before the arguments are parsed, so the help texts are ready when cobra
// prints them.
func applyLocale() {
	rootCmd.Short, rootCmd.Long = T("root.short"), T("root.long")
	runCmd.Use, runCmd.Short = T("run.use"), T("run.short")
	buildCmd.Use, buildCmd.Short = T("build.use"), T("build.short")
	disasmCmd.Use, disasmCmd.Short = T("disasm.use"), T("disasm.short")
	lspCmd.Short = T("lsp.short")
	initCmd.Use, initCmd.Short = T("init.use"), T("init.short")
	packCmd.Use, packCmd.Short = T("pack.use"), T("pack.short")

	for cmd, flags := range map[*cobra.Command]map[string]string{
		rootCmd:  {"locale": "flag.locale"},
		runCmd:   {"allow-file": "run.flag.allowFile", "allow-net": "run.flag.allowNet", "timing": "run.flag.timing", "bc": "run.flag.bc"},
		buildCmd: {"output": "build.flag.output"},
		initCmd:  {"lang": "init.flag.lang"},
		packCmd:  {"output": "pack.flag.output", "runtime": "pack.flag.runtime", "embed": "pack.flag.embed", "target": "pack.flag.target"},
	} {
		set := cmd.Flags()
		if cmd == rootCmd {
			set = cmd.PersistentFlags()
		}
		for name, key := range flags {
			set.Lookup(name).Usage = T(key)
		}
	}
	for _, cmd := range []*cobra.Command{rootCmd, runCmd, buildCmd, disasmCmd, lspCmd, initCmd, packCmd} {
		cmd.InitDefaultHelpFlag()
		cmd.Flags().Lookup("help").Usage = T("flag.help", cmd.Name())
	}
	rootCmd.SetHelpCommand(&cobra.Command{
		Use:   "help [command]",
		Short: map[errs.Locale]string{errs.Korean: "명령의 도움말을 보여줘요", errs.Japanese: "コマンドのヘルプを表示します"}[uiLocale],
		Run: func(cmd *cobra.Command, args []string) {
			found, _, err := cmd.Root().Find(args)
			if found == nil || err != nil {
				cmd.Root().Help()
				return
			}
			found.Help()
		},
	})

	template := rootCmd.UsageTemplate()
	for _, h := range help {
		word := h.ko
		if uiLocale == errs.Japanese {
			word = h.ja
		}
		template = strings.ReplaceAll(template, h.english, word)
	}
	rootCmd.SetUsageTemplate(template)
}

// registerLocaleFlag adds the flag every command accepts.
func registerLocaleFlag(cmd *cobra.Command) {
	cmd.PersistentFlags().String("locale", "", "")
}
