package tests

import "testing"

// Runtime spec 3.1.4: if a child class declares no constructor of its own,
// the parent's constructor must be found and run automatically. Regression
// test for a bug where NewExpression only looked in the leaf class's own
// body and never walked the BaseClass chain, so the parent constructor
// silently never ran.
func TestInheritedConstructorRunsAutomatically(t *testing.T) {
	interp, err := runHari(t, `
[동물]을 설계하자:
    '이름'을 [문자열]인 ""로 정하여 물려주자
    처음 만들어질 때 ([문자열]인 '초기이름') 다음과 같이 하자:
        '나'의 '이름'을 '초기이름'으로 정하자

[동물]을 바탕으로 [개]를 설계하자:
    [문자열]을 돌려주는 <소개>를 만들자 ():
        '나'의 '이름'을 돌려주자

'개'를 [개]인 새로운 [개]("바둑이")로 정하자
'결과'를 '개'의 <소개>()로 정하자
`)
	if err != nil {
		t.Fatalf("unexpected runtime error: %v", err)
	}
	if got, _ := interp.GlobalEnv.Get("결과"); got != "바둑이" {
		t.Errorf("결과 = %v, want \"바둑이\" (parent constructor should have set '이름')", got)
	}
}

// The explicit form (`부모의 <처음 만들어질 때>(...)`) must still work and
// still enforce MissingArgumentError for its own parameters.
func TestExplicitSuperConstructorCall(t *testing.T) {
	interp, err := runHari(t, `
[동물]을 설계하자:
    '이름'을 [문자열]인 ""로 정하여 물려주자
    처음 만들어질 때 ([문자열]인 '초기이름') 다음과 같이 하자:
        '나'의 '이름'을 '초기이름'으로 정하자

[동물]을 바탕으로 [개]를 설계하자:
    처음 만들어질 때 ([문자열]인 '이름2') 다음과 같이 하자:
        부모의 <처음 만들어질 때>('이름2')을 실행하자
    [문자열]을 돌려주는 <소개>를 만들자 ():
        '나'의 '이름'을 돌려주자

'개'를 [개]인 새로운 [개]("바둑이")로 정하자
'결과'를 '개'의 <소개>()로 정하자
`)
	if err != nil {
		t.Fatalf("unexpected runtime error: %v", err)
	}
	if got, _ := interp.GlobalEnv.Get("결과"); got != "바둑이" {
		t.Errorf("결과 = %v, want \"바둑이\"", got)
	}

	_, err = runHari(t, `
[동물]을 설계하자:
    처음 만들어질 때 ([문자열]인 '초기이름') 다음과 같이 하자:
        '나'의 '이름'을 '초기이름'으로 정하자

[동물]을 바탕으로 [개]를 설계하자:
    처음 만들어질 때 ([문자열]인 '이름2') 다음과 같이 하자:
        부모의 <처음 만들어질 때>()을 실행하자

'개'를 [개]인 새로운 [개]("바둑이")로 정하자
`)
	requireErrorContains(t, err, "MissingArgumentError")
}
