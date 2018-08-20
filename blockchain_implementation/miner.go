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

var quit chan struct{}

func BlockIsValid(block *pb.Block) bool {
    // Check whether the block is mined, its previous block is 
    // mined and all transactions are valid
    // Not sure if this check is actually needed as long as each
    // block is checked to be mined before adding to the chain in the first place
    if block.Header.PrevBlockHash != nil {
        if !CheckHashMined(block.Header.PrevBlockHash)  {
            return false
        }
    } 
    if !CheckHashMined(getBlockHash(block)) {
        return false
    }
    for _, trans := range block.Transactions {
        if !TransactionVerify(trans) {
            return false
        }
    }  
    return true
}

func CheckHashMined(hash []byte) bool {
    return bytes.Equal(hash[0:2], []byte{0x00, 0x00})
}

func MineBlock(block *pb.Block) {
    // Increment the nonce until the hash starts with 3 zeros.
    for {
        if ! CheckHashMined(getBlockHash(block)) {
            // Increment the nonce, append the block data to it then hash it
//             fmt.Printf("Mining attempt not successful %v\n", getBlockHash(block))
            block.Header.Nonce += 1
        } else {
            break
        }
    }
    fmt.Println("Mined block: ", block)
}

func (s *server) StartMining(ctx context.Context, in *pb.Empty) (*pb.Empty, error) {
    var reply pb.Empty
    quit = make(chan struct{})
    go Mine(quit)
    return &reply, nil
}

func (s *server) StopMining(ctx context.Context, in *pb.Empty) (*pb.Empty, error) {
    fmt.Println("Stop mining")
    var reply pb.Empty
    if quit != nil {
        fmt.Println("Closing channel")
        quit <- struct{}{}
    } else {
        fmt.Println("Not already mining")
    }
    return &reply, nil
}

// Go routine to continuously mine while still accumulating blocks in the mempool
// note that we can mine a block without anything in the block and still get paid
func Mine(quit chan struct{}) {
    // Take whatever is in the mempool right now and start mining it in a block
    for {
        select {
        case <- quit:
            fmt.Println("stop mining")
            close(quit)
            return 
        default:
            fmt.Println("mining")
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
            // Hard coded for just one chain right now, can't actually handle a fork
            newBlock.Header.PrevBlockHash = getBlockHash(tipsOfChains[0])
            blockChain[string(getBlockHash(&newBlock))] = &newBlock
            tipsOfChains[0] = &newBlock
        }
    }
}


