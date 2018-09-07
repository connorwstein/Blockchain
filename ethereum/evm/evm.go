// Goal: implement the basics of the EVM
// Should be able to input a compiled contract (contract bytecode) and get the new evm state
// https://github.com/trailofbits/evm-opcodes
// 256-bit words

package evm

import (
	"bufio"
	"encoding/hex"
	"log"
	"os"
	//     "strings"
)

const (
	PUSH = 0x60
	ADD  = 0x01
)

type EVMState struct {
	stack []int
}

func parse(input string) []byte {
	// Take a hex string evm program and return the bytes
	if len(input)%2 != 0 {
		log.Print("Invalid EVM program, need even number of chars")
		return nil
	}
	// program is just a big hex string
	// convert this into an array of bytes for execution
	instructions := make([]byte, len(input)/2)
	var tmp []byte = make([]byte, 1)
	for i := 0; i < len(input); i += 2 {
		hex.Decode(tmp, []byte{input[i], input[i+1]})
		instructions[i/2] = tmp[0]
	}
	return instructions
}

// Walk through the bytes interpreting the opcodes
func process(evmProgram []byte) {
	for i := 0; i < len(evmProgram); i++ {
		switch evmProgram[i] {
			case PUSH:
				log.Printf("Push %v", evmProgram[i+1])
				i++
			case ADD:
				log.Printf("Add")
			default:
				log.Print("Unsupported")
		}
	}		
}

func main() {
	reader := bufio.NewReader(os.Stdin)
	log.Print("Enter EVM program: ")
	program, _ := reader.ReadString('\n')
	program = program[:(len(program) - 1)]
	instructions := parse(program)
	log.Print(instructions)
}
