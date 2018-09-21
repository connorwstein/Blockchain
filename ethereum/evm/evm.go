// Goal: implement the basics of the EVM
// Should be able to input a compiled contract (contract bytecode) and get the new evm state
// https://github.com/trailofbits/evm-opcodes
// 256-bit words
// A new EVM instance is instantiated every time a contract account is called.
// The EVM then executes that contract code along with the arguments provided by the transaction
// which invoked the contract (gas supply, data payload, sender, receiver/contract address, etc.)
// and the existing contract storage and state (from the last blocks state merkle root)
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
)

type ContractCall struct {
	ContractCode string `json:"contractCode"`
	CallValue    string `json:"callValue"`
	CallData     string `json:"callData"`
	CallDataSize string `json:"callDataSize"`
	Gas 		 string `json:"gas"`
}

const (
	WORD_SIZE = 32
	MAX_STACK_SIZE = 1024
)

type OutOfGasError struct {
	msg string
	pc  int // pc pointing to last instruction
}

type InvalidOpError struct {
	msg string
}

func (e OutOfGasError) Error() string  { return e.msg }
func (e InvalidOpError) Error() string { return e.msg }

type Word [WORD_SIZE]byte

func (w Word) String() string {
	return hex.EncodeToString(w[:])
}

func (e *Word) Write(p []byte) (int, error) {
	n := 0
	for i := 0; i < len(p); i++ {
		// ex: 0x10 0x20 --> 0x0000...0120 (32 byte)
		e[31-i] = p[len(p)-1-i]
		n += 1
	}
	return n, nil
}

type EVMStack struct {
	stack []Word // should actually be 32 byte i.e. word size 256 bit values, max size 1024
}

func (s *EVMStack) init() {
	s.stack = make([]Word, 0)
}

func (s *EVMStack) push(value Word) {
	s.stack = append(s.stack, value)
	log.Printf("Append %v stack now %v", value, s.stack)
}

func (s EVMStack) String() string {
	var buf bytes.Buffer
	for i := range s.stack {
		buf.WriteString(" ")
		buf.WriteString(s.stack[i].String())
	}
	return buf.String()
}

func (s *EVMStack) pop() (Word, error) {
	length := len(s.stack)
	if length == 0 {
		return Word{}, errors.New("stack empty")
	}
	res := s.stack[length-1]
	s.stack = s.stack[:length-1]
	return res, nil
}

type EVMMem struct {
	mem []Word
}

func (m *EVMMem) init() {
	m.mem = make([]Word, 0)
}

func (m *EVMMem) grow(desiredSize int) {
// 	if desiredSize > 
	if desiredSize < len(m.mem) {
		return
	}
	m.mem = append(m.mem, make([]Word, desiredSize-len(m.mem))...)
}

type OpHandler func(evm *EVM, args []byte)

type OpCode struct {
	code    byte
	name    string
	numArgs int
	gasCost int
	handler OpHandler
}

func (op OpCode) String() string {
	return op.name
}

type EVM struct {
	stack *EVMStack
	// This memory just grows as needed
	storage map[Word]Word // persistent key-value mappings sstore/ssload, actually stored on-chain
	// Sort of like registers:
	memory  *EVMMem // mstore/mload, freshly cleared per message call, expanded when accessing a previously untouched word
	opCodes map[byte]OpCode
	pc      int  // pointer to the next instruction
	jump    bool // flag indicating a jump just happened
	call    ContractCall
}

func (evm *EVM) init() {
	evm.stack.init()
	evm.memory.init() // can grow indefinitely
	evm.opCodes = opCodeInit()
	evm.storage = make(map[Word]Word) // can grow indefinitely
	// just becomes expensive
	// Missing codecopy, log1 and sha3
	evm.pc = 0
}

func hexStringToWord(hexString string) Word {
	b, _ := hex.DecodeString(hexString)
	var data Word
	binary.Write(&data, binary.BigEndian, b)
	return data
}

func (evm EVM) parse(input string) []byte {
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

// Family type meaning PUSHx SWAPx etc.
// same function just different size
func isFamilyType(opCode byte) bool {
	return (int(opCode) >= PUSH1 && int(opCode) <= PUSH32) || (int(opCode) >= DUP1 && int(opCode) <= DUP16) || (int(opCode) >= SWAP1 && int(opCode) <= SWAP16)
}

// Process an op, return an error if unrecognized instruction op code
func (evm *EVM) handleOp(evmProgram []byte) error {
	log.Print(evmProgram[evm.pc])
	op, ok := evm.opCodes[evmProgram[evm.pc]]
	if !ok {
		log.Print("Unknown op code")
		return InvalidOpError{"Invalid op code"}
	}
	log.Printf("OP: %v %x at index %d", op, op.code, evm.pc)
	nextInstruction := evm.pc + op.numArgs + 1
	// get however many arguments this op needs and call its function
	// for push/swap/dup add the family index
	args := make([]byte, 0)
	if isFamilyType(op.code) {
		log.Printf("add family index for %v\n", op.code)
		args = append(args, op.code)
	}
	args = append(args, evmProgram[evm.pc+1:nextInstruction]...)
	op.handler(evm, args)
	// if the op is a jump or something that manipulates the pc
	// don't deal with it as that handler will update the pc
	if op.code != JUMPI && op.code != JUMP {
		evm.pc = nextInstruction
	} else {
		// if it is a jump just make sure dst is a jumpdest
		// jumpdest's should always be jumped to,
		// we shouldn't be reading a jumpdest randomly
		if evm.jump && evm.opCodes[evmProgram[evm.pc]].code != JUMPDEST {
			return InvalidOpError{fmt.Sprintf("Jumped to a non-jump dest %d instruction there is %s", evm.pc, evm.opCodes[evmProgram[evm.pc]])}
		} else {
			// all is well, skip to after the jumpdest instruction
			evm.pc += 1
		}
		evm.jump = false
	}
	return nil
}

// Walk through the bytes interpreting the opcodes
// TODO: stop if we run out of gas
func (evm EVM) execute(evmProgram []byte) {
	gasUse := 0
	for evm.pc < len(evmProgram) {
		// Break if we run out of gas or read a
		// stop instruction or reach the end of the program
		// handleOp will update the pc
		op := evm.opCodes[evmProgram[evm.pc]]
		if op.code == STOP {
			log.Printf("Execution stopped by stop instruction at index %d total gas use %d storage %v\n", evm.pc, gasUse, evm.storage)
			break
		}
		if op.code == RETURN {
			// return value is at the address in memory at the top of the stack
			log.Printf("Execution hit return at index %d gas use %d stack %v memory %v storage %v len mem %v\n", evm.pc, gasUse, evm.stack, evm.memory, evm.storage, len(evm.memory.mem))
			address, _ := evm.stack.pop() // pops a word
			log.Printf("Return value at address %v is %v\n", address[31], evm.memory.mem[address[31]][31])
			break
		}
		if op.code == REVERT {
			log.Printf("Execution hit revert at index %d  gas use %d stack %v\n", evm.pc, gasUse, evm.stack)
			break
		}
		gasUse += op.gasCost
		totalGas, _ := strconv.Atoi(evm.call.Gas)
		if gasUse > totalGas {
			log.Printf("Out of gas exception, used more than the provided %v\n", evm.call.Gas)
			break
		}
		err := evm.handleOp(evmProgram)
		if err != nil {
			log.Printf("Execution stoped due to %v", err)
			break
		}
	}
}

func main() {
	// Take in input and contract code as a json file and execute
	// Milestone one - be able to see the return value, done
	// Other milestone ideas - be able to update the contract storage
	// via requests would need some kind of persistence
	if len(os.Args) < 2 {
		log.Print("Pass json file for smart contract execution")
		return
	}
	jsonFile, err := os.Open(os.Args[1])
	defer jsonFile.Close()
	if err != nil {
		fmt.Println(err)
	}
	byteValue, _ := ioutil.ReadAll(jsonFile)
	var call ContractCall
	json.Unmarshal(byteValue, &call)
	evm := EVM{call: call, stack: &EVMStack{}, memory: &EVMMem{}}
	evm.init()
	instructions := evm.parse(call.ContractCode)
	log.Print(instructions)
	evm.execute(instructions)
}
