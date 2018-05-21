package main

import (
    "testing"
    "fmt"
)

func TestBlockMine(t *testing.T){
    // Create a block, mine it and ensure its valid
	b := Block{blockNumber: 0, data:[]byte("Genesis"), prevHash:&[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 
                                                                       0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
                                                                       0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
                                                                       0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}}
    b.Mine()
    if ! b.IsValid() {
        t.Log("Mined block, but still not valid")
        t.Fail()
    }
}

func TestModifyUpstream(t *testing.T){
    // Build a blockchain, modify an upstream block and ensure that all downstream blocks of that are invalidated
	blockCounter := 0
	gen := Block{blockNumber: 0, data:[]byte("Genesis"), prevHash:&[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 
                                                                          0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
                                                                          0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
                                                                          0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}}
    gen.Mine()
    prevHash := &gen.hash
    prevBlock := &gen
    blocks := make([]*Block, 0)
    blocks = append(blocks, &gen)
    for i := 0; i < 100; i++ {
        blockCounter += 1
	    newBlock := Block{blockNumber: blockCounter, data:[]byte(fmt.Sprintf("test block %d", i)), prevHash: prevHash, prevBlock: prevBlock}
        newBlock.Mine()
        prevHash = &newBlock.hash
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
