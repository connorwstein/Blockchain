package evm

import (
	"testing"
)

func TestStack(t *testing.T) {
	// Test stack operations pop, mstore, mload, sload/store, msize, pushx, dupx, swapx

	// push 0x10, push 0x20 then add
	// check stack contains 0x10 + 0x20 = 0x30
	evm := EVM{stack: &EVMStack{}}
	evm.init()
	instructions := evm.parse("6010602001")
	t.Log(instructions)
	evm.execute(instructions)
	if evm.stack.stack[len(evm.stack.stack)-1] != 0x30 {
		t.Logf("Got %v on the stack, expecting %v", evm.stack.stack[len(evm.stack.stack)-1], 0x30)
		t.Fail()
	}

	// test
	//     instructions := evm.parse("")
	//     t.Log(instructions)
	// 	evm.execute(instructions)
}

func TestProcessFlow(t *testing.T) {
	// Stop, jump, jumpi, pc, jumpdest
}

func TestLogical(t *testing.T) {
	// LT, GT, SLT, SGT, EQ, ISZERO, AND, OR, XOR, NOT, BYTE
}

func TestEnviron(t *testing.T) {
	// gas, address, balance, origin, caller, callvalue, calldataload, calldatasize
	// calldatacopy, codesize, codecopy, gasprice, extcodesize, extcodecopy
	// returndatasize, returndatacopy
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
