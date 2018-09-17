package main

import (
	"bytes"
	"encoding/hex"
	"testing"
)

type EVMTest struct {
	inputBytes    string
	expectedStack []Word
	expectedMem   []Word
}

func testRunner(t *testing.T, tests []EVMTest) {
	for _, tt := range tests {
		evm := EVM{call: ContractCall{CallValue: "01", CallData: "23cd4e55"}, stack: &EVMStack{}, memory: &EVMMem{}}
		evm.init()
		instructions := evm.parse(tt.inputBytes)
		t.Log(instructions)
		evm.execute(instructions)
		// evm.stack.stack will be a []Word for each word
		if len(evm.stack.stack) != len(tt.expectedStack) {
			t.Logf("Expected %v stack length got %v", len(tt.expectedStack), len(evm.stack.stack))
			t.Fail()
		}
		for i := range evm.stack.stack {
			if !bytes.Equal(evm.stack.stack[i][:], tt.expectedStack[i][:]) {
				t.Logf("Expected %v on stack got %v", tt.expectedStack[i], evm.stack.stack[i])
				t.Fail()
			}
		}
		if len(evm.memory.mem) != len(tt.expectedMem) {
			t.Logf("Expected %v memory length got %v", len(tt.expectedMem), len(evm.memory.mem))
			t.Fail()
		}
		for i := range evm.memory.mem {
			if !bytes.Equal(evm.memory.mem[i][:], tt.expectedMem[i][:]) {
				t.Logf("Expected %v on stack got %v", tt.expectedMem[i], evm.memory.mem[i])
				t.Fail()
			}
		}
	}
}

func paddWord(input ...byte) Word {
	var res Word
	for i := len(input) - 1; i >= 0; i-- {
		res[31-i] = input[i]
	}
	return res
}

func TestStack(t *testing.T) {
	// Test stack operations pop, mstore, mload, sload/store, msize, pushx, dupx, swapx
	var stackTests = []EVMTest{
		// Push1
		EVMTest{hex.EncodeToString([]byte{byte(PUSH1), byte(0x10), byte(PUSH1), byte(0x20)}),
			[]Word{paddWord(0x10), paddWord(0x20)}, []Word{}},
		// Push2
		EVMTest{hex.EncodeToString([]byte{byte(PUSH2), byte(0x10), byte(0x10)}),
			[]Word{paddWord(0x10, 0x10)}, []Word{}},
		// Pop
		EVMTest{hex.EncodeToString([]byte{byte(PUSH1), byte(0x10), byte(POP)}),
			[]Word{}, []Word{}},
		// MSTORE 0x10 in address 0x04
		EVMTest{hex.EncodeToString([]byte{byte(PUSH1), byte(0x10), byte(PUSH1), byte(0x03), byte(MSTORE)}),
			[]Word{}, []Word{paddWord(0x00), paddWord(0x00), paddWord(0x00), paddWord(0x10)}},
		// Mload
		EVMTest{hex.EncodeToString([]byte{byte(PUSH1), byte(0x10), byte(PUSH1), byte(0x01), byte(MSTORE),
			byte(PUSH1), byte(0x01), byte(MLOAD)}),
			[]Word{paddWord(0x10)}, []Word{paddWord(0x00), paddWord(0x10)}},
		// Add
		EVMTest{hex.EncodeToString([]byte{byte(PUSH1), byte(0x01), byte(PUSH1), byte(0x02), byte(ADD)}),
			[]Word{paddWord(0x03)}, []Word{}},
		// Dup1
		EVMTest{hex.EncodeToString([]byte{byte(PUSH1), byte(0x01), byte(DUP1)}),
			[]Word{paddWord(0x01), paddWord(0x01)}, []Word{}},
		// Dup 2
		EVMTest{hex.EncodeToString([]byte{byte(PUSH1), byte(0x02), byte(PUSH1), byte(0x01), byte(DUP2)}),
			[]Word{paddWord(0x02), paddWord(0x01), paddWord(0x02)}, []Word{}},
		// swap1
		EVMTest{hex.EncodeToString([]byte{byte(PUSH1), byte(0x01), byte(PUSH1), byte(0x02), byte(SWAP1)}),
			[]Word{paddWord(0x02), paddWord(0x01)}, []Word{}},
		// swap2
		EVMTest{hex.EncodeToString([]byte{byte(PUSH1), byte(0x01), byte(PUSH1), byte(0x02), byte(PUSH1), byte(0x03), byte(SWAP2)}),
			[]Word{paddWord(0x03), paddWord(0x02), paddWord(0x01)}, []Word{}},
		EVMTest{hex.EncodeToString([]byte{byte(PUSH1), byte(0x01), byte(PUSH1), byte(0x02), byte(PUSH1), byte(0x03), byte(SWAP2)}),
			[]Word{paddWord(0x03), paddWord(0x02), paddWord(0x01)}, []Word{}}}
	testRunner(t, stackTests)
}

func TestProcessFlow(t *testing.T) {
	// Stop, jump, jumpi, pc, jumpdest
	var logicalTests = []EVMTest{
		// jumpi test - jump to a later part of the code which contains a stack push
		// make sure the instructions in between are skipped
		EVMTest{hex.EncodeToString([]byte{byte(PUSH1), byte(0x01), byte(PUSH1), byte(0x09), // jump to push 3
			byte(JUMPI), byte(PUSH1), byte(0x01), byte(PUSH1),
			byte(0x02), byte(JUMPDEST), byte(PUSH1), byte(0x03)}),
			[]Word{paddWord(0x03)}, []Word{}},
		// stop test - should break out early and not execute the last push
		EVMTest{hex.EncodeToString([]byte{byte(STOP), byte(PUSH1), byte(0x01)}),
			[]Word{}, []Word{}}}
	testRunner(t, logicalTests)
}

func TestLogical(t *testing.T) {
	// LT, GT, SLT, SGT, EQ, ISZERO, AND, OR, XOR, NOT, BYTE
	var logicalTests = []EVMTest{
		// iszero(0x10) --> push 0 on stack
		EVMTest{hex.EncodeToString([]byte{byte(PUSH1), byte(0x10), byte(ISZERO)}),
			[]Word{paddWord(0x00)}, []Word{}},
		// iszero(0x00) --> push 1 on stack
		EVMTest{hex.EncodeToString([]byte{byte(PUSH1), byte(0x00), byte(ISZERO)}),
			[]Word{paddWord(0x01)}, []Word{}},
		// LT  false
		EVMTest{hex.EncodeToString([]byte{byte(PUSH1), byte(0x10), byte(PUSH1), byte(0x20), byte(LT)}),
			[]Word{paddWord(0x00)}, []Word{}},
		// LT true
		EVMTest{hex.EncodeToString([]byte{byte(PUSH1), byte(0x20), byte(PUSH1), byte(0x10), byte(LT)}),
			[]Word{paddWord(0x01)}, []Word{}},
		// GT true
		EVMTest{hex.EncodeToString([]byte{byte(PUSH1), byte(0x01), byte(PUSH1), byte(0x10), byte(GT)}),
			[]Word{paddWord(0x01)}, []Word{}},
		// EQ true
		EVMTest{hex.EncodeToString([]byte{byte(PUSH1), byte(0x20), byte(PUSH1), byte(0x20), byte(EQ)}),
			[]Word{paddWord(0x01)}, []Word{}}}
	testRunner(t, logicalTests)
}

func TestEnviron(t *testing.T) {
	// gas, address, balance, origin, caller, callvalue, calldataload, calldatasize
	// calldatacopy, codesize, codecopy, gasprice, extcodesize, extcodecopy
	// returndatasize, returndatacopy

	// Hardcode the callvalue (input ether), call data etc.
	var environTests = []EVMTest{
		EVMTest{hex.EncodeToString([]byte{byte(CALLVALUE)}),
			[]Word{hexStringToWord("01")}, []Word{}},
		EVMTest{hex.EncodeToString([]byte{byte(CALLDATALOAD)}),
			[]Word{hexStringToWord("23cd4e55")}, []Word{}}} // msg_data should be the first 32 bytes of calldata
	testRunner(t, environTests)
}

func TestFunctionCall(t *testing.T) {
	// Say the input was
	// to do a simple function call like in simple_storage get
	// we need push, mstore, push1, calldatasize, lt, jumpi, calldataload
	// push29, swap1, div, push4, and, dup1, eq, revert, mload
	// dup2, dup3, swap2, sub, return, log1, push6, sha3, push15, codecopy
	// dup9, swap11
	return
}

func TestStorageContract(t *testing.T) {
}

func TestRealContract(t *testing.T) {
}
