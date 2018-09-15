package main

import (
	"bytes"
	"encoding/binary"
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
		evm := EVM{stack: &EVMStack{}, memory: &EVMMem{}}
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
	for i := len(input) - 1; i >=0; i-- {
		res[31 - i] = input[i]
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
	b, _ := hex.DecodeString(MSG_DATA)
	var data Word	
 	binary.Write(&data, binary.BigEndian, b)
	var environTests = []EVMTest{
		EVMTest{hex.EncodeToString([]byte{byte(CALLVALUE)}), 
				[]Word{hexStringToWord(MSG_CALLVALUE)}, []Word{}}, 
		EVMTest{hex.EncodeToString([]byte{byte(CALLDATALOAD)}), 
				[]Word{hexStringToWord(MSG_DATA)}, []Word{}}} // msg_data should be the first 32 bytes of calldata
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

	// Instructions needed for this: callvalue, mstore, dup1, iszero, 
	// jumpi, 0x0, revert, pop, dataSize, dup1, dataOffset, codecopy, return, stop

	// EVM assembly:
	//     /* "simple_storage.sol":25:72  contract SimpleStorage {... */
	//   mstore(0x40, 0x80)
	//   callvalue
	//     /* "--CODEGEN--":8:17   */
	//   dup1
	//     /* "--CODEGEN--":5:7   */
	//   iszero
	//   tag_1
	//   jumpi
	//     /* "--CODEGEN--":30:31   */
	//   0x0
	//     /* "--CODEGEN--":27:28   */
	//   dup1
	//     /* "--CODEGEN--":20:32   */
	//   revert
	//     /* "--CODEGEN--":5:7   */
	// tag_1:
	//     /* "simple_storage.sol":25:72  contract SimpleStorage {... */
	//   pop
	//   dataSize(sub_0)
	//   dup1
	//   dataOffset(sub_0)
	//   0x0
	//   codecopy
	//   0x0
	//   return
	// stop
	//
	// sub_0: assembly {
	//         /* "simple_storage.sol":25:72  contract SimpleStorage {... */
	//       mstore(0x40, 0x80)
	//       0x0
	//       dup1
	//       revert
	//
	//     auxdata: 0xa165627a7a72305820039a32ae510dd9ca7064ace05e604144eef026b1be4d6e5456071d9a92c312cc0029
	// }
	//
	// Binary of the runtime part:
// 	push 80 push 40 mstore callvalue iszero push 0x0f jumpi 
// 60 80 60 40 52 34 80 15 60 0f 57 600080fd5b50603580601d6000396000f3006080604052600080fd00a165627a7a72305820039a32ae510dd9ca7064ace05e604144eef026b1be4d6e5456071d9a92c312cc0029
	//push 80 push 40 mstore push 00 dup1 
	// 60 80 60 40 52 60 00 80 fd 00 a1 65 62 7a 7a 72 30 58 20 03 9a 32 ae510dd9ca7064ace05e604144eef026b1be4d6e5456071d9a92c312cc0029
}

func TestRealContract(t *testing.T) {
	//Byte code from the following smart contract
	// pragma solidity ^0.4.19;
	//
	// contract example {
	//
	//   address contractOwner;
	//
	//   function example() {
	//     contractOwner = msg.sender;
	//   }
	// }
	// Assembly:
	// EVM assembly:
	//     /* "add.sol":26:132  contract example {... */
	//   mstore(0x40, 0x80) --> really pushes the two values on the stack then calls mstore which push 0x80 in address 0x40
	//     /* "add.sol":74:130  function example() {... */
	//   callvalue
	//     /* "--CODEGEN--":8:17   */
	//   dup1
	//     /* "--CODEGEN--":5:7   */
	//   iszero
	//   tag_1
	//   jumpi
	//     /* "--CODEGEN--":30:31   */
	//   0x0
	//     /* "--CODEGEN--":27:28   */
	//   dup1
	//     /* "--CODEGEN--":20:32   */
	//   revert
	//     /* "--CODEGEN--":5:7   */
	// tag_1:
	//     /* "add.sol":74:130  function example() {... */
	//   pop
	//     /* "add.sol":115:125  msg.sender */
	//   caller
	//     /* "add.sol":99:112  contractOwner */
	//   0x0
	//   dup1
	//     /* "add.sol":99:125  contractOwner = msg.sender */
	//   0x100
	//   exp
	//   dup2
	//   sload
	//   dup2
	//   0xffffffffffffffffffffffffffffffffffffffff
	//   mul
	//   not
	//   and
	//   swap1
	//   dup4
	//   0xffffffffffffffffffffffffffffffffffffffff
	//   and
	//   mul
	//   or
	//   swap1
	//   sstore
	//   pop
	//     /* "add.sol":26:132  contract example {... */
	//   dataSize(sub_0)
	//   dup1
	//   dataOffset(sub_0)
	//   0x0
	//   codecopy
	//   0x0
	//   return
	// stop
	//
	// sub_0: assembly {
	//         /* "add.sol":26:132  contract example {... */
	//       mstore(0x40, 0x80)
	//       0x0
	//       dup1
	//       revert
	//
	//     auxdata: 0xa165627a7a723058200c169ab676d3371eed99077a197af9efcc29501f04c9ce7d41593a184a4398a70029
	// }
	// 6080604052348015600f57600080fd5b50336000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550603580605d6000396000f3006080604052600080fd00a165627a7a723058200c169ab676d3371eed99077a197af9efcc29501f04c9ce7d41593a184a4398a70029
}
