package main

import (
	"fmt"
	"os"
	"strings"

	"teleivo/nand2tetris/hack-assembler"
)

func main() {
	if err := run(os.Args); err != nil {
		fmt.Printf("assembly failed due to:\n%v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("expected one arg pointing to an '.asm' file, got %d args instead", len(args)-1)
	}

	assemblyFile := args[1]
	fin, err := os.Open(assemblyFile)
	if err != nil {
		return err
	}
	defer fin.Close()

	name, _, found := strings.Cut(assemblyFile, ".asm")
	if !found {
		return fmt.Errorf("expected assembly file with filename ending in '.asm', instead got %q", assemblyFile)
	}
	machineFile := name + ".hack"
	fout, err := os.Create(machineFile)
	if err != nil {
		return err
	}
	defer fout.Close()

	return hack.Assemble(fin, fout)
}
