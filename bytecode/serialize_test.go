// External test package (not `package bytecode`) so it can freely import
// bcvm to actually execute a round-tripped Program — bcvm itself imports
// bytecode, so an internal test file here would be a genuine import cycle.
package bytecode_test

import (
	"bytes"
	"testing"

	"github.com/soumt-r/hana/bcvm"
	"github.com/soumt-r/hana/bytecode"
	lexer "github.com/soumt-r/hana/lexer/haja"
	parser "github.com/soumt-r/hana/parser/haja"
)

func compileSource(t *testing.T, code string) *bytecode.Program {
	t.Helper()
	l := lexer.New(code)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse error: %v", p.Errors())
	}
	c := bytecode.NewCompiler()
	bcProg := c.Compile(prog)
	if len(c.Errors()) > 0 {
		t.Fatalf("compile error: %v", c.Errors())
	}
	return bcProg
}

func runProgram(t *testing.T, prog *bytecode.Program) []string {
	t.Helper()
	vm := bcvm.New(prog)
	if err := vm.Run(); err != nil {
		t.Fatalf("run error: %v", err)
	}
	return vm.Output
}

// TestProgramRoundTrip exercises every kind of operand a Program can hold —
// plain opcodes, CallOperand, NewObjectOperand, MemberOperand,
// StaticFieldOperand, TryOperand (with a nested FinallyChunk),
// ImportOperand — plus classes (fields with defaults, a constructor,
// methods, a setter) and functions, so a gap in gob.Register or an
// unexported field would show up here rather than only on some future real
// .hn file.
func TestProgramRoundTrip(t *testing.T) {
	prog := compileSource(t, `
[숫자]를 돌려주는 <더하기>를 만들자 ([숫자]인 '왼쪽', [숫자]인 '오른쪽' = 10):
    ('왼쪽' + '오른쪽')를 돌려주자

[오류]를 바탕으로 [내오류]를 설계하자:
    처음 만들어질 때 ([문자열]인 '메시지') 다음과 같이 하자:
        부모의 <처음 만들어질 때>('메시지')을 실행하자

[상자]를 설계하자:
    '내용물'을 [(문자열)목록]인 []로 정하자
    '우리'의 '개수'를 [숫자]인 0으로 정하자

    '이름' 정하자:
        정할 때 ('값'):
            '나'의 '내용물' 뒤에 '값'을 추가하자

    [숫자]를 돌려주는 '우리'의 <증가>를 만들자 ():
        '우리'의 '개수'에 1을 더하자
        '우리'의 '개수'를 돌려주자

'상자1'을 [상자]인 새로운 [상자]()로 정하자
'상자1'의 '이름'을 "테스트"로 정하자

일단 해보자:
    새로운 [내오류]("문제 발생")를 던지자
[내오류]가 발생했다면 ('에러'):
    틀"잡힘: {'에러'의 '메시지'}"를 출력하자
마무리는 항상:
    "마무리"를 출력하자

틀"더하기: {<더하기>(1, 2)}"를 출력하자
틀"증가: {[상자]의 <증가>()}"를 출력하자
`)

	var buf bytes.Buffer
	if err := prog.Encode(&buf, bytecode.LangHaja); err != nil {
		t.Fatalf("Encode: %v", err)
	}

	decoded, _, err := bytecode.ReadProgram(&buf)
	if err != nil {
		t.Fatalf("ReadProgram: %v", err)
	}

	if len(decoded.Main.Instructions) != len(prog.Main.Instructions) {
		t.Errorf("Main instruction count = %d, want %d", len(decoded.Main.Instructions), len(prog.Main.Instructions))
	}
	if len(decoded.Functions) != len(prog.Functions) {
		t.Errorf("Functions count = %d, want %d", len(decoded.Functions), len(prog.Functions))
	}
	if len(decoded.Classes) != len(prog.Classes) {
		t.Errorf("Classes count = %d, want %d", len(decoded.Classes), len(prog.Classes))
	}
	box, ok := decoded.Classes["상자"]
	if !ok {
		t.Fatalf("class '상자' missing after round-trip")
	}
	var nameField *bytecode.FieldInfo
	for i := range box.Fields {
		if box.Fields[i].Name == "이름" {
			nameField = &box.Fields[i]
		}
	}
	if nameField == nil || nameField.Setter == nil {
		t.Errorf("class '상자' field '이름' or its setter missing after round-trip (Fields=%+v)", box.Fields)
	}
	if _, ok := box.StaticMethods["증가"]; !ok {
		t.Errorf("class '상자' static method '증가' missing after round-trip")
	}

	cls, ok := decoded.Classes["내오류"]
	if !ok || cls.Constructor == nil {
		t.Fatalf("class '내오류' or its constructor missing after round-trip")
	}
}

// TestProgramRoundTripExecutesIdentically is the real proof: run the
// program before and after a Encode/ReadProgram round-trip and check the
// output is byte-for-byte the same, not just that the struct shape looks
// right — structural equality (TestProgramRoundTrip) could miss a gob
// quirk that only shows up when an instruction's operand is actually used
// at runtime.
func TestProgramRoundTripExecutesIdentically(t *testing.T) {
	prog := compileSource(t, `
[숫자]를 돌려주는 <더하기>를 만들자 ([숫자]인 '왼쪽', [숫자]인 '오른쪽' = 10):
    ('왼쪽' + '오른쪽')를 돌려주자

[오류]를 바탕으로 [내오류]를 설계하자:
    처음 만들어질 때 ([문자열]인 '메시지') 다음과 같이 하자:
        부모의 <처음 만들어질 때>('메시지')을 실행하자

[상자]를 설계하자:
    '내용물'을 [(문자열)목록]인 []로 정하자

    '이름' 정하자:
        정할 때 ('값'):
            '나'의 '내용물' 뒤에 '값'을 추가하자

'상자1'을 [상자]인 새로운 [상자]()로 정하자
'상자1'의 '이름'을 "테스트"로 정하자
틀"내용물: {'상자1'의 '내용물'}"를 출력하자

일단 해보자:
    새로운 [내오류]("문제 발생")를 던지자
[내오류]가 발생했다면 ('에러'):
    틀"잡힘: {'에러'의 '메시지'}"를 출력하자
마무리는 항상:
    "마무리"를 출력하자

틀"더하기: {<더하기>(1, 2)}"를 출력하자
'i'를 1로 정하자
(('i'가 3보다 작다) 또는 ('i'와 3이 같다)) 인 동안 반복하자:
    'i'에 따라 나누자:
        1 인 경우:
            "하나"를 출력하자
        나머지는:
            틀"기타({'i'})"를 출력하자
    'i'에 1을 더하자
`)

	before := runProgram(t, prog)

	var buf bytes.Buffer
	if err := prog.Encode(&buf, bytecode.LangHaja); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	decoded, _, err := bytecode.ReadProgram(&buf)
	if err != nil {
		t.Fatalf("ReadProgram: %v", err)
	}

	after := runProgram(t, decoded)

	if len(before) != len(after) {
		t.Fatalf("output line count = %d after round-trip, want %d\nbefore: %v\nafter:  %v", len(after), len(before), before, after)
	}
	for i := range before {
		if before[i] != after[i] {
			t.Errorf("line %d = %q after round-trip, want %q", i, after[i], before[i])
		}
	}
}

// A wrong magic number or an unsupported version byte should fail with a
// clear .hn-specific error, not a raw gob decode error.
func TestReadProgramRejectsBadHeader(t *testing.T) {
	if _, _, err := bytecode.ReadProgram(bytes.NewReader([]byte("not a bytecode file at all"))); err == nil {
		t.Error("expected an error for a non-.hn file, got nil")
	}
}
