package bytecode

import (
	"sync/atomic"

	"github.com/soumt-r/hana/symbol"
)

func internAll(names []string) []symbol.Symbol {
	out := make([]symbol.Symbol, len(names))
	for i, n := range names {
		out[i] = symbol.Intern(n)
	}
	return out
}

// symbolCache holds a lazily computed []Symbol next to the names it came from:
// a chunk works out its Symbols the first time it runs (a .hn file stores names).
type symbolCache struct {
	p atomic.Pointer[[]symbol.Symbol]
}

func (c *symbolCache) get(names []string) []symbol.Symbol {
	if p := c.p.Load(); p != nil && len(*p) == len(names) {
		return *p
	}
	syms := internAll(names)
	c.p.Store(&syms)
	return syms
}

// Symbols are the Symbols of Names, index for index.
func (c *Chunk) Symbols() []symbol.Symbol { return c.symbols.get(c.Names) }

// ParamSymbols are the Symbols of the parameters' names.
func (f *Function) ParamSymbols() []symbol.Symbol {
	if p := f.paramSymbols.p.Load(); p != nil && len(*p) == len(f.Params) {
		return *p
	}
	names := make([]string, len(f.Params))
	for i, p := range f.Params {
		names[i] = p.Name
	}
	return f.paramSymbols.get(names)
}
