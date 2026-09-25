# hana — Hari/Kanade Go 런타임

Go로 작성된 '하리(Hari)'와 '카나데(Kanade)' 언어의 인터프리터/컴파일러/CLI입니다. 언어의 규칙은
하자·카나데 문서 사이트의 기술 스펙(개요, AST, 런타임)이 기준이고, 이 문서와 코드 주석의 "스펙 2.2"
같은 번호는 그 스펙의 절 번호입니다 — 구현이 스펙과 다르게 동작하면 버그입니다.

## 코드 수정 규칙

- **소스 파일을 직접 Edit하세요.** `patch1.py`처럼 정규식/문자열 치환으로 소스를 고치는
  1회성 스크립트를 새로 만들지 마세요. 과거 세션에서 이런 스크립트가 147개 쌓여 커밋까지
  되었고, 실행 후 아무도 지우지 않아 저장소가 정리 불가능한 상태가 됐습니다.
- 실험적으로 동작을 확인하려고 만드는 `.hr` 스크립트나 디버그 출력은 커밋하지 마세요.
  `.gitignore`에 `debug_*`, `temp_*` 패턴이 이미 등록되어 있습니다. 재사용 가치가 있는
  예제/회귀 테스트라면 `tests/`에 의미 있는 이름으로 두고 의도적으로 커밋하세요.
- 빌드 산출물(`hana.exe` 등)은 커밋하지 않습니다. `go build ./cmd` 등으로 로컬에서 생성하세요.
- 커밋 전에는 `git status`로 스테이징된 파일을 확인해, 위 규칙에 걸리는 파일이 섞여
  들어가지 않았는지 보세요.

## 언어-무관 설계 (LangConfig / LangProfile)

`vm/config.go`의 `LangConfig`는 VM이 한국어 하드코딩 없이 동작하도록 만든 것입니다.
새 내장 함수명·표시 문자열을 추가할 때는 Go 코드에 한국어 리터럴을 직접 박지 말고
`LangConfig` 필드로 등록하세요 (`KoreanConfig`/`JapaneseConfig` 둘 다 채우기).

### 명령줄이 하는 말: `cmd/i18n.go` 카탈로그

도움말, 플래그 설명, `-t` 시간 표시, `init`/`pack`/`build`의 안내와 오류 문구는 한국어와 일본어로 나옵니다. 문구를 코드에 한국어로 직접 쓰지 말고 `cmd/i18n.go`의 `catalog`에 `{ko, ja}` 쌍으로 넣은 뒤 `T("키", 값…)`으로 쓰세요(서식 동사 개수가 두 언어에서 같아야 하고 `cmd/i18n_test.go`가 검사합니다). 명령과 플래그의 설명은 `Execute()`가 언어를 정한 뒤 `applyLocale()`이 채우므로, 명령 정의에는 `Use`의 이름만 두고 `Short`와 플래그 설명(`""`)은 카탈로그에서 옵니다. 언어는 `--locale`, 환경 변수 `HANA_LANG`, 스크립트(`.knd`)나 `init --lang kanade`, 시스템 언어(`LANG` 등, 윈도우는 사용자 로캘) 순이고 못 정하면 한국어입니다. 스크립트가 낸 런타임 오류의 언어(`localeFor`)는 스크립트의 것이라 이 규칙과 별개입니다. cobra 자체의 문구(`Usage:` 등)는 `help` 표로 바꾸고, 기본 `completion` 명령은 영어 문구뿐이라 끕니다.

### 런타임 에러: `errs` 패키지 (던지는 쪽은 언어를 모른다)

VM/stdlib(트리워커 `vm/`·`stdlib/`, 바이트코드 `bcvm/`·`bcstdlib/`)에서 런타임 에러를 낼 때는
`fmt.Errorf("…한국어…")`를 쓰지 말고 `return nil, errs.New(errs.SomeCode, args...)`로 던지세요.
`errs.Error`는 "무슨 에러인가"(`Code` + 인자)만 담고, 실제 문구는 언어별 카탈로그
(`errs/ko.go` 해요체, `errs/ja.go` です・ます調, `errs/en.go` 폴백)가 갖습니다.
문구가 실제로 텍스트가 되는 경계는 딱 세 곳이고 거기서만 `errs.Localize(locale, err)`를 부릅니다:
`vm/exec_stmt.go`의 TryStatement(잡힌 에러의 `메시지`), `bcvm/errors.go`의 `errorObjectFor`,
`cmd/run.go`의 최상위 출력(+ `errs.LabelText`로 "런타임 오류" 같은 제목도 같은 로케일로).
로케일은 `vm.LangConfig.Locale`/`bcvm.VM.Locale`(`UseJapaneseWords()`가 함께 설정)로 정해집니다.

새 에러를 추가하는 법: `errs/errs.go`에 `Code` 상수를 만들고(`"<Kind>.<Name>"` — Kind가 출력 접두사
`TypeError:` 등이며 언어와 무관하게 영어 유지) `allCodes`에 등록한 뒤 **세 카탈로그 모두**에 문구를
넣으세요. `errs/errs_test.go`가 (1) 세 카탈로그가 `allCodes`와 정확히 일치하는지, (2) 언어별 서식 동사
개수가 같은지, (3) 한국어가 `요.`로, 일본어가 ます/です 계열로 끝나는지(톤 린트)를 검사합니다. 인자 순서가
언어마다 자연스럽지 않으면 영어 템플릿에서 `%[2]s`처럼 인덱스를 쓰세요.

`(*errs.Error).Error()`는 영어를 돌려줍니다(로그·`strings.Contains(err.Error(), "TypeError")` 같은
접두사 검사용). 한국어/일본어 문구를 테스트에서 확인하려면 `errs.Localize`나 잡힌 `메시지`를 보세요.
docs가 이미 인용하는 문구(접근 위반·목록 범위 초과·사전 키 없음·상수 재대입)는 카탈로그가 docs 문구를
그대로 씁니다 — 바꾸면 docs도 같이 고쳐야 합니다. `bcvm`은 일반 문자열 `TypeError`를 `errs`로 안 바꾼
곳(`.hn` 포맷 에러, `unimplemented opcode`)이 일부러 남아 있습니다(개발자/툴링용).

파서 계층도 같은 패턴으로 이미 추상화되어 있습니다: `parser/hari.Parser`는 실제로는
언어-무관한 공유 구현체이고, 어떤 리터럴을 인식할지는 생성자에 넘기는
`parser/hari.LangProfile`이 결정합니다 (`hariProfile`). `parser/kanade`는 이 Parser에
자기 것(`kanadeProfile`)과 `lexer/kanade`가 만든 토큰 스트림을 넘기는 얇은 wrapper일
뿐, 별도 AST나 별도 파서 구현이 아닙니다 — `카나데(Kanade)` 스크립트도 하리와 완전히
같은 `*ast.Program`을 만들어내므로 bytecode 컴파일러/VM도 수정 없이 그대로 씁니다.

### 카나데 구두점은 전각 문자 — ASCII가 아님

**중요**: 카나데는 전각 문자를 씁니다 — `「」`(STRING), `『』`(VAR), `【】`(TYPE·리스트
리터럴 둘 다), `〈〉`(FUNCTION), `枠「...」`(TEMPLATE_STRING). 괄호/제네릭 인자는 ASCII
`()`를 그대로 씁니다(`【(文字列)リスト】`). 이건 취향이 아니라 **실제
kanade-docs(`kanade-docs/src/pages/docs`)가 이렇게 쓰기 때문** — 세션 초반엔 ASCII
구두점(`"`/`'`/`[]`/`<>`)으로 만들었다가, `doctest`를 kanade-docs에 실제로 돌려보고서야
전혀 안 맞는다는 걸 발견하고 전면 재작성했습니다. **kanade-docs의 실제 예제 없이 카나데
문법을 추측으로 만들지 마세요** — 반드시 `kanade-docs/src/pages/docs`의 ` ```kanade `
블록을 먼저 찾아 확인하세요.

이 전각 구두점 때문에 `parser/hari/parser.go`가 예전엔 델리미터를 `literal[1:len-1]`로
하드코딩해서 벗기던 자리 전부가 `LangProfile.DelimLen`(하리 1바이트, 카나데는 모든
델리미터가 3바이트로 통일)을 쓰도록 바뀌었습니다. `TypeOpen`/`TypeClose`도 TYPE 판별용
prefix/suffix 문자를 담습니다(하자 `[`/`]`, 카나데 `【`/`】`).

카나데 렉서(`lexer/kanade`)를 건드릴 때 꼭 알아야 할 것: 일본어 텍스트는 띄어쓰기가
없어서, "조사/마커 단어 + 별도 동사" 처럼 하리 문법을 그대로 토큰 2개로 나누면 일반
IDENT 정규식이 둘을 하나로 욕심껏 삼켜버립니다(예: "동안 반복하자"를 그대로 옮기면
"間繰り返そう"가 통째로 한 IDENT가 됨). 그래서 루프 종류(while/foreach/range)는
동사 리터럴 자체에 融合되어 있고(`KW_LOOP` 정규식 3종, 뒤에서 꺼낸 값 표현식용 KW_FRONT/
KW_BACK도 "から" 변형까지 융합), 접두사 관계인 키워드(`もしくは` vs `もし`, `それ以外ならば`
vs `それ以外なら`)는 반드시 **긴 것부터** 스펙 배열에 넣어야 합니다 — 안 그러면 짧은 쪽이
먼저 매치돼 나머지가 엉뚱한 IDENT로 남습니다(실제로 겪은 버그: `もしくは`가 `もし`+`くは`로
쪼개짐).

카나데는 TYPE_IN(`[타입]인 값`의 "인")과 MemberParticle(멤버 접근의 "의")에 **같은 단어
"の"를 재사용**합니다(kanade-docs 실측). 이 때문에 `parser/hari/parser.go`의
`parseMemberAndCall`이 "の"를 보면 곧장 멤버 접근으로 먹어버리기 전에, 바로 다음이
KW_FRONT/KW_BACK인지(목록 앞/뒤 조작, "『果物』の前に..." 처럼 카나데는 이 위치에도 조사를
붙임) 먼저 살펴보고 아니면 넘기는 lookahead가 들어가 있습니다.

같은 "の" 중복은 `TYPE의 〈메서드〉()`(정적 메서드를 클래스명으로 직접 호출 — kanade-docs
전역에서 흔히 씀)에서도 문제였습니다: `parsePrimary`의 TYPE_IN 분기가 원래 "TYPE_IN 다음은
항상 값"이라고 가정해 TYPE을 통째로 버리고 값만 반환했는데, 다음이 FUNCTION이면 그건
값이 아니라 정적 멤버 접근입니다. 고친 내용:
- `peek(1)`이 FUNCTION이면 TYPE을 버리지 않고 `MemberExpression{Object: TYPE, Property: ...}`로
  남김. `peek(1)`이 또 TYPE이면(kanade-docs가 종종 `TYPE의 TYPE의 〈메서드〉()`처럼 같은
  클래스명을 중복해서 씀) 바깥 TYPE을 버리고 안쪽을 재귀 파싱 — Property가
  MemberExpression 자체가 되는 이중 중첩을 피하기 위함.
- 문제는 TYPE이 **실제 등록된 클래스가 아닐 때**(`【文字列】の〈文字列に〉(123)`처럼 내장
  타입 이름을 장식으로 쓴 빌트인 변환 호출) — 파싱 시점엔 클래스 등록 여부를 알 방법이
  없어서, `vm/eval_expr.go`·`bytecode/compiler.go` 양쪽 CallExpression 평가/컴파일에
  **런타임/컴파일타임 폴백**을 추가: Callee가 `MemberExpression{TypeReference, FunctionReference}`
  형태인데 그 TypeReference가 실제 클래스가 아니면, TYPE을 버리고 FunctionReference만
  평범한 전역 함수 호출로 처리합니다. (안 그러면 `TypeReference` 평가가 미등록 타입 이름을
  그냥 문자열로 반환해버려서 — 그 문자열에 대고 `BoundStringMethod`를 만들려다
  `MethodNotFoundError`로 깨짐.)

인덱싱 관련해서도 실제 문서 기준으로 두 개 더 고쳤습니다:
- **`番目`(서수 조사)가 카나데 PARTICLE 정규식에 빠져 있었음** — `길이`/`값`과 달리 이건
  토큰화 자체가 안 돼서 일반 IDENT로 떨어지고, 그 뒤로 엉뚱한 SOV 컴포넌트가 하나 더
  생겨 파싱이 깨졌습니다(하리의 PARTICLE 목록엔 "번째"가 이미 있었음 — 카나데 쪽만 놓침).
- **인덱싱 뒤에 조사로 연결된 장식용 "の値"**(`『果物』の1番目の値` = "그 목록의 1번째")를
  `parseMemberAndCall`에서 미리 감지해 건너뛰게 했습니다. 하리는 "1번째 값"처럼 조사 없이
  붙여 써서 바깥 조사 수집 루프가 그냥 버리는데, 카나데는 "의"로 연결해 일반 멤버 접근처럼
  보이는 바람에 그냥 두면 인덱싱 결과(문자열)에 없는 '값'이라는 필드를 찾으려 듭니다.
- **"値にしよう"가 통째로 IDENT 하나로 삼켜짐** — `もしくは`/`もし`와 같은 원인(짧은 예약어
  뒤에 띄어쓰기 없이 동사가 바로 붙음). "値"(PoppedValueWord)에 전용 정규식을 먼저 넣어서
  해결. **새 카나데 짧은 예약어를 추가할 때마다 이 함정을 의심하세요** — 뒤에 동사가 바로
  붙는 자리라면 generic IDENT보다 먼저 매치되는 자기만의 정규식이 필요합니다.
- **동적 리플렉션(`<'변수'>()`/`〈『変数』〉()`, 스펙 3.8)이 ASCII 홑따옴표 `'`에
  하드코딩**돼 있었음(`vm/eval_expr.go` 두 곳 — `FunctionReference` 케이스와
  `MemberExpression`의 메서드-이름 케이스, 둘 다 따로 체크). `LangConfig.VarQuoteOpen/
  VarQuoteClose`("`'`"/"`'`" 하자, `"『"`/`"』"` 카나데)로 일반화. 트리워커 전용 —
  바이트코드는 원래도 지원 범위 밖(`bytecode.Compiler`의 CallExpression 주석 참고).

**리스트/문자열의 "길이"·문자열 유사메서드 4개도 전부 LangConfig화했습니다**(전엔
`vm/eval_expr.go`·`bcvm/vm.go` 양쪽에 `"길이"`/`"자르기"`/`"바꾸기"`/`"분리하기"`/
`"포함확인"`이 하드코딩): `LangConfig.LengthWord`, `StringSliceMethod`,
`StringReplaceMethod`, `StringSplitMethod`, `StringContainsMethod`. `bcvm.VM`은
`vm.LangConfig` 같은 구조체가 없어서 필드를 직접 들고(`LengthWord`, 4개 문자열메서드,
+ `NullString`/`TrueString`/`FalseString`/`ObjectFormat` — `formatValue`도 하드코딩이라
같이 고침) `vm.UseJapaneseWords()` 한 번 호출로 전부 카나데 값으로 바꿉니다 — 호출부마다
필드를 하나씩 따로 설정하지 않는 이유는 바로 아래 문단 때문입니다.

**언어별 값을 여러 곳에 나눠 적는 데서 생긴 실수**: `PluralSelfWords`("우리"에 해당,
카나데는 "私たち")를 `parser/kanade`(파싱용)에서만 고치고 `vm.JapaneseConfig`·
`bytecode.kanadeLang`(둘 다 런타임/컴파일용)에는 예전 값("私達")이 그대로 남아있던 적이
있습니다 — 정적 필드 선언은 파서만 거치니 통과했는데 정적 필드 *대입*(`'私たち'의 X에
Y를 더하자`처럼 런타임에 평가되는 경로)은 조용히 "Variable '私たち' not found"로
깨졌습니다. 같은 어휘가 세 곳(`parser/kanade`, `vm/config.go`, `bytecode/compiler.go`)에
따로 적혀 있다는 걸 기억하고, 카나데 단어 하나를 고칠 땐 **세 곳 다** 확인하세요. 이게 바로
위 `bcvm.UseJapaneseWords()` 같은 "한 번에 묶어서 설정" 헬퍼를 만든 이유이기도 합니다 —
흩어진 필드 중 하나를 빠뜨리는 게 실수의 반복되는 형태였습니다.

현재 상태: **`go run ./cmd/doctest ../kanade-docs/src/pages/docs`로 실제 문서 66블록
전부(100%) 통과**, `../hari-docs/...`도 여전히 66/66. `hana/tests/kanade_test.go`
18개(트리워커 12개, 바이트코드 6개 — 정적 메서드 호출, 빌트인 변환, 서수+장식값 인덱싱,
문자열 유사메서드, 동적 리플렉션 등 이번에 고친 것마다 회귀 테스트 추가)도 전부 통과.
오류 객체를 출력하면 `[클래스 객체]` 형태로 찍히고 `__toString__` 같은 변환 훅은 없습니다 — 스펙에 없는 기능이라
지원하지 않는 것이 의도입니다(이전에는 한계로 적어 두었으나 버그가 아님). `bytecode.compileForEach`의 기본 항목 이름은
`bcLang.defaultItemName`으로 언어별로 처리합니다(`tests/bytecode_parity_test.go`가 지킴).

`hana/parser/hari/langprofile.go`의 `LangProfile`이 지금까지 뽑아낸 언어별 항목 전체
목록(새 문법 추가 시 참고): ErrorLiterals, PluralSelfWords, ConditionThenWords,
PoppedValueWord, MemberParticle, TypeInWord, TemplatePrefix/Suffix,
ConstructorFunctionName, FrontMarker, AccessModifierFromVerb, IsConstVerb,
IsPrintInlineVerb, ClassifyLoop, NormalizeCompareOpSOV/SVO, ImportAsParticles,
DelimLen, TypeOpen/TypeClose.

## 바이트코드 쪽도 카나데를 지원함

트리워커와 별개로 `bytecode`/`bcvm`/`bcstdlib`도 카나데를 지원합니다 — 처음엔
`bcstdlib`가 한국어 빌트인 이름만 등록하고 `bytecode.Compiler`가 내장 `[오류]` 클래스를
항상 하드코딩된 한국어 소스로 컴파일해서 `hana run --bc`/`hana build`를 `.knd`에
막아뒀었는데, 그 두 곳을 마저 연동했습니다:

- `bcstdlib.LangConfig`(+ `Korean`/`Japanese` 값) — `RegisterStandardLibrary(vm, cfg...)`.
  `vm.LangConfig`를 재사용하지 않고 별도 타입으로 둔 이유는 bcstdlib 패키지 설명 그대로
  (두 엔진의 callable/value 표현이 호환 안 됨) — 값만 두 곳에 나눠 적혀 있으므로 이름
  하나 추가/변경 시 양쪽 다 고쳐야 함을 기억하세요.
- `bytecode.Compiler`에 `NewKanadeCompiler()` 추가(`NewCompiler()`는 그대로 하자용).
  내부적으로 `lang *bcLang`(selfWord/pluralSelfWord/parseExpr/builtinErrorClass)을 들고
  다니며 자기참조(`나`/`私`), 정적참조(`우리`/`私たち`), 템플릿 문자열 `{}` 보간용
  렉서/파서, 내장 오류 클래스(하리는 `[오류]`+`'메시지'`, 카나데는 `【エラー】`+
  `『メッセージ』`)를 결정합니다. `compileLocalImport`(로컬 파일 임포트)는 이 필드가 아니라
  **가져오는 파일 자신의 확장자**(`.knd` 여부)로 렉서/파서를 고르므로, 하리 파일이
  카나데 파일을 가져오거나 그 반대도 됩니다.
- `.hn` 포맷(`bytecode/serialize.go`)이 v2로 올라가면서 매직+버전 뒤에 1바이트
  `Lang`(`LangHari`/`LangKanade`)이 추가됐습니다 — `.hn`엔 원래 소스 언어 정보가 전혀
  없어서, `hana build`로 만든 `.knd` 유래 `.hn`을 나중에 `hana run`이 열 때 어떤
  bcstdlib 이름 세트를 등록해야 하는지 알 방법이 없었기 때문입니다. `Encode`/
  `ReadProgram` 시그니처가 이 때문에 바뀌었습니다(`Encode(w, lang)`,
  `ReadProgram(r) (*Program, Lang, error)`). v1로 저장된 `.hn`은 이제 못 읽지만, 이
  변경 시점에 저장소에 커밋되거나 남아있는 `.hn`이 없었으므로 실질적인 하위 호환 이슈는
  없었습니다.

`hana/tests/kanade_test.go`(총 18개, 트리워커/바이트코드 나눠서)에 `TestKanadeBytecode*`
접두사로 바이트코드 전용 케이스들이 있습니다. `disasm`은 `.knd`를 항상 지원했고
(bcstdlib을 안 건드리는 읽기 전용 작업이라), 이제 `run --bc`/`build`도 `.knd`를 지원합니다.

## 리스트 pseudo-method `<비우기>`

스펙 2.6: 리스트는 기본적으로 객체지향 메서드가 아니라 네이티브 구문(추가하자/꺼내자)으로
조작합니다. 예외가 딱 하나 있는데, TS 참조 구현(hari-docs)에도 있는 `<비우기>()`(카나데:
`<空にする>()`, 목록을 즉석에서 비움, 인자 없음, 반환값 없음)입니다.

트리워커는 `vm/eval_expr.go`가 리스트에 대한 `FunctionReference` 접근을 `BoundListMethod`로 만들고
`CallExpression`에서 `비우기`만 받아 그 목록을 제자리에서 비웁니다(`Target`은 상수 변수 검사용).
바이트코드는 컴파일러가 `X의 <비우기>()` 패턴(메서드 이름이 `c.lang.listClearMethod`이고 인자 0개)을 감지해
`compileConstCheck → compileExpression(목록) → LIST_CLEAR → PUSH_NULL`로 컴파일합니다. 메서드 *이름*으로
컴파일 타임에 감지하지만 `LIST_CLEAR` 자체가 런타임에 값 타입을 다시 확인하므로, 목록이 아닌 값에 대해서는
조용히 오작동하는 대신 TypeError로 떨어집니다(같은 이름의 메서드를 가진 커스텀 클래스의 `<비우기>`가 가려지는 것이
알려진 트레이드오프이고, TS 참조 구현은 애초에 이 경우가 생기지 않습니다).

## 목록은 참조 타입(`value.List`)입니다 (런타임 스펙 1.2)

목록 값은 `[]interface{}`가 아니라 `*value.List`(`value/list.go`, `Items` 슬라이스 하나)입니다. 목록은 객체라서 변수·매개변수·필드가 같은 목록을 가지면 하나를 공유합니다: 함수 안에서 `추가하자`/`꺼내자`/`<비우기>()`/`'목'의 1번째를 …로 정하자`를 하면 부른 쪽의 목록이 바뀝니다(사전·객체는 원래 참조). 목록은 `List.Push/Pop`으로 제자리에서 바꾸고, 예전의 "새 목록을 만들어 원래 자리에 되쓰기"(`assignListBack`, `SET_LIST_VAR`가 값을 저장하던 것)는 없어졌습니다. 새 목록을 만드는 표준 라이브러리 함수는 `value.NewList(out)`를 돌려주고 `listArg`는 `Items`를 줍니다(목록 안의 목록은 `*value.List`로 그대로 두어야 별칭이 유지됩니다). `==`는 목록의 참조가 같은지를 봅니다(객체와 같음). 목록이 자기 자신을 담을 수 있으므로 `FormatValue`와 JSON 쓰기는 깊이 한도(`maxFormatDepth`, `maxJSONDepth`)가 있습니다.

- **선언한 타입과 추가**: 추가는 먼저 목록을 바꾸고, 그 목록을 담은 변수·필드의 선언 타입에 맞는지 **새 원소만** 검사하며(`CheckAppended`), 안 맞으면 되돌리고(`Unpush`) 오류를 냅니다. 트리워커는 `checkListPush`, 바이트코드는 `LIST_PUSH`가 목록을 스택에 남기고 `SET_LIST_VAR`(변수)/`CHECK_LIST_FIELD`(객체 필드)가 검사·되돌리기를 합니다. 별칭으로 추가하면 그 이름의 타입만 검사됩니다.
- **상수**: `고정하자` 변수가 든 목록의 추가·꺼내기·비우기는 `ConstantAssignmentError`(스펙 2.1)입니다 — 트리워커 `requireMutable`, 바이트코드 `CHECK_CONST_VAR`(대상이 변수일 때만, 컴파일러의 `compileConstCheck`). 원소 대입(`'상수목록'의 1번째를 …`)은 막지 않습니다.
- **반복**: `마다 반복하자`는 시작할 때의 목록을 복사해 돕니다(트리워커·`TO_ITERABLE`·브라우저 엔진 모두). 반복 중에 추가해도 끝나지 않는 반복이 생기지 않습니다.
- `.hn` 형식은 `LIST_PUSH`/`LIST_POP`/`LIST_CLEAR`의 뜻이 바뀌어 버전 3입니다. 네이티브 라이브러리 ABI(JSON)는 그대로이고 `native`가 목록과 JSON 배열을 오갑니다.
- 브라우저 엔진(TS)은 JS 배열이 원래 참조라 같은 규칙으로 맞췄습니다(`listOps.ts`: `pushOnto`/`takeBack`/`popFromList`/`requireMutable`/`checkListPush`, 반복은 `Array.from`으로 복사). `tests/list_reference_test.go`와 `compare_tests.ts`의 '반복: 목록 참조:'가 지킵니다.

## 바이트코드 VM은 이름을 `Symbol`로 찾음

변수와 멤버를 문자열로 찾던 것을 프로세스 전체에서 유일한 정수 `symbol.Symbol`(`symbol/symbol.go`)로 바꿨습니다: `bcvm/frame.go`의 프레임은 `Symbol`을 비교하고(큰 프레임은 `Symbol`로 인덱싱하는 표), 클래스 멤버 캐시도 `Symbol`이 열쇠입니다.
`.hn` 파일에는 여전히 이름 문자열만 있고(형식 변경 없음), `Chunk.Symbols()`/`Function.ParamSymbols()`가 처음 실행할 때 한 번 계산해 둡니다. VM에서 이름 글자가 필요한 곳(오류 문구, 클래스 이름으로 찾기)만 `Symbol.String()`을 씁니다.
새 명령을 만들 때 변수 이름을 다루면 `chunk.Names[i]` 대신 `syms[i]`(exec 첫머리의 `chunk.Symbols()`)를 쓰세요.
트리워커도 같은 `symbol.Symbol`을 씁니다: `ast.Identifier.Symbol()`/`ast.FunctionReference.Symbol()`이 노드에 한 번 계산해 두고(`FunctionReference`는 동적 리플렉션이 `Name`을 바꿀 수 있어 이름이 같을 때만 재사용), `Environment`의 `GetSym`/`DeclareSym`/`AssignSym`이 Symbol로 찾습니다. 글자만 가진 곳(임포트, 내장 함수 등록)은 `Get`/`Declare`/`Assign`이 대신 Intern합니다. `HariObject.class`(`Interpreter.classOf`)가 클래스 선언을 기억하고 클래스 멤버 캐시는 (클래스, Symbol)이 열쇠입니다.

## 패키지 형식(`hana.pkg.json`)과 네이티브 라이브러리(`.dll`/`.so`/`.dylib`)

무엇을 `std`에 두고 무엇을 패키지로 두는지는 이렇게 나눕니다. `std`는 대부분의 프로그램이 쓰고, 작고, 무거운 의존성이 없고, 브라우저 엔진과 같은 결과가 필요하거나 언어별 이름이 필요한 것(`[수학]`, `[JSON]`, `[파일]`)이고, 패키지는 그 밖의 전부(크거나, 선택적이거나, 다른 언어로 만든 네이티브 라이브러리가 필요한 것)입니다. `std`는 `dlopen`도 C 컴파일러도 필요 없는 단일 바이너리를 지키는 쪽이고, 패키지는 hana를 다시 빌드하지 않고 늘리는 쪽입니다. 어떤 패키지가 거의 모든 프로그램이 쓰는 것이 되면 `std`로 올립니다. 패키지 폴더는 실행한 폴더의 `./packages` → `$HANA_PACKAGES` → `hana` 실행 파일 옆 `packages/` 순으로 찾습니다(`pkg.Dir`; 엔진 코드에 `"packages"` 경로를 직접 쓰지 마세요).

`packages/<모듈>/`은 hana와 **함께 배포하는 builtin 패키지**입니다(`hana add` 같은 설치 명령과는 무관 — 서드파티 설치는 나중의 별개 주제). `pkg` 패키지(`pkg/manifest.go`)가
그 안의 `hana.pkg.json`(이름·버전·언어별 진입점 `entry`·플랫폼별 미리 빌드한 네이티브 파일 `native`·`dependencies`)을 읽고 검증합니다. 매니페스트는
없어도 되고(없으면 `hari/index.hr`, `kanade/index.knd`와 옛 `<모듈>.dll` 관례), 있으면 두 엔진이 진입점 경로와 이 플랫폼의 네이티브 파일을 거기서 정합니다.
잘못된 매니페스트는 `ImportManifestInvalid`, 이 플랫폼용 파일이 없으면 `ImportNativeMissing`입니다. `url`/`sha256`은 나중에 내려받기가 생기면 쓸 자리이고
`hana run`은 안 봅니다.

네이티브 라이브러리는 **ABI v1**을 따르면 어떤 언어로든 만들 수 있고(규칙은 `native/native.go` 맨 위) 엔진에 묶이지 않습니다: `HanaAlloc`/`HanaFree`(필수),
`HanaInit(hostcall)`(콜백이 필요할 때), 그리고 함수마다 `char* Name(char* argsJSON)` → `{"ok": 값}`/`{"error": "글"}` 봉투. hana 함수를 인자로 넘기면
`{"$fn": "cb_1"}`로 가고(같은 함수는 같은 id를 다시 씀), 라이브러리가 `hostcall(id, argsJSON)`을 부르면 그 함수를 실행해 봉투를 `HanaAlloc` 메모리에 돌려줍니다.
**콜백은 프로그램과 절대 동시에 hana 코드를 실행하지 않습니다**: 엔진(`native.Host`를 구현한 `vm.Interpreter`, `bcvm.VM`)이 실행하는 동안 `native.ExecLock`을 쥐고 있다가
네이티브 함수 안에서 기다리는 동안(`Host.BeginNative`)에만 풀기 때문에, 다른 스레드의 콜백은 프로그램이 그런 호출 안에 있거나 끝날 때까지 차례를 기다립니다
(`tests/native_plugin_test.go`의 `TestCallbacksWaitForTheProgramToBlock`이 지킴 — 락을 빼 보면 실패합니다). 소스 패키지 안에서는
`[모듈]에서 <네이티브_이름>을 가져오자`(`LangConfig.NativePrefix`)로 그 함수를 들여옵니다.

**두 엔진 모두 패키지 폴더 임포트를 지원합니다.** 트리워커는 실행 중에(`vm/import.go`), 바이트코드는 컴파일 때(`bytecode/package_import.go`) 진입점을 컴파일해 요청한
함수·클래스를 합치고, 그 진입점의 네이티브 바인딩은 임포트 자리에서 `IMPORT_NATIVE`(`ImportOperand.Library`)로 실행 시점에 묶습니다. 함수를 값으로 넘기는
`<함수>`는 `PUSH_FUNC_REF`입니다. 요청한 항목만 합치므로 진입점 안의 도우미 함수는 함께 임포트해야 합니다(트리워커와 같은 제한).

로더는 OS별로 갈립니다: Windows는 `syscall`(`native/raw_windows.go`), Linux/macOS는 `github.com/ebitengine/purego`(`native/raw_unix.go`) — cgo 없이 `dlopen`하므로
`hana`는 계속 `CGO_ENABLED=0`으로 어느 OS에서나 빌드됩니다(FreeBSD 등은 `raw_other.go`가 `ImportPluginUnsupported`). **엔진 쪽 파일(`vm/`, `bcvm/`)에 OS 전용 API를
넣지 마세요.** 확인: `CGO_ENABLED=0 GOOS=linux go build .`(darwin도). 네이티브 라이브러리를 실제로 불러 보는 테스트(`tests/native_plugin_test.go`,
`tests/http_server_test.go`)는 두 엔진에서 돌고 C 컴파일러가 있어야 하며 없으면 건너뜁니다. Linux는 WSL에서 Go를 깔아 cgo 켠/끈 두 방식으로 통과를 확인했고,
**macOS는 아직 실제로 돌려 보지 못했습니다**. `hana` 내장 네이티브 모듈(`[수학]`, `[파일]` 등)은 이 경로가 아니라 바이너리에 컴파일되는 `std`입니다.

`packages/timezone/`은 시간대 패키지입니다(IANA 데이터베이스가 `time/tzdata`로 라이브러리 안에 있어 컴퓨터에 따로 필요 없음, 약 2.7MB). `std`가 아니라 패키지인 이유는 3.5의 기준대로 데이터가 크고 서머타임 규칙까지 브라우저 엔진과 맞춰야 해서입니다. 네이티브 함수 `Offset`·`Format`·`Parse`·`Weekday`(`native/main.go`)를 `hari/index.hr`·`kanade/index.knd`가 `<시간대오프셋>`·`<시간대서식>`·`<시간대읽기>`·`<시간대요일>`(카나데 `〈時間帯オフセット〉` …)로 감쌉니다. 시각은 [날짜]와 같은 유닉스 초이고 서식 토큰(`YYYY MM DD HH mm ss`)과 검증도 같습니다. 시간대는 IANA 이름이나 `UTC`나 고정 오프셋(`+09:00`)이고 컴퓨터의 시간대(`""`, `Local`)는 일부러 거절합니다. 읽기에서 시계가 건너뛴 시각은 에러, 두 번 나오는 시각은 이른 쪽입니다. 네이티브 에러 문구는 영어입니다. `tests/timezone_test.go`가 라이브러리를 직접 빌드해(C 컴파일러가 없으면 건너뜀) 14개 시간대의 옛 날짜와 전환 경계 수천 건을 Go 자신의 데이터와 두 엔진에서 대조합니다.

`packages/http_server/`가 본보기 패키지입니다: `hana.pkg.json` + `hari/index.hr` + `kanade/index.knd` + `native/main.go`(Go의 `net/http`를 ABI v1 뒤에 둔 c-shared 라이브러리,
`native/build.sh`·`build.ps1`로 이 플랫폼 것만 빌드, 결과물은 git이 무시). 핸들러는 요청 딕셔너리(`method`, `path`, `query`, `headers`, `body`, `remote`)를 받아 글자나 `<응답>`(`status`,
`body`, `type`, `headers`)을 돌려줍니다.

## 패키지 매니저: `hana add`/`install`/`remove`/`list` (`pkg` 패키지)

서드파티 패키지는 git 경로가 이름입니다: `[github.com/owner/repo]에서 <함수>를 가져오자`(렉서의 TYPE 정규식이 대괄호 안의 경로 꼴 `host.tld/owner/repo`를 받고, 두 docs의 TS 렉서도 같습니다). 경로 판별은 `pkg.IsPackagePath` 한 곳이고, `pkg.Dir(module)`이 그 경로를 프로젝트의 `hana.json`(가까운 위쪽 폴더에서 찾음)에 따라 `replace` 폴더나 `hana-lock.json`이 고정한 캐시 폴더(`$HANA_HOME/pkg` 또는 `~/.hana/pkg`의 `<경로>@<버전>`)로 풀어 줍니다. 그래서 두 엔진·`pack`·네이티브 로더는 패키지 폴더를 어디서 찾는지 모르고 `pkg.Dir`만 부릅니다. 이름이 경로가 아니면(`timezone`) 예전처럼 hana와 함께 배포한 `packages/`입니다.

- **`hana.json`**은 프로젝트가 원하는 것(`dependencies`: 경로 → **최소 버전**, `replace`, `trustedScripts`), **`hana-lock.json`**은 고른 것(경로 → 정확한 버전, 태그가 가리키던 커밋, 승인한 설치 스크립트의 해시)이고 둘 다 git에 커밋합니다. 버전은 `1.2.3`뿐(범위·`^`·프리릴리스 없음).
- **버전 고르기는 MVS**(`Installer.Resolve`): 직접·전이 의존성 전체에서 같은 경로에 요구된 최소 버전 중 가장 높은 것. 요구되는 모든 버전을 내려받아 살펴보고(`hana.pkg.json`의 `dependencies`), 순환은 `Circular` 오류입니다. `replace`는 이 프로젝트에서만 유효하고 잠그지 않습니다.
- **`hana add 경로[@버전]`**은 최소 버전을 적고 다시 고르고 두 파일을 씁니다(버전을 안 주면 가장 높은 태그). **`hana install`**은 lock이 hana.json을 덮으면 lock대로, 아니면 다시 골라 내려받습니다. `hana run`/`build`는 **절대 내려받지 않습니다**(설치 안 됐으면 `ImportPackageNotInstalled`로 `hana install`을 안내). 내려받기는 `git clone --depth 1 --branch <태그>` 뒤 `.git`을 지우고 커밋을 `.hana-commit`에 적어 lock의 커밋과 대조합니다(태그가 옮겨졌으면 `CommitMismatch`).
- **네이티브 라이브러리와 설치 스크립트**: `hana.pkg.json`의 `native.<플랫폼>.url`+`sha256`이면 이 플랫폼 것만 내려받아 해시를 검증합니다(sha256 없이 url만 있으면 거부). `scripts.install`(명령 배열, 셸 없음)은 `hana.json`의 `trustedScripts`에 있거나 `hana add … --allow-scripts`로 승인한 패키지에서만, **설치 명령 안에서만** 돌고, 그 해시를 lock에 적어 스크립트가 바뀌면 `ScriptChanged`로 거부합니다. 전이 의존성의 스크립트는 자동으로 허용되지 않습니다.
- **`pkg`는 `net/http`를 부르지 않습니다**: `hana-runtime`이 `pkg`를 가져오므로 `Installer.Fetch`를 필드로 받고 진짜 HTTP는 `cmd/pkgcmd.go`에 있습니다(`net/http`를 runtime에 들이면 크기가 크게 늘어납니다). `git`은 명령을 부르고, 테스트(`pkg/install_test.go`)는 `GIT_CONFIG_*`의 `url.insteadOf`로 `https://example.test/…`를 디스크의 저장소로 돌려 진짜 git으로 검증합니다.
- 임포트한 코드는 자기 모듈의 **이름 공간과 상태**를 가집니다(아래 '모듈' 절).
- **`hana why <경로>`**(`Installer.Why`: lock과 캐시 속 `hana.pkg.json`의 dependencies를 따라 직접 추가한 패키지에서 그 패키지까지의 길, 최대 10개)와 **`hana outdated`**(`Installer.Outdated`: 잠근 것보다 높은 태그가 있는 패키지, 태그를 물어보므로 네트워크를 씁니다)가 있습니다.
- **비공개 저장소**: 내려받기가 git 명령이라 git에 설정한 로그인(credential helper, ssh 키, `url.<기준>.insteadOf`)이 그대로 쓰입니다. 표준 입력이 터미널이면 git이 로그인을 물어볼 수 있고(`interactive`), 아니면 `GIT_TERMINAL_PROMPT=0`입니다(사용자가 `GIT_TERMINAL_PROMPT`를 정해 두었으면 그대로). `HANA_GIT_PROTOCOL=ssh`면 `git@호스트:주인/저장소`로 받습니다. git의 불평이 로그인 문제(`needsLogin`: Authentication failed, could not read Username, Permission denied (publickey), Repository not found …)면 `AuthFailed`로 바꿔 안내합니다.
- CLI 문구는 `cmd/i18n.go`의 `pkg.err.<코드>`(오류)와 `add.`/`install.` 등 키, 오류 코드는 `pkg/errors.go`입니다.

## `마다 반복하자`는 문자열도 돕니다

스펙 5.4: `마다 반복하자`는 문자열도 돕니다. 코드포인트마다 길이 1의 문자열이고, 트리워커·브라우저 엔진은 목록으로 바꿔 돌며 바이트코드는 `TO_ITERABLE` 옵코드로 바꿉니다(목록이나 문자열이 아니면 `NotIterable`). `따라 나누자`에 맞는 경우도 `나머지는:`도 없으면 오류 없이 아무 일도 하지 않습니다(스펙에 있던 `UnhandledSwitchCaseError`는 필요 없다고 결정해 뺐습니다). `tests/switch_foreach_test.go`와 `compare_tests.ts`의 '분기:'·'반복:'이 지킵니다.

## 모듈: 임포트한 코드는 자기 모듈의 이름 공간과 상태를 가집니다

`"파일"`이나 `[패키지]`에서 가져온 함수·메서드·생성자는 안에서 같은 모듈의 다른 함수(도우미)와 그 모듈이 스스로 가져온 것(다른 패키지, `[모듈]`의 네이티브 함수)을 부를 수 있고, 가져오는 쪽이 그것들을 가져오지 않아도 됩니다. 가져오는 쪽에 같은 이름의 함수가 있어도 서로 가리지 않고, 두 모듈이 같은 도우미 이름을 써도 됩니다. **모듈 최상위의 변수와 문장도 모듈 것**입니다: 모듈은 처음 임포트될 때 한 번만 적재되어 최상위 코드가 한 번 실행되고(다시 임포트해도 같은 상태), 그 모듈의 함수는 그 변수를 읽고 바꿉니다. 가져오는 쪽의 변수는 모듈 함수에서 보이지 않습니다(`ReferenceError`). 서로를 가져오는 순환 임포트는 그 시점까지 적재된 것을 돌려받아 멈추지 않습니다(스펙 4.3).

- **트리워커**(`vm/module_scope.go`): 가져온 함수는 여전히 가져온 쪽 인터프리터에서 실행되지만, `loadModule`이 모듈을 한 번 적재하고(키: 파일 경로나 패키지 진입점, 캐시는 하위 인터프리터들이 공유하는 `moduleCache`, 실행 전에 캐시에 넣어 순환을 막음) `markModule`이 모듈의 선언(함수, 클래스, 메서드, 생성자)에 `Module`(그 모듈의 하위 인터프리터)을 적어 둡니다. 함수 본문은 호출 환경의 부모가 `globalOf(decl.Module)` — 즉 **모듈의 `GlobalEnv`**라서 모듈 변수가 보이고 가져오는 쪽의 변수는 안 보입니다. 함수 이름은 `scopedFunction`이 `i.scope`(실행 중인 본문의 모듈)의 선언과 환경을 먼저 봅니다(`runFunctionBody`, `runConstructor`, 필드 초기값, getter/setter가 `enterModule`로 켜고 끔). 가져오는 쪽의 함수는 `Module`이 nil이라 원래 범위로 돌아갑니다. 모듈 최상위가 출력한 글은 `outputSink`로 프로그램의 `Output`에 모입니다.
- **바이트코드**(`bytecode/module_scope.go`): 함수 표가 하나뿐이라 모듈의 함수를 `<모듈>::<이름>`으로 넣고 모듈 자기 코드의 `CALL`/`PUSH_FUNC_REF`를 그 이름으로 고칩니다(`exportModule`, 가져온 항목은 별칭으로도 등록). 모듈이 부른 네이티브 바인드(`IMPORT_NATIVE`)도 같은 이름으로 바뀝니다. 이미 `::`가 있는 이름은 다른 모듈 것이라 건드리지 않습니다. 모듈의 최상위 문장은 `INIT_MODULE` 옵코드(`ModuleInit`)로 컴파일되어 임포트 자리에서(네이티브 바인드 다음, 그 모듈이 가져온 모듈들의 것 뒤에) 한 번만 실행되고, VM은 모듈마다 `frame`을 하나 두어(`moduleFrame`) 그 모듈 함수의 호출 프레임(`frame.module`)이 이름을 찾을 때 프로그램의 전역 대신 그 프레임을 봅니다(`globalsFor`). `Function.Module`/`ClassInfo.Module`이 어느 모듈 것인지 기억합니다(`.hn`에 저장됨). 새 옵코드로 함수 이름을 다루게 되면 `rewriteChunk`에 넣으세요. 로컬 파일이 `[모듈]`을 가져오는 것도 되며 그 바인드는 로컬 임포트 자리에서 실행됩니다.
- **클래스 이름은 프로그램 하나에 하나**입니다(오류 문구·타입 검사가 이름으로 클래스를 가리키기 때문에). 모듈이 선언한(또는 스스로 가져온) 클래스와 인터페이스는 가져오는 쪽에 그 이름이 없을 때 자기 이름으로 등록되어 모듈의 함수가 만들 수 있고, **이미 다른 모듈(이나 프로그램 자신)의 같은 이름이 있으면 `ImportClassConflict`(`…Own`)**입니다. `<이름>을 <별칭>으로 가져오자`로 그 클래스에 다른 이름을 붙이면 충돌하지 않습니다(`bringModuleTypes`, 바이트코드는 `exportModule`의 `classOwner`). 같은 모듈을 두 번 임포트하는 것은 충돌이 아닙니다.
- 새 문법으로 함수를 부르는 자리를 만들면 트리워커는 `scopedFunction`을, 바이트코드는 `rewriteChunk`를 지나가게 하세요. `tests/module_scope_test.go`, `module_state_test.go`, `class_conflict_test.go`, `local_module_import_test.go`가 두 엔진(하자·카나데)을 같이 지킵니다.

## 릴리스(CD): 태그를 푸시하면 `.github/workflows/release.yml`이 내보냅니다

`v0.1.0` 같은 태그(`v*`)를 푸시하면 5개 플랫폼(linux amd64·arm64, darwin amd64·arm64, windows amd64)에서 빌드해 GitHub Release를 만듭니다. 손으로 돌리면(`workflow_dispatch`) 같은 빌드와 시험만 하고 올리지는 않습니다. 플랫폼마다 자기 러너에서 만드는 이유는 기본 패키지의 네이티브 라이브러리(`packages/*/native`)가 cgo와 그 OS의 C 컴파일러를 필요로 해서입니다(darwin/amd64만 arm64 러너에서 clang으로 크로스 빌드하고 실행 시험은 건너뜀). 릴리스 파일: `hana-<버전>-<os>-<arch>.tar.gz`(윈도우는 `.zip`)에 `hana`, `hana-runtime`, `packages/`(소스와 그 플랫폼의 네이티브 라이브러리만), `LICENSE`, README가 들어 있고, `hana pack --target`이 `hana` 옆에서 찾는 `hana-runtime-<os>-<arch>` 파일은 따로도 올리며, `checksums.txt`가 붙습니다. 버전은 `-ldflags "-X github.com/soumt-r/hana/cmd.Version=<태그>"`로 넣고 `hana version`이 보여 줍니다(`go install`로 만든 것은 모듈 버전). 빌드마다 압축 전에 `version`, 기본 패키지(`[timezone]`) 실행, `--bc` 실행, `pack`을 시험합니다. 새 기본 패키지를 `packages/`에 추가하면 `native/`가 있는 한 워크플로를 고칠 필요 없이 함께 실립니다.

## `hana pack`: 프로그램을 실행 파일 하나로

`hana pack 앱.hr -o 앱`은 `hana-runtime`(`cmd/hana-runtime`: 바이트코드 VM + 표준 라이브러리만, 파서·트리워커·LSP·cobra 없음, `-ldflags="-s -w" -trimpath`로 약 6MB — 표준 라이브러리에 네트워크(`[HTTP]`의 https)가 들어오기 전에는 3.5MB였습니다)을 복사한 뒤 뒤에 페이로드를 붙입니다.
형식은 `[런타임][zip 페이로드][트레일러 24바이트: 매직 | 크기 | 페이로드 SHA-256 앞 8바이트]`이고 `pack` 패키지가 읽고 씁니다. 페이로드에는 `program.hn`과 `libraries.json`(프로그램이 **실행 때** 네이티브 라이브러리를 여는 패키지 이름들, `Compiler.LibraryModules()`)이 들어갑니다
(하리 소스로 된 패키지 코드는 컴파일 때 이미 프로그램 안에 있음). 라이브러리 파일은 매니페스트 없이 이름만 `<패키지><확장자>`로 바꿔 기본적으로 실행 파일 **옆 `libraries/`**에 내보냅니다(`pack.Install`) — 실행 파일은 작게 남고 파일 구성이 단순합니다.
`--embed`면 페이로드 안에 `libraries/<패키지><확장자>`로 넣어 파일 하나로 만들고(압축되어 더 작아짐), 실행할 때 사용자 캐시 폴더 `hr-packed/<id>/libraries`에 한 번 풉니다(같은 크기 파일이 있으면 그대로 둠 — 윈도우는 로드된 DLL을 못 덮어씀).
어느 쪽이든 `Activate`가 `native.Bundled`(패키지 이름 → 라이브러리 경로)를 채우고, `native.OpenModule`(`bcvm`이 부름)이 패키지 폴더보다 먼저 그걸 봅니다. `--target 플랫폼`(예 `linux-amd64`)과 `--runtime`으로 다른 OS용도 만들 수 있고(그 OS용 런타임을 `hana-runtime-<플랫폼>`으로 hana 옆에 두거나 `--runtime`),
`hana run x.hn`과 패키지 실행은 `runner.NewVM`을 함께 씁니다. 서명은 붙인 다음에 하세요(뒤에 덧붙이면 서명이 깨짐). 테스트는 `tests/pack_test.go`(실제 `hana`와 `hana-runtime`을 빌드해서 돌림).

## 임포트는 가져오는 파일/패키지의 언어로 실행됨

`vm/config.go`의 `LangConfig`가 `Name`(`hari`/`kanade`), `SourceExt`(`.hr`/`.knd`), `ParseProgram`을 갖고,
`vm/import.go`가 이걸로 언어를 고릅니다: 로컬 파일 임포트(`"파일"에서 …`)는 **그 파일의 확장자**로
(`ConfigForFile`), 패키지 임포트(`[모듈]에서 …`)는 **임포트하는 쪽 언어**의 진입점
`packages/<모듈>/<Name>/index<SourceExt>`로 갑니다. 그 언어 진입점만 없으면 `ImportUnsupportedLocale`.
임포트된 모듈은 `newSubInterpreter`가 만드는 자기 인터프리터에서 실행되고, `Interpreter.Bootstrap`
(`stdlib.RegisterStandardLibrary`가 설정)으로 자기 언어의 표준 라이브러리를 받습니다 — 새 언어를 추가하면
`allConfigs`에 넣고 `Name`/`SourceExt`/`ParseProgram`을 채우세요.
바이트코드도 패키지 폴더 임포트를 지원합니다(아래 '패키지 형식' 참고).
`ast.ImportStatement`는 `Items`(이름+선택적 별칭)와 `All`을 가집니다: `<a>와 <b>를 가져오자`(조사 `LangProfile.ImportJoinParticles`), `[모듈]에서 전부 가져오자`(`ImportAllWord`; 카나데 `全部`는 렉서에 전용 규칙이 있음 — 뒤에 동사가 바로 붙어서). 별칭은 항목이 하나일 때만. 모듈은 한 번만 적재해서 항목마다 바인딩하고, `전부`는 그 파일/패키지가 선언한 함수·클래스·인터페이스(내장 모듈은 함수 전부)를 가져옵니다(변수는 항목으로 지정해야 함, 바이트코드와 같은 범위). 바이트코드는 로컬 파일을 컴파일 시점에, `[모듈]`은 `IMPORT_NATIVE`(`ImportOperand.All`)로 처리합니다.

## 네이티브 표준 라이브러리 이름은 `std` 패키지에 한 번만

`hana`가 내장하는 네이티브 모듈(`[수학]`/`【数学】` 등)은 언어 중립 ID(`math.ceil`)와 언어별 이름표로
나뉩니다: `std/std.go`의 `Modules`(ID 목록)와 `names`(언어 → ID → 이름)가 **이름의 유일한 출처**이고,
트리워커(`stdlib.nativeImpls`)·바이트코드(`bcstdlib.nativeImpls`)·docs TS 엔진이 전부 여기서 이름을 찾습니다.
`LangConfig`에 모듈/함수 이름 필드를 다시 추가하지 마세요.

새 네이티브 함수 추가: ① `std.go`에 ID 상수 + `Modules` + 두 언어 이름 → ② `std/stdimpl`의 `Impls`에
구현(두 엔진이 같은 함수를 이름만 바꿔 씁니다: 값은 float64/string/bool/`[]interface{}`/`map[interface{}]interface{}`/nil뿐이라 공유 가능. 새 에러는 `errs`에 코드+세 카탈로그) → ③ `tests/std_names_test.go`의 `stdCases`에 기대값(+동작은 `tests/std_library_test.go`) → ④ `go run ./cmd/stdgen -hari
../hari-docs/src/utils/hari/stdNames.ts -kanade ../kanade-docs/src/utils/kanade/stdNames.ts`(`-check`로 검증) →
⑤ docs TS 엔진(`stdImpls.ts`의 `nativeImpls`)에 구현하고 `compare_tests.ts`에 '표준:' 케이스 추가.
운영체제가 필요한 모듈(파일, 소켓)은 `std.Module{NativeOnly: true}`로 표시합니다: Go 엔진 둘은 평소처럼 등록하고(구현도 `stdimpl` 한 곳), `stdgen`이 `stdNames.ts`에 `nativeOnly: true`를 내보내면 브라우저 엔진이 구현 없이 모듈만 기억했다가 임포트 때 `ImportNativeOnly`("브라우저에서 사용할 수 없어요")를 냅니다. 이 표시가 붙은 함수는 TS `stdImpls.ts`에 구현하지 않고 `compare_tests.ts`의 '표준:' 케이스도 만들지 않습니다(브라우저에서 그 에러가 나는지만 확인).
`[소켓]`(`socket.*`, `stdimpl/net.go`)과 `[HTTP]`(`http.*`, `stdimpl/http.go`)은 `[파일]` 다음의 NativeOnly 모듈입니다. 소켓은 값이 숫자뿐이라 **번호**(`stdimpl.sockets` 표)로 다루고, 듣기는 기본으로 `127.0.0.1`에서만 받습니다.
모든 네트워크 함수는 `stdimpl.NetAccess(op, 대상)`을 먼저 부르고(`hana run --allow-net=false`가 `DenyNet`으로 바꿈), 에러는 OS 문구 대신 `NetworkError.*` 코드로 옮깁니다. `[HTTP]`는 `net/http`를 쓰지 않고 `net`(+https는 `crypto/tls`) 위에서 HTTP/1.1을 직접 말합니다(요청 한 번에 연결 한 번, 길이/청크/끝까지 읽기 본문, 리다이렉트 10번까지):
`net/http`를 들이면 HTTP/2·프록시·쿠키까지 딸려 와 `hana-runtime`이 3.5MB에서 7MB가 되고, 직접 하면 6MB입니다(`crypto/tls`가 약 2.5MB — https가 필요 없다면 빌드 태그로 뺄 수 있는 자리). 테스트(`tests/net_test.go`)는 손으로 쓴 응답 바이트로 파서를 확인합니다.
브라우저 엔진은 두 모듈의 이름만 알고 가져오면 `ImportNativeOnly`를 냅니다. 여러 번의 호출이 필요해 한 번의 호출로 보일 수 없는 함수는 `stdCase.elsewhere`에 어느 테스트가 지키는지 적습니다.
사용자 함수를 받는 표준 함수(`[목록]`의 변환하기·걸러내기·접기·찾기·하나라도·모두·기준정렬)는 `stdimpl.Impls`가 아니라 `stdimpl.HostImpls`(`hostfuncs.go`)에 구현합니다: `func(call Caller, args)`가 엔진이 준 `call`로 함수 값을 부릅니다(트리워커 `Interpreter.CallFunction`, 바이트코드 `VM.CallValue`, TS `evalExpr.ts`의 `callValue`). 프로그램이 이미 실행 중이라 락을 다시 잡지 않는 것이 `native.Host.Call`과 다른 점입니다. 조건 함수가 참/거짓이 아닌 값을 돌려주면 `CallbackNotBoolean`. TS 쪽은 `stdImpls.ts`의 `hostImpls`와 `stdlib.ts`가 같은 일을 하고, `tests/std_names_test.go`의 `stdCase.prelude`가 넘길 함수를 정의합니다.
`[경로]`(`path.*`, `stdimpl/path.go`)는 반대로 파일을 보지 않는 순수 글자 함수라 브라우저에서도 돕니다: 구분자는 `/`(입력의 `\`도 `/`로 봄), 맨 앞 `C:` 꼴은 드라이브, 결과는 항상 `/`, 새 에러 코드 없음 — 경계 사례는 `tests/path_test.go`와 `compare_tests.ts`의 '표준: 경로'가 지킵니다.
`[CSV]`(`csv.*`, `stdimpl/csv.go`)도 글자만 다루는 순수 함수라 브라우저에서 돕니다: `파싱`은 RFC 4180대로 읽어 글자 목록의 목록을 돌려주고(칸은 늘 글자, 따옴표 안의 줄바꿈은 칸의 일부, 끝의 줄바꿈은 행을 더하지 않고, 빈 줄은 빈 칸 하나짜리 행, 닫는 따옴표 뒤에 이상한 글자가 오면 `CSVInvalid`), `문자열화`는 글자·숫자·`비어있음` 칸만 받아 필요한 칸에만 따옴표를 붙이고 행 사이에만 줄바꿈을 넣습니다. 구분자는 글자 하나(따옴표·줄바꿈 제외, `CSVDelimiter`). 경계 사례는 `tests/csv_test.go`와 `compare_tests.ts`의 '표준: CSV'가 지킵니다.
`[파일]`(`file.*`)은 첫 NativeOnly 모듈입니다. 모든 파일 함수는 `stdimpl.FileAccess(op, path)`(기본은 전부 허용, `hana run --allow-file=false`가 `stdimpl.DenyFiles`로 바꿈)를 먼저 부르므로 폴더 화이트리스트 같은 더 세밀한 정책도 그 한 곳에 꽂을 수 있습니다. OS 에러는 `fileError`가 `FileNotFound`/`FileAccessDenied`/`FileFailed`로 옮깁니다(OS 문구는 OS 언어라서 쓰지 않음). 테스트에서 이 정책을 바꿀 땐 반드시 `t.Cleanup`으로 되돌리세요. `"파일"에서 가져오자`(로컬 파일 임포트)는 언어 기능이라 이 정책의 대상이 아닙니다.
표준 모듈은 파이썬 표준 라이브러리에서 훔쳐오되 **파일·네트워크·콜백 없이 값만 다루는 것**만 둡니다(브라우저 엔진도 똑같이 돌려야 해서). 함수 결과는 `-0`을 `0`으로 정리(`clean`)하고, 글자 수·위치는 코드포인트 단위, 정렬은 코드포인트 순서, 아주 크거나 작은 수는 두 엔진 모두 과학적 표기 없이 찍습니다(TS는 `plainNumber`). 삼각함수·로그는 언어 구현마다 마지막 자리가 다를 수 있어 테스트는 정확히 떨어지는 값만 씁니다.
표준 모듈의 규칙: 날짜는 유닉스 초(숫자)·컴퓨터의 시간대, 서식 토큰 `YYYY MM DD HH mm ss`; 정규식은 Go와 JS의 공통 문법(맨 앞 `(?ims)`만 옵션)이고 빈 매치는 모두찾기/치환/분할에서 무시; JSON 사전 키는 문자열만, 정렬해서 출력. ①~⑤ 중 빠진 게 있으면 테스트나 `stdgen -check`가 잡습니다.

## 구문 오류는 파서가 `Diagnostics()`로 보고하고 `Errors()`에도 들어감

`parser/hari`(카나데 파서도 이걸 씀)는 알 수 없는 토큰을 만나면 `Diagnostic`(줄, 글자 열, 길이)을 기록하고 계속 파싱합니다.
`Errors()`는 그 진단을 영어 문장으로 포함하고, `LocalizedErrors(loc)`는 사용자 언어로 돌려줍니다(CLI가 씀).
문구는 `errs`의 `SyntaxError.*` 코드라 에러 카탈로그·TS 생성 파일과 같은 어휘입니다. 입력 끝에서 나온 진단은 앞선 진짜
잘못된 토큰의 후폭풍이면 숨기고, 단독일 때만 한 번 보고합니다(`Diagnostics()`). 그래서 문법이 틀린 파일은 이제
`hana run`이 실행 없이 "구문 분석 중 오류"로 끝나고, 임포트한 파일·패키지의 구문 오류는 `ImportFileSyntax`/
`ImportPackageSyntax`가 됩니다. LSP도 같은 진단과 `errs.Message`를 씁니다(별도 문구 없음).

## 자동완성 키워드는 `hana/lsp/language.go` 한 곳에서 (브라우저 에디터 포함)

언어 서버가 제안하는 낱말은 `lsp/language.go`의 세 표가 원본입니다: `keywords`(렉서에서 KW_ 토큰 하나로 읽히는 낱말 —
`TestKeywordsLexAsKeywords`가 검증), `comparisons`(COMPARE 토큰), `words`(같은 뜻이지만 자기 토큰이 없는 문법 낱말: 돌려주는,
우리, 値 등). 키워드를 추가하면 ① 호버 설명(`hoverdocs.go`의 해당 KW_ 토큰 타입) ② VS Code 문법(`vscode-hari/syntaxes/*.tmLanguage.json`)
③ 생성 파일을 함께 맞춰야 하고, 빠지면 테스트(호버 문서·문법 드리프트·생성 파일 최신 여부)가 잡습니다.
docs 저장소의 브라우저 에디터용 `lspKeywords.ts`는 `go run ./cmd/lspgen -hari ../hari-docs/src/utils/hari/lspKeywords.ts
-kanade ../kanade-docs/src/utils/kanade/lspKeywords.ts`로 생성합니다(`-check`로 최신 여부 확인).

## `입력받자`는 입력 소스에서 한 줄을 읽음

`Interpreter.ReadLine`(트리워커)/`VM.ReadLine`(바이트코드, `INPUT` 옵코드)이 다음 한 줄을 주고, `hana run`이 stdin으로 연결합니다
(`cmd/run.go`의 `stdinLines`). 변환은 `conv.ParseInput`(두 엔진 공용, 브라우저 엔진이 손으로 미러링): 숫자는 앞뒤 공백만 무시하는
엄격한 형식이라 `""`, `0x10`, `1e3`은 실패하고(`InputConversionError`), 논리는 언어의 참/거짓 낱말, 타입을 생략하면 문자열입니다.
입력 소스가 없으면(`ReadLine == nil`: 테스트, doctest) 빈 줄을 읽어서 절대 멈추지 않습니다. 타입 이름은 `LangConfig.TypeString/TypeNumber/TypeBoolean`.

## 속도: 재는 법과 이미 들어간 최적화

`bench/*.hr`(피보나치·중첩 반복·목록·문자열·클래스·사전)를 `tests/bench_test.go`의 `BenchmarkPrograms`가 두 엔진에서 돌립니다(파싱·컴파일은 재는 시간 밖, 출력은 버림): `go test ./tests -run XXX -bench Programs -benchtime 4x`. 느려 보이면 `-cpuprofile`로 `go tool pprof -top`부터 보세요 — 짐작으로 고친 최적화 중 실제로 느려진 것도 있었습니다(호출 인자를 스택 뷰로 넘긴 바이트코드 CALL). 최적화를 넣은 뒤에는 `go test ./...`와 두 docs의 `compare_tests.ts`(엔진 간 출력 비교)로 의미가 안 바뀌었는지 확인하세요.

- **`num.Box`**: 작은 정수 float64를 `interface{}`로 만들 때마다 8바이트를 할당하던 것을 [-1024, 1<<20)의 값은 미리 만든 표에서 돌려줍니다. 산술 결과·`NumberLiteral.Boxed`·범위 반복 변수가 씁니다.
- **트리워커**: 리터럴은 노드에 미리 계산해 둡니다(`NumberLiteral.Boxed`, `StringLiteral.Cooked`, `TemplateLiteral.Parts`). 함수·반복 범위는 `newScope`/`freeScope`(작은 free list)로 재사용하므로, **범위를 만든 뒤 아무도 그 `Environment`를 붙잡지 않아야** 합니다(클로저 값이 없는 지금은 성립). 호출 인자는 `Interpreter.argStack`에 쌓고 프로그램 안에서 선언한 함수(`*ast.FunctionDeclaration`)에만 뷰로 넘깁니다(매개변수 바인딩이 복사) — 다른 호출 대상은 인자를 붙잡을 수 있어 복사본을 받습니다. 최상위 함수는 `topFunction`(이름 색인), 클래스 멤버는 클래스별 Symbol 표(`classMembers`)로 찾고, `evaluate`/`Execute`에 `defer`를 넣지 마세요(모든 식 평가가 느려집니다).
- **바이트코드**: `bytecode/peephole.go`의 `Optimize`가 컴파일 끝에 `LOAD_VAR/PUSH_CONST + 산술·비교 (+ SET_VAR / JUMP_IF_FALSE)`를 `BIN` 하나로 합칩니다(피연산자 `BinOperand`, 못 빠르게 하는 경우는 합치지 않은 명령과 같은 결과·에러로 처리). 점프 목표가 가운데에 있으면 합치지 않고, 합친 뒤 점프 목표를 옮깁니다. `SET_LIST_VAR`/`CHECK_LIST_FIELD`는 추가한 원소의 선언 타입 검사, `Instruction.Hint`는 실행 중 채우는 캐시(직렬화 안 함), `CallOperand.CachedFunction`은 호출 대상 캐시입니다. 새 옵코드가 점프나 변수 슬롯을 다루면 `jumpTargets`/`optimizeChunk`를 같이 고치세요. `run`/`exec` 분리는 `defer` 없이 프레임을 정리하기 위한 것입니다. 프로그램 안의 함수·메서드·생성자를 부르는 `CALL`/`CALL_METHOD`/`NEW_OBJECT`는 인자를 새 슬라이스에 옮기지 않고 피연산자 스택 위의 뷰로 넘깁니다(매개변수 바인딩이 프레임으로 복사하므로 안전하고 피보나치가 25% 빨라짐, 네이티브·문자열 메서드는 인자를 붙잡을 수 있어 복사본을 받습니다). `op, SET_VAR`/`op, JUMP_IF_FALSE`도 `BIN`으로 합칩니다. 세는 반복의 뒤끝(`BIN i+걸음 -> i`, `JUMP`, 그 점프 목표의 반복 머리)은 `fuseLoopSteps`가 `FOR_STEP` 하나로 만듭니다: 더하고 비교하고 본문이나 끝으로 한 번에 갑니다. 머리는 `BIN i<끝 ; 거짓이면`(상수 범위, `동안 반복`)이거나, 양 끝이 둘 다 리터럴은 아닌 범위(`1부터 'n'까지`)의 `끝 - i`, `* 걸음`, `>= 0` 세 BIN(`Span`, 같은 계산 순서로 처리)입니다. 걸음과 끝은 상수나 카운터가 아닌 변수입니다. 숫자가 아니거나 상수·타입 검사에 걸리면 아무것도 바꾸지 않고 원래 `BIN`처럼 처리한 뒤 남겨 둔 `JUMP`로 넘어가서 결과·에러가 같습니다(명령 위치가 안 바뀌어 점프 목표를 다시 계산하지 않음, `tests/for_step_test.go`, loops 벤치 약 23%, `1부터 'n'까지` 약 2배). 본문이 변수를 선언하는 반복도 `POP_SCOPE`가 `ADD` 앞에 오므로 그대로 합쳐집니다. 피연산자 스택을 풀에서 재사용해 보았지만 할당보다 이득이 없어 넣지 않았습니다.
- **typecheck**: 타입 표기의 분해 결과를 `Spec`으로 캐시하고(`parsed`) 공개 함수가 `*Names`를 받습니다(값 복사 비용 제거). 흔한 경우(`[숫자]`에 float64 등)는 `QuickAccepts`가 바로 답합니다.

## 문자열 `+`는 `strcat.Join`으로

Go 문자열은 바꿀 수 없어서 `s = s + 조각`을 반복하면 매번 s 전체를 복사해 길이의 제곱만큼 걸렸습니다(20만 번에 약 4초). 두 엔진의 문자열 `+`(`vm/eval_expr.go`, `bcvm/vm.go`의 `binaryOp`)는 `strcat.Join`을 씁니다: 256바이트 이상의 결과는 여유를 둔 버퍼에 만들어 기억해 두었다가, 다음 `Join`의 왼쪽이 정확히 그 결과이면 여유 자리에 조각만 써 넣습니다(값 표현은 그대로). 이미 나눠 준 문자열의 바이트는 절대 다시 쓰지 않고 그 뒤쪽만 쓰므로 안전하고, 같은 문자열에서 갈라져 붙이거나 부분 문자열을 잇는 경우는 그냥 복사합니다(`strcat/strcat_test.go`가 지킴). 문자열을 잇는 새 경로를 만들면 `a + b` 대신 이걸 쓰세요.

## 출력은 `console` 패키지로 (파이프면 모아서 씀)

두 엔진의 `출력하자`는 `fmt.Print` 대신 `console.Print/Println`을 씁니다. 표준 출력이 터미널이면 예전처럼 바로 쓰고, 파일이나 파이프면 모았다가 32KB가 되거나 25ms가 지나면 씁니다 —
줄마다 시스템 호출을 하던 비용이 없어져 출력이 많은 프로그램이 2배 이상 빨라집니다. 순서와 즉시성은 `Flush`가 지킵니다: 입력을 읽기 전(`conv.NewLineReader`), 표준 에러에 쓰기 전(`runner.PrintRuntimeError`, `cmd/run.go`),
프로그램이 끝날 때. 프로그램이 출력하고 나서 서버나 `기다리기`에서 멈춰도 25ms 안에 나옵니다(`tests/console_test.go`). 새로 `os.Exit`로 끝나는 실행 경로를 만들면 그 전에 `console.Flush()`를 부르세요.

## 선언한 타입은 실행 때 검사됨 (`typecheck` 패키지)

Runtime 스펙 2.2: `[타입]`이 붙은 선언은 값이 그 타입인지 검사하고(`typecheck.Check`), 타입은 선언 뒤에도 남아 재대입·목록 추가(`추가하자`, 제네릭은
요소 하나하나)·매개변수(`CheckArgument`)·클래스 필드와 그 초기값에도 적용됩니다. `비어있음`은 모든 타입에 통과하고, 타입을 생략하거나 `[아무거나]`면 다 통과,
자식/인터페이스 구현체는 부모·인터페이스 타입에 통과(업캐스팅). 트리워커는 `vm/types.go`(`assignVariable`, `declareParam`, `checkField`, `Environment.DeclareType`),
바이트코드는 `bcvm/types.go`와 `SET_VAR_TYPED` 옵코드(+`Param.Type`, `FieldInfo.Type`)에서 같은 `typecheck`를 씁니다. 새로 변수를 쓰는 경로를 만들면 반드시
`assignVariable`(바이트코드는 `assignOrDeclare`)을 거치게 하세요 — 안 그러면 그 경로만 타입 제약이 새어 나갑니다. 타입 이름은 `LangConfig.Types`(`typecheck.Names`).
브라우저 엔진은 `typecheck.ts`/`types.ts`로 미러링하고 `compare_tests.ts`가 Go와 문구까지 비교합니다. 연산자도 같은 스펙을 따릅니다: 묵시적 형변환 금지(`10 + "안녕"`은 `OperandTypeMismatch`, `+`는 문자열끼리만 잇기),
`비어있음` 피연산자는 `NullOperand`(동등 비교 `같다/다르다`만 예외). 트리워커·브라우저 엔진이 모르는 연산자(`x 이다 참` 같은 오타)를 만나면 조용히 `비어있음`을 내지 않고 `UnknownOperator`를 냅니다(바이트코드는 컴파일 때 거절). 함수 반환 타입은 **적으면 검사하고 안 적어도 됩니다**: `typecheck.CheckReturn`(`ReturnTypeMismatch`)을 트리워커 `Interpreter.runFunctionBody`, 바이트코드 `Function.ReturnType`+`VM.checkedReturn`(함수·인스턴스 메서드·정적 메서드), TS `types.ts`의 `checkReturn`이 부릅니다(`tests/return_type_test.go`, `compare_tests.ts`의 '타입: 반환'). `비어있음`(반환 없이 끝남 포함)은 통과합니다.

조건도 같은 방침입니다: `만약`·`동안 반복`의 조건과 `그리고`/`또는`의 양쪽 피연산자는 `참`/`거짓`이어야 하고 그 밖의 값은 `ConditionNotBoolean`(`TypeError`)입니다 — 예전엔 조용히 거짓이 되어 `(('x' % 2) == 0)` 같은 파서 버그를 가렸습니다. 트리워커는 `Interpreter.requireBool`, 바이트코드는 `VM.requireBool`(`JUMP_IF_*`/`NOT`)과 `CHECK_BOOL`(그리고/또는의 우항, `.hn` 형식은 그대로 — 옵코드만 뒤에 추가), TS는 `types.ts`의 `requireBool`이고 `tests/condition_paren_test.go`와 `compare_tests.ts`의 '조건:'이 지킵니다.

## 변수 범위: 반복문과 오류 처리기는 반복마다·처리마다 새 범위

런타임 스펙 1.1: 반복문(반복할 때마다), `오류가 발생했다면` 처리기, 함수 호출이 새 범위를 열고, `만약`·`따라 나누자` 블록은 열지 않습니다. 트리워커는 `NewEnvironment`, 바이트코드는 프레임(선언 순서대로 쌓는 슬라이스)을 되감는 `PUSH_SCOPE`/`POP_SCOPE`(`.hn` 형식은 그대로, 옵코드만 뒤에 추가)로 같은 결과를 냅니다: `PUSH_SCOPE`는 숨은 변수(`__tmpN_scope`)에 "지금 변수가 몇 개인가"를 적어 두고, `POP_SCOPE`는 그 뒤에 선언된 변수를 표식까지 통째로 버립니다. 표식이 변수라서 `반복을 끝내자`나 예외로 건너뛰어도 안쪽 표식이 바깥 표식을 어지럽히지 않고, 바깥 표식을 버리면 안쪽 것도 함께 사라집니다. 반복문 본문에 선언할 수 있는 노드가 없으면(`mayDeclare`) 옵코드를 아예 내지 않습니다. 최상위에서는 변수가 `vm.globals`에 있어서 `declaringFrame`이 그쪽을 씁니다. 반복 변수(`횟수`, `아이템`)와 숨은 끝값·색인은 바깥 표식 하나(`__tmpN_outer`)에 묶여 반복이 끝나면 사라집니다. `tests/scope_test.go`가 지키고 브라우저 엔진도 같은 결과입니다(예전에는 바이트코드만 변수를 함수 끝까지 들고 있어서, 트리워커에서는 오류가 나는 프로그램이 바이트코드에서 조용히 돌았습니다).

반복 변수(범위의 카운터·반복 변수, `마다`의 항목)와 처리기의 오류 변수는 **자기 범위에 새로 선언**되어 같은 이름의 바깥 변수를 가리고, 범위가 끝나면 바깥 것이 그대로 돌아옵니다(`'가'`가 100일 때 `1부터 2까지 반복하자 ('가')` 뒤의 `'가'`는 100). 바이트코드는 `DECLARE_VAR`(이미 있어도 새 칸, `frame.declare`)로 선언하고, 가려진 칸은 `hidden`(힌트 캐시 `probe`도 건너뜀), 새 칸은 `shadows`에 가린 칸을 적어 `POP_SCOPE`(`dropFrom`)가 되살립니다. 이름 찾기(`frame.find`)는 뒤에서부터라 가장 최근 선언이 이깁니다. 범위의 양 끝은 반복 변수를 선언하기 전에 계산합니다(`1부터 '가'까지 반복하자 ('가')`의 끝은 바깥 `'가'`). 오류가 반복·처리기를 뚫고 나가 `일단 해보자`에 잡히면 `POP_SCOPE`를 건너뛰므로, `TRY_PUSH` 때의 프레임 크기(`tryHandler.frameDepth`) 위에 남은 첫 범위 표식(`varSlot.marker`)부터 버립니다(`unwindScopes`, `일단 해보자` 본문이 직접 선언한 변수는 남음). 상수 여부는 그 이름을 가진 **가장 가까운 범위**가 정합니다(트리워커 `Environment.isConst` — 예전엔 부모까지 거슬러 올라가 가려진 바깥 상수 때문에 반복 변수에 `더하자`가 `ConstantAssignmentError`였음).

`[클래스]인 <함수>()`는 타입이 붙은 함수 호출입니다. 파서(`parsePrimary`의 TYPE_IN 갈래)는 TYPE_IN 낱말이 멤버 접근 조사와 같을 때만(카나데의 `の`) "TYPE의 <정적메서드>()"로 읽습니다. 하리의 정적 호출은 `[클래스]의 <함수>()`(조사 `의`)이고 `인`은 언제나 타입 표시입니다(`tests/typed_call_test.go`). TS 파서 둘도 같습니다.

산술 연산자는 일반 수학 순서입니다: `parser/hari`의 `parseBinary`(TS는 두 docs의 `parser.ts`)가 `* / %`를 `+ -`보다 먼저 묶고 같은 단계는 왼쪽부터 묶습니다(예전에는 전부 왼쪽에서 오른쪽이라 `2 + 3 * 4`가 20이었습니다). 파서는 하나라서 트리워커·바이트코드·카나데가 함께 바뀌고, 비교(`같다`, `크다`)와 `그리고`/`또는`은 산술을 다 읽은 뒤에 붙는 별개의 구조라 그대로입니다. `tests/precedence_test.go`와 `compare_tests.ts`의 '연산: 우선순위'가 지킵니다.

인터페이스와 추상 클래스(`밑설계하자`/`下設計しよう`, 파서가 `LangProfile.IsAbstractClassVerb`로 `ClassDeclaration.IsAbstract`를 채움)는 `새로운`으로 만들 수 없고(`InstantiationError`), 문자열 글자 재대입은 `ImmutableAssignmentError`입니다(스펙 3.1.1, 5.4). 트리워커·바이트코드(`ClassInfo.IsAbstract`)·브라우저 엔진 모두 같습니다.
