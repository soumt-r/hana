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
	"root.short": {"Hana는 하리(Hari)와 카나데(Kanade)를 위한 통합 도구예요.", "Hanaは、ハジャ（Hari）とカナデ（Kanade）のための統合ツールです。"},
	"root.long": {"하자(한국어)와 카나데(일본어) 프로그래밍 언어를 위한 빠르고 유연한 명령줄 도구이자 가상 머신이에요.\n프로그램 실행, 바이트코드 컴파일, 실행 파일로 묶기, 언어 서버까지 한 곳에서 해요.",
		"ハジャ（韓国語）とカナデ（日本語）のプログラミング言語のための、高速で柔軟なコマンドラインツール兼仮想マシンです。\nプログラムの実行、バイトコードへのコンパイル、実行ファイルへのパッケージ化、言語サーバーまで、これ一つでできます。"},
	"flag.locale": {"화면에 나오는 안내 문구의 언어: ko 또는 ja (기본값: 환경 변수 HANA_LANG, 시스템 언어)", "画面に表示する案内文の言語：koまたはja（既定：環境変数HANA_LANG、システムの言語）"},
	"flag.help":   {"%s 도움말", "%sのヘルプ"},

	"run.use":            {"run <파일명.hr|파일명.knd|파일명.hn>", "run <ファイル名.hr|ファイル名.knd|ファイル名.hn>"},
	"run.short":          {"Hari(.hr) 또는 Kanade(.knd) 스크립트를 실행합니다 (.hn이면 이미 컴파일된 바이트코드를 바로 불러와 실행)", "Hari（.hr）またはKanade（.knd）のスクリプトを実行します（.hnなら、コンパイル済みのバイトコードをそのまま読み込んで実行）"},
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

	"build.use":         {"build <파일명.hr|파일명.knd>", "build <ファイル名.hr|ファイル名.knd>"},
	"build.short":       {"Hari/Kanade 스크립트를 바이트코드(.hn)로 컴파일해 파일로 저장합니다", "Hari/Kanadeのスクリプトをバイトコード（.hn）にコンパイルしてファイルに保存します"},
	"build.flag.output": {"출력 파일 경로 (기본값: 입력 파일명 + .hn)", "出力ファイルのパス（既定：入力ファイル名＋.hn）"},
	"build.createFail":  {"출력 파일 생성 오류: %v", "出力ファイルを作成できませんでした: %v"},
	"build.saveFail":    {"바이트코드 저장 오류: %v", "バイトコードを保存できませんでした: %v"},

	"disasm.use":   {"disasm <파일명.hr|파일명.knd|파일명.hn>", "disasm <ファイル名.hr|ファイル名.knd|ファイル名.hn>"},
	"disasm.short": {"Hari/Kanade 스크립트(또는 이미 컴파일된 .hn 파일)를 디스어셈블 결과로 출력합니다", "Hari/Kanadeのスクリプト（またはコンパイル済みの.hnファイル）を逆アセンブルした結果を出力します"},

	"lsp.short": {"에디터용 언어 서버(LSP)를 표준 입출력으로 실행합니다 (Hari .hr, Kanade .knd)", "エディター用の言語サーバー（LSP）を標準入出力で実行します（Hari .hr、Kanade .knd）"},

	"init.use":       {"init [폴더]", "init [フォルダ]"},
	"init.short":     {"새 하리/카나데 프로젝트를 만들어요 (폴더를 안 주면 지금 폴더에 만들어요)", "新しいHari/Kanadeプロジェクトを作ります（フォルダを指定しないと、今のフォルダに作ります）"},
	"init.flag.lang": {"프로젝트의 언어: hari 또는 kanade", "プロジェクトの言語：hariまたはkanade"},
	"init.tryRun":    {"\n실행해 보세요: hana run %s\n", "\n次のコマンドで実行してみましょう: hana run %s\n"},
	"init.badLang":   {"--lang은 hari나 kanade여야 해요: %s", "--langはhariかkanadeにしてください: %s"},
	"init.exists":    {"%s이(가) 이미 있어요. 덮어쓰지 않아요", "%sはすでにあります。上書きしません"},

	"pack.use":          {"pack <파일명.hr|파일명.knd>", "pack <ファイル名.hr|ファイル名.knd>"},
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

	"add.use":               {"add <git 경로>[@버전]", "add <gitパス>[@バージョン]"},
	"add.short":             {"패키지를 프로젝트에 추가해요 (예: hana add github.com/주인/저장소@1.2.0). 버전을 안 주면 가장 높은 버전이에요", "パッケージをプロジェクトに追加します（例：hana add github.com/owner/repo@1.2.0）。バージョンを指定しないと最新のバージョンになります"},
	"add.badVersion":        {"버전은 1.2.3처럼 적어요: %s", "バージョンは1.2.3のように書いてください: %s"},
	"add.done":              {"%s %s을(를) 추가했어요 (hana.json, hana-lock.json)", "%s %sを追加しました（hana.json、hana-lock.json）"},
	"add.alsoNeeded":        {"함께 필요한 패키지 %d개도 내려받았어요", "あわせて必要なパッケージ%d個もダウンロードしました"},
	"install.use":           {"install", "install"},
	"install.short":         {"hana-lock.json에 적힌 패키지를 모두 내려받아요 (hana.json에 비해 lock이 모자라면 버전을 다시 골라요)", "hana-lock.jsonに書かれたパッケージをすべてダウンロードします（hana.jsonに対してlockが足りなければバージョンを選び直します）"},
	"install.done":          {"패키지 %d개를 준비했어요", "パッケージ%d個を用意しました"},
	"remove.use":            {"remove <git 경로>", "remove <gitパス>"},
	"remove.short":          {"패키지를 프로젝트에서 빼요", "パッケージをプロジェクトから外します"},
	"remove.done":           {"%s을(를) 뺐어요", "%sを外しました"},
	"remove.notThere":       {"%s은(는) 이 프로젝트의 의존 패키지가 아니에요", "%sはこのプロジェクトの依存パッケージではありません"},
	"why.use":               {"why <git 경로>", "why <gitパス>"},
	"why.short":             {"패키지가 프로젝트에 들어온 이유를 보여줘요 (직접 추가했는지, 어떤 패키지가 필요로 하는지)", "パッケージがプロジェクトに入っている理由を表示します（直接追加したか、どのパッケージが必要としているか）"},
	"why.direct":            {"%s: 직접 추가했어요 (hana.json)", "%s: 直接追加しました（hana.json）"},
	"why.replacedOnly":      {"%s은(는) 내 폴더로 바꿔 쓰고 있어요 (replace)", "%sは自分のフォルダに置き換えて使っています（replace）"},
	"outdated.use":          {"outdated", "outdated"},
	"outdated.short":        {"더 높은 버전이 나온 패키지를 보여줘요 (*는 직접 추가한 것)", "より新しいバージョンが出ているパッケージを表示します（*は直接追加したもの）"},
	"outdated.none":         {"모두 최신이에요", "すべて最新です"},
	"outdated.hint":         {"올리려면 hana add <경로>@<버전>을 실행하세요", "上げるにはhana add <パス>@<バージョン>を実行してください"},
	"version.use":           {"version", "version"},
	"version.short":         {"hana의 버전을 보여줘요", "hanaのバージョンを表示します"},
	"list.use":              {"list", "list"},
	"list.short":            {"이 프로젝트가 쓰는 패키지와 버전을 보여줘요 (*는 직접 추가한 것)", "このプロジェクトが使うパッケージとバージョンを表示します（*は直接追加したもの）"},
	"list.replaced":         {"→ %s (로컬 폴더)", "→ %s（ローカルフォルダ）"},
	"list.empty":            {"설치된 패키지가 없어요", "インストールされたパッケージはありません"},
	"pkg.noProject":         {"hana.json이 없어요. 먼저 hana add로 패키지를 추가하세요", "hana.jsonがありません。まずhana addでパッケージを追加してください"},
	"add.flag.allowScripts": {"이 패키지의 설치 스크립트를 실행해도 된다고 승인해요 (hana.json의 trustedScripts에 적혀요)", "このパッケージのインストールスクリプトの実行を承認します（hana.jsonのtrustedScriptsに書かれます）"},
	"pkg.native":            {"네이티브 라이브러리를 내려받는 중: %s (%s)", "ネイティブライブラリをダウンロード中: %s（%s）"},
	"pkg.script":            {"설치 스크립트를 실행해요: %s: %s", "インストールスクリプトを実行します: %s: %s"},
	"pkg.skipScript":        {"%s의 설치 스크립트를 실행하지 않았어요 (%s). 믿는다면 hana add <경로> --allow-scripts로 승인하세요", "%sのインストールスクリプトは実行しませんでした（%s）。信頼できるならhana add <パス> --allow-scriptsで承認してください"},
	"pkg.httpsOnly":         {"https 주소만 내려받을 수 있어요", "httpsのアドレスだけダウンロードできます"},
	"pkg.download":          {"내려받는 중: %s %s", "ダウンロード中: %s %s"},

	"pkg.err.GitMissing":         {"git을 찾을 수 없어요. git을 설치하고 PATH에 넣어 주세요", "gitが見つかりません。gitをインストールしてPATHに追加してください"},
	"pkg.err.TagsFailed":         {"%s의 버전 목록을 가져오지 못했어요: %s", "%sのバージョン一覧を取得できませんでした: %s"},
	"pkg.err.NoVersions":         {"%s에는 v1.2.0 같은 버전 태그가 없어요", "%sにはv1.2.0のようなバージョンタグがありません"},
	"pkg.err.VersionMissing":     {"%s에는 %s 버전이 없어요", "%sにはバージョン%sがありません"},
	"pkg.err.DownloadFailed":     {"%s %s을(를) 내려받지 못했어요: %s", "%s %sをダウンロードできませんでした: %s"},
	"pkg.err.CommitMismatch":     {"%s %s은(는) 커밋 %s로 잠겨 있는데 태그가 이제 %s를 가리켜요. 태그가 옮겨졌는지 확인하세요", "%s %sはコミット%sでロックされていますが、タグは今%sを指しています。タグが動かされていないか確認してください"},
	"pkg.err.NameMismatch":       {"%s에 있는 패키지가 hana.pkg.json에서는 자기 이름을 %q라고 해요", "%sにあるパッケージがhana.pkg.jsonでは自分の名前を%qとしています"},
	"pkg.err.BadManifest":        {"%s %s의 hana.pkg.json이 올바르지 않아요: %s", "%s %sのhana.pkg.jsonが正しくありません: %s"},
	"pkg.err.Circular":           {"패키지가 서로를 돌고 돌아 필요로 해요: %s", "パッケージが互いを循環して必要としています: %s"},
	"pkg.err.NativeNoHash":       {"%s는 %s용 네이티브 라이브러리를 내려받으라고 하면서 sha256을 적지 않았어요", "%sは%s用のネイティブライブラリをダウンロードするとしていますが、sha256がありません"},
	"pkg.err.NativeDownloadFail": {"%s의 네이티브 라이브러리를 %s에서 내려받지 못했어요: %s", "%sのネイティブライブラリを%sからダウンロードできませんでした: %s"},
	"pkg.err.NativeHashMismatch": {"%s의 %s용 네이티브 라이브러리가 sha256과 달라요. 파일이 바뀌었을 수 있어요", "%sの%s用ネイティブライブラリがsha256と一致しません。ファイルが書き換えられた可能性があります"},
	"pkg.err.ScriptChanged":      {"%[1]s %[2]s의 설치 스크립트가 승인했던 것과 달라요. 새 스크립트를 확인한 뒤 hana add %[1]s --allow-scripts로 다시 승인하세요", "%[1]s %[2]sのインストールスクリプトが承認したものと異なります。新しいスクリプトを確認してから、hana add %[1]s --allow-scriptsで再承認してください"},
	"pkg.err.ScriptFailed":       {"%s의 설치 스크립트가 실패했어요: %s", "%sのインストールスクリプトが失敗しました: %s"},
	"pkg.err.NotInProject":       {"%s은(는) 이 프로젝트의 패키지가 아니에요", "%sはこのプロジェクトのパッケージではありません"},
	"pkg.err.AuthFailed": {"git이 %s을(를) 가져오지 못했어요. 로그인이 필요한 비공개 저장소이거나 저장소가 없어요 (%s). git에 로그인해 두거나 ssh 키를 등록하고, ssh로 받으려면 HANA_GIT_PROTOCOL=ssh를 설정하세요", "gitが%sを取得できませんでした。ログインが必要な非公開リポジトリか、リポジトリが存在しません（%s）。gitにログインしておくかsshキーを登録し、sshで取得するにはHANA_GIT_PROTOCOL=sshを設定してください"},
	"pkg.err.NotPath":            {"%q은(는) github.com/주인/저장소 같은 패키지 경로가 아니에요", "%qはgithub.com/owner/repoのようなパッケージパスではありません"},
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
	addCmd.Use, addCmd.Short = T("add.use"), T("add.short")
	installCmd.Use, installCmd.Short = T("install.use"), T("install.short")
	removeCmd.Use, removeCmd.Short = T("remove.use"), T("remove.short")
	listCmd.Use, listCmd.Short = T("list.use"), T("list.short")
	whyCmd.Use, whyCmd.Short = T("why.use"), T("why.short")
	outdatedCmd.Use, outdatedCmd.Short = T("outdated.use"), T("outdated.short")
	versionCmd.Use, versionCmd.Short = T("version.use"), T("version.short")

	for cmd, flags := range map[*cobra.Command]map[string]string{
		rootCmd:  {"locale": "flag.locale"},
		runCmd:   {"allow-file": "run.flag.allowFile", "allow-net": "run.flag.allowNet", "timing": "run.flag.timing", "bc": "run.flag.bc"},
		buildCmd: {"output": "build.flag.output"},
		initCmd:  {"lang": "init.flag.lang"},
		addCmd:   {"allow-scripts": "add.flag.allowScripts"},
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
	for _, cmd := range []*cobra.Command{rootCmd, runCmd, buildCmd, disasmCmd, lspCmd, initCmd, packCmd, addCmd, installCmd, removeCmd, listCmd, whyCmd, outdatedCmd, versionCmd} {
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
