<h1 align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="assets/banner-dark.svg">
    <img alt="hana" src="assets/banner-light.svg">
  </picture>
</h1>

<p align="center">
  <a href="README.md">한국어</a> | <a href="README.ja.md">日本語</a>
</p>

<p align="center">
  <a href="https://github.com/soumt-r/hana/actions/workflows/ci.yml"><img alt="CI" src="https://github.com/soumt-r/hana/actions/workflows/ci.yml/badge.svg"></a>
  <img alt="Go 1.25+" src="https://img.shields.io/badge/go-1.25%2B-00ADD8?logo=go&logoColor=white">
  <a href="LICENSE"><img alt="MIT License" src="https://img.shields.io/badge/license-MIT-ff5c8a"></a>
</p>

<p align="center">
  <b>Hana</b> runs two programming languages written in Korean and Japanese:<br>
  <b>Haja</b> (ハジャ) and <b>Kanade</b> (カナデ). One Go binary.<br>
  <sub>名前の<b>ハナ</b>は、韓国語の「1つ(하나)」と、日本語の「花(はな)」の両方の意味を持ちます。</sub>
</p>

<br>

<table>
<tr>
<th align="center">ハジャ (Haja) · 韓国語</th>
<th align="center">カナデ (Kanade) · 日本語</th>
</tr>
<tr>
<td valign="top">

```haja
<인사>를 만들자 ([문자열]인 '이름'):
    틀"안녕, {'이름'}!"을 출력하자

'이름들'을 [(문자열)목록]인 ["하자", "카나데"]로 정하자
'이름들'의 '이름'마다 반복하자:
    <인사>('이름')을 실행하자
```

</td>
<td valign="top">

```kanade
〈挨拶〉を作ろう(【文字列】の『名前』):
    枠「こんにちは、{『名前』}！」を出力しよう

『名前たち』を【(文字列)リスト】の【「ハジャ」,「カナデ」】にしよう
『名前たち』の『名前』ごとに繰り返そう:
    〈挨拶〉(『名前』)を実行しよう
```

</td>
</tr>
</table>

2つの言語は同じASTを作るので、どちらで書いても実行エンジンを1つ共有します。

<br>

## クイックスタート

[Go](https://go.dev/dl/) 1.25以上が必要です。

```bash
go build -o hana .                       # Windowsでは hana.exe ができます
./hana init hello --lang kanade          # スタートファイルのある新しいプロジェクト
./hana run hello/main.knd
```

ビルドせずに使うには、[Releases](https://github.com/soumt-r/hana/releases)からプラットフォームに合った`hana-<バージョン>-<os>-<arch>`ファイルをダウンロードして展開します。`hana`、`hana-runtime`、標準パッケージ（`packages/`）が入っていて、`hana version`でバージョンを確認できます。ほかのOS用の実行ファイルを`hana pack --target`で作るには、そのプラットフォームの`hana-runtime-<os>-<arch>`ファイルもダウンロードして`hana`の隣に置きます。

ソースからビルドして、プログラムを1つの実行ファイルにまとめる`hana pack`を使うには、小さなランタイム（[`cmd/hana-runtime`](cmd/hana-runtime)）も`hana`の隣に置く必要があります。

```bash
go build -ldflags="-s -w" -trimpath -o hana-runtime ./cmd/hana-runtime
```

ドキュメント: [カナデ ドキュメント](https://kanade.soumt.moe) · [ハジャ ドキュメント](https://haja.soumt.moe)

## コマンド

| コマンド | できること |
| --- | --- |
| `hana run <ファイル>` | `.hj`（Haja）、`.knd`（Kanade）、`.hn`（コンパイル済みバイトコード）を実行します |
| `hana build <ファイル>` | バイトコード（`.hn`）にコンパイルして保存します |
| `hana disasm <ファイル>` | バイトコードを逆アセンブルした結果を表示します |
| `hana pack <ファイル>` | プログラムを1つの実行ファイルにまとめます。ほかのOS用も作れます（`--target linux-amd64`） |
| `hana init [フォルダ]` | スタートファイルと`.gitignore`のある新しいプロジェクトを作ります（`--lang haja`または`kanade`） |
| `hana lsp` | エディター用の言語サーバー（LSP）を標準入出力で実行します |
| `hana add <gitパス>[@バージョン]` | パッケージをプロジェクトに追加し、`hana.json`と`hana-lock.json`に書きます |
| `hana install` | `hana-lock.json`に書かれたパッケージをすべてダウンロードします |
| `hana remove <gitパス>` | パッケージをプロジェクトから外します |
| `hana list` | このプロジェクトが使うパッケージとバージョンを表示します |

`run`は、標準ではツリーウォーク方式のインタープリタで実行し、`--bc`を付けるとバイトコードコンパイラとVMで実行します。`-t`はパースと実行にかかった時間を表示します。`--allow-file=false`と`--allow-net=false`で、`【ファイル】`、`【ソケット】`、`【HTTP】`モジュールを止められます。

コマンドラインが表示する文言（ヘルプや案内）は、韓国語と日本語に対応しています。言語は`--locale ko|ja`、環境変数`HANA_LANG`、扱うスクリプト（`.knd`なら日本語）、システムの言語の順に決まります。スクリプトが出すエラーは、そのスクリプトの言語で表示されます。

## 言語と標準モジュール

Hajaは`'나이'를 [숫자]인 20으로 정하자`のように型を角括弧で書き、助詞（`을`、`를`、`로`、`의`など）が文法の一部になっている言語です。Kanadeは同じ構造を日本語の語順で書きます。

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

外部パッケージはgitのパスが名前です。`hana add github.com/owner/repo`で追加し、`【github.com/owner/repo】から〈道具〉を持ってこよう`で使います。バージョンは最小バージョンとして書き（`hana.json`）、選ばれたバージョンとコミットは`hana-lock.json`に固定されます。パッケージの作り方は[ドキュメントサイト](https://kanade.soumt.moe/docs/hana/packages)にあります。

ネイティブライブラリは、各パッケージの`native/`で`build.sh`（または`build.ps1`）を使って作ります。GoとCコンパイラ（cgo）が必要です。`hana pack`は、プログラムが使うパッケージのライブラリを実行ファイルの隣（`libraries/`）に書き出すか、`--embed`で実行ファイルの中に入れます。

## リポジトリ案内

<details>
<summary><b>フォルダ構成</b></summary>

<br>

| フォルダ | 内容 |
| --- | --- |
| [`lexer/`](lexer)、[`parser/`](parser)、[`ast/`](ast) | HajaとKanadeのレキサー・パーサー（同じASTを作ります） |
| [`vm/`](vm)、[`stdlib/`](stdlib) | ツリーウォーク方式のインタープリタとその標準ライブラリ |
| [`bytecode/`](bytecode)、[`bcvm/`](bcvm)、[`bcstdlib/`](bcstdlib) | バイトコードコンパイラ、スタックVM、その標準ライブラリ |
| [`std/`](std) | 標準モジュールの名前表（[`std.go`](std/std.go)）と共有実装（[`stdimpl/`](std/stdimpl)） |
| [`value/`](value) | リストのように参照で扱う値 |
| [`errs/`](errs) | 3つの言語（韓国語・日本語・英語）で出るエラー文言のカタログ |
| [`typecheck/`](typecheck)、[`symbol/`](symbol)、[`strcat/`](strcat) | 型検査、名前を整数にしたSymbol、長い文字列の連結 |
| [`native/`](native)、[`pkg/`](pkg)、[`pack/`](pack)、[`runner/`](runner) | ネイティブライブラリのローダー、パッケージのマニフェスト、実行ファイルへのパッケージ化、共通の実行処理 |
| [`lsp/`](lsp) | 言語サーバー（補完、診断、ホバー） |
| [`cmd/`](cmd) | コマンドラインと開発ツール（[`doctest`](cmd/doctest)、[`errsgen`](cmd/errsgen)、[`stdgen`](cmd/stdgen)、[`lspgen`](cmd/lspgen)） |
| [`packages/`](packages)、[`tests/`](tests)、[`bench/`](bench) | 一緒に配布するパッケージ、統合テスト、速度測定用のプログラム |

</details>

<details>
<summary><b>テストと速度測定</b></summary>

<br>

```bash
go test ./...
go test ./tests -run XXX -bench Programs -benchtime 4x     # bench/*.hj を2つのエンジンで測ります
```

ネイティブライブラリを実際に読み込むテストは、Cコンパイラがないとスキップされます。ツリーウォーク方式のインタープリタとバイトコードVMが同じ結果になるかを確認するテストが多くあります。2つのエンジンは常に同じように動く必要があります。[`bench.hj`](bench.hj)は、`hana run -t bench.hj`と`hana run --bc -t bench.hj`で2つのエンジンの速度を比べるための負荷プログラムです。

</details>

<details>
<summary><b>ドキュメントサイトとブラウザエンジン</b></summary>

<br>

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

</details>

<details>
<summary><b>開発ノート</b></summary>

<br>

内部構造と守るべきルール（エラー文言の追加方法、標準関数を追加する順序、2つのエンジンを揃える方法など）は、[CLAUDE.md](CLAUDE.md)にまとめています（韓国語）。

</details>

## ライセンス

[MIT License](LICENSE)です。
