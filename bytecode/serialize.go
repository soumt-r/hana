package bytecode

import (
	"encoding/gob"
	"fmt"
	"io"
)

// hnMagic/hnVersion form a .hn file's fixed-size header, ahead of the
// gob-encoded *Program payload: 4 magic bytes so a wrong/corrupt file is
// rejected immediately with a clear error instead of a cryptic gob decode
// failure, then 1 version byte so a future breaking change to the
// instruction set or Program's shape (this session added a dozen opcodes —
// it'll happen again) fails the same way on an old hana build reading a
// newer file, rather than silently misreading it.
//
// The payload itself is plain encoding/gob, not a hand-rolled binary
// layout: Instruction.Operand is an interface{} holding one of several
// concrete *Operand struct types (or a bare int/bool/string, or nil), and
// gob already solves exactly this "heterogeneous value behind an
// interface" problem via type registration (see init below) — writing a
// custom encoder/decoder for every opcode's operand shape by hand would
// just be reimplementing gob worse. The trade-off is that .hn is a Go/gob
// format tied to these struct definitions, not a portable, documented wire
// format — acceptable since hana itself is the only reader, same as how
// Go's own build cache uses gob for internal artifacts nobody outside the
// toolchain needs to parse.
const (
	hnMagic   = "HNB\x00"
	hnVersion = 2 // v1 had no Lang byte — bumped rather than made optional, per the version byte's own stated purpose
)

// Lang records which source language a .hn file's Program was compiled
// from, since Program itself (just instructions/constants/names) carries
// no trace of it but bcstdlib still needs to know which builtin names to
// register before running it (see bcstdlib.LangConfig / NewKanadeCompiler).
type Lang byte

const (
	LangHaja Lang = iota
	LangKanade
)

func init() {
	gob.Register(&CallOperand{})
	gob.Register(&NewObjectOperand{})
	gob.Register(&MemberOperand{})
	gob.Register(&StaticFieldOperand{})
	gob.Register(&TryOperand{})
	gob.Register(&ImportOperand{})
	gob.Register(&TypedSetOperand{})
	gob.Register(&Chunk{}) // RUN_FINALLY's operand is a bare *Chunk
}

// Encode writes prog to w as a .hn file, tagged with lang.
func (prog *Program) Encode(w io.Writer, lang Lang) error {
	if _, err := io.WriteString(w, hnMagic); err != nil {
		return err
	}
	if _, err := w.Write([]byte{hnVersion, byte(lang)}); err != nil {
		return err
	}
	if err := gob.NewEncoder(w).Encode(prog); err != nil {
		return fmt.Errorf(".hn: failed to encode program: %w", err)
	}
	return nil
}

// ReadProgram reads a *Program and its Lang tag, as previously written by
// (*Program).Encode.
func ReadProgram(r io.Reader) (*Program, Lang, error) {
	header := make([]byte, len(hnMagic)+2)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, 0, fmt.Errorf(".hn: failed to read header: %w", err)
	}
	if string(header[:len(hnMagic)]) != hnMagic {
		return nil, 0, fmt.Errorf(".hn: not a hana bytecode file (bad magic)")
	}
	if version := header[len(hnMagic)]; version != hnVersion {
		return nil, 0, fmt.Errorf(".hn: unsupported format version %d (this hana build reads version %d)", version, hnVersion)
	}
	lang := Lang(header[len(hnMagic)+1])

	var prog Program
	if err := gob.NewDecoder(r).Decode(&prog); err != nil {
		return nil, 0, fmt.Errorf(".hn: failed to decode program: %w", err)
	}
	return &prog, lang, nil
}
