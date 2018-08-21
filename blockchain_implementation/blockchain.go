package main

import (
    "crypto/sha256"
    "fmt"
    "time"
	"bytes"
    "errors"
	"encoding/binary"
    "golang.org/x/net/context"
    "google.golang.org/grpc"
    "crypto/ecdsa"
    "crypto/rand"
    "crypto/elliptic"
    "net"
    "strings"
    "google.golang.org/grpc/peer"
    pb "./protos"
//     "math/big"
//     "encoding/hex"
//     "strconv"
)

const (
    PORT = "8333"
    PEER_CHECK = 2000
    BLOCK_REWARD = 10
)

var (
    // Don't have a large list of peers, no need to use pointers to structs
    peerList map[string]BlockchainPeer
    ips []net.IPNet
    // TODO: this stuff can probably be moved into its own type 
    // Mempool is a big map of unconfirmed transactions
    // keyed by their stringified
    // Used pointers because it could be large
    // Use a map because we will have to remove transactions
    // from here based on new blocks received
    memPool map[string]*pb.Transaction
    // Giant linked list (with potentially multiple children
    // per node in the case of a temporary fork)
    // Keyed on block hash, value is a pointer to a block  
    // This makes it easy to look up a specific block
    // Need a way to know which blocks are part of the our main chain
    // and which ones are secondary/other competing chains
    blockChain map[string]*pb.Block
    // Maintain a list of blocks which are the tips of various chains, one of which is the main chain?
    // Also need to maintain a list of orphaned blocks to be added to chain once their parent arrives
    // For simplicity lets assume that length of chain represents work that went into it
    // (not always true as forks can span re-targets (difficulty increases). This way we can just check the 
    // height in the block and use that.
    tipsOfChains []*pb.Block 
    key *ecdsa.PrivateKey
    // I think we will need a pool of orphan blocks as well
    blockNum int
)

type server struct{}

type BlockchainPeer struct {
    conn *grpc.ClientConn 
    peerIP string
    sourceIP string
}

// Make sure block is well formed
// and all transactions are valid in the block
func verifyNewBlock(block *pb.Block) bool {
    return true
}

// Walk all the tips of the chains looking for the longest one
func getLongestChain() *pb.Block {
    return nil
}

func startGrpc() {
    lis, err := net.Listen("tcp", strings.Join([]string{":", PORT}, ""))
    if err != nil {
        fmt.Printf("gRPC server failed to start listening: %v", err)
    }
    s := grpc.NewServer()
    pb.RegisterTransactionsServer(s, &server{})
    pb.RegisterPeeringServer(s, &server{})
    pb.RegisterStateServer(s, &server{})
    pb.RegisterWalletServer(s, &server{})
    pb.RegisterMinerServer(s, &server{})
    pb.RegisterBlocksServer(s, &server{})
    if err := s.Serve(lis); err != nil {
        fmt.Printf("gRPC server failed to start serving: %v", err)
    }
}

func getOutgoingIP(peerIP string) (string, error) {
    // Determine which one of our IPs is in the same network as the peer
    ipPeer := net.ParseIP(peerIP)
    for _, ip := range ips {
        if ip.Contains(ipPeer) {
            return ip.IP.String(), nil
        }
    }
    // Shouldn't happen
    return "", errors.New("Can't find outgoing IP for peer") 
}

func addTransactionToMemPool(transaction *pb.Transaction) {
    tx := GetHash(transaction)
    memPool[string(tx[:])] = transaction
    fmt.Printf("Added transaction to mempool:")
    fmt.Println(getTransactionString(transaction))
}

func getSenderIP(ctx context.Context) string { 
    var result string
//     var senderAddr *net.TCPAddr
    peerIP, _ := peer.FromContext(ctx)
    if peerIP == nil {
        return ""
    }
    switch senderAddr := peerIP.Addr.(type) {
        case *net.TCPAddr:
            // Expected case
            fmt.Printf("Receive Transaction %v", senderAddr.IP.String())
            result = senderAddr.IP.String()
        default:
            fmt.Println("Receive Transaction (no sender IP)")
            result = "" 
    }
    return result 
}

// Need to verify a transaction before propagating. This ensures that invalid transactions
// are dropped at the first node which receives it
func (s *server) ReceiveTransaction(ctx context.Context, in *pb.Transaction) (*pb.Empty, error) {
    var reply pb.Empty
    senderIP := getSenderIP(ctx)
    if !TransactionVerify(in)  {
        fmt.Println("Reject transaction, invalid signature")
        return &reply, nil
    }
    addTransactionToMemPool(in) 
    for _, myPeer := range peerList {
        if senderIP == "" || myPeer.peerIP == senderIP {
            // Don't send back to the receiver
            continue
        }
        ipAddr, _ := net.ResolveIPAddr("ip", myPeer.sourceIP)
        ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: ipAddr})
        c := pb.NewTransactionsClient(myPeer.conn)
        c.ReceiveTransaction(ctx, in)
    }
    return &reply, nil
}

// Tell all of our peers about this new block we either
// received or mined 
// TODO do we need this send to be part of the grpc server ?
func (s *server) SendBlock(ctx context.Context, in *pb.Block) (*pb.Empty, error) {
    var reply pb.Empty
    if !BlockIsValid(in) {
        return &reply, nil
    }
    // Send block to all peers. Note we only forward valid blocks
    for _, myPeer := range peerList {
        // Find which one of our IP addresses is in the same network as the peer
        ipAddr, _ := net.ResolveIPAddr("ip", myPeer.sourceIP)
        // This cast works because ipAddr is a pointer and the pointer to ipAddr does implement 
        // the Addr interface
        ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: ipAddr})
        c := pb.NewBlocksClient(myPeer.conn)
        c.ReceiveBlock(ctx, in)
    }
    return &reply, nil
}

func (s *server) ReceiveBlock(ctx context.Context, in *pb.Block) (*pb.Empty, error) {
    var reply pb.Empty
    senderIP := getSenderIP(ctx)
    // Add this block to our chain after verifying it. Since
    // the majority of the nodes are honest and doing this validation
    // miners are incentivized to be honest otherwise the block with their reward won't actually be included in the longest chain and is 
    // thus unusable
    // Verify: block is actually mined and transactions are valid
    if !BlockIsValid(in) {
        fmt.Println("Block is invalid")
        fmt.Println(getBlockString(in))
        return &reply, nil
    }
    // TODO: If we are a miner and we receive a new block for the same
    // number, we need to abandon mining that block and start on the
    // next one

    // Add this guy to the longest chain 
    // If we don't already have the block
    // TODO: handle forks and orphans
    blockHash := string(getBlockHash(in))
    if _, ok := blockChain[blockHash]; ok {
        fmt.Printf("Already have block %v", blockHash)
        return &reply, nil
    }
    // If the block number the next block we were looking for update it
    // TODO: handle out of order (orphan blocks)
    if int(in.Header.Height) == blockNum {
        blockNum = int(in.Header.Height) + 1
    }
    fmt.Println("Received valid new block adding to local chain\n", 
               getBlockString(in))
    blockChain[blockHash] = in
    tipsOfChains[0] = in 
    // Forward this new block along
    for _, myPeer := range peerList {
        if senderIP ==  "" || myPeer.peerIP == senderIP {
            // Don't send back to the receiver
            continue
        }
        // Find which one of our IP addresses is in the same network as the peer
        ipAddr, _ := net.ResolveIPAddr("ip", myPeer.sourceIP)
        // This cast works because ipAddr is a pointer and the pointer to ipAddr does implement 
        // the Addr interface
        ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: ipAddr})
        c := pb.NewBlocksClient(myPeer.conn)
        c.ReceiveBlock(ctx, in)
    }
    return &reply, nil
}

func getPubKey() []byte {
    // Concatenate the 2 32 byte ints representing our public key
    pubkey := make([]byte, 0)
    pubkey = append(pubkey, key.PublicKey.X.Bytes()...)
    pubkey = append(pubkey,  key.PublicKey.Y.Bytes()...)
    return pubkey
}

// Walk the blockchain looking for references to our key 
// Maybe the wallet software normally just caches the utxos
// associated with our keys?
func getUTXOs() []*pb.Transaction {
    // Kind of wasteful on space, surely a better way
    sent := make([]*pb.Transaction, 0)
    received := make([]*pb.Transaction, 0)
    utxos := make([]*pb.Transaction, 0)
    // Right now we walk every block and transaction
    // Maybe there is a way to use the merkle root here?
    // Make two lists --> inputs from our pubkey and outputs to our pubkey
    // Then walk the outputs looking to see if that output transaction is referenced
    // anywhere in an input, then the utxo was spent
    for _, block := range blockChain {
        for _, transaction := range block.Transactions {
            if bytes.Equal(transaction.ReceiverPubKey, getPubKey()) {
                received = append(received, transaction) 
            }
            if bytes.Equal(transaction.SenderPubKey, getPubKey()) {
                sent = append(sent, transaction) 
            }
        } 
    }
    spent := false
    for _, candidateUTXO := range received {
        fmt.Println(getTransactionString(candidateUTXO))
        spent = false
        for _, spentTX := range sent {
            if bytes.Equal(spentTX.InputUTXO, GetHash(candidateUTXO)) {
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

// Find a specific UTXO of ours to reference in a new transaction
// needs to be > desiredAmount.
func getUTXO(desiredAmount uint64) *pb.Transaction {
    for _, utxo := range getUTXOs() {
        if utxo.Value >= desiredAmount {
            return utxo
        } 
    }
    // No such utxo
    return nil
}

func getBalance() uint64 {
    var balance uint64
    for _, utxo := range getUTXOs() {
        balance += utxo.Value
    }
    return balance
}

func (s *server) GetBalance(ctx context.Context, in *pb.Empty) (*pb.Balance, error) {
    var balance pb.Balance
    balance.Balance = getBalance()
    return &balance, nil
}

func (s *server) GetAddress(ctx context.Context, in *pb.Empty) (*pb.AccountCreated, error) {
    var account pb.AccountCreated
    addr := strings.Join([]string{key.X.String(), key.Y.String()}, "")
    fmt.Println(addr)
    account.Address = addr
    return &account, nil
}


// Note this is an honest node, need to find a way to test a malicious node
func (s *server) SendTransaction(ctx context.Context, in *pb.Transaction) (*pb.Empty, error) {
    var reply pb.Empty
    if key == nil {
        fmt.Println("Make an account first")
        return &reply, nil
    }
    // Find some UTXO we can use to cover the transaction
    // If we cannot, then we have to reject the transactionk
    inputUTXO := getUTXO(in.Value)
    if inputUTXO == nil {
        fmt.Printf("Not enough coin for the transaction in the wallet, balance is %d\n", getBalance())
        return &reply, nil
    }
    // Reference to our unspent output being used in this transaction
    in.InputUTXO = GetHash(inputUTXO)
    // Our pub key gets added as part of the signing
    signTransaction(in)
    fmt.Printf("Send transaction %v\n", getTransactionString(in))
    addTransactionToMemPool(in) 
    // Send this transaction to all the list of clients we are connected to
    // Need to include the source, so that the peer doesn't send it back to us
    for _, myPeer := range peerList {
        // Find which one of our IP addresses is in the same network as the peer
        ipAddr, _ := net.ResolveIPAddr("ip", myPeer.sourceIP)
        // This cast works because ipAddr is a pointer and the pointer to ipAddr does implement 
        // the Addr interface
        ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: ipAddr})
        c := pb.NewTransactionsClient(myPeer.conn)
        c.ReceiveTransaction(ctx, in)
    }
    return &reply, nil
}

func (s *server) GetTransactions(in *pb.Empty, stream pb.State_GetTransactionsServer) error {
    fmt.Println("Get transactions")
    // Walk the mempool 
    for _, transaction := range memPool {
        stream.Send(transaction)
    }
    return nil
}

func (s *server) GetBlocks(in *pb.Empty, stream pb.State_GetBlocksServer) error {
    fmt.Println("Get blocks ", len(blockChain))
    // Walk the mempool 
    // This is the only slow part, is building a sorted list
    orderedBlocks := make([]*pb.Block, len(blockChain))
    for _, block := range blockChain {
        orderedBlocks[block.Header.Height - 1] = block
    }
    for _, block := range orderedBlocks {
        if block != nil {
        fmt.Println("Sending: ", getBlockString(block))
        stream.Send(block)
        }
    }
    return nil
}

func (s *server) Connect(ctx context.Context, in *pb.Hello) (*pb.Ack, error) {
    var reply pb.Ack
    fmt.Println("Peer connect")
    return &reply, nil
}

func (s *server) NewAccount(ctx context.Context, in *pb.Account) (*pb.AccountCreated, error) {
    var reply pb.AccountCreated
    fmt.Println("New Account for: ", in.Name)
    // Create a key pair 
    // Get a keypair
    // Need the curve for our trapdoor
    // Allocate memory for a private key
    key = new(ecdsa.PrivateKey)
    // Generate the keypair based on the curve
    key, _ = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
//     var pubkey ecdsa.PublicKey = key.PublicKey
    // TODO: Maybe we return the public key so user knows
    // key itself is two big ints which we convert to and from bigints
    // for sending over the wire
    addr := strings.Join([]string{key.X.String(), key.Y.String()}, "")
    fmt.Println(addr)
    reply.Address = addr
    return &reply, nil
}

func connectToPeers(nodeList []string) {
    for _, node := range nodeList {
        if _, ok := peerList[node]; ok {
            continue    
        }
        conn, err := grpc.Dial(strings.Join([]string{node, ":", PORT}, ""), grpc.WithInsecure())
        if err != nil {
            fmt.Printf("Failed to connect to gRPC server: %v", err)
        } else {
            client := pb.NewPeeringClient(conn)
            ctx, _ := context.WithTimeout(context.Background(), 500 * time.Millisecond)
            _, err = client.Connect(ctx, &pb.Hello{})
            if err == nil {
                // Save that connection, will send new transactions to peers to flood the network
                fmt.Printf("New peer %v!\n", node)
                outgoingIP, _ := getOutgoingIP(node)
                peerList[node] = BlockchainPeer{conn: conn, peerIP: node, sourceIP: outgoingIP}
            }
        }
    }
    fmt.Println("My peer list: ")
    for _, myPeer := range peerList {
        fmt.Printf("Peer %v outgoing interface %v\n", myPeer.peerIP, myPeer.sourceIP)
    }
}

func removeOurAddress(nodeList []string) []string {
    ifaces, _ := net.Interfaces()
    // Remove our own address from the node list
    for _, i := range ifaces {
        // Ignore loopback interfaces
        if i.Name == "lo" {
            continue
        }
        addrs, _ := i.Addrs()
        for _, a := range addrs {
            switch v := a.(type) {
            case *net.IPNet: 
                if v.IP.To4() != nil {
                    ips = append(ips, *v)
                    for i, val := range nodeList {
                        if val == v.IP.String() {
                            nodeList = append(nodeList[:i], nodeList[i+1:]...)
                            break
                        }
                    }
                }
            }
        }
    }
    return nodeList
}

func addGenesisBlock() {
    var genesis pb.Block
    var genesisHeader pb.BlockHeader
    // Block # 1
    genesisHeader.Height = 1
    genesisHeader.PrevBlockHash = make([]byte, 32)
    genesisHeader.MerkleRoot = make([]byte, 32)
    genesis.Header = &genesisHeader
    blockNum = 2 // Next block num
    blockChain[string(getBlockHash(&genesis))] = &genesis
    // Currently the longest chain is this block to build on
    // top of
    tipsOfChains = append(tipsOfChains, &genesis) 
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
	    toHash = append(toHash, GetHash(trans)...)
    }
	sum := sha256.Sum256(toHash)
    return sum[:]
}



func main() {
    fmt.Println("Listening")
    // TODO: pass an argument to indicate whether we are a miner or not
    ips = make([]net.IPNet, 0)
    peerList = make(map[string]BlockchainPeer, 0)
    nodeList := []string{"172.27.0.2", "172.27.0.3", "172.26.0.2", 
                         "172.26.0.4", "172.25.0.2", "172.25.0.3", 
                         "172.24.0.2", "172.24.0.3"}
    nodeList = removeOurAddress(nodeList)
    memPool = make(map[string]*pb.Transaction, 0)
    blockChain = make(map[string]*pb.Block, 0)
    // List of blocks at the ends of chains
    tipsOfChains = make([]*pb.Block, 0)
    addGenesisBlock() 
    // Problem: if they all have a different number of peers but need to connect to all of them because
    // the whole network will not be reachable. However, since we have a list of the all the nodes ips (cheating) 
    // you can periodically try to connect to all of them. This way eventually everyone will be peering with everyone they can.
    // In the real network you would ask some DNS seeds for stable nodes to connect to as peers for the first time.
    ticker := time.NewTicker(PEER_CHECK * time.Millisecond)
    go func() {
        for _ = range ticker.C {
            // TODO: handle peers dying (nice to have)
            connectToPeers(nodeList)
        }
    }()
    startGrpc() 
}
