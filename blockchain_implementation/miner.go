// need functions to mine blocks
// accumultate transactions into blocks
// and broadcast new blocks
package main

import (
    pb "./protos"
    "fmt"
    "golang.org/x/net/context"
    "bytes"
)

func BlockIsValid(block *pb.Block) bool {
    // Check if my block is valid i.e. hash starts with a zero 
    // And the upstream block must be valid unless it is the genesis block
    blockHash := getBlockHash(block)
    if block.Header.PrevBlockHash != nil {
        if bytes.Equal(block.Header.PrevBlockHash[0:4], []byte{0x00, 0x00, 0x00}) {
            return false
        }
    } 
    return bytes.Equal(blockHash[0:3], []byte{0x00,0x00, 0x00}) 
}

func MineBlock(block *pb.Block) {
    // Increment the nonce until the hash starts with 4 zeros.
    for {
        if ! BlockIsValid(block) {
            // Increment the nonce, append the block data to it then hash it
            block.Header.Nonce += 1
        } else {
            break
        }
    }
    fmt.Println("Mined block: ", block)
}

func (s *server) StartMining(ctx context.Context, in *pb.Empty) (*pb.Empty, error) {
    var reply pb.Empty
    go Mine()
    return &reply, nil
}


// Go routine to continuously mine while still accumulating blocks in the mempool
// note that we can mine a block without anything in the block and still get paid
func Mine() {
    // Take whatever is in the mempool right now and start mining it in a block
    for {
        var newBlock pb.Block
        var newBlockHeader pb.BlockHeader
        newBlock.Header = &newBlockHeader
        newBlock.Transactions = make([]*pb.Transaction, 0)
        // Add a transaction to ourselves
        // leave inputUTXO and senderPubKey as zeroed bytes
        // as this is minted 
        var mint pb.Transaction
        // Receiver is our key (note need an account before you can mine)
        // Coinbase transaction is actually unsigned
        mint.ReceiverPubKey = getPubKey()
        mint.Value = BLOCK_REWARD
        newBlock.Transactions = append(newBlock.Transactions, &mint)
        // Now add all the other ones (could be empty)
        for _, transaction := range memPool {
            newBlock.Transactions = append(newBlock.Transactions, transaction)
        } 
        // Blocks until mining is complete
        MineBlock(&newBlock) 
        // Add new block to the chain 
        // Blocks previous hash is the tip of the currently longest chain
        // TODO: broadcast this new block
        newBlock.Header.PrevBlockHash = getBlockHash(tipsOfChains[0])
        blockChain[string(getBlockHash(&newBlock))] = &newBlock
        tipsOfChains[0] = &newBlock
    }
}
