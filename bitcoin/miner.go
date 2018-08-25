// Need functions to mine blocks
// accumultate transactions into blocks
// and broadcast new blocks
package main

import (
    pb "./protos"
    "fmt"
    "golang.org/x/net/context"
    "bytes"
    "time"
    "net"
    "google.golang.org/grpc/peer"
)

func checkHashMined(target []byte, hash []byte) bool {
    return bytes.Compare(hash, target) < 0
}

func mineBlock(target []byte, block *pb.Block, stop chan struct{}) bool {
    // Increment the nonce until the hash starts with some
    // leading zeroes (depends on the difficulty)
    for {
        select {
        case <- stop:
            fmt.Println("Stop mining")
            return false 
        default:
            if ! checkHashMined(target, getBlockHash(block)) {
                // Increment the nonce, append the block data to it then hash it
                block.Header.Nonce += 1
            } else {
                fmt.Printf("Mined block: %s\n", getBlockString(block))
                return true
            }
       }
        time.Sleep(MINE_SPEED * time.Millisecond)
    }
}

func (s *Server) StartMining(ctx context.Context, in *pb.Empty) (*pb.Empty, error) {
    var reply pb.Empty
    go mine(s)
    return &reply, nil
}

func (s *Server) StopMining(ctx context.Context, in *pb.Empty) (*pb.Empty, error) {
    var reply pb.Empty
    if s.isMining {
        fmt.Println("Stop mining")
        s.stopMining <- struct{}{} // creating a new empty struct, good for signalling
    } else {
        fmt.Println("Already not mining")
    }
    return &reply, nil
}

// Go routine to continuously mine while still accumulating blocks in the mempool
// note that we can mine a block without anything in the block and still get paid
func mine(s *Server) {
    // Take whatever is in the mempool right now and start mining it in a block
    fmt.Println("Start mining")
    s.isMining = true
    for {
        var newBlock pb.Block
        var newBlockHeader pb.BlockHeader
        newBlock.Header = &newBlockHeader
        newBlock.Header.TimeStamp = uint64(time.Now().UnixNano())
        newBlock.Header.PrevBlockHash = getBlockHash(s.Blockchain.tipsOfChains[0])
        newBlock.Header.Height = uint64(s.Blockchain.nextBlockNum)
        newBlock.Transactions = make([]*pb.Transaction, 0)
        var mint pb.Transaction
        // Receiver is our key (note need an account before you can mine)
        // Coinbase transaction is actually unsigned
        var TXO pb.TXO
        TXO.ReceiverPubKey = getPubKeyBytes(s.Wallet.key)
        TXO.Value = BLOCK_REWARD
        mint.Height = uint64(s.Blockchain.nextBlockNum)
        mint.Vout = make([]*pb.TXO, 0)
        mint.Vout = append(mint.Vout, &TXO)
        newBlock.Transactions = append(newBlock.Transactions, &mint)
        // Now add all the other ones (could be empty)
        for _, transaction := range s.MemPool.transactions {
            newBlock.Transactions = append(newBlock.Transactions, transaction)
        } 
        // Blocks until mining is complete
        // Need a way to abort if a new block at the same number is received while mining
        result := mineBlock(s.Blockchain.target, &newBlock, s.stopMining) 
        // After mining we cannot modify the block, otherwise its hash will no longer
        // be valid
        if result {
            blockHash := string(getBlockHash(&newBlock))
            for i := range newBlock.Transactions {
                delete(s.MemPool.transactions, string(getTransactionHash(newBlock.Transactions[i])))
            } 
            // index all transactions 
            for i := range newBlock.Transactions {
                s.Blockchain.txIndex[string(getTransactionHash(newBlock.Transactions[i]))] = TxIndex{blockHash : blockHash, 
                                                                                               index: i}
            }
            s.Blockchain.blocks[blockHash] = &newBlock
            s.Blockchain.tipsOfChains[0] = &newBlock
            s.Blockchain.nextBlockNum += 1
            // Broadcast this block
            // Send block to all peers. Block is valid since we just mined it
            for _, myPeer := range s.peerList {
                // Find which one of our IP addresses is in the same network as the peer
                ipAddr, _ := net.ResolveIPAddr("ip", myPeer.sourceIP)
                // This cast works because ipAddr is a pointer and the pointer to ipAddr does implement 
                // the Addr interface
                ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: ipAddr})
                c := pb.NewBlocksClient(myPeer.conn)
                c.ReceiveBlock(ctx, &newBlock)
            }
        } else {
            fmt.Println("Aborted mining")
            break
        }
    }
}
