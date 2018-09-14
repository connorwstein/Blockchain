// Goal: implement the basics of the EVM
// Should be able to input a compiled contract (contract bytecode) and get the new evm state
// https://github.com/trailofbits/evm-opcodes
// 256-bit words
// A new EVM instance is instantiated every time a contract account is called.
// The EVM then executes that contract code along with the arguments provided by the transaction
// which invoked the contract (gas supply, data payload, sender, receiver/contract address, etc.)
// and the existing contract storage and state (from the last blocks state merkle root)
package evm

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"log"
	"os"
	//     "strings"
)

const (
	MSG_CALLVALUE = "0a" // Amount of ether sent with the call
	MSG_DATA = "6d4ce63c" // Call to get()
	WORD_SIZE = 32
// 	MSG_DATA = 0x2e1a7d4d000000000000000000000000000000000000000000000000000000000000000a // Contains first 4 bytes of keccak hash of the ascii form of the method
// 				// signature + a single unit8 representing the value to be withdrawn
// 				// in the case of the faucet function
// 2e1a7d4d13322e7b96f9a57413e1525c250fb7a9021cf91d1540d5b69f16a49f
// 2e1a7d4d withdraw(uint256) 
// 2e1a7d4d000000000000000000000000000000000000000000000000000000000000000a --> first 4 bytes + 0x0a (dummy withdrawal of 0x0a)
// 2e1a7d4d0000000000000000000000000000000000000000000000000000000000000005

//000000000000000000000000000000000000000000000000000000000000000a withdrawal of a
// function withdraw(uint withdraw_amount)
)

const (
	// Arithmetic
	ADD        = 0x01 //Add the top two stack items
	MUL               //Multiply the top two stack items
	SUB               //Subtract the top two stack items
	DIV               //Integer division
	SDIV              //Signed integer division
	MOD               //Modulo (remainder) operation
	SMOD              //Signed modulo operation
	ADDMOD            //Addition modulo any number
	MULMOD            //Multiplication modulo any number
	EXP               //Exponential operation
	SIGNEXTEND        //Extend the length of a two’s complement signed integer

	// Precomipled contract for this?
	SHA3 //Compute the Keccak-256 hash of a block of memory

	// Stack
	POP     = 0x50 //Remove the top item from the stack
	MLOAD   = 0x51 //Load a word from memory
	MSTORE  = 0x52 //Save a word to memory
	MSTORE8        //Save a byte to memory
	SLOAD          //Load a word from storage
	SSTORE         //Save a word to storage
	MSIZE          //Get the size of the active memory in bytes
	// 	PUSHx   //Place x-byte item on the stack, where x can be any integer from 1 to 32 (full word) inclusive
PUSH1 = 0x60
PUSH2 = 0x61
PUSH3 = 0x62
PUSH4 = 0x63
PUSH5 = 0x64
PUSH6 = 0x65
PUSH7 = 0x66
PUSH8 = 0x67
PUSH9 = 0x68
PUSH10 = 0x69
PUSH11 = 0x6a
PUSH12 = 0x6b
PUSH13 = 0x6c
PUSH14 = 0x6d
PUSH15 = 0x6e
PUSH16 = 0x6f
PUSH17 = 0x70
PUSH18 = 0x71
PUSH19 = 0x72
PUSH20 = 0x73
PUSH21 = 0x74
PUSH22 = 0x75
PUSH23 = 0x76
PUSH24 = 0x77
PUSH25 = 0x78
PUSH26 = 0x79
PUSH27 = 0x7a
PUSH28 = 0x7b
PUSH29 = 0x7c
PUSH30 = 0x7d
PUSH31 = 0x7e
PUSH32 = 0x7f
	// 	DUPx    //Duplicate the x-th stack item, where x can be any integer from 1 to 16 inclusive
DUP1 = 0x80
DUP2 = 0x81
DUP3 = 0x82
DUP4 = 0x83
DUP5 = 0x84
DUP6 = 0x85
DUP7 = 0x86
DUP8 = 0x87
DUP9 = 0x88
DUP10 = 0x89
DUP11 = 0x8a
DUP12 = 0x8b
DUP13 = 0x8c
DUP14 = 0x8d
DUP15 = 0x8e
DUP16 = 0x8f
	// 	SWAPx   //Exchange 1st and (x+1)-th stack items, where x can by any integer from 1 to 16 inclusive
SWAP1 = 0x90
SWAP2 = 0x91
SWAP3 = 0x92
SWAP4 = 0x93
SWAP5 = 0x94
SWAP6 = 0x95
SWAP7 = 0x96
SWAP8 = 0x97
SWAP9 = 0x98
SWAP10 = 0x99
SWAP11 = 0x9a
SWAP12 = 0x9b
SWAP13 = 0x9c
SWAP14 = 0x9d
SWAP15 = 0x9e
SWAP16 = 0x9f
	// Process Flow
	STOP     = 0x00 //Halts execution
	JUMP     = 0x56 //Set the program counter to any value
	JUMPI    = 0x57 //Conditionally alter the program counter
	PC       = 0x58 //Get the value of the program counter (prior to the increment corresponding to this instruction)
	JUMPDEST = 0x5b // Mark a valid destination for jumps

	// System
	// 	LOGx          //Append a log record with +x+ topics, where +x+ is any integer from 0 to 4 inclusive
	CREATE              //Create a new account with associated code
	CALL         = 0xf1 //Message-call into another account, i.e. run another account's code
	CALLCODE     = 0xf2 //Message-call into this account with an another account’s code
	RETURN       = 0xf3 //Halt execution and return output data
	DELEGATECALL        //Message-call into this account with an alternative account’s code, but persisting the current values for sender and value
	STATICCALL          //Static message-call into an account
	REVERT       = 0xfd //Halt execution reverting state changes but returning data and remaining gas
	INVALID            //The designated invalid instruction
	SELFDESTRUCT        //Halt execution and register account for deletion

	// Logic
	LT            = 0x10 //Less-than comparison
	GT            = 0x11 //Greater-than comparison
	SLT           //Signed less-than comparison
	SGT           //Signed greater-than comparison
	EQ         = 0x14   //Equality comparison
	ISZERO = 0x15 //Simple not operator
	AND           //Bitwise AND operation
	OR            //Bitwise OR operation
	XOR           //Bitwise XOR operation
	NOT           //Bitwise NOT operation
	BYTE          //Retrieve a single byte from a full-width 256 bit word

	// Environment
	GAS            = 0x5a //Get the amount of available gas (after the reduction for this instruction)
	ADDRESS               //Get the address of the currently executing account
	BALANCE               //Get the account balance of any given account
	ORIGIN                //Get the address of the EOA that initiated this EVM execution
	CALLER                //Get the address of the caller immediately responsible for this execution
	CALLVALUE      = 0x34 //Get the ether amount deposited by the caller responsible for this execution
	CALLDATALOAD   = 0x35 //Get the input data sent by the caller responsible for this execution
	CALLDATASIZE   = 0x36 //Get the size of the input data
	CALLDATACOPY          //Copy the input data to memory
	CODESIZE              //Get the size of code running in the current environment
	CODECOPY       = 0x39 //Copy the code running in the current environment to memory
	GASPRICE              //Get the gas price specified by the originating transaction
	EXTCODESIZE           //Get the size of any account's code
	EXTCODECOPY           //Copy any account’s code to memory
	RETURNDATASIZE        //Get the size of the output data from the previous call in the current environment
	RETURNDATACOPY        //Copy of data output from the previous call to memory

	// Block
	BLOCKHASH  //Get the hash of one of the 256 most recently completed blocks
	COINBASE   //Get the block’s beneficiary address for the block reward
	TIMESTAMP  //Get the block’s timestamp
	NUMBER     //Get the block’s number
	DIFFICULTY //Get the block’s difficulty
	GASLIMIT   //Get the block’s gas limit
)

type OpHandler func(evm *EVM, args []byte)

type OpCode struct {
	code    byte
	numArgs int
	gasCost int
	handler OpHandler
}

type Word [WORD_SIZE]byte

func (e *Word) Write(p []byte) (int, error) {
	n := 0
	for i := 0; i < len(p); i++ {
		// ex: 0x10 0x20 --> 0x0000...0120 (32 byte)
		e[31 - i] = p[len(p) - 1 - i]
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
	log.Printf("Append %v %v", value, s.stack)
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
	if desiredSize < len(m.mem) {
		return
	}
	m.mem = append(m.mem, make([]Word, desiredSize-len(m.mem))...)
}

type EVM struct {
	stack *EVMStack
	// This memory just grows as needed
	storage map[byte]byte // persistent key-value mappings sstore/ssload
	// Sort of like registers:
	memory  *EVMMem // mstore/mload, freshly cleared per message call, expanded when accessing a previously untouched word
	opCodes map[byte]OpCode
	pc int // pointer to the next instruction 
}

func (evm *EVM) init() {
	evm.stack.init()
	evm.memory.init()
	evm.opCodes = make(map[byte]OpCode)
	evm.opCodes[ADD] = OpCode{ADD, 0, 3, add}
	evm.opCodes[CALLVALUE] = OpCode{CALLVALUE, 0, 2, callValue}
	evm.opCodes[MSTORE] = OpCode{MSTORE, 0, 3, mstore}
	evm.opCodes[MLOAD] = OpCode{MLOAD, 0, 3, mload}

	for i := 0; i < 32; i++ {
		evm.opCodes[byte(PUSH1 + i)] = OpCode{byte(PUSH1 + i), i + 1, 3, push}
	}

	for i := 0; i < 16; i++ {
		evm.opCodes[byte(DUP1 + i)] = OpCode{byte(DUP1 + i), 0, 3, dup}
	}

	for i := 0; i < 16; i++ {
		evm.opCodes[byte(SWAP1 + i)] = OpCode{byte(SWAP1 + i), 0, 3, swap}
	}

	evm.opCodes[ISZERO] = OpCode{ISZERO, 0, 3, iszero}
	evm.opCodes[JUMPI] = OpCode{JUMPI, 0, 10, jumpi}
	evm.opCodes[JUMPDEST] = OpCode{JUMPDEST, 0, 1, nil}
	evm.opCodes[REVERT] = OpCode{REVERT, 0, 0, revert}
	evm.opCodes[POP] = OpCode{CALLVALUE, 0, 2, pop}
	evm.opCodes[CALLDATASIZE] = OpCode{CALLVALUE, 0, 2, callDataSize}
	// 	evm.opCodes[DATAOFFSET] = OpCode{CALLVALUE, 0, 2, callValue} seems to be a mystery
	evm.opCodes[CODECOPY] = OpCode{CODECOPY, 0, 3, codeCopy}
	evm.opCodes[RETURN] = OpCode{RETURN, 0, 0, returnF}
	evm.opCodes[STOP] = OpCode{STOP, 0, 0, nil}
	evm.opCodes[CALLDATALOAD] = OpCode{CALLDATALOAD, 0, 3, callDataLoad}
	evm.opCodes[LT] = OpCode{LT, 0, 3, lt} // check whether top stack item is lt the second item
	evm.opCodes[EQ] = OpCode{EQ, 0, 3, eq} 
	evm.opCodes[GT] = OpCode{GT, 0, 3, gt} 

	// we need push, mstore, push1, calldatasize, lt, jumpi, calldataload
	// push29, swap1, div, push4, and, dup1, eq, revert, mload
	// dup2, dup3, swap2, sub, return, log1, push6, sha3, push15, codecopy
	// dup9, swap11
	evm.pc = 0
}

func hexStringToWord(hexString string) Word {
	b, _ := hex.DecodeString(hexString)
	var data Word	
 	binary.Write(&data, binary.BigEndian, b)
	return data
}

func lt(evm *EVM, args []byte) {
	val1, err1 := evm.stack.pop()
	val2, err2 := evm.stack.pop()
	if err1 != nil || err2 != nil {
		log.Printf("Error in execution invalid evm program")
	}
	x1 := binary.BigEndian.Uint64(val1[24:])
	x2 := binary.BigEndian.Uint64(val2[24:])
	var element Word
	if x1 < x2 {
 		binary.Write(&element, binary.BigEndian, []byte{0x01})
	} else {
 		binary.Write(&element, binary.BigEndian, []byte{0x00})
	}
	evm.stack.push(element)
}

func eq(evm *EVM, args []byte) {
	val1, err1 := evm.stack.pop()
	val2, err2 := evm.stack.pop()
	if err1 != nil || err2 != nil {
		log.Printf("Error in execution invalid evm program")
	}
	x1 := binary.BigEndian.Uint64(val1[24:])
	x2 := binary.BigEndian.Uint64(val2[24:])
	var element Word
	if x1 == x2 {
 		binary.Write(&element, binary.BigEndian, []byte{0x01})
	} else {
 		binary.Write(&element, binary.BigEndian, []byte{0x00})
	}
	evm.stack.push(element)
}

func gt(evm *EVM, args []byte) {
	val1, err1 := evm.stack.pop()
	val2, err2 := evm.stack.pop()
	if err1 != nil || err2 != nil {
		log.Printf("Error in execution invalid evm program")
	}
	x1 := binary.BigEndian.Uint64(val1[24:])
	x2 := binary.BigEndian.Uint64(val2[24:])
	var element Word
	if x1 > x2 {
 		binary.Write(&element, binary.BigEndian, []byte{0x01})
	} else {
 		binary.Write(&element, binary.BigEndian, []byte{0x00})
	}
	evm.stack.push(element)
}

func callDataLoad(evm *EVM, args []byte) {
	// push 32 bytes (padded if less onto the stack) 
	evm.stack.push(hexStringToWord(MSG_DATA))
}

func callValue(evm *EVM, args []byte) {
	// For now just hardcode, future could actually take in a message arguments
	// from somewhere
	// Push onto the stack the amount of ether sent with message call which initiated this execution
	evm.stack.push(hexStringToWord(MSG_CALLVALUE))
}

func returnF(evm *EVM, args []byte) {
}

func codeCopy(evm *EVM, args []byte) {
}

func mstore(evm *EVM, args []byte) {
	// pop two values on the stack, first one is the address of where we store stuff in memory
	// second is the actual value we put in there
	address, err1 := evm.stack.pop()
	val, err2 := evm.stack.pop()
	if err1 != nil || err2 != nil {
		panic("Error in execution, mstore invalid")
	}
	// check if the address is available, grow to that address if needed
	addressVal := binary.BigEndian.Uint64(address[24:])
	log.Printf("MSTORE value %v in %v", val, addressVal)
	evm.memory.grow(int(addressVal) + 1)
	evm.memory.mem[addressVal] = val
}

func mload(evm *EVM, args []byte) {
	// pop address from the stack and load value with that address, push on the stack
	address, err1 := evm.stack.pop()
	if err1 != nil {
		panic("Error in execution, mload invalid")
	}
	addressVal := binary.BigEndian.Uint64(address[24:])
	if addressVal >= uint64(len(evm.memory.mem)) {
		panic("Try to load out of bounds address")
	}
	evm.stack.push(evm.memory.mem[addressVal])	
}

func iszero(evm *EVM, args []byte) {
	val, err := evm.stack.pop()
	if err != nil {
		panic("Could not find value on stack to iszero")
	}
	var result Word
	if bytes.Equal(val[:], make([]byte, WORD_SIZE)) {
		binary.Write(&result, binary.BigEndian, []byte{0x01})
	} else {
		binary.Write(&result, binary.BigEndian, []byte{0x00})
	}
	evm.stack.push(result)
}

func jumpi(evm *EVM, args []byte) {
	// Pop two values off the stack
	// first value is the destination and the second value is the condition
	// if the condition is 1 then we jump there
	dst, err1 := evm.stack.pop()
	cond, err2 := evm.stack.pop()
	if err1 != nil || err2 != nil {
		log.Printf("JUMPI Error in execution invalid evm program")
	}
	if !bytes.Equal(cond[:], make([]byte, WORD_SIZE)) {
		evm.pc = int(binary.BigEndian.Uint64(dst[24:]))
	} 
}

func revert(evm *EVM, args []byte) {
	// Something bad happened, rollback everything
}

func pop(evm *EVM, args []byte) {
	_, err := evm.stack.pop()
	if err != nil {
		log.Print("tried to pop off empty stack")
	}
}

func callDataSize(evm *EVM, args []byte) {
}

func push(evm *EVM, args []byte) {
	familyIndex := int(args[0]) - PUSH1 + 1 // 1 means PUSH1, 2 means PUSH2 etc.
	var element Word
	// args[1:familyIndex] will have the bytes for the item to push
	binary.Write(&element, binary.BigEndian, args[1:1+familyIndex])
	log.Printf("Word pushed %v\n", element)
	evm.stack.push(element)
}

func swap(evm *EVM, args []byte) {
	// swap 1st and 2nd stack items
	familyIndex := int(args[0]) - SWAP1  + 1 // 1 means SWAP1 etc.
	log.Printf("Swap %d called\n", familyIndex)
	if len(evm.stack.stack) < familyIndex + 1 {  // need at least 2 elements for a swap1
		panic("Insufficient stack for swap")
	}
	pops := make([]Word, familyIndex + 1)
	// Pop all the way to familyIndex
	for i := 0; i <= familyIndex; i++ {
		pops[i], _ = evm.stack.pop()
	}
	// pops now as s1, s2, s2 ... s<familyIndex + 1>
	// need s<familyIndex +1> to go on the top then everything else in the same order it was
	// first push everything back except familyIndex + 1
	// push the top item first
	log.Printf("swap pops %v stack %v\n", pops, evm.stack.stack)
	evm.stack.push(pops[0])
	for i := familyIndex - 1; i > 0; i-- {
		evm.stack.push(pops[i])
	}
	// push our familyIndex+1 element (indexed at familyIndex)
	evm.stack.push(pops[familyIndex])
}

func dup(evm *EVM, args []byte) {
	// dup 1st item and put it on the stack
	familyIndex := int(args[0]) - DUP1 + 1 // 1 means DUP1
	log.Printf("Dup %v called args %v dup1 %v\n", familyIndex, args[0], DUP1)
	if len(evm.stack.stack) < familyIndex { // need at least 1 element for a dup1
		panic("Insufficient stack for dup")
	}
	pops := make([]Word, familyIndex)
	// Pop all the way to familyIndex, now this element pops[familyIndex - 1] needs to be 
	//  2 1 --> pops 1 2 
	for i := 0; i < familyIndex; i++ {
		pops[i], _ = evm.stack.pop()
	}
	// pops[familyIndex - 1] now contains the element we want to dup
	// push everything back except that element
	log.Printf("pops size %d", len(pops))
	// push 2 on the stack
	duped := pops[familyIndex - 1]
	for i := familyIndex - 1; i >= 0; i-- {
		evm.stack.push(pops[i])
	}
	evm.stack.push(duped) // push duped
}


func add(evm *EVM, args []byte) {
	// Pop two items from the stack, add them and push the result on the stack
	// (full words)
	val1, err1 := evm.stack.pop()
	val2, err2 := evm.stack.pop()
	if err1 != nil || err2 != nil {
		log.Printf("Error in execution invalid evm program")
	}
	// Add two values. The sum should not be larger than a 64-bit (8 byte int)
	// or there is something corrupted on the stack
	// most of the time this would be like a push1 push1 add meaning
	// only the last byte of each word actually has the number
	x1 := binary.BigEndian.Uint64(val1[24:])
	x2 := binary.BigEndian.Uint64(val2[24:])
	var element Word
 	binary.Write(&element, binary.BigEndian, x1 + x2)
	evm.stack.push(element)
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

type OutOfGasError struct {
	msg string
	pc int // pc pointing to last instruction
} 

type InvalidOpError struct {
	msg string
} 

func (e OutOfGasError) Error() string { return e.msg }
func (e InvalidOpError) Error() string { return e.msg }

func isFamilyType(opCode byte) bool {
	return (int(opCode) >= PUSH1 && int(opCode) <= PUSH32) ||  (int(opCode) >= DUP1 && int(opCode) <= DUP16) || (int(opCode) >= SWAP1 && int(opCode) <= SWAP16) 
}

// Process an op, return an error if unrecognized instruction op code 
func (evm *EVM) handleOp(evmProgram []byte) error {
	log.Print(evmProgram[evm.pc])
	op, ok := evm.opCodes[evmProgram[evm.pc]]
	if !ok {
		log.Print("Unknown op code")
		return InvalidOpError{"Invalid op code"}
	}
	log.Printf("op code found %x", op.code)
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
	if op.code != JUMPI {
		evm.pc = nextInstruction
	} else {
		// if it is a jump just make sure dst is a jumpdest
		// jumpdest's should always be jumped to,
		// we shouldn't be reading a jumpdest randomly
		if evm.opCodes[evmProgram[evm.pc]].code != JUMPDEST {
			return InvalidOpError{"Jumped to a non-jump dest"}
		} else {
			// all is well, skip to after the jumpdest instruction
			evm.pc += 1
		}
	}
	return nil
}

// Walk through the bytes interpreting the opcodes
// TODO: stop if we run out of gas
func (evm EVM) execute(evmProgram []byte) {
	// Need to support the program counter and jumps
	for evm.pc < len(evmProgram) {
		// Break if we run out of gas or read a 
		// stop instruction or reach the end of the program
		// handleOp will update the pc
		if evm.opCodes[evmProgram[evm.pc]].code == STOP {
			log.Printf("Execution stopped by stop instruction")
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
	// First milestone: should be able to pass 
	// the byte code for simple storage along with a input data
	// to call it and get back a 0x01
	reader := bufio.NewReader(os.Stdin)
	log.Print("Enter EVM program: ")
	program, _ := reader.ReadString('\n')
	program = program[:(len(program) - 1)]
	evm := EVM{stack: &EVMStack{}, memory: &EVMMem{}}
	evm.init()
	instructions := evm.parse(program)
	log.Print(instructions)
	evm.execute(instructions)
}
