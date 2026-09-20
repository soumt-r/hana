// Package symbol interns names as small integers for the whole process. Both
// engines find variables and class members by Symbol — comparing and indexing
// integers — instead of by string, which is what a program spends much of its
// time doing. A Symbol is never stored in a file: .hn files and source keep
// names, and each engine works out the Symbols it needs when it first runs them.
package symbol

import (
	"sync"
	"sync/atomic"
)

// A Symbol stands for one name.
type Symbol uint32

var table struct {
	mu    sync.Mutex
	ids   map[string]Symbol
	names atomic.Pointer[[]string] // read without locking; only Intern appends
}

func init() {
	table.ids = map[string]Symbol{}
	empty := []string{}
	table.names.Store(&empty)
}

// Intern returns the Symbol for name, creating it the first time.
func Intern(name string) Symbol {
	table.mu.Lock()
	defer table.mu.Unlock()
	if s, ok := table.ids[name]; ok {
		return s
	}
	names := append(*table.names.Load(), name)
	s := Symbol(len(names) - 1)
	table.ids[name] = s
	table.names.Store(&names)
	return s
}

// String is the name the Symbol stands for.
func (s Symbol) String() string { return (*table.names.Load())[s] }

// Count is how many Symbols exist; every Symbol is smaller than it.
func Count() int { return len(*table.names.Load()) }
