# hana

[한국어](README.md) | [日本語](README.ja.md)

**Hana**は、2つのプログラミング言語 **Haja**（ハジャ、韓国語で書きます）と **Kanade**（カナデ、日本語で書きます）のGo実装です。インタープリタ、バイトコードコンパイラとVM、コマンドラインツール、言語サーバーが1つのバイナリに入っています。

```haja
'이름'을 [문자열]인 "하자"로 정하자
틀"안녕, {'이름'}!"을 출력하자
```

```kanade
『名前』を【文字列】の「カナデ」にしよう
枠「こんにちは、{『名前』}！」を出力しよう
```

## ビルドと実行

[Go](https://go.dev/dl/) 1.25以上が必要です。

```bash
go build -o hana .
./hana run hello.knd
```

Windowsでは`hana.exe`ができます。プログラムを1つの実行ファイルにまとめる`hana pack`を使うには、小さなランタイム（[`cmd/hana-runtime`](cmd/hana-runtime)）も`hana`の隣に置く必要があります。

```bash
go build -ldflags="-s -w" -trimpath -o hana-runtime ./cmd/hana-runtime
```

## コマンド

| コマンド | できること |
| --- | --- |
| `hana run <ファイル>` | `.hj`（Haja）、`.knd`（Kanade）、`.hn`（コンパイル済みバイトコード）を実行します |
| `hana build <ファイル>` | バイトコード（`.hn`）にコンパイルして保存します |
| `hana disasm <ファイル>` | バイトコードを逆アセンブルした結果を表示します |
| `hana pack <ファイル>` | プログラムを1つの実行ファイルにまとめます。ほかのOS用も作れます（`--target linux-amd64`） |
| `hana init [フォルダ]` | スタートファイルと`.gitignore`のある新しいプロジェクトを作ります（`--lang haja`または`kanade`） |
| `hana lsp` | エディター用の言語サーバー（LSP）を標準入出力で実行します |

`run`は、標準ではツリーウォーク方式のインタープリタで実行し、`--bc`を付けるとバイトコードコンパイラとVMで実行します。`-t`はパースと実行にかかった時間を表示します。`--allow-file=false`と`--allow-net=false`で、`【ファイル】`、`【ソケット】`、`【HTTP】`モジュールを止められます。

コマンドラインが表示する文言（ヘルプや案内）は、韓国語と日本語に対応しています。言語は`--locale ko|ja`、環境変数`HANA_LANG`、扱うスクリプト（`.knd`なら日本語）、システムの言語の順に決まります。スクリプトが出すエラーは、そのスクリプトの言語で表示されます。

## 言語と標準モジュール

Hajaは`'나이'를 [숫자]인 20으로 정하자`のように型を角括弧で書き、助詞（`을`、`를`、`로`、`의`など）が文法の一部になっている言語です。Kanadeは同じ構造を日本語の語順で書きます。2つの言語は同じASTを作るので、実行エンジンは1つを共有しています。

インポートなしで使えるのは構文といくつかの型変換だけで、残りは`【モジュール】から〈道具〉を持ってこよう`で持ってきます。

| モジュール | できること |
| --- | --- |
| `【数学】` `【統計】` | 計算、数字のリストの要約 |
| `【テキスト】` `【リスト】` `【正規表現】` `【パス】` | 文字列、リスト、正規表現、パス文字列の操作 |
| `【JSON】` `【CSV】` `【エンコード】` `【ハッシュ】` | データ形式と変換 |
| `【乱数】` `【日時】` | ランダムな値、時刻（Unix秒） |
| `【ファイル】` `【ソケット】` `【HTTP】` | OSが必要なモジュール（ブラウザエンジンでは使えません） |

モジュールと道具の名前は、[`std`](std)パッケージの[`std.go`](std/std.go)に、言語に依存しないIDと言語別の名前表として置かれています。実際の動作は[`std/stdimpl`](std/stdimpl)に一度だけ実装して、2つのエンジンで共有しています。

## パッケージ

[`packages/`](packages)は、hanaと一緒に配布するパッケージです。Haja/Kanadeのソースで作るか、ネイティブライブラリ（`.dll`/`.so`/`.dylib`、簡単なC ABI）を添えられます。

| パッケージ | できること |
| --- | --- |
| [`http_server`](packages/http_server) | Goの`net/http`によるHTTPサーバー |
| [`timezone`](packages/timezone) | IANAタイムゾーンによる時刻の書式・読み取り・曜日・オフセット |

ネイティブライブラリは、各パッケージの`native/`で`build.sh`（または`build.ps1`）を使って作ります。GoとCコンパイラ（cgo）が必要です。`hana pack`は、プログラムが使うパッケージのライブラリを実行ファイルの隣（`libraries/`）に書き出すか、`--embed`で実行ファイルの中に入れます。

## フォルダ構成

| フォルダ | 内容 |
| --- | --- |
| [`lexer/`](lexer)、[`parser/`](parser)、[`ast/`](ast) | HajaとKanadeのレキサー・パーサー（同じASTを作ります） |
| [`vm/`](vm)、[`stdlib/`](stdlib) | ツリーウォーク方式のインタープリタとその標準ライブラリ |
| [`bytecode/`](bytecode)、[`bcvm/`](bcvm)、[`bcstdlib/`](bcstdlib) | バイトコードコンパイラ、スタックVM、その標準ライブラリ |
| [`std/`](std) | 標準モジュールの名前表（[`std.go`](std/std.go)）と共有実装（[`stdimpl/`](std/stdimpl)） |
| [`errs/`](errs) | 3つの言語（韓国語・日本語・英語）で出るエラー文言のカタログ |
| [`typecheck/`](typecheck)、[`symbol/`](symbol)、[`strcat/`](strcat) | 型検査、名前を整数にしたSymbol、長い文字列の連結 |
| [`native/`](native)、[`pkg/`](pkg)、[`pack/`](pack)、[`runner/`](runner) | ネイティブライブラリのローダー、パッケージのマニフェスト、実行ファイルへのパッケージ化、共通の実行処理 |
| [`lsp/`](lsp) | 言語サーバー（補完、診断、ホバー） |
| [`cmd/`](cmd) | コマンドラインと開発ツール（[`doctest`](cmd/doctest)、[`errsgen`](cmd/errsgen)、[`stdgen`](cmd/stdgen)、[`lspgen`](cmd/lspgen)） |
| [`packages/`](packages)、[`tests/`](tests) | 一緒に配布するパッケージ、統合テスト |

## テスト

```bash
go test ./...
```

ネイティブライブラリを実際に読み込むテストは、Cコンパイラがないとスキップされます。ツリーウォーク方式のインタープリタとバイトコードVMが同じ結果になるかを確認するテストが多くあります。2つのエンジンは常に同じように動く必要があります。[`bench.hj`](bench.hj)は、2つのエンジンの速度を比べるための負荷プログラムです（`hana run -t bench.hj`と`hana run --bc -t bench.hj`）。

## ドキュメントサイトとブラウザエンジン

ドキュメントサイトは別のリポジトリです。サイトの実行ボタンは、このリポジトリのエンジンをTypeScriptに移したブラウザエンジンを使っています。

- Haja: [soumt-r/haja-document](https://github.com/soumt-r/haja-document)
- Kanade: [soumt-r/kanade-document](https://github.com/soumt-r/kanade-document)

2つのリポジトリを`haja-docs`、`kanade-docs`というフォルダ名で`hana`の隣に並べて置くと、次のことができます。

```bash
git clone https://github.com/soumt-r/haja-document haja-docs
git clone https://github.com/soumt-r/kanade-document kanade-docs

# ドキュメント内のすべてのコードブロックを実際に実行して確認
go run ./cmd/doctest ../haja-docs/src/pages/docs
go run ./cmd/doctest ../kanade-docs/src/pages/docs
```

エラー文言、標準モジュールの名前表、補完キーワードは、ここで作ったファイルをドキュメントのリポジトリが使っています（[`cmd/errsgen`](cmd/errsgen)、[`cmd/stdgen`](cmd/stdgen)、[`cmd/lspgen`](cmd/lspgen)）。隣にドキュメントのリポジトリがあれば、`go test`がそのファイルが最新かどうかも検査します。

## 開発ノート

内部構造と守るべきルール（エラー文言の追加方法、標準関数を追加する順序、2つのエンジンを揃える方法など）は、[CLAUDE.md](CLAUDE.md)にまとめています（韓国語）。
