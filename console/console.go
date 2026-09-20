// Package console is where a running program's output goes.
//
// On a terminal every print is written at once, as it always was. When standard
// output is a file or a pipe, prints are collected and written in larger pieces:
// a program that prints a lot spends most of its time in one system call per
// line otherwise. The collected text is written when it grows large, before the
// program reads a line of input, before anything is written to standard error,
// when the program ends (Flush, Exit), and in any case within flushDelay of the
// print — so a program that prints and then waits (a server, a sleep) still shows
// its output in time.
package console

import (
	"os"
	"sync"
	"time"
)

const (
	flushDelay = 25 * time.Millisecond
	flushSize  = 32 << 10
)

var (
	mu       sync.Mutex
	buf      []byte
	timer    *time.Timer
	checked  bool
	buffered bool
)

// Print writes s to standard output.
func Print(s string) { write(s, false) }

// Println writes s and a line break to standard output.
func Println(s string) { write(s, true) }

func write(s string, newline bool) {
	mu.Lock()
	defer mu.Unlock()
	if !checked {
		checked = true
		buffered = !isTerminal(os.Stdout)
	}
	if !buffered {
		if newline {
			s += "\n"
		}
		os.Stdout.WriteString(s)
		return
	}
	buf = append(buf, s...)
	if newline {
		buf = append(buf, '\n')
	}
	if len(buf) >= flushSize {
		flushLocked()
	} else if timer == nil {
		timer = time.AfterFunc(flushDelay, Flush)
	}
}

// Flush writes what has been collected.
func Flush() {
	mu.Lock()
	defer mu.Unlock()
	flushLocked()
}

func flushLocked() {
	if timer != nil {
		timer.Stop()
		timer = nil
	}
	if len(buf) > 0 {
		os.Stdout.Write(buf)
		buf = buf[:0]
	}
}

// Exit flushes and ends the process.
func Exit(code int) {
	Flush()
	os.Exit(code)
}

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
