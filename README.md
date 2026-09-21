<h1 align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="assets/banner-dark.svg">
    <img alt="hana" src="assets/banner-light.svg">
  </picture>
</h1>

<p align="center"><a href="README.md">한국어</a> | <a href="README.ja.md">日本語</a></p>

<p align="center"><sub>이름 <b>하나</b>는 숫자 하나(one)와 花(はな, 꽃)를 함께 뜻해요.</sub></p>

**Hana** is a Go implementation of two programming languages: **Haja** (하자, written in Korean) and **Kanade** (カナデ, written in Japanese). It is the interpreter, the bytecode compiler and VM, the command line tool, and the language server, in one binary.

하자(Haja)와 카나데(Kanade)를 위한 Go 구현이에요. 인터프리터, 바이트코드 컴파일러와 VM, 명령줄 도구, 언어 서버가 바이너리 하나에 들어 있어요.

```haja
'이름'을 [문자열]인 "하자"로 정하자
틀"안녕, {'이름'}!"을 출력하자
```

```kanade
『名前』を【文字列】の「カナデ」にしよう
枠「こんにちは、{『名前』}！」を出力しよう
```

## 만들고 실행하기

[Go](https://go.dev/dl/) 1.25 이상이 필요해요.

```bash
go build -o hana .
./hana run hello.hj
```

윈도우에서는 `hana.exe`가 만들어져요. 실행 파일 하나로 묶는 `hana pack`을 쓰려면 작은 런타임([`cmd/hana-runtime`](cmd/hana-runtime))도 `hana` 옆에 함께 두어야 해요.

```bash
go build -ldflags="-s -w" -trimpath -o hana-runtime ./cmd/hana-runtime
```

## 명령

| 명령 | 하는 일 |
| --- | --- |
| `hana run <파일>` | `.hj`(하자), `.knd`(카나데), `.hn`(컴파일된 바이트코드)를 실행해요 |
| `hana build <파일>` | 바이트코드(`.hn`)로 컴파일해서 저장해요 |
| `hana disasm <파일>` | 바이트코드를 디스어셈블한 결과를 보여줘요 |
| `hana pack <파일>` | 프로그램을 실행 파일 하나로 묶어요. 다른 운영체제용도 만들 수 있어요(`--target linux-amd64`) |
| `hana init [폴더]` | 시작 파일과 `.gitignore`가 있는 새 프로젝트를 만들어요(`--lang haja` 또는 `kanade`) |
| `hana lsp` | 에디터용 언어 서버(LSP)를 표준 입출력으로 실행해요 |
| `hana add <git 경로>[@버전]` | 패키지를 프로젝트에 추가하고 `hana.json`, `hana-lock.json`에 적어요 |
| `hana install` | `hana-lock.json`에 적힌 패키지를 모두 내려받아요 |
| `hana remove <git 경로>` | 패키지를 프로젝트에서 빼요 |
| `hana list` | 이 프로젝트가 쓰는 패키지와 버전을 보여줘요 |

`run`은 기본으로 트리워킹 인터프리터로 실행하고, `--bc`를 주면 바이트코드 컴파일러와 VM으로 실행해요. `-t`는 파싱과 실행에 걸린 시간을 보여줘요. `--allow-file=false`와 `--allow-net=false`는 `[파일]`, `[소켓]`, `[HTTP]` 모듈을 막아요.

명령줄이 하는 말(도움말과 안내 문구)은 한국어와 일본어를 지원해요. 언어는 `--locale ko|ja`, 환경 변수 `HANA_LANG`, 다루는 스크립트(`.knd`는 일본어), 시스템 언어 순으로 정해요. 스크립트가 낸 오류는 그 스크립트의 언어로 나와요.

## 언어와 표준 모듈

하자는 `'나이'를 [숫자]인 20으로 정하자`처럼 타입을 대괄호로 쓰고, 조사(`을`, `를`, `로`, `의` …)가 문법의 일부인 언어예요. 카나데는 같은 구조를 일본어 어순으로 써요. 두 언어는 같은 AST를 만들어서, 실행 엔진은 하나를 함께 써요.

임포트 없이 쓰는 것은 문법과 몇 가지 형변환뿐이고, 나머지는 `[모듈]에서 <도구>를 가져오자`로 가져와요.

| 모듈 | 하는 일 |
| --- | --- |
| `[수학]` `[통계]` | 계산, 숫자 목록 요약 |
| `[텍스트]` `[목록]` `[정규식]` `[경로]` | 글자, 목록, 정규식, 경로 글자 다루기 |
| `[JSON]` `[CSV]` `[인코딩]` `[해시]` | 데이터 형식과 바꿔 담기 |
| `[무작위]` `[날짜]` | 무작위 값, 시각(유닉스 초) |
| `[파일]` `[소켓]` `[HTTP]` | 운영체제가 필요한 모듈(브라우저 엔진에서는 쓸 수 없어요) |

모듈과 도구의 이름은 [`std`](std) 패키지의 [`std.go`](std/std.go) 한 곳에 언어 중립 ID와 언어별 이름표로 있고, 실제 동작은 [`std/stdimpl`](std/stdimpl)에 한 번만 구현해서 두 엔진이 함께 써요.

## 패키지

[`packages/`](packages)는 hana와 함께 배포하는 패키지예요. 하자/카나데 소스로 만들거나, 네이티브 라이브러리(`.dll`/`.so`/`.dylib`, 간단한 C ABI)를 함께 둘 수 있어요.

| 패키지 | 하는 일 |
| --- | --- |
| [`http_server`](packages/http_server) | Go `net/http` 위의 HTTP 서버 |
| [`timezone`](packages/timezone) | IANA 시간대로 시각 서식·읽기·요일·오프셋 |

외부 패키지는 git 경로가 이름이에요. `hana add github.com/주인/저장소`로 추가하고 `[github.com/주인/저장소]에서 <함수>를 가져오자`로 써요. 버전은 최소 버전으로 적고(`hana.json`), 고른 버전과 커밋은 `hana-lock.json`에 고정해요. 패키지 만드는 법은 [문서 사이트](https://haja.soumt.moe/docs/hana/packages)에 있어요.

네이티브 라이브러리는 각 패키지의 `native/`에서 `build.sh`(또는 `build.ps1`)로 만들어요. Go와 C 컴파일러(cgo)가 필요해요. `hana pack`은 프로그램이 쓰는 패키지의 라이브러리를 실행 파일 옆(`libraries/`)에 내보내거나, `--embed`로 실행 파일 안에 넣어요.

## 폴더 구성

| 폴더 | 내용 |
| --- | --- |
| [`lexer/`](lexer), [`parser/`](parser), [`ast/`](ast) | 하자와 카나데의 렉서·파서(같은 AST를 만들어요) |
| [`vm/`](vm), [`stdlib/`](stdlib) | 트리워킹 인터프리터와 그 표준 라이브러리 |
| [`bytecode/`](bytecode), [`bcvm/`](bcvm), [`bcstdlib/`](bcstdlib) | 바이트코드 컴파일러, 스택 VM, 그 표준 라이브러리 |
| [`std/`](std) | 표준 모듈의 이름표([`std.go`](std/std.go))와 공유 구현([`stdimpl/`](std/stdimpl)) |
| [`errs/`](errs) | 세 언어(한국어·일본어·영어)로 나오는 오류 문구 카탈로그 |
| [`typecheck/`](typecheck), [`symbol/`](symbol), [`strcat/`](strcat) | 타입 검사, 이름을 정수로 바꾼 Symbol, 긴 글자 이어붙이기 |
| [`native/`](native), [`pkg/`](pkg), [`pack/`](pack), [`runner/`](runner) | 네이티브 라이브러리 로더, 패키지 매니페스트, 실행 파일로 묶기, 공통 실행 |
| [`lsp/`](lsp) | 언어 서버(자동완성, 진단, 호버) |
| [`cmd/`](cmd) | 명령줄과 개발 도구([`doctest`](cmd/doctest), [`errsgen`](cmd/errsgen), [`stdgen`](cmd/stdgen), [`lspgen`](cmd/lspgen)) |
| [`packages/`](packages), [`tests/`](tests) | 함께 배포하는 패키지, 통합 테스트 |

## 테스트

```bash
go test ./...
```

네이티브 라이브러리를 실제로 불러 보는 테스트는 C 컴파일러가 없으면 건너뛰어요. 트리워킹 인터프리터와 바이트코드 VM이 같은 결과를 내는지 확인하는 테스트가 많아요. 두 엔진은 언제나 같게 동작해야 해요. [`bench.hj`](bench.hj)는 두 엔진의 속도를 비교하는 부하 프로그램이에요(`hana run -t bench.hj`와 `hana run --bc -t bench.hj`).

## 문서 사이트와 브라우저 엔진

문서 사이트는 별도 저장소예요. 사이트의 실행 버튼은 이 저장소의 엔진을 TypeScript로 옮긴 브라우저 엔진을 써요.

- 하자: [soumt-r/haja-document](https://github.com/soumt-r/haja-document)
- 카나데: [soumt-r/kanade-document](https://github.com/soumt-r/kanade-document)

두 저장소를 `haja-docs`, `kanade-docs`라는 폴더 이름으로 `hana` 옆에 나란히 두면 다음이 동작해요.

```bash
git clone https://github.com/soumt-r/haja-document haja-docs
git clone https://github.com/soumt-r/kanade-document kanade-docs

# 문서 속 모든 코드 블록을 실제로 실행해서 확인
go run ./cmd/doctest ../haja-docs/src/pages/docs
go run ./cmd/doctest ../kanade-docs/src/pages/docs
```

오류 문구, 표준 모듈 이름표, 자동완성 키워드는 여기서 만든 파일을 문서 저장소가 가져다 써요([`cmd/errsgen`](cmd/errsgen), [`cmd/stdgen`](cmd/stdgen), [`cmd/lspgen`](cmd/lspgen)). 옆에 문서 저장소가 있으면 `go test`가 그 파일이 최신인지도 검사해요.

## 개발 노트

내부 구조와 지켜야 할 규칙(오류 문구를 새로 넣는 법, 표준 함수를 추가하는 순서, 두 엔진을 맞추는 법 등)은 [CLAUDE.md](CLAUDE.md)에 정리해 두었어요.

## 라이선스

[MIT License](LICENSE)예요.
