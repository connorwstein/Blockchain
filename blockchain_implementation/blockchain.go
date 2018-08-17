package main

import (
//     "crypto/sha256"
    "fmt"
    "time"
// 	"bytes"
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
)

const (
    PORT = "8333"
    PEER_CHECK = 2000
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
    fmt.Printf("Added transaction to mempool %v\n", memPool)
}

// Need to verify a transaction before propagating. This ensures that invalid transactions
// are dropped at the first node which receives it
func (s *server) ReceiveTransaction(ctx context.Context, in *pb.Transaction) (*pb.Empty, error) {
    var reply pb.Empty
    var senderAddr *net.TCPAddr
    peerIP, _ := peer.FromContext(ctx)
    if peerIP == nil {
        return &reply, nil
    }
    switch senderAddr := peerIP.Addr.(type) {
        case *net.TCPAddr:
            // Expected case
            fmt.Printf("Receive Transaction %v %v", in, senderAddr.IP.String())
        default:
            senderAddr = nil
            fmt.Println("Receive Transaction %v (no sender IP)", in)
    }
    // TODO: Verify before adding to mempool and relaying
    TransactionVerify(in) 
    addTransactionToMemPool(in) 
    for _, myPeer := range peerList {
        if senderAddr == nil || myPeer.peerIP == senderAddr.IP.String() {
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

func (s *server) SendTransaction(ctx context.Context, in *pb.Transaction) (*pb.Empty, error) {
    var reply pb.Empty
    fmt.Printf("Send transaction %v\n", in)
//     signTransaction(in)
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
//     var t pb.Transaction
    fmt.Println("Get transactions")
    // Walk the mempool 
    for _, transaction := range memPool {
        fmt.Println(transaction)
        stream.Send(transaction)
    }
    return nil
}

func (s *server) Connect(ctx context.Context, in *pb.Hello) (*pb.Ack, error) {
    var reply pb.Ack
    fmt.Println("Peer connect")
    return &reply, nil
}

func (s *server) NewAccount(ctx context.Context, in *pb.Account) (*pb.Empty, error) {
    var reply pb.Empty
    fmt.Println("New Account for: ", in.Name)
    // Create a key pair 
    // Get a keypair
    // Need the curve for our trapdoor
    // Allocate memory for a private key
    key = new(ecdsa.PrivateKey)
    // Generate the keypair based on the curve
    key, _ = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
    var pubkey ecdsa.PublicKey = key.PublicKey
    fmt.Println("Private Key :", key)
    fmt.Println("Public Key :", pubkey)
    // TODO: Maybe we return the public key so user knows
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
