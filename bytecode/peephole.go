package bytecode

// The peephole pass: after a chunk is compiled, the arithmetic and comparisons that
// read variables and constants are fused into single BIN instructions. A statement
// like `'합'에 '가' * '나'를 더하자` compiles to LOAD_VAR, LOAD_VAR, MUL, ..., SET_VAR:
// each of those pushes and pops the operand stack and takes a trip through the
// dispatch loop. Fused, the VM reads the two variables straight from their slots and
// does the arithmetic in one step.
//
// What is fused (a binary operator is ADD, SUB, MUL, DIV, MOD, EQ, NEQ, LT, LTE, GT, GTE):
//
//	LOAD_VAR a, LOAD_VAR b, op      -> BIN{L: var a, R: var b}
//	LOAD_VAR a, PUSH_CONST c, op    -> BIN{L: var a, R: const c}
//	PUSH_CONST c, op                -> BIN{L: stack, R: const c}
//	LOAD_VAR b, op                  -> BIN{L: stack, R: var b}
//	BIN, SET_VAR x                  -> the same BIN storing into x
//	BIN, JUMP_IF_FALSE t            -> the same BIN jumping to t when the result is false
//	op, SET_VAR x                   -> BIN{L: stack, R: stack} storing into x (likewise JUMP_IF_FALSE)
//	BIN i+c -> i, JUMP t (t: BIN i<N ; if false) -> FOR_STEP (a counting loop's back edge, fuseLoopSteps)
//
// A sequence is only fused when no jump lands in the middle of it, and every jump
// target is moved to where its instruction went. Whatever the fused instruction cannot
// do quickly (an operand that is not a number, an error) it does the way the unfused
// instructions would have, so the result and the error are the same.

// Optimize fuses the instructions of every chunk of a program that a run spends its
// time in: the main chunk and the bodies of functions, methods and constructors.
func Optimize(p *Program) {
	seen := map[*Chunk]bool{}
	visit := func(c *Chunk) {
		if c != nil && !seen[c] {
			seen[c] = true
			optimizeChunk(c)
		}
	}
	visit(p.Main)
	for _, fn := range p.Functions {
		visitFunction(fn, visit)
	}
	for _, cls := range p.Classes {
		for _, m := range cls.Methods {
			visitFunction(m, visit)
		}
		for _, m := range cls.StaticMethods {
			visitFunction(m, visit)
		}
		visitFunction(cls.Constructor, visit)
	}
}

func visitFunction(fn *Function, visit func(*Chunk)) {
	if fn != nil {
		visit(fn.BodyChunk)
	}
}

func isBinaryOp(op Opcode) bool {
	switch op {
	case ADD, SUB, MUL, DIV, MOD, EQ, NEQ, LT, LTE, GT, GTE:
		return true
	}
	return false
}

// jumpTargets are the instruction indexes some jump or catch handler goes to.
func jumpTargets(ins []Instruction) map[int]bool {
	targets := map[int]bool{}
	for _, in := range ins {
		switch in.Op {
		case JUMP, JUMP_IF_FALSE, JUMP_IF_TRUE:
			targets[in.Operand.(int)] = true
		case TRY_PUSH:
			for _, c := range in.Operand.(*TryOperand).Catches {
				targets[c.HandlerPc] = true
			}
		case BIN:
			if b := in.Operand.(*BinOperand); b.Jump >= 0 {
				targets[b.Jump] = true
			}
		case FOR_STEP:
			f := in.Operand.(*ForStepOperand)
			targets[f.Body] = true
			targets[f.Test.Jump] = true
		}
	}
	return targets
}

func optimizeChunk(c *Chunk) {
	old := c.Instructions
	if len(old) < 2 {
		return
	}
	targets := jumpTargets(old)
	// free reports whether instructions i..i+n-1 can be fused: no jump goes into the
	// middle of them (a jump to the first is fine).
	free := func(i, n int) bool {
		if i+n > len(old) {
			return false
		}
		for k := i + 1; k < i+n; k++ {
			if targets[k] {
				return false
			}
		}
		return true
	}

	out := make([]Instruction, 0, len(old))
	moved := make([]int, len(old)+1) // old index -> new index
	for i := 0; i < len(old); {
		in := old[i]
		emit := func(fused Instruction, n int) {
			for k := 0; k < n; k++ {
				moved[i+k] = len(out)
			}
			out = append(out, fused)
			i += n
		}
		switch {
		case in.Op == LOAD_VAR && free(i, 3) && old[i+1].Op == LOAD_VAR && isBinaryOp(old[i+2].Op):
			emit(Instruction{Op: BIN, Operand: &BinOperand{Op: old[i+2].Op, L: BinArg{ArgVar, in.Operand.(int)}, R: BinArg{ArgVar, old[i+1].Operand.(int)}, Set: -1, Jump: -1}}, 3)
		case in.Op == LOAD_VAR && free(i, 3) && old[i+1].Op == PUSH_CONST && isBinaryOp(old[i+2].Op):
			emit(Instruction{Op: BIN, Operand: &BinOperand{Op: old[i+2].Op, L: BinArg{ArgVar, in.Operand.(int)}, R: BinArg{ArgConst, old[i+1].Operand.(int)}, Set: -1, Jump: -1}}, 3)
		case in.Op == PUSH_CONST && free(i, 2) && isBinaryOp(old[i+1].Op):
			emit(Instruction{Op: BIN, Operand: &BinOperand{Op: old[i+1].Op, L: BinArg{ArgStack, 0}, R: BinArg{ArgConst, in.Operand.(int)}, Set: -1, Jump: -1}}, 2)
		case in.Op == LOAD_VAR && free(i, 2) && isBinaryOp(old[i+1].Op):
			emit(Instruction{Op: BIN, Operand: &BinOperand{Op: old[i+1].Op, L: BinArg{ArgStack, 0}, R: BinArg{ArgVar, in.Operand.(int)}, Set: -1, Jump: -1}}, 2)
		case isBinaryOp(in.Op) && i+1 < len(old) && !targets[i+1] && (old[i+1].Op == SET_VAR || old[i+1].Op == JUMP_IF_FALSE):
			emit(Instruction{Op: BIN, Operand: &BinOperand{Op: in.Op, L: BinArg{ArgStack, 0}, R: BinArg{ArgStack, 0}, Set: -1, Jump: -1}}, 1)
		default:
			moved[i] = len(out)
			out = append(out, in)
			i++
			continue
		}
		// what the fused instruction can absorb after it
		last := &out[len(out)-1]
		b := last.Operand.(*BinOperand)
		if i < len(old) && !targets[i] && old[i].Op == SET_VAR {
			b.Set = old[i].Operand.(int)
			moved[i] = len(out) - 1
			i++
		} else if i < len(old) && !targets[i] && old[i].Op == JUMP_IF_FALSE {
			b.Jump = old[i].Operand.(int)
			moved[i] = len(out) - 1
			i++
		}
	}
	moved[len(old)] = len(out)

	if len(out) != len(old) {
		retarget(out, moved)
	}
	fuseLoopSteps(c, out)
	c.Instructions = out
}

// retarget moves every jump target in out from its old instruction index to its new one.
func retarget(out []Instruction, moved []int) {
	for k := range out {
		switch out[k].Op {
		case JUMP, JUMP_IF_FALSE, JUMP_IF_TRUE:
			out[k].Operand = moved[out[k].Operand.(int)]
		case TRY_PUSH:
			t := out[k].Operand.(*TryOperand)
			for ci := range t.Catches {
				t.Catches[ci].HandlerPc = moved[t.Catches[ci].HandlerPc]
			}
		case BIN:
			if b := out[k].Operand.(*BinOperand); b.Jump >= 0 {
				b.Jump = moved[b.Jump]
			}
		case FOR_STEP:
			// Test is the BIN at the loop's head, which moves its own Jump.
			f := out[k].Operand.(*ForStepOperand)
			f.Body = moved[f.Body]
		}
	}
}

// fuseLoopSteps turns the back edge of a counting loop into one FOR_STEP. After the
// fusing above, `1부터 N까지 반복하자` ends each pass with
//
//	k:    BIN ADD var i, const 1 -> i
//	k+1:  JUMP -> t
//	t:    BIN LTE var i, const N ; if false -> end   (N may also be a variable)
//
// and FOR_STEP at k does all three: add, compare, and go to t+1 or end. A range whose
// ends are not both literals (`1부터 'n'까지`) counts by a hidden step variable and its
// head is three BINs instead, `end - i`, `* step`, `>= 0`; FOR_STEP (Span) does those
// too and goes to t+3 or end. The JUMP stays where it is, so no instruction moves;
// FOR_STEP falls back to it when it cannot take the fast path (see the VM), and a jump
// that lands on it still works.
func fuseLoopSteps(c *Chunk, out []Instruction) {
	for k := 0; k+1 < len(out); k++ {
		if out[k].Op != BIN || out[k+1].Op != JUMP {
			continue
		}
		inc := out[k].Operand.(*BinOperand)
		if inc.Op != ADD || inc.L.Kind != ArgVar || inc.R.Kind == ArgStack || inc.R == inc.L || inc.Set != inc.L.Index || inc.Jump >= 0 {
			continue
		}
		t := out[k+1].Operand.(int)
		if f := compareHead(out, t, inc); f != nil {
			out[k] = Instruction{Op: FOR_STEP, Operand: f}
		} else if f := spanHead(c, out, t, inc); f != nil {
			out[k] = Instruction{Op: FOR_STEP, Operand: f}
		}
	}
}

// binAt is the BIN at out[i], or nil.
func binAt(out []Instruction, i int) *BinOperand {
	if i < 0 || i >= len(out) || out[i].Op != BIN {
		return nil
	}
	return out[i].Operand.(*BinOperand)
}

// compareHead matches a loop head `counter <cmp> end ; if false -> exit` at t.
func compareHead(out []Instruction, t int, inc *BinOperand) *ForStepOperand {
	test := binAt(out, t)
	if test == nil {
		return nil
	}
	switch test.Op {
	case LT, LTE, GT, GTE:
	default:
		return nil
	}
	if test.L != inc.L || test.R.Kind == ArgStack || test.R == inc.L || test.Set >= 0 || test.Jump < 0 {
		return nil
	}
	return &ForStepOperand{Inc: inc, Test: test, End: test.R, Body: t + 1}
}

// spanHead matches the head compileForRange gives a range whose ends are not both
// literals: `end - counter`, `* step`, `>= 0 ; if false -> exit` at t, t+1, t+2, where
// step is what inc adds.
func spanHead(c *Chunk, out []Instruction, t int, inc *BinOperand) *ForStepOperand {
	sub, mul, test := binAt(out, t), binAt(out, t+1), binAt(out, t+2)
	if sub == nil || mul == nil || test == nil {
		return nil
	}
	if sub.Op != SUB || sub.L.Kind == ArgStack || sub.R != inc.L || sub.Set >= 0 || sub.Jump >= 0 {
		return nil
	}
	if mul.Op != MUL || mul.L.Kind != ArgStack || mul.R != inc.R || mul.Set >= 0 || mul.Jump >= 0 {
		return nil
	}
	if test.Op != GTE || test.L.Kind != ArgStack || test.R.Kind != ArgConst || test.Set >= 0 || test.Jump < 0 {
		return nil
	}
	if zero, ok := c.Constants[test.R.Index].(float64); !ok || zero != 0 {
		return nil
	}
	return &ForStepOperand{Inc: inc, Test: test, Span: true, End: sub.L, Body: t + 3}
}
