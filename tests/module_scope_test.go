package tests

// Module scope: an imported function, method or constructor keeps the scope of the
// module it came from. It can call the module's other functions, use what the module
// itself imported, and create the module's other classes, none of which the importer
// asked for; and the importer's own functions of the same names do not get in the way.
// Both engines must agree.

import (
	"strings"
	"testing"
)

const scopedLib = `
<두배>를 만들자 ('값'):
    ('값' * 2)를 돌려주자

<더하기>를 만들자 ('값'):
    '결과'를 ('값' + <두배>('값'))로 정하자
    '결과'를 돌려주자

<설명하기>를 만들자 ('값'):
    '글'을 틀"값의 두배는 {<두배>('값')}"으로 정하자
    '글'을 돌려주자

[알갱이]를 설계하자:
    '무게'를 [숫자]인 0으로 정하여 물려주자
    처음 만들어질 때 ([숫자]인 '값') 다음과 같이 하자:
        '나'의 '무게'를 <두배>('값')로 정하자

[상자]를 설계하자:
    '크기'를 [숫자]인 0으로 정하여 물려주자
    처음 만들어질 때 ([숫자]인 '값') 다음과 같이 하자:
        '나'의 '크기'를 <두배>('값')로 정하자
    [숫자]를 돌려주는 <세배크기>를 만들자 ():
        ('나'의 '크기' + <두배>('나'의 '크기'))를 돌려주자
    [숫자]를 돌려주는 <알갱이무게>를 만들자 ():
        '알'을 [알갱이]인 새로운 [알갱이](4)로 정하자
        '알'의 '무게'를 돌려주자
`

const scopedMain = `
"lib.hr"에서 <더하기>를 가져오자
"lib.hr"에서 <설명하기>를 가져오자
"lib.hr"에서 '상자'를 가져오자

<두배>를 만들자 ('값'):
    (0 - 1)를 돌려주자

<더하기>(5)를 출력하자
<설명하기>(4)를 출력하자
'상자'를 [상자]인 새로운 [상자](3)로 정하자
'상자'의 <세배크기>()를 출력하자
'상자'의 <알갱이무게>()를 출력하자
<두배>(4)를 출력하자
`

func TestImportedCodeUsesItsOwnModulesFunctions(t *testing.T) {
	inTempDir(t, map[string]string{"lib.hr": scopedLib})
	want := "15|값의 두배는 8|18|8|-1"

	tree, err := runHari(t, scopedMain)
	if err != nil {
		t.Fatalf("tree-walker: %v", err)
	}
	bc, err := runBytecode(t, scopedMain)
	if err != nil {
		t.Fatalf("bytecode: %v", err)
	}
	if got := strings.Join(tree.Output, "|"); got != want {
		t.Errorf("tree-walker printed %q, want %q", got, want)
	}
	if got := strings.Join(bc.Output, "|"); got != want {
		t.Errorf("bytecode printed %q, want %q", got, want)
	}
}

func TestAModuleUsesWhatItImportedItself(t *testing.T) {
	inTempDir(t, map[string]string{
		"low.hr": "<낮은>을 만들자 ('값'):\n    ('값' + 100)를 돌려주자\n",
		"mid.hr": "\"low.hr\"에서 <낮은>을 가져오자\n<중간>을 만들자 ('값'):\n    '결과'를 <낮은>('값')로 정하자\n    '결과'를 돌려주자\n",
	})
	main := "\"mid.hr\"에서 <중간>을 가져오자\n<중간>(1)을 출력하자\n"
	tree, err := runHari(t, main)
	if err != nil {
		t.Fatalf("tree-walker: %v", err)
	}
	bc, err := runBytecode(t, main)
	if err != nil {
		t.Fatalf("bytecode: %v", err)
	}
	if got := strings.Join(tree.Output, "|") + "#" + strings.Join(bc.Output, "|"); got != "101#101" {
		t.Errorf("printed %q, want 101#101", got)
	}
}

func TestTwoModulesMayUseTheSameHelperName(t *testing.T) {
	inTempDir(t, map[string]string{
		"a.hr": "<도우미>를 만들자 ():\n    \"A\"를 돌려주자\n<가>를 만들자 ():\n    <도우미>()를 돌려주자\n",
		"b.hr": "<도우미>를 만들자 ():\n    \"B\"를 돌려주자\n<나>를 만들자 ():\n    <도우미>()를 돌려주자\n",
	})
	main := "\"a.hr\"에서 <가>를 가져오자\n\"b.hr\"에서 <나>를 가져오자\n<가>()를 출력하자\n<나>()를 출력하자\n"
	tree, err := runHari(t, main)
	if err != nil {
		t.Fatalf("tree-walker: %v", err)
	}
	bc, err := runBytecode(t, main)
	if err != nil {
		t.Fatalf("bytecode: %v", err)
	}
	if got := strings.Join(tree.Output, "|") + "#" + strings.Join(bc.Output, "|"); got != "A|B#A|B" {
		t.Errorf("printed %q, want A|B#A|B", got)
	}
}

func TestAPackageUsesItsOwnHelpersAndItsDependencies(t *testing.T) {
	const title = "example.test/owner/title"
	installed(t, map[string]string{
		title + "@2.0.0/hari/index.hr": "<제목붙이기>를 만들자 ('글'):\n    '결과'를 틀\"== {'글'} ==\"으로 정하자\n    '결과'를 돌려주자\n",
		greetPath + "@1.0.0/hari/index.hr": "[" + title + "]에서 <제목붙이기>를 가져오자\n" +
			"<큰소리>를 만들자 ('글'):\n    (('글' + \"!\"))를 돌려주자\n" +
			"<인사하기>를 만들자 ('이름'):\n    '결과'를 <제목붙이기>(<큰소리>('이름'))로 정하자\n    '결과'를 돌려주자\n",
		greetPath + "@1.0.0/kanade/index.knd": "【" + title + "】から〈見出し〉を持ってこよう\n" +
			"〈強調〉を作ろう(『文』):\n    『結果』を(『文』 + 「！」)にしよう\n    『結果』を返そう\n" +
			"〈挨拶〉を作ろう(『名前』):\n    『結果』を〈見出し〉(〈強調〉(『名前』))にしよう\n    『結果』を返そう\n",
		title + "@2.0.0/kanade/index.knd": "〈見出し〉を作ろう(『文』):\n    『結果』を枠「== {『文』} ==」にしよう\n    『結果』を返そう\n",
	}, map[string]string{
		"hana.json": `{"dependencies": {"` + greetPath + `": "1.0.0"}}`,
		"hana-lock.json": `{"` + greetPath + `": {"version": "1.0.0", "commit": "a"},
"` + title + `": {"version": "2.0.0", "commit": "b"}}`,
	})

	hari := "[" + greetPath + "]에서 <인사하기>를 가져오자\n<인사하기>(\"하나\")를 출력하자\n"
	tree, err := runHari(t, hari)
	if err != nil {
		t.Fatalf("hari tree-walker: %v", err)
	}
	bc, err := runBytecode(t, hari)
	if err != nil {
		t.Fatalf("hari bytecode: %v", err)
	}
	if got := strings.Join(tree.Output, "|") + "#" + strings.Join(bc.Output, "|"); got != "== 하나! ==#== 하나! ==" {
		t.Errorf("hari printed %q", got)
	}

	kanade := "【" + greetPath + "】から〈挨拶〉を持ってこよう\n〈挨拶〉(「カナデ」)を出力しよう\n"
	tree2, err := runKanade(t, kanade)
	if err != nil {
		t.Fatalf("kanade tree-walker: %v", err)
	}
	bc2, err := runKanadeBytecode(t, kanade)
	if err != nil {
		t.Fatalf("kanade bytecode: %v", err)
	}
	if got := strings.Join(tree2.Output, "|") + "#" + strings.Join(bc2.Output, "|"); got != "== カナデ！ ==#== カナデ！ ==" {
		t.Errorf("kanade printed %q", got)
	}
}
