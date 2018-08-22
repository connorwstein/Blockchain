package main

import (
//     "crypto/sha256"
    "fmt"
    "time"
    "errors"
// 	"encoding/binary"
    "golang.org/x/net/context"
    "google.golang.org/grpc"
    "crypto/ecdsa"
    "crypto/rand"
    "crypto/elliptic"
    "net"
    "strings"
    "google.golang.org/grpc/peer"
    pb "./protos"
    "encoding/hex"
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
//     blockChain map[string]*pb.Block
    // Maintain a list of blocks which are the tips of various chains, one of which is the main chain?
    // Also need to maintain a list of orphaned blocks to be added to chain once their parent arrives
    // For simplicity lets assume that length of chain represents work that went into it
    // (not always true as forks can span re-targets (difficulty increases). This way we can just check the 
    // height in the block and use that.
//     tipsOfChains []*pb.Block 
    key *ecdsa.PrivateKey
    // I think we will need a pool of orphan blocks as well
    // This blockNum is the number of the next block we expect to add to the chain either by
    // mining or by receiving a new block
//     blockNum int
)

type server struct{}

type BlockchainPeer struct {
    conn *grpc.ClientConn 
    peerIP string
    sourceIP string
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

    blockHash := string(getBlockHash(in))
    if _, ok := blockChain.blocks[blockHash]; ok {
        fmt.Printf("Already have block %v", blockHash)
        return &reply, nil
    }
    // If the block number the next block we were looking for update it
    // TODO: handle out of order (orphan blocks)
    if int(in.Header.Height) == blockChain.nextBlockNum {
        blockChain.nextBlockNum = int(in.Header.Height) + 1
    }
    fmt.Println("Received valid new block adding to local chain\n", 
               getBlockString(in))
    blockChain.blocks[blockHash] = in
    blockChain.tipsOfChains[0] = in 
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
    pubkey := make([]byte, 64)
    pubkey = append(pubkey, key.PublicKey.X.Bytes()...)
    pubkey = append(pubkey,  key.PublicKey.Y.Bytes()...)
    return pubkey
}

func setKey(newKey *ecdsa.PrivateKey) {
    key = newKey
}

func (s *server) GetAddress(ctx context.Context, in *pb.Empty) (*pb.AccountCreated, error) {
    var account pb.AccountCreated
    addr := strings.Join([]string{key.X.String(), key.Y.String()}, "")
    fmt.Println(addr)
    account.Address = addr
    return &account, nil
}

func (s *server) GetBlocks(in *pb.Empty, stream pb.State_GetBlocksServer) error {
    fmt.Println("Get blocks ", len(blockChain.blocks))
    // Walk the mempool 
    // This is the only slow part, is building a sorted list
    orderedBlocks := make([]*pb.Block, len(blockChain.blocks))
    for _, block := range blockChain.blocks {
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

func createKey() *ecdsa.PrivateKey {
    // Create a key pair 
    // Get a keypair
    // Need the curve for our trapdoor
    // Allocate memory for a private key
    ellipticKey := new(ecdsa.PrivateKey)
    // Generate the keypair based on the curve
    ellipticKey, _ = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
    return ellipticKey 
}

func (s *server) NewAccount(ctx context.Context, in *pb.Account) (*pb.AccountCreated, error) {
    var reply pb.AccountCreated
    fmt.Println("New Account for: ", in.Name)
    key = createKey()
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

func initBlockChain() []string {
    // Returns a list of peers
    ips = make([]net.IPNet, 0)
    peerList = make(map[string]BlockchainPeer, 0)
    nodeList := []string{"172.27.0.2", "172.27.0.3", "172.26.0.2", 
                         "172.26.0.4", "172.25.0.2", "172.25.0.3", 
                         "172.24.0.2", "172.24.0.3"}
    nodeList = removeOurAddress(nodeList)
    memPool = make(map[string]*pb.Transaction, 0)
    blockChain = &Blockchain{blocks: make(map[string]*pb.Block, 0), 
                             tipsOfChains: make([]*pb.Block, 0), 
                             nextBlockNum: 1}

    target, _ := hex.DecodeString(strings.Repeat("f", 19))
    blockChain.setTarget(target)
    blockChain.addGenesisBlock()
    return nodeList
}

func main() {
    fmt.Println("Listening")
    nodeList := initBlockChain()
    ticker := time.NewTicker(PEER_CHECK * time.Millisecond)
    go func() {
        for _ = range ticker.C {
            connectToPeers(nodeList)
        }
    }()
    startGrpc() 
}
