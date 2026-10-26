package tests

import (
	"reflect"
	"testing"
)

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

// 부모 must start looking above the class that declares the running method,
// not above the receiver's class. Regression test for a bug where, three
// classes deep, the middle class's 부모 call (made on a grandchild object)
// found the middle class's own method again and recursed until RecursionError
// — for methods and for 부모의 <처음 만들어질 때>(...) alike.
func TestSuperCallThreeLevelsDeep(t *testing.T) {
	c := dualRun{
		hari: `[가]를 설계하자:
    <이름>을 만들자 ():
        "가"를 돌려주자

[가]를 바탕으로 [나]를 설계하자:
    <이름>을 만들자 ():
        틀"{부모의 <이름>()}>나"를 돌려주자

[나]를 바탕으로 [다]를 설계하자:
    <이름>을 만들자 ():
        틀"{부모의 <이름>()}>다"를 돌려주자

[다]를 바탕으로 [라]를 설계하자:
    <별명>을 만들자 ():
        틀"{부모의 <이름>()}!"를 돌려주자

(새로운 [나]())의 <이름>()을 출력하자
(새로운 [다]())의 <이름>()을 출력하자
(새로운 [라]())의 <별명>()을 출력하자

[갑]을 설계하자:
    '기록'을 [문자열]인 ""로 정하자
    처음 만들어질 때 ([문자열]인 '값') 다음과 같이 하자:
        '나'의 '기록'을 틀"갑({'값'})"으로 정하자

[갑]을 바탕으로 [을]을 설계하자:
    처음 만들어질 때 ([문자열]인 '값') 다음과 같이 하자:
        부모의 <처음 만들어질 때>('값')을 실행하자
        '나'의 '기록'을 틀"{'나'의 '기록'}>을"로 정하자

[을]을 바탕으로 [병]을 설계하자:
    처음 만들어질 때 ([문자열]인 '값') 다음과 같이 하자:
        부모의 <처음 만들어질 때>('값')을 실행하자
        '나'의 '기록'을 틀"{'나'의 '기록'}>병"으로 정하자

(새로운 [병]("x"))의 '기록'을 출력하자
`,
		kanade: `【甲】を設計しよう:
    〈名前〉を作ろう():
        「甲」を返そう

【甲】をもとにして【乙】を設計しよう:
    〈名前〉を作ろう():
        枠「{親の〈名前〉()}>乙」を返そう

【乙】をもとにして【丙】を設計しよう:
    〈名前〉を作ろう():
        枠「{親の〈名前〉()}>丙」を返そう

【丙】をもとにして【丁】を設計しよう:
    〈あだ名〉を作ろう():
        枠「{親の〈名前〉()}!」を返そう

(新しい【乙】())の〈名前〉()を出力しよう
(新しい【丙】())の〈名前〉()を出力しよう
(新しい【丁】())の〈あだ名〉()を出力しよう

【子】を設計しよう:
    『記録』を【文字列】の「」にしよう
    最初に作られる時 (【文字列】の『値』) 次のようにしよう:
        私の『記録』を枠「子({『値』})」にしよう

【子】をもとにして【丑】を設計しよう:
    最初に作られる時 (【文字列】の『値』) 次のようにしよう:
        親の〈最初に作られる時〉(『値』)を実行しよう
        私の『記録』を枠「{私の『記録』}>丑」にしよう

【丑】をもとにして【寅】を設計しよう:
    最初に作られる時 (【文字列】の『値』) 次のようにしよう:
        親の〈最初に作られる時〉(『値』)を実行しよう
        私の『記録』を枠「{私の『記録』}>寅」にしよう

(新しい【寅】(「x」))の『記録』を出力しよう
`,
	}
	wantKo := []string{"가>나", "가>나>다", "가>나>다!", "갑(x)>을>병"}
	wantJa := []string{"甲>乙", "甲>乙>丙", "甲>乙>丙!", "子(x)>丑>寅"}
	outs, errs := runAllFour(t, c)
	for i, name := range []string{"hari tree", "hari bytecode", "kanade tree", "kanade bytecode"} {
		if errs[i] != nil {
			t.Errorf("%s: unexpected error: %v", name, errs[i])
			continue
		}
		want := wantKo
		if i >= 2 {
			want = wantJa
		}
		if !reflect.DeepEqual(outs[i], want) {
			t.Errorf("%s: output = %v, want %v", name, outs[i], want)
		}
	}
}

// 부모 in a class with no parent looks in that class itself — on every
// engine (the bytecode VM used to raise MethodNotFoundError instead).
func TestSuperCallWithoutParent(t *testing.T) {
	c := dualRun{
		hari: `[가]를 설계하자:
    <이름>을 만들자 ():
        "가"를 돌려주자
    <부름>을 만들자 ():
        부모의 <이름>()을 돌려주자

[가]를 바탕으로 [나]를 설계하자:
    <이름>을 만들자 ():
        "나"를 돌려주자

(새로운 [가]())의 <부름>()을 출력하자
(새로운 [나]())의 <부름>()을 출력하자
`,
		kanade: `【甲】を設計しよう:
    〈名前〉を作ろう():
        「甲」を返そう
    〈呼び〉を作ろう():
        親の〈名前〉()を返そう

【甲】をもとにして【乙】を設計しよう:
    〈名前〉を作ろう():
        「乙」を返そう

(新しい【甲】())の〈呼び〉()を出力しよう
(新しい【乙】())の〈呼び〉()を出力しよう
`,
	}
	wantKo, wantJa := []string{"가", "가"}, []string{"甲", "甲"}
	outs, errs := runAllFour(t, c)
	for i, name := range []string{"hari tree", "hari bytecode", "kanade tree", "kanade bytecode"} {
		if errs[i] != nil {
			t.Errorf("%s: unexpected error: %v", name, errs[i])
			continue
		}
		want := wantKo
		if i >= 2 {
			want = wantJa
		}
		if !reflect.DeepEqual(outs[i], want) {
			t.Errorf("%s: output = %v, want %v", name, outs[i], want)
		}
	}
}
