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
	"encoding/hex"
	"errors"
	"log"
	"os"
	//     "strings"
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
	// 	DUPx    //Duplicate the x-th stack item, where x can be any integer from 1 to 16 inclusive
	DUP1 = 0x80
	// 	SWAPx   //Exchange 1st and (x+1)-th stack items, where x can by any integer from 1 to 16 inclusive
	SWAP1 = 0x90

	// Process Flow
	STOP     = 0x00 //Halts execution
	JUMP     = 0x56 //Set the program counter to any value
	JUMPI    = 0x57 //Conditionally alter the program counter
	PC       = 0x58 //Get the value of the program counter (prior to the increment corresponding to this instruction)
	JUMPDEST = 0x5b //Mark a valid destination for jumps

	// System
	// 	LOGx          //Append a log record with +x+ topics, where +x+ is any integer from 0 to 4 inclusive
	CREATE              //Create a new account with associated code
	CALL         = 0xf1 //Message-call into another account, i.e. run another account's code
	CALLCODE     = 0xf2 //Message-call into this account with an another account’s code
	RETURN       = 0xf3 //Halt execution and return output data
	DELEGATECALL        //Message-call into this account with an alternative account’s code, but persisting the current values for sender and value
	STATICCALL          //Static message-call into an account
	REVERT       = 0xfd //Halt execution reverting state changes but returning data and remaining gas
	INVALID             //The designated invalid instruction
	SELFDESTRUCT        //Halt execution and register account for deletion

	// Logic
	LT            //Less-than comparison
	GT            //Greater-than comparison
	SLT           //Signed less-than comparison
	SGT           //Signed greater-than comparison
	EQ            //Equality comparison
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
	CALLDATALOAD          //Get the input data sent by the caller responsible for this execution
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

type EVMStack struct {
	stack []byte // should actually be 32 byte i.e. word size 256 bit values, max size 1024
}

func (s *EVMStack) init() {
	s.stack = make([]byte, 0)
}

func (s *EVMStack) push(value byte) {
	s.stack = append(s.stack, value)
	log.Printf("Append %v %v", value, s.stack)
}

func (s *EVMStack) pop() (byte, error) {
	length := len(s.stack)
	if length == 0 {
		return 0, errors.New("stack empty")
	}
	res := s.stack[length-1]
	s.stack = s.stack[:length-1]
	return res, nil
}

type EVMMem struct {
	mem []byte
}

func (m *EVMMem) init() {
	m.mem = make([]byte, 0)
}

func (m *EVMMem) grow(desiredSize int) {
	if desiredSize < len(m.mem) {
		return
	}
	m.mem = append(m.mem, make([]byte, desiredSize-len(m.mem))...)
}

type EVM struct {
	stack *EVMStack
	// This memory just grows as needed
	storage map[byte]byte // persistent key-value mappings sstore/ssload
	// Sort of like registers:
	memory  *EVMMem // mstore/mload, freshly cleared per message call, expanded when accessing a previously untouched word
	opCodes map[byte]OpCode
}

func (evm *EVM) init() {
	evm.stack.init()
	evm.memory.init()
	evm.opCodes = make(map[byte]OpCode)
	evm.opCodes[PUSH1] = OpCode{PUSH1, 1, 3, push1}
	evm.opCodes[ADD] = OpCode{ADD, 0, 3, add}
	evm.opCodes[CALLVALUE] = OpCode{CALLVALUE, 0, 2, callValue}
	evm.opCodes[MSTORE] = OpCode{MSTORE, 0, 3, mstore}
	evm.opCodes[DUP1] = OpCode{DUP1, 0, 3, dup1}
	evm.opCodes[ISZERO] = OpCode{ISZERO, 0, 3, iszero}
	evm.opCodes[JUMPI] = OpCode{JUMPI, 0, 10, jumpi}
	evm.opCodes[REVERT] = OpCode{REVERT, 0, 0, revert}
	evm.opCodes[POP] = OpCode{CALLVALUE, 0, 2, pop}
	evm.opCodes[CALLDATASIZE] = OpCode{CALLVALUE, 0, 2, callDataSize}
	// 	evm.opCodes[DATAOFFSET] = OpCode{CALLVALUE, 0, 2, callValue} seems to be a mystery
	evm.opCodes[CODECOPY] = OpCode{CODECOPY, 0, 3, codeCopy}
	evm.opCodes[RETURN] = OpCode{RETURN, 0, 0, returnF}
	evm.opCodes[STOP] = OpCode{STOP, 0, 0, stop}
}

func stop(evm *EVM, args []byte) {
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
		log.Printf("Error in execution, mstore invalid")
	}
	log.Printf("MSTORE value %v in %v", val, address)
	// check if the address is available, grow to that address if needed
	evm.memory.grow(int(address) + 1)
	evm.memory.mem[address] = val
}
func dup1(evm *EVM, args []byte) {
}
func iszero(evm *EVM, args []byte) {
}
func jumpi(evm *EVM, args []byte) {
}
func revert(evm *EVM, args []byte) {
}
func pop(evm *EVM, args []byte) {
	_, err := evm.stack.pop()
	if err != nil {
		log.Print("tried to pop off empty stack")
	}
}
func callDataSize(evm *EVM, args []byte) {
}

func push1(evm *EVM, args []byte) {
	log.Print("Pushing onto the stack")
	evm.stack.push(args[0])
}

func add(evm *EVM, args []byte) {
	// Pop two items from the stack, add them and push the result on the stack
	val1, err1 := evm.stack.pop()
	val2, err2 := evm.stack.pop()
	if err1 != nil || err2 != nil {
		log.Printf("Error in execution invalid evm program")
	}
	evm.stack.push(val1 + val2)
}

func callValue(evm *EVM, args []byte) {
	// For now just hardcode, future could actually take in a message arguments
	// from somewhere
	// Push onto the stack the amount of ether sent with message call which initiated this execution
	etherSent := byte(0x0a)
	evm.stack.push(etherSent)
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

// Process an op, return new index in the byte code
// startIndex should always point to an instruction
func (evm *EVM) handleOp(evmProgram []byte, startIndex int) (int, error) {
	log.Print(evmProgram[startIndex])
	op, ok := evm.opCodes[evmProgram[startIndex]]
	if !ok {
		log.Print("Unknown op code")
		return -1, errors.New("Invalid op code")
	}
	nextInstruction := startIndex + op.numArgs + 1
	// get however many arguments this op needs and call its function
	op.handler(evm, evmProgram[startIndex+1:nextInstruction])
	return nextInstruction, nil
}

// Walk through the bytes interpreting the opcodes
// TODO: stop if we run out of gas
func (evm EVM) execute(evmProgram []byte) {
	for curr := 0; curr < len(evmProgram); {
		// Handle each op
		next, err := evm.handleOp(evmProgram, curr)
		if err != nil {
			log.Printf("Execution failed on op %v", evmProgram[curr])
			break
		}
		curr = next
	}
}

func main() {
	reader := bufio.NewReader(os.Stdin)
	log.Print("Enter EVM program: ")
	program, _ := reader.ReadString('\n')
	program = program[:(len(program) - 1)]

	evm := EVM{stack: &EVMStack{}}
	evm.init()
	instructions := evm.parse(program)
	log.Print(instructions)
}
