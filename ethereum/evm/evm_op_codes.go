package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
)

const (
	// Arithmetic
	ADD        = 0x01 //Add the top two stack items
	MUL        = 0x02 //Multiply the top two stack items
	SUB        = 0x03 //Subtract the top two stack items
	DIV        = 0x04 //Integer division
	SDIV              //Signed integer division
	MOD               //Modulo (remainder) operation
	SMOD              //Signed modulo operation
	ADDMOD            //Addition modulo any number
	MULMOD            //Multiplication modulo any number
	EXP               //Exponential operation
	SIGNEXTEND        //Extend the length of a two’s complement signed integer

	// Precomipled contract for this?
	SHA3 = 0x20 //Compute the Keccak-256 hash of a block of memory

	// Stack
	POP     = 0x50 //Remove the top item from the stack
	MLOAD   = 0x51 //Load a word from memory
	MSTORE  = 0x52 //Save a word to memory
	MSTORE8        //Save a byte to memory
	SLOAD          //Load a word from storage
	SSTORE         //Save a word to storage
	MSIZE          //Get the size of the active memory in bytes
	// 	PUSHx   //Place x-byte item on the stack, where x can be any integer from 1 to 32 (full word) inclusive
	PUSH1  = 0x60
	PUSH2  = 0x61
	PUSH3  = 0x62
	PUSH4  = 0x63
	PUSH5  = 0x64
	PUSH6  = 0x65
	PUSH7  = 0x66
	PUSH8  = 0x67
	PUSH9  = 0x68
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
	DUP1  = 0x80
	DUP2  = 0x81
	DUP3  = 0x82
	DUP4  = 0x83
	DUP5  = 0x84
	DUP6  = 0x85
	DUP7  = 0x86
	DUP8  = 0x87
	DUP9  = 0x88
	DUP10 = 0x89
	DUP11 = 0x8a
	DUP12 = 0x8b
	DUP13 = 0x8c
	DUP14 = 0x8d
	DUP15 = 0x8e
	DUP16 = 0x8f
	// 	SWAPx   //Exchange 1st and (x+1)-th stack items, where x can by any integer from 1 to 16 inclusive
	SWAP1  = 0x90
	SWAP2  = 0x91
	SWAP3  = 0x92
	SWAP4  = 0x93
	SWAP5  = 0x94
	SWAP6  = 0x95
	SWAP7  = 0x96
	SWAP8  = 0x97
	SWAP9  = 0x98
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
	LOG0         = 0xa0
	LOG1         = 0xa1
	LOG2         = 0xa2
	LOG3         = 0xa3
	LOG4         = 0xa4
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
	LT     = 0x10 //Less-than comparison
	GT     = 0x11 //Greater-than comparison
	SLT           //Signed less-than comparison
	SGT           //Signed greater-than comparison
	EQ     = 0x14 //Equality comparison
	ISZERO = 0x15 //Simple not operator
	AND    = 0x16 //Bitwise AND operation
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

func opCodeInit() map[byte]OpCode {
	opCodes := make(map[byte]OpCode)

	opCodes[LOG1] = OpCode{LOG1, "LOG1", 0, 750, log1}

	opCodes[ADD] = OpCode{ADD, "ADD", 0, 3, add}
	opCodes[SUB] = OpCode{SUB, "SUB", 0, 3, sub}
	opCodes[CODECOPY] = OpCode{CODECOPY, "CODECOPY", 0, 3, codeCopy}
	opCodes[CALLVALUE] = OpCode{CALLVALUE, "CALLVALUE", 0, 2, callValue}
	opCodes[MSTORE] = OpCode{MSTORE, "MSTORE", 0, 3, mstore}
	opCodes[MLOAD] = OpCode{MLOAD, "MLOAD", 0, 3, mload}

	for i := 0; i < 32; i++ {
		opCodes[byte(PUSH1+i)] = OpCode{byte(PUSH1 + i), fmt.Sprintf("PUSH%d", i+1), i + 1, 3, push}
	}

	for i := 0; i < 16; i++ {
		opCodes[byte(DUP1+i)] = OpCode{byte(DUP1 + i), fmt.Sprintf("DUP%d", i+1), 0, 3, dup}
	}

	for i := 0; i < 16; i++ {
		opCodes[byte(SWAP1+i)] = OpCode{byte(SWAP1 + i), fmt.Sprintf("SWAP%d", i+1), 0, 3, swap}
	}

	opCodes[ISZERO] = OpCode{ISZERO, "ISZERO", 0, 3, iszero}
	opCodes[JUMPI] = OpCode{JUMPI, "JUMPI", 0, 10, jumpi}
	opCodes[JUMP] = OpCode{JUMP, "JUMP", 0, 8, jump}
	opCodes[JUMPDEST] = OpCode{JUMPDEST, "JUMPDEST", 0, 1, jumpdst}
	opCodes[REVERT] = OpCode{REVERT, "REVERT", 0, 0, revert}
	opCodes[POP] = OpCode{POP, "POP", 0, 2, pop}
	opCodes[CALLDATASIZE] = OpCode{CALLDATASIZE, "CALLDATASIZE", 0, 2, callDataSize}
	// 	opCodes[DATAOFFSET] = OpCode{CALLVALUE, 0, 2, callValue} seems to be a mystery
	opCodes[CODECOPY] = OpCode{CODECOPY, "CODECOPY", 0, 3, codeCopy}
	opCodes[RETURN] = OpCode{RETURN, "RETURN", 0, 0, returnF}
	opCodes[STOP] = OpCode{STOP, "STOP", 0, 0, nil}
	opCodes[CALLDATALOAD] = OpCode{CALLDATALOAD, "CALLDATALOAD", 0, 3, callDataLoad}
	opCodes[LT] = OpCode{LT, "LT", 0, 3, lt} // check whether top stack item is lt the second item
	opCodes[EQ] = OpCode{EQ, "EQ", 0, 3, eq}
	opCodes[GT] = OpCode{GT, "GT", 0, 3, gt}

	opCodes[DIV] = OpCode{DIV, "DIV", 0, 5, div}
	opCodes[SHA3] = OpCode{SHA3, "SHA3", 0, 30, sha3}
	opCodes[AND] = OpCode{AND, "AND", 0, 3, and}
	return opCodes
}

func log1(evm *EVM, args []byte) {
	log.Printf("Log 1")
}

func codeCopy(evm *EVM, args []byte) {
	// Copy the code to memory
	log.Printf("Copying code to memory")
}

func sub(evm *EVM, args []byte) {
	// Pop two items from the stack, subtract them and push the result on the stack
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
	binary.Write(&element, binary.BigEndian, x1-x2)
	evm.stack.push(element)
}

func jumpdst(evm *EVM, args []byte) {
	log.Println("In jumpdst, noop")
}

func div(evm *EVM, args []byte) {
	// div stack[0] / stack[1] push result
	// stack[1] should be some multiple of 16 meaning some byte is a 1
	// find the index of this byte and do a right shift of stack[0] byte that amount
	// OR just read up until you find the 1 in stack[1] (cheating a bit)
	if len(evm.stack.stack) < 2 {
		panic("Stack underflow")
	}
	numerator, _ := evm.stack.pop()
	denominator, _ := evm.stack.pop()
	res := make([]byte, 0)
	// Loop until we hit a 1 in the denominator, then copy everything after that
	i := 0
	for i = len(denominator) - 1; i >= 0 && denominator[i] == 0; i-- {
	}
	// last byte (includes the 1)
	if i > 0 {
		for j := 0; j <= i; j++ {
			res = append(res, numerator[j])
		}
	}
	var data Word
	binary.Write(&data, binary.BigEndian, res)
	evm.stack.push(data)
}

func sha3(evm *EVM, args []byte) {
}

func jump(evm *EVM, args []byte) {
	// Set the program counter to first itm on the stack
	if len(evm.stack.stack) < 1 {
		panic("Stack underflow")
	}
	v, _ := evm.stack.pop()
	x1 := binary.BigEndian.Uint64(v[24:])
	log.Printf("Jumping to %v\n", int(x1))
	evm.pc = int(x1)
}

func and(evm *EVM, args []byte) {
	// bitwise and stack[0] and stack[1] push result
	if len(evm.stack.stack) < 2 {
		panic("Stack underflow")
	}
	x1, _ := evm.stack.pop()
	x2, _ := evm.stack.pop()
	var res Word
	for i := range x1 {
		res[i] = x1[i] & x2[i]
	}
	evm.stack.push(res)
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
	// first item on the stack is the offset with with to read the msg data
	evm.stack.pop() // don't use index right now
	evm.stack.push(hexStringToWord(evm.call.CallData))
}

func callValue(evm *EVM, args []byte) {
	// For now just hardcode, future could actually take in a message arguments
	// from somewhere
	// Push onto the stack the amount of ether sent with message call which initiated this execution
	evm.stack.push(hexStringToWord(evm.call.CallValue))
}

func returnF(evm *EVM, args []byte) {
	log.Printf("Return stack values %v\n", evm.stack)
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
		evm.jump = true
		log.Printf("JUMPI condition is TRUE jumping to %d", evm.pc)
	} else {
		log.Printf("JUMPI condition is FALSE")
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
	evm.stack.push(hexStringToWord(evm.call.CallDataSize))
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
	familyIndex := int(args[0]) - SWAP1 + 1 // 1 means SWAP1 etc.
	log.Printf("Swap %d called\n", familyIndex)
	if len(evm.stack.stack) < familyIndex+1 { // need at least 2 elements for a swap1
		panic("Insufficient stack for swap")
	}
	pops := make([]Word, familyIndex+1)
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
	duped := pops[familyIndex-1]
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
	binary.Write(&element, binary.BigEndian, x1+x2)
	evm.stack.push(element)
}
