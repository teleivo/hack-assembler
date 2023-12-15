package hack

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestAssemble(t *testing.T) {
	tests := map[string]struct {
		in   string
		want string
	}{
		"Add": {
			in: `
// This file is part of www.nand2tetris.org
// and the book "The Elements of Computing Systems"
// by Nisan and Schocken, MIT Press.
// File name: projects/06/pong/Pong.asm

// Computes R0 = 2 + 3  (R0 refers to RAM[0])

@2
D=A
@3
D=D+A
@0
M=D
`,
			want: `0000000000000010
1110110000010000
0000000000000011
1110000010010000
0000000000000000
1110001100001000
`,
		},
		"Max": {
			in: `
// This file is part of www.nand2tetris.org
// and the book "The Elements of Computing Systems"
// by Nisan and Schocken, MIT Press.
// File name: projects/06/pong/Pong.asm

// Computes R2 = max(R0, R1)  (R0,R1,R2 refer to RAM[0],RAM[1],RAM[2])

	@R0
	D=M              // D = first number
	@R1
	D=D-M            // D = first number - second number
	@OUTPUT_FIRST
	D;JGT            // if D>0 (first is greater) goto output_first
	@R1
	D=M              // D = second number
	@OUTPUT_D
	0;JMP            // goto output_d
(OUTPUT_FIRST)
	@R0             
	D=M              // D = first number
(OUTPUT_D)
	@R2
	M=D              // M[2] = D (greatest number)
(INFINITE_LOOP)
	@INFINITE_LOOP
	0;JMP            // infinite loop
`,
			want: `0000000000000000
1111110000010000
0000000000000001
1111010011010000
0000000000001010
1110001100000001
0000000000000001
1111110000010000
0000000000001100
1110101010000111
0000000000000000
1111110000010000
0000000000000010
1110001100001000
0000000000001110
1110101010000111
`,
		},
		"Rect": {
			in: `
// This file is part of www.nand2tetris.org
// and the book "The Elements of Computing Systems"
// by Nisan and Schocken, MIT Press.
// File name: projects/06/pong/Pong.asm

// Draws a rectangle at the top-left corner of the screen.
// The rectangle is 16 pixels wide and R0 pixels high.

			   @0
			   D=M
			   @INFINITE_LOOP
			   D;JLE 
			   @counter
			   M=D
			   @SCREEN
			   D=A
			   @address
			   M=D
			(LOOP)
			   @address
			   A=M
			   M=-1
			   @address
			   D=M
			   @32
			   D=D+A
			   @address
			   M=D
			   @counter
			   MD=M-1
			   @LOOP
			   D;JGT
			(INFINITE_LOOP)
			   @INFINITE_LOOP
			   0;JMP
`,
			want: `0000000000000000
1111110000010000
0000000000010111
1110001100000110
0000000000010000
1110001100001000
0100000000000000
1110110000010000
0000000000010001
1110001100001000
0000000000010001
1111110000100000
1110111010001000
0000000000010001
1111110000010000
0000000000100000
1110000010010000
0000000000010001
1110001100001000
0000000000010000
1111110010011000
0000000000001010
1110001100000001
0000000000010111
1110101010000111
`,
		},
	}

	for _, tc := range tests {
		var got bytes.Buffer
		err := Assemble(strings.NewReader(tc.in), &got)
		assertNoError(t, err)

		assertDeepEquals(t, "Assemble", tc.in, got.String(), tc.want)
	}

	file := "testdata/Pong.asm"
	f, err := os.Open(file)
	assertNoError(t, err)

	want, err := os.ReadFile("testdata/Pong.hack.golden")
	assertNoError(t, err)

	var got bytes.Buffer
	err = Assemble(f, &got)
	assertNoError(t, err)

	assertDeepEquals(t, "Assemble", file, got.String(), string(want))

	// TODO add error test cases.
}

func TestParse(t *testing.T) {
	tests := map[string]struct {
		in   string
		want []instruction
	}{
		"IgnoresEmptylines": {
			in: `
`,
			want: nil,
		},
		"IgnoresCommentLines": {
			in:   `// This is a comment`,
			want: nil,
		},
		"IgnoresTrailingComments": {
			in: `@2 // this is a comment`,
			want: []instruction{
				&aInstruction{
					Literal: "2",
					Value:   2,
				},
			},
		},
		"IgnoresSpaces": {
			in: `  @2`,
			want: []instruction{
				&aInstruction{
					Literal: "2",
					Value:   2,
				},
			},
		},
		"IgnoresTabs": {
			in: `	D=M`,
			want: []instruction{
				&cInstruction{
					Dest: "D",
					Comp: "M",
					Jump: "",
				},
			},
		},
	}

	for _, tc := range tests {
		got, err := parse(strings.NewReader(tc.in))
		assertNoError(t, err)

		assertDeepEquals(t, "Parse", tc.in, got, tc.want)
	}

	// TODO add error test cases.
}

func TestParseCInstruction(t *testing.T) {
	tests := map[string]struct {
		in   string
		want instruction
	}{
		"DestAndComp": {
			in: `D=M`,
			want: &cInstruction{
				Dest: "D",
				Comp: "M",
				Jump: "",
			},
		},
		"CompAndJump": {
			in: `D;JEQ`,
			want: &cInstruction{
				Dest: "",
				Comp: "D",
				Jump: "JEQ",
			},
		},
		"CompAndJumpWithWhitespace": {
			in: `D ; JEQ`,
			want: &cInstruction{
				Dest: "",
				Comp: "D",
				Jump: "JEQ",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := parseCInstruction(tc.in)
			assertNoError(t, err)

			assertDeepEquals(t, "parseCInstruction", tc.in, got, tc.want)
		})
	}
	// TODO add some error cases.
	// mnemonics need to be upper case.
	// dest or jump can be omitted not both
}

func TestParseAInstruction(t *testing.T) {
	tests := map[string]struct {
		in   string
		want instruction
	}{
		"ParseConstantValue": {
			in: `@2`,
			want: &aInstruction{
				Literal: "2",
				Value:   2,
			},
		},
		"ParsePredifinedSymbol": {
			in: `@R0`,
			want: &aInstruction{
				Literal:  "R0",
				IsSymbol: true,
			},
		},
		"ParseUserdefinedSymbol": {
			in: `@_0.$:var`,
			want: &aInstruction{
				Literal:  "_0.$:var",
				IsSymbol: true,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := parseAInstruction(tc.in)
			assertNoError(t, err)

			assertDeepEquals(t, "parseAInstruction", tc.in, got, tc.want)
		})
	}

	errTests := map[string]struct {
		in string
	}{
		"Reject@WithoutSymbolOrConstant": {
			in: `@`,
		},
		"RejectNegativeConstants": {
			in: `@-2`,
		},
		"RejectConstantsExceeding15Bits": {
			in: `@32768`,
		},
		"RejectFloats": {
			in: `@3.14`,
		},
		"RejectSymbolWithLeadingDigit": {
			in: `@2Avar`,
		},
		"RejectSymbolWithIllegalChar": {
			in: `@var\`,
		},
	}

	for name, tc := range errTests {
		t.Run(name, func(t *testing.T) {
			_, err := parseAInstruction(tc.in)
			assertError(t, err)
		})
	}
}

func TestParseLabel(t *testing.T) {
	tests := map[string]struct {
		in   string
		want instruction
	}{
		"ParseLabel": {
			in: `(INFINITE_LOOP)`,
			want: &label{
				Literal: "INFINITE_LOOP",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := parseLabel(tc.in)
			assertNoError(t, err)

			assertDeepEquals(t, "parseLabel", tc.in, got, tc.want)
		})
	}

	errTests := map[string]struct {
		in string
	}{
		"RejectLabelDefinitionWithoutSymbol": {
			in: `()`,
		},
		"RejectMissingClosingParenthesis": {
			in: `(INFINITE_LOOP`,
		},
		"RejectSymbolWithIllegalChar": {
			in: `(INFINITE-LOOP)`,
		},
	}

	for name, tc := range errTests {
		t.Run(name, func(t *testing.T) {
			_, err := parseLabel(tc.in)
			assertError(t, err)
		})
	}
}

func TestCode(t *testing.T) {
	tests := map[string]struct {
		in   []instruction
		want string
	}{
		"@5": {
			in: []instruction{
				&aInstruction{
					Value: 5,
				},
			},
			want: "0000000000000101",
		},
		"@variable": {
			in: []instruction{
				&aInstruction{
					Literal:  "variable",
					IsSymbol: true,
				},
				&aInstruction{
					Literal:  "variable2",
					IsSymbol: true,
				},
			},
			want: `0000000000010000
0000000000010001`,
		},
		"D=A": {
			in: []instruction{
				&cInstruction{
					Dest: "D",
					Comp: "A",
				},
			},
			want: "1110110000010000",
		},
		"D=D+A": {
			in: []instruction{
				&cInstruction{
					Dest: "D",
					Comp: "D+A",
				},
			},
			want: "1110000010010000",
		},
		"M=D": {
			in: []instruction{
				&cInstruction{
					Dest: "M",
					Comp: "D",
				},
			},
			want: "1110001100001000",
		},
		"LabelDeclarationDoesNotResultInAnInstruction": {
			in: []instruction{
				&label{
					Literal: "INFINITE_LOOP",
				},
				&cInstruction{
					Dest: "D",
					Comp: "A",
				},
			},
			want: "1110110000010000",
		},
		"LabelUseBeforeDeclaration": {
			in: []instruction{
				&aInstruction{
					Literal:  "OUTPUT",
					IsSymbol: true,
				},
				&aInstruction{
					Literal:  "R15",
					IsSymbol: true,
				},
				&label{
					Literal: "OUTPUT",
				},
				&cInstruction{
					Dest: "D",
					Comp: "A",
				},
			},
			want: `0000000000000010
0000000000001111
1110110000010000`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			b := new(bytes.Buffer)
			err := code(tc.in, b)
			assertNoError(t, err)

			want := tc.want + "\n"
			assertDeepEquals(t, "Code", tc.in, b.String(), want)
		})
	}

	errTests := map[string]struct {
		in []instruction
	}{
		"RejectRedeclarationOfLabel": {
			in: []instruction{
				&label{
					Literal: "INFINITE_LOOP",
				},
				&label{
					Literal: "INFINITE_LOOP",
				},
			},
		},
		"RejectLabelDeclarationOfPredefinedSymbol": {
			in: []instruction{
				&label{
					Literal: "R0",
				},
			},
		},
	}

	for name, tc := range errTests {
		t.Run(name, func(t *testing.T) {
			b := new(bytes.Buffer)
			err := code(tc.in, b)
			assertError(t, err)
		})
	}
}

func assertError(t *testing.T, err error) {
	if err == nil {
		t.Fatal("expected error instead got nil instead", err)
	}
}

func assertNoError(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("expected no error instead got: %q", err)
	}
}

func assertEquals(t *testing.T, method string, in, want, got any) {
	if got != want {
		t.Errorf("%s(%q) = %d; want %d", method, in, got, want)
	}
}

func assertDeepEquals(t *testing.T, method string, in, got, want any) {
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("%s(%q) mismatch (-want +got):\n%s", method, in, diff)
	}
}
