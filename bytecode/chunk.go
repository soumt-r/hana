package bytecode

// Instruction is one bytecode op plus its (optional) operand. Operand's
// concrete type depends on Op — see the comments in opcode.go.
type Instruction struct {
	Op      Opcode
	Operand interface{}
}

// Chunk is a compiled sequence of instructions plus the constant and name
// pools its PUSH_CONST/LOAD_VAR/SET_VAR/CALL/... instructions index into.
type Chunk struct {
	Instructions []Instruction
	Constants    []interface{}
	Names        []string
	symbols      symbolCache // Names as Symbols, worked out on first use (see Symbols)
}

func newChunk() *Chunk {
	return &Chunk{}
}

// emit appends an instruction and returns its index (useful for later
// back-patching a jump target once it's known).
func (c *Chunk) emit(op Opcode, operand interface{}) int {
	c.Instructions = append(c.Instructions, Instruction{Op: op, Operand: operand})
	return len(c.Instructions) - 1
}

// patchOperand rewrites an already-emitted instruction's operand — used to
// back-patch a forward jump once its target address is known.
func (c *Chunk) patchOperand(index int, operand interface{}) {
	c.Instructions[index].Operand = operand
}

// nextIndex is the instruction index the *next* emit call will produce,
// i.e. "here" for a jump target.
func (c *Chunk) nextIndex() int {
	return len(c.Instructions)
}

func (c *Chunk) addConstant(v interface{}) int {
	c.Constants = append(c.Constants, v)
	return len(c.Constants) - 1
}

// addName interns a name into the pool, reusing an existing entry if the
// same name was already added (keeps Names small and stable-indexed).
func (c *Chunk) addName(name string) int {
	for i, n := range c.Names {
		if n == name {
			return i
		}
	}
	c.Names = append(c.Names, name)
	return len(c.Names) - 1
}

// Function is a compiled function/method: its parameter list (name and
// optional default-value expression, compiled into their own tiny chunk so
// defaults can reference earlier parameters and the caller's scope, same as
// the tree-walking interpreter) and its body's Chunk. Access is only
// meaningful for class methods ("public"/"private"/"protected"); plain
// top-level functions leave it "".
type Function struct {
	Name       string
	Params     []Param
	BodyChunk  *Chunk
	Access     string
	ReturnType string // the [타입]을 돌려주는 annotation as written ("" = none); the returned value is checked against it

	paramSymbols symbolCache // the parameters' names as Symbols (see ParamSymbols)
}

// Param is one compiled function parameter.
type Param struct {
	Name    string
	Type    string // the [타입] annotation as written ("" = none / [아무거나]); checked on every call
	Default *Chunk // nil if no default; otherwise a tiny chunk that pushes the default value
}

// Program is a whole compiled source file: the top-level Chunk plus every
// function/class/interface declared in it, keyed by name.
type Program struct {
	Main       *Chunk
	Functions  map[string]*Function
	Classes    map[string]*ClassInfo
	Interfaces map[string]*InterfaceInfo
}

// ClassInfo is a compiled class: its own declared fields/methods/
// constructor only (never merged with ancestors — method/field lookup
// walks BaseClass at call time, exactly matching vm.findInClassChain, and
// deliberately reproducing the tree-walker's existing quirk that a
// NewExpression only initializes the leaf class's own declared fields, not
// inherited ones with no constructor assignment).
type ClassInfo struct {
	Name          string
	BaseClass     string // "" if none
	Interfaces    []string
	IsAbstract    bool // 밑설계하자: cannot be instantiated
	Fields        []FieldInfo
	StaticFields  []FieldInfo
	Methods       map[string]*Function
	StaticMethods map[string]*Function
	Constructor   *Function // nil if this class doesn't declare its own
}

// FieldInfo is one instance or static field declaration. Default is nil for
// a field declared with 준비하자 (no initializer, PUSH_NULL semantics).
type FieldInfo struct {
	Name    string
	Type    string // the field's [타입] annotation ("" = none); enforced on every write
	Access  string
	Default *Chunk
	Setter  *SetterInfo // non-nil if this field declared a setter
	Getter  *Chunk      // non-nil if this field declared a getter (가져올 때); runs with 나 bound, its return value is the read
}

// SetterInfo is a field's custom setter body (ParamName is "" if the setter
// declared no parameter to bind the assigned value to).
type SetterInfo struct {
	ParamName string
	Body      *Chunk
}

// InterfaceInfo is a compiled interface: just the method names it requires,
// checked against implementing classes' own Methods (not the chain) at
// vm.Run() start — exactly matching vm.Interpreter.Run()'s pre-flight scan
// of cls.Body only.
type InterfaceInfo struct {
	Name    string
	Methods []string
}
