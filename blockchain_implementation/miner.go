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
    "google.golang.org/grpc/peer"
)

var quit chan struct{}

type Blockchain struct {
    blocks map[string]*pb.Block
    tipsOfChains []*pb.Block
    nextBlockNum int
    target []byte // difficulty for mining
}

var blockChain *Blockchain

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


func BlockIsValid(block *pb.Block) bool {
    // Check whether the block is mined, its previous block is 
    // mined and all transactions are valid
    if !CheckHashMined(getBlockHash(block)) {
        return false
    }
    for _, trans := range block.Transactions {
        if !verifyTransaction(trans) {
            return false
        }
    }  
    return true
}

func CheckHashMined(hash []byte) bool {
//     var b byte = 20 
//     return bytes.Equal(hash[0:2], []byte{0x00, 0x00}) && hash[2] < b
    return bytes.Compare(hash, blockChain.target) < 0
}

func MineBlock(block *pb.Block, quit chan struct{}) bool {
    // Increment the nonce until the hash starts with 3 zeros.
    for {
        select {
        case <- quit:
            fmt.Println("stop mining")
            close(quit)
            return false 
        default:
            if ! CheckHashMined(getBlockHash(block)) {
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
        fmt.Println("mining")
        var newBlock pb.Block
        var newBlockHeader pb.BlockHeader
        newBlock.Header = &newBlockHeader
        newBlock.Header.TimeStamp = uint64(time.Now().UnixNano())
        newBlock.Header.PrevBlockHash = getBlockHash(blockChain.tipsOfChains[0])
        newBlock.Header.Height = uint64(blockChain.nextBlockNum)
        var mint pb.Transaction
        // Receiver is our key (note need an account before you can mine)
        // Coinbase transaction is actually unsigned
        mint.ReceiverPubKey = getPubKey()
        mint.Value = BLOCK_REWARD
        mint.Height = uint64(blockChain.nextBlockNum)
        newBlock.Transactions = append(newBlock.Transactions, &mint)
        // Now add all the other ones (could be empty)
        for _, transaction := range memPool {
            newBlock.Transactions = append(newBlock.Transactions, transaction)
        } 
        // Blocks until mining is complete
        // Need a way to abort if a new block at the same number is received while mining
        result := MineBlock(&newBlock, quit) 
        // After mining we cannot modify the block, otherwise its hash will no longer
        // be valid
        if result {
            for i := range newBlock.Transactions {
                delete(memPool, string(getTransactionHash(newBlock.Transactions[i])))
            } 
            blockChain.blocks[string(getBlockHash(&newBlock))] = &newBlock
            blockChain.tipsOfChains[0] = &newBlock
            blockChain.nextBlockNum += 1
            // Broadcast this block
            // Send block to all peers. Block is valid since we just mined it
            for _, myPeer := range peerList {
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
