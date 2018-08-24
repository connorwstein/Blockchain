// need functions to mine blocks
// accumultate transactions into blocks
// and broadcast new blocks
package main

import (
    pb "./protos"
    "fmt"
    "golang.org/x/net/context"
    "bytes"
    "encoding/hex"
    "crypto/sha256"
    "encoding/binary"
    "strconv"
    "time"
    "net"
    "crypto/ecdsa"
    "google.golang.org/grpc/peer"
)

var quit chan struct{}

type Blockchain struct {
    blocks map[string]*pb.Block
    tipsOfChains []*pb.Block
    nextBlockNum int
    target []byte // difficulty for mining
    // This an index to lookup a block hash by transaction hash,
    // the real bitcoin implementation has something similar but heavily cached/optimized
    // see bitcoin/src/index/txindex.h
    txIndex map[string]TxIndex
}

// Block and index of transaction 
// to quickly look up a transaction
type TxIndex struct {
    blockHash string
    index int
}

// Wrapper with details about the location
// of a specific TXI / TXO
type UTXO struct {
    transaction *pb.Transaction // transaction which has the UTXO
    index int // vout index
}

func (b *Blockchain) setTarget(inputTarget []byte) {
    b.target = inputTarget
}

func (b Blockchain) String() string {
    // Print the blocks in sorted order
    return ""
}

func (b *Blockchain) addGenesisBlock() {
    var genesis pb.Block
    var genesisHeader pb.BlockHeader
    // Block # 1
    genesisHeader.Height = 1
    genesisHeader.PrevBlockHash = make([]byte, 32)
    genesisHeader.MerkleRoot = make([]byte, 32)
    genesis.Header = &genesisHeader
    b.nextBlockNum = 2 // Next block num
    b.blocks[string(getBlockHash(&genesis))] = &genesis
    // Currently the longest chain is this block to build on
    // top of
    b.tipsOfChains = append(b.tipsOfChains, &genesis) 
}

// Determine a set of UTXOs which can cover the transaction amount
// return nil if it is not possible
func (blockChain Blockchain) getUTXOsToCoverTransaction(key *ecdsa.PrivateKey, desiredAmount uint64) []*UTXO {
    var currentAmount uint64
    var results []*UTXO
    for _, utxo := range blockChain.getUTXOs(&key.PublicKey) {
        if currentAmount >= desiredAmount {
            return results 
        } else {
            currentAmount += blockChain.getValueUTXO(utxo)
            results = append(results, utxo) 
        }
    }
    // No such utxo
    return results 
}

func (blockChain Blockchain) getBalance(key *ecdsa.PublicKey) uint64 {
    var balance uint64
    for _, utxo := range blockChain.getUTXOs(key) {
        balance += blockChain.getValueUTXO(utxo)
    }
    return balance
}

func (blockChain Blockchain) getTransaction(hash []byte) *pb.Transaction {
    idx := blockChain.txIndex[string(hash)] 
    return blockChain.blocks[idx.blockHash].Transactions[idx.index]
}

func (blockChain Blockchain) getValueUTXO(utxo *UTXO) uint64 {
    txIndex := blockChain.txIndex[string(getTransactionHash(utxo.transaction))]
    block := blockChain.blocks[txIndex.blockHash]
    return block.Transactions[txIndex.index].Vout[utxo.index].Value
}

func (blockChain Blockchain) getTXO(utxo *UTXO) *pb.TXO {
    txIndex := blockChain.txIndex[string(getTransactionHash(utxo.transaction))]
    block := blockChain.blocks[txIndex.blockHash]
    return block.Transactions[txIndex.index].Vout[utxo.index]
}

// Given a transaction input, lookup the pubkey of the transaction hash
// that input references
func (blockChain Blockchain) getSenderPubKey(txi *pb.TXI) []byte {
    txIndex := blockChain.txIndex[string(txi.TxID)] 
    block := blockChain.blocks[txIndex.blockHash]
    trans := block.Transactions[txIndex.index]
    txo := trans.Vout[txi.Index]
    return txo.ReceiverPubKey
}

// Walk the blockchain looking for references to our key 
// Maybe the wallet software normally just caches the utxos
// associated with our keys?
func (blockChain Blockchain) getUTXOs(key *ecdsa.PublicKey) []*UTXO {
    // Kind of wasteful on space, surely a better way
    sent := make([]*UTXO, 0)
    received := make([]*UTXO, 0)
    utxos := make([]*UTXO, 0)
    // Right now we walk every block and transaction
    // Maybe there is a way to use the merkle root here?
    // Make two lists --> inputs from our pubkey and outputs to our pubkey
    // Then walk the outputs looking to see if that output transaction is referenced
    // anywhere in an input, then the utxo was spent
    for _, block := range blockChain.blocks {
        fmt.Printf("Block (%d)\n", len(block.Transactions))
        for _, transaction := range block.Transactions {
            // Within a transaction there can be multiple inputs and outputs
            fmt.Println("transaction ", getTransactionString(transaction))
            for _, inputUTXO := range transaction.Vin {
                // If the transaction hash and index in this vin references an output which has our pub key
                // that means we spent that index
                if bytes.Equal(blockChain.getSenderPubKey(inputUTXO), getPubKeyBytesFromPublicKey(key)) {
                    fmt.Printf("\nSpent %s", getTXIString(inputUTXO))
                    sent = append(sent, &UTXO{transaction: blockChain.getTransaction(inputUTXO.TxID),
                                              index: int(inputUTXO.Index)})
                }
            }
            for i, outputTX := range transaction.Vout {
                if bytes.Equal(outputTX.ReceiverPubKey, getPubKeyBytesFromPublicKey(key)) {
                    fmt.Printf("\nReceived %s", getTXOString(outputTX))
                    received = append(received, &UTXO{transaction: transaction,
                                                      index: i}) 
                }
            }
        } 
    }
    spent := false
    // Walk through all transactions which reference us as an output 
    // check if each one is spent, if not it is a UTXO
    for _, candidateUTXO := range received {
//         fmt.Println(getTXOString(blockChain.getTXO(candidateUTXO)))
        spent = false
        // Loop over the Vin's with our pubkey as the sender (spent)
        // If the transaction hash of the candidateUTXO matches, we cannot use it
        for _, spentTX := range sent {
            fmt.Printf("\nCandidate %s", getTransactionString(candidateUTXO.transaction))
            fmt.Printf("\nSpent %s", getTransactionString(spentTX.transaction))
            if bytes.Equal(getTransactionHash(spentTX.transaction), getTransactionHash(candidateUTXO.transaction)) && spentTX.index == candidateUTXO.index {
                spent = true 
                fmt.Println("spent ^")
            }
        }
        if !spent {
            utxos = append(utxos, candidateUTXO)
        }
    }
    return utxos 
}

func getBlockString(block *pb.Block) string {
    var buf bytes.Buffer
    buf.WriteString("\nBlock Hash: ")
    buf.WriteString(hex.EncodeToString(getBlockHash(block)))
    buf.WriteString("\nBlock Header: ")
    buf.WriteString("\n  prevBlockHash: ")
    buf.WriteString(hex.EncodeToString(block.Header.PrevBlockHash[:]))
    buf.WriteString("\n  timestamp: ")
    buf.WriteString(time.Unix(0, int64(block.Header.TimeStamp)).String())
    buf.WriteString("\n  nonce: ")
    buf.WriteString(strconv.Itoa(int(block.Header.Nonce)))
    buf.WriteString("\n  height: ")
    buf.WriteString(strconv.Itoa(int(block.Header.Height)))
    buf.WriteString("\nTransactions:\n\n")
    for i := range block.Transactions {
        buf.WriteString("\n")
        buf.WriteString(getTransactionString(block.Transactions[i]))
        buf.WriteString("\n")
    }
    return buf.String()
}

func (blockChain Blockchain) blockIsValid(target []byte, block *pb.Block) bool {
    // Check whether the block is mined, its previous block is 
    // mined and all transactions are valid
    if ! checkHashMined(target, getBlockHash(block)) {
        fmt.Println("invalid block hash not mined, target:", target)
        return false
    }
    for _, trans := range block.Transactions {
        if ! blockChain.verifyTransaction(trans) {
            fmt.Println("transaction invalid in block")
            return false
        }
    }  
    return true
}

func checkHashMined(target []byte, hash []byte) bool {
    return bytes.Compare(hash, target) < 0
}

func mineBlock(target []byte, block *pb.Block, quit chan struct{}) bool {
//     fmt.Println("hash target: ", target)
    // Increment the nonce until the hash starts with 3 zeros.
    for {
        select {
        case <- quit:
            fmt.Println("stop mining")
            close(quit)
            return false 
        default:
            if ! checkHashMined(target, getBlockHash(block)) {
                // Increment the nonce, append the block data to it then hash it
//                 fmt.Printf("Mining attempt not successful %v\n", getBlockHash(block))
                block.Header.Nonce += 1
            } else {
                fmt.Printf("Mined block: %s\n", getBlockString(block))
                return true
            }
       }
        time.Sleep(20 * time.Millisecond)
    }
}

func (s *Server) StartMining(ctx context.Context, in *pb.Empty) (*pb.Empty, error) {
    var reply pb.Empty
    quit = make(chan struct{})
    go mine(s, quit)
    return &reply, nil
}

func (s *Server) StopMining(ctx context.Context, in *pb.Empty) (*pb.Empty, error) {
//     fmt.Println("Stop mining")
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
func mine(s *Server, quit chan struct{}) {
    // Take whatever is in the mempool right now and start mining it in a block
    fmt.Println("mining")
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
        result := mineBlock(s.Blockchain.target, &newBlock, quit) 
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
//             fmt.Println("BLOCKS after mining")
//             for _, block := range s.Blockchain.blocks {
//                 fmt.Println(getBlockString(block))
//             }
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

// Walk all the tips of the chains looking for the longest one
func getLongestChain() *pb.Block {
    return nil
}

func getBlockHash(block *pb.Block) []byte {
    // TODO: split into getting the headers hash
    // and the transactions hash 
	toHash := make([]byte, 0)
    // PrevBlockHash can be nil if it is the genesis block
	toHash = append(toHash, block.Header.PrevBlockHash...)
	toHash = append(toHash, block.Header.MerkleRoot...)
    value := make([]byte, 8)
    binary.LittleEndian.PutUint64(value, block.Header.TimeStamp)
    toHash = append(toHash, value...)
    binary.LittleEndian.PutUint64(value, block.Header.Height)
    toHash = append(toHash, value...)
    value = make([]byte, 4)
    binary.LittleEndian.PutUint32(value, block.Header.DifficultyTarget)
    toHash = append(toHash, value...)
    binary.LittleEndian.PutUint32(value, block.Header.Nonce)
    toHash = append(toHash, value...)
    for _, trans := range block.Transactions {
	    toHash = append(toHash, getTransactionHash(trans)...)
    }
	sum := sha256.Sum256(toHash)
    return sum[:]
}
