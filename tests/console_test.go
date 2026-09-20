package tests

// Output is collected when standard output is a pipe or a file (package
// console). These tests run the real hana with its output piped and check that
// nothing is late, lost or out of order.

import (
	"bufio"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestPipedOutputKeepsItsOrderWithErrors(t *testing.T) {
	hana, _ := packTools(t)
	inTempDir(t, map[string]string{"boom.hj": "\"먼저\"를 출력하자\n새로운 [오류](\"터졌어요\")를 발생시키자\n"})
	for _, flags := range [][]string{{}, {"--bc"}} {
		out, err := exec.Command(hana, append([]string{"run"}, append(flags, "boom.hj")...)...).CombinedOutput()
		text := string(out)
		if err == nil || strings.Index(text, "먼저") < 0 || strings.Index(text, "먼저") > strings.Index(text, "터졌어요") {
			t.Errorf("%v: the output should come before the error, got %q (%v)", flags, text, err)
		}
	}
}

func TestPipedOutputAppearsWhileTheProgramWaits(t *testing.T) {
	hana, _ := packTools(t)
	inTempDir(t, map[string]string{"wait.hj": "[날짜]에서 <기다리기>를 가져오자\n\"안녕\"을 출력하자\n<기다리기>(3)을 실행하자\n\"끝\"을 출력하자\n"})
	cmd := exec.Command(hana, "run", "wait.hj")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer cmd.Process.Kill()
	first := make(chan string, 1)
	go func() {
		line, _ := bufio.NewReader(stdout).ReadString('\n')
		first <- line
	}()
	select {
	case line := <-first:
		if strings.TrimSpace(line) != "안녕" {
			t.Errorf("got %q", line)
		}
	case <-time.After(2 * time.Second):
		t.Error("the first line should be visible long before the program ends")
	}
}

func TestPromptIsVisibleBeforeTheProgramReadsInput(t *testing.T) {
	hana, _ := packTools(t)
	inTempDir(t, map[string]string{"ask.hj": "\"이름: \"을 이어출력하자\n'이름'을 [문자열]로 입력받자\n틀\"안녕, {'이름'}\"을 출력하자\n"})
	cmd := exec.Command(hana, "run", "ask.hj")
	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer cmd.Process.Kill()
	prompt := make(chan string, 1)
	go func() {
		buf := make([]byte, len("이름: "))
		n, _ := stdout.Read(buf)
		prompt <- string(buf[:n])
	}()
	select {
	case got := <-prompt:
		if got != "이름: " {
			t.Errorf("got %q", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the prompt should be visible before the program waits for input")
	}
	stdin.Write([]byte("하나\n"))
	stdin.Close()
	rest, _ := bufio.NewReader(stdout).ReadString('\n')
	if strings.TrimSpace(rest) != "안녕, 하나" {
		t.Errorf("got %q", rest)
	}
	cmd.Wait()
}
