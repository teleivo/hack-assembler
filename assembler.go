// Package hack implements an assembler for the hack assembly language as documented in
// https://www.nand2tetris.org/project04.
package hack

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"
)

type instruction interface {
	Instruction()
}

// A-instruction represents a constant or symbol which can be pre- or user-defined.
type aInstruction struct {
	Literal  string
	IsSymbol bool
	Value    uint16
}

func (a aInstruction) Instruction() {}

// C-instruction represents a computation in the form of dest=comp;jump.
type cInstruction struct {
	Dest string
	Comp string
	Jump string
}

func (c cInstruction) Instruction() {}

// label represents a label declaration. It is a pseudo-instruction that will not be translated into
// machine code. It is used as a reference to instruction memory location holding the next command
// in the program.
type label struct {
	Literal string
}

func (l label) Instruction() {}

// Assemble translates hack assembly into machine code for the hack CPU. The machine code is written
// as text instead of binary as that is what was required in https://www.nand2tetris.org/project06.
func Assemble(r io.Reader, w io.Writer) error {
	instructions, err := parse(r)
	if err != nil {
		return err
	}

	err = code(instructions, w)
	if err != nil {
		return err
	}

	return nil
}

// parse parses hack assembly into instructions including the pseudo-instruction label. Symbolic
// declarations in labels or symbolic references in A-instructions will not have been resolved at
// this stage.
func parse(r io.Reader) ([]instruction, error) {
	var instructions []instruction
	s := bufio.NewScanner(r)
	for s.Scan() {
		line, _, _ := strings.Cut(s.Text(), "//")
		command := strings.TrimSpace(line)

		if len(command) == 0 {
			continue
		}
		if command[0] == '@' {
			ins, err := parseAInstruction(command)
			if err != nil {
				return nil, err
			}
			instructions = append(instructions, ins)
		} else if command[0] == '(' {
			ins, err := parseLabel(command)
			if err != nil {
				return nil, err
			}
			instructions = append(instructions, ins)
		} else {
			ins, err := parseCInstruction(command)
			if err != nil {
				return nil, err
			}
			instructions = append(instructions, ins)
		}
	}

	// TODO error handling of s.Error()
	return instructions, nil
}

func parseAInstruction(in string) (*aInstruction, error) {
	if len(in) < 2 {
		return nil, errors.New("failed to parse A-instruction: @ needs to be followed by a constant or symbol")
	}
	in = in[1:] // drop the @

	// symbols cannot start with a digit; as a starting digit indicates a constant
	if unicode.IsDigit(rune(in[0])) {
		v, err := strconv.ParseUint(in, 10, 15)
		if err != nil {
			return nil, fmt.Errorf("failed to parse A-instruction: expected unsigned 15-bit value: %v", err)
		}
		return &aInstruction{Literal: in, Value: uint16(v)}, nil
	}

	ok := containsOnly(in, validSymbolChars)
	if !ok {
		return nil, errors.New(`failed to parse A-instruction: literal contains illegal character. A user-deﬁned symbol can be any sequence of letters, digits, underscore ( _ ),
dot (.), dollar sign ($), and colon (:) that does not begin with a digit`)
	}

	return &aInstruction{Literal: in, IsSymbol: true}, nil
}

// validSymbolChars ensures that user-deﬁned symbol can only be any sequence of letters, digits,
// underscore ( _ ), dot (.), dollar sign ($), and colon (:).
func validSymbolChars(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '.' || r == '$' || r == ':'
}

// containsOnly returns true if every rune in s satisfies given f.
func containsOnly(s string, f func(rune) bool) bool {
	for _, r := range s {
		ok := f(r)
		if !ok {
			return false
		}
	}
	return true
}

func parseCInstruction(in string) (*cInstruction, error) {
	var dest, comp, jump string
	dest, rest, foundEquals := strings.Cut(in, "=")
	if !foundEquals {
		// this is to accommodate for Cut behavior
		dest = ""
		rest = in
	}
	comp, jump, foundSemicolon := strings.Cut(rest, ";")
	if !foundEquals && !foundSemicolon {
		// TODO this is illegal; add a test and implement
	}
	if !foundSemicolon {

	}

	return &cInstruction{
		Dest: strings.TrimSpace(dest),
		Comp: strings.TrimSpace(comp),
		Jump: strings.TrimSpace(jump),
	}, nil
}

func parseLabel(in string) (*label, error) {
	if len(in) < 3 {
		return nil, errors.New("failed to parse label: label definitions need to define symbols enclosed in ().")
	}
	if in[0] != '(' {
		return nil, errors.New("failed to parse label: label definitions need to be enclosed in (). Missing leading (")
	}
	if in[len(in)-1] != ')' {
		return nil, errors.New("failed to parse label: label definitions need to be enclosed in (). Missing closing )")
	}
	in = strings.Trim(in, "()")
	ok := containsOnly(in, validSymbolChars)
	if !ok {
		return nil, errors.New(`failed to parse A-instruction: literal contains illegal character. A user-deﬁned symbol can be any sequence of letters, digits, underscore ( _ ),
dot (.), dollar sign ($), and colon (:) that does not begin with a digit`)
	}

	return &label{Literal: in}, nil
}

var predefinedSymbols map[string]uint16 = map[string]uint16{
	"SP":     0,
	"LCL":    1,
	"ARG":    2,
	"THIS":   3,
	"THAT":   4,
	"R0":     0,
	"R1":     1,
	"R2":     2,
	"R3":     3,
	"R4":     4,
	"R5":     5,
	"R6":     6,
	"R7":     7,
	"R8":     8,
	"R9":     9,
	"R10":    10,
	"R11":    11,
	"R12":    12,
	"R13":    13,
	"R14":    14,
	"R15":    15,
	"SCREEN": 16384,
	"KBD":    24576,
}

// code translates instructions into machine code. Labels do not result in an instruction in machine
// code. Symbolic references in A-instructions are resolved into memory addresses at this stage.
func code(instructions []instruction, w io.Writer) error {
	var nextVariableAddress uint16 = 16
	var pc uint16
	symbolTable := make(map[string]uint16)
	for _, instruction := range instructions {
		switch v := instruction.(type) {
		case *label:
			if _, ok := symbolTable[v.Literal]; ok {
				return fmt.Errorf("failed to encode label %q: label re-declared", v.Literal)
			}
			if _, ok := predefinedSymbols[v.Literal]; ok {
				return fmt.Errorf("failed to encode label: %q is a pre-defined symbol which cannot be used as a label", v.Literal)
			}
			symbolTable[v.Literal] = pc
		default:
			pc++
		}
	}
	for k, v := range predefinedSymbols {
		symbolTable[k] = v
	}

	for _, instruction := range instructions {
		switch ins := instruction.(type) {
		case *aInstruction:
			ains := ins
			if ins.IsSymbol {
				v, ok := symbolTable[ins.Literal]
				if !ok {
					v = nextVariableAddress
					symbolTable[ins.Literal] = v
					nextVariableAddress++
				}
				ains = &aInstruction{Value: v}
			}

			n, err := fmt.Fprintf(w, "%016b\n", codeAInstruction(ains))
			if n != 17 {
				return fmt.Errorf("failed to write entire a-instruction %v: wrote %d instead of 17 bytes/chars", ins, n)
			}
			if err != nil {
				return fmt.Errorf("failed to write a-instruction %v: %v", ins, err)
			}
		case *cInstruction:
			code, err := codeCInstruction(ins)
			if err != nil {
				return fmt.Errorf("failed to encode c-instruction %v: %v", ins, err)
			}
			n, err := fmt.Fprintf(w, "%s\n", code)
			if n != 17 {
				return fmt.Errorf("failed to write entire c-instruction %v: wrote %d instead of 16 bytes/chars", ins, n)
			}
			if err != nil {
				return fmt.Errorf("failed to write c-instruction %v: %v", ins, err)
			}
		}
	}

	return nil
}

func codeAInstruction(instruction *aInstruction) uint16 {
	return instruction.Value
}

var compToA map[string]string = map[string]string{
	"0":   "0",
	"1":   "0",
	"-1":  "0",
	"D":   "0",
	"A":   "0",
	"!D":  "0",
	"!A":  "0",
	"-D":  "0",
	"-A":  "0",
	"D+1": "0",
	"A+1": "0",
	"D-1": "0",
	"A-1": "0",
	"D+A": "0",
	"D-A": "0",
	"A-D": "0",
	"D&A": "0",
	"D|A": "0",
	"M":   "1",
	"!M":  "1",
	"-M":  "1",
	"M+1": "1",
	"M-1": "1",
	"D+M": "1",
	"D-M": "1",
	"M-D": "1",
	"D&M": "1",
	"D|M": "1",
}

var compToC map[string]string = map[string]string{
	"0":   "101010",
	"1":   "111111",
	"-1":  "111010",
	"D":   "001100",
	"A":   "110000",
	"!D":  "001101",
	"!A":  "110001",
	"-D":  "001111",
	"-A":  "110011",
	"D+1": "011111",
	"A+1": "110111",
	"D-1": "001110",
	"A-1": "110010",
	"D+A": "000010",
	"D-A": "010011",
	"A-D": "000111",
	"D&A": "000000",
	"D|A": "010101",
	"M":   "110000",
	"!M":  "110001",
	"-M":  "110011",
	"M+1": "110111",
	"M-1": "110010",
	"D+M": "000010",
	"D-M": "010011",
	"M-D": "000111",
	"D&M": "000000",
	"D|M": "010101",
}

var destToD map[string]string = map[string]string{
	"M":   "001",
	"D":   "010",
	"MD":  "011",
	"A":   "100",
	"AM":  "101",
	"AD":  "110",
	"AMD": "111",
}

var jumpToJ map[string]string = map[string]string{
	"JGT": "001",
	"JEQ": "010",
	"JGE": "011",
	"JLT": "100",
	"JNE": "101",
	"JLE": "110",
	"JMP": "111",
}

func codeCInstruction(instruction *cInstruction) ([]byte, error) {
	aBit, ok := compToA[instruction.Comp]
	if !ok {
		return nil, fmt.Errorf("failed to encode a-bit from comp field %q", instruction.Comp)
	}

	cBits, ok := compToC[instruction.Comp]
	if !ok {
		return nil, fmt.Errorf("failed to encode c-bits from comp field %q", instruction.Comp)
	}

	dBits := "000"
	if instruction.Dest != "" {
		dBits, ok = destToD[instruction.Dest]
		if !ok {
			return nil, fmt.Errorf("failed to encode d-bits from dest field %q", instruction.Dest)
		}
	}

	jBits := "000"
	if instruction.Jump != "" {
		jBits, ok = jumpToJ[instruction.Jump]
		if !ok {
			return nil, fmt.Errorf("failed to encode j-bits from jump field %q", instruction.Jump)
		}
	}

	return []byte("111" + aBit + cBits + dBits + jBits), nil
}
