package strcat

import (
	"strings"
	"sync"
	"testing"
)

func TestJoinMatchesPlainConcatenation(t *testing.T) {
	acc := ""
	want := ""
	for i := 0; i < 5000; i++ {
		piece := strings.Repeat("가", i%7) + "x"
		acc = Join(acc, piece)
		want += piece
		if acc != want {
			t.Fatalf("step %d: got %d bytes, want %d", i, len(acc), len(want))
		}
	}
}

func TestBranchingFromTheSameStringKeepsBothResults(t *testing.T) {
	base := Join(strings.Repeat("a", 300), "b")
	first := Join(base, "1")
	second := Join(base, "2")
	third := Join(first, "3")
	if first != base+"1" || second != base+"2" || third != base+"13" {
		t.Fatal("a join changed a string that was already handed out")
	}
	if base != strings.Repeat("a", 300)+"b" {
		t.Fatal("the left side changed")
	}
}

func TestStringsBuiltAlongsideEachOtherStayApart(t *testing.T) {
	var accs [9]string
	var wants [9]string
	for i := 0; i < 2000; i++ {
		k := i % len(accs)
		piece := string(rune('a' + k))
		accs[k] = Join(accs[k], strings.Repeat(piece, 3))
		wants[k] += strings.Repeat(piece, 3)
	}
	for k := range accs {
		if accs[k] != wants[k] {
			t.Fatalf("accumulator %d differs", k)
		}
	}
}

func TestSlicesOfAResultAreNotMistakenForIt(t *testing.T) {
	long := Join(strings.Repeat("z", 400), "end")
	prefix := long[:len(long)-3]
	joined := Join(prefix, "!")
	if joined != strings.Repeat("z", 400)+"!" || long != strings.Repeat("z", 400)+"end" {
		t.Fatal("a slice of a result was extended in place")
	}
}

func TestJoinFromManyGoroutines(t *testing.T) {
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			acc, want := "", ""
			for i := 0; i < 2000; i++ {
				acc = Join(acc, "ab")
				want += "ab"
			}
			if acc != want {
				t.Error("goroutine result differs")
			}
		}()
	}
	wg.Wait()
}
