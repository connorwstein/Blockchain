package main

import (
    "testing"
    "fmt"
//     pb "./protos"
//     "golang.org/x/net/context"
//     "crypto/ecdsa"
//     "crypto/elliptic"
//     "crypto/rand"
)

func TestBlockMine(t *testing.T){
    // Create a block, mine it and ensure its valid
	b := Block{blockNumber: 0, data:[]byte("Genesis"), prevBlock: &Block{hash:[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 
                                                                           0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
                                                                           0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
                                                                           0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}}}
    b.Mine()
    if ! b.IsValid() {
        t.Log("Mined block, but still not valid")
        t.Fail()
    }
}

func TestModifyUpstream(t *testing.T){
    // Build a blockchain, modify an upstream block and ensure that all downstream blocks of that are invalidated
	blockCounter := 0
	gen := Block{blockNumber: 0, data:[]byte("Genesis"), prevBlock:&Block{hash:[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 
                                                                                  0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
                                                                                  0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
                                                                                  0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}}}
    gen.Mine()
    prevBlock := &gen
    blocks := make([]*Block, 0)
    blocks = append(blocks, &gen)
    for i := 0; i < 100; i++ {
        blockCounter += 1
	    newBlock := Block{blockNumber: blockCounter, data:[]byte(fmt.Sprintf("test block %d", i)), prevBlock: prevBlock}
        newBlock.Mine()
        prevBlock = &newBlock
        blocks = append(blocks, &newBlock)
    }
    blockToModify := 10
    blocks[blockToModify].Modify([]byte("Some messed up data"))
    // Check all downstream blocks are invalidated
    for i := blockToModify; i < 100; i++ {
        if blocks[i].IsValid() {
            t.Fail()
        }
    }
}

