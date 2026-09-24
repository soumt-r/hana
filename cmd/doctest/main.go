package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/soumt-r/hana/console"
	"github.com/soumt-r/hana/lexer/hari"
	kanadelexer "github.com/soumt-r/hana/lexer/kanade"
	parser "github.com/soumt-r/hana/parser/hari"
	kanadeparser "github.com/soumt-r/hana/parser/kanade"
	"github.com/soumt-r/hana/stdlib"
	"github.com/soumt-r/hana/vm"
)

var (
	hariBlockRegex   = regexp.MustCompile("(?s)```hari([^\\n]*)\\n(.*?)```")
	kanadeBlockRegex = regexp.MustCompile("(?s)```kanade([^\\n]*)\\n(.*?)```")
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("사용법: go run main.go <문서 디렉토리 경로>")
		os.Exit(1)
	}

	docsDir := os.Args[1]
	fileCount := 0
	blockCount := 0
	errorCount := 0

	fmt.Println("🔍 마크다운(Markdown) 문서 내 Hari 코드 블록 런타임 검증을 시작합니다...")

	err := filepath.Walk(docsDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			fileCount++
			blocks := extractBlocks(string(content))

			for _, b := range blocks {
				// `skip` marks an example that needs the network or a peer, so it is only shown.
				if strings.Contains(b.meta, "skip") {
					continue
				}
				blockCount++
				expectError := strings.Contains(b.meta, "fail")

				var err error
				if b.isKanade {
					err = runKanadeCode(b.code)
				} else {
					err = runCode(b.code)
				}

				if expectError {
					if err == nil {
						errorCount++
						fmt.Printf("\n❌ [예상된 에러 누락] 파일: %s\n", path)
						fmt.Printf("코드:\n%s\n", b.code)
						fmt.Printf("기대: 에러가 발생해야 하는데 정상 실행되었습니다!\n")
					}
				} else {
					if err != nil {
						errorCount++
						fmt.Printf("\n❌ [예상치 못한 에러] 파일: %s\n", path)
						fmt.Printf("코드:\n%s\n", b.code)
						fmt.Printf("발생한 에러: %v\n", err)
					}
				}
			}
		}
		return nil
	})

	if err != nil {
		fmt.Printf("디렉토리 순회 중 에러 발생: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n=========================================")
	fmt.Printf("총 검증된 파일: %d개\n", fileCount)
	fmt.Printf("총 실행된 코드 블록: %d개\n", blockCount)

	if errorCount > 0 {
		fmt.Printf("🚨 총 %d개의 테스트가 실패했습니다.\n", errorCount)
		os.Exit(1)
	} else {
		fmt.Println("✅ 모든 코드 블록이 완벽하게 검증되었습니다! (의도된 에러 포함)")
	}
}

type docBlock struct {
	meta     string
	code     string
	isKanade bool
}

// extractBlocks finds every ```hari and ```kanade fenced block in content,
// in source order (a file could in principle mix both, though today it
// won't — hari-docs only has ```hari, kanade-docs only ```kanade).
func extractBlocks(content string) []docBlock {
	var blocks []docBlock
	for _, m := range hariBlockRegex.FindAllStringSubmatch(content, -1) {
		blocks = append(blocks, docBlock{meta: strings.TrimSpace(m[1]), code: strings.TrimSpace(m[2])})
	}
	for _, m := range kanadeBlockRegex.FindAllStringSubmatch(content, -1) {
		blocks = append(blocks, docBlock{meta: strings.TrimSpace(m[1]), code: strings.TrimSpace(m[2]), isKanade: true})
	}
	return blocks
}

func runKanadeCode(code string) error {
	l := kanadelexer.New(code)
	p := kanadeparser.New(l)
	prog := p.ParseProgram()

	if len(p.Errors()) > 0 {
		return fmt.Errorf("파싱 에러: %v", p.Errors())
	}

	interpreter := vm.NewInterpreter(prog, vm.JapaneseConfig)
	stdlib.RegisterStandardLibrary(interpreter)

	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := interpreter.Run()
	console.Flush()

	w.Close()
	os.Stdout = oldStdout

	return err
}

func runCode(code string) error {
	l := hari.New(code)
	p := parser.New(l)
	prog := p.ParseProgram()

	if len(p.Errors()) > 0 {
		return fmt.Errorf("파싱 에러: %v", p.Errors())
	}

	interpreter := vm.NewInterpreter(prog)
	stdlib.RegisterStandardLibrary(interpreter)

	// Suppress output
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := interpreter.Run()
	console.Flush()

	w.Close()
	os.Stdout = oldStdout

	return err
}
