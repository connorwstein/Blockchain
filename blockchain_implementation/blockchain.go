package main

import (
    "crypto/sha256"
    "fmt"
    "time"
	"bytes"
	"encoding/binary"
    "golang.org/x/net/context"
    "google.golang.org/grpc"
    "net"
    "strings"
    "google.golang.org/grpc/peer"
    pb "./protos"
)

const (
    PORT = "8333"
)

type server struct{}

var peerList []*BlockchainPeer
var ips []net.IPNet

type BlockchainPeer struct {
    conn *grpc.ClientConn 
    peerIP string
    sourceIP string
}

type Block struct {
	blockNumber int
	nonce uint32 // 4 byte nonce in bitcoin
	data []byte
	hash []byte // 32 bytes for SHA256 digest
	prevBlock *Block // So we can walk the DAG
}

func (block *Block) GetNonceBytes() []byte {
	nonceBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(nonceBytes, block.nonce)
	return nonceBytes
}

func (block *Block) ComputeHash() []byte {
	toHash := make([]byte, 0)
	toHash = append(toHash, block.GetNonceBytes()...)
	toHash = append(toHash, block.prevBlock.hash...)
	toHash = append(toHash, block.data...)
	sum := sha256.Sum256(toHash)
	return sum[:]
}

func (block *Block) Mine() {
	// Increment the nonce until the hash starts with 4 zeros.
    block.hash = block.ComputeHash() 
	for {
		if ! block.IsValid() {
			// Increment the nonce, append the block data to it then hash it
			block.nonce += 1
    		block.hash = block.ComputeHash()
		} else {
			break
		}
	}
}

func (block *Block) Modify(newData []byte) {
	// Add some new data and recompute the blocks hash
	block.data = newData
	block.hash = block.ComputeHash()
}

func (block *Block) IsValid() bool {
	// Check if my block is valid i.e. hash starts with a zero 
	// And the upstream block must be valid unless it is the genesis block
	return bytes.Equal(block.hash[0:1], []byte{0x00}) && (block.prevBlock == nil || block.prevBlock.IsValid())
}

func (block *Block) ToString() string {
	return fmt.Sprintf("Block Number %v Data %s Validity %v", block.blockNumber, string(block.data), block.IsValid())
}

func PrintBlockChain(current *Block) {
	// From the current block, walk all the way to the genesis block
	currentBlock := current
	for currentBlock.prevBlock != nil {
		fmt.Println(currentBlock.ToString())
		currentBlock = currentBlock.prevBlock
	}
	fmt.Println(currentBlock.ToString())
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
    if err := s.Serve(lis); err != nil {
        fmt.Printf("gRPC server failed to start serving: %v", err)
    }
}

func getOutgoingIP(peerIP string) string {
    // Determine which one of our IPs is in the same network as the peer
    ipPeer := net.ParseIP(peerIP)
    fmt.Println(ipPeer)
    for _, ip := range ips {
        fmt.Println(ip)
        if ip.Contains(ipPeer) {
            return ip.IP.String()
        }
    }
    return ""
}

func (s *server) ReceiveTransaction(ctx context.Context, in *pb.Transaction) (*pb.Empty, error) {
    var reply pb.Empty
    peerIP, _ := peer.FromContext(ctx)
    // TODO: Make this safer
    senderAddr := peerIP.Addr.(*net.TCPAddr)
    fmt.Printf("Receive Transaction %v %v", in, senderAddr.IP.String())
    // If we are a miner, then we need to accumulate these transactions and build a block
    // to verify then broadcast
    // Otherwise we just forward the transactions along to our peers as part of flooding
    // forward to everyone except who we received it from 
    for _, bcPeer := range peerList {
        if bcPeer.peerIP == senderAddr.IP.String() {
            // Don't send back to the receiver
            continue
        }
        ipAddr, _ := net.ResolveIPAddr("ip", bcPeer.sourceIP)
        ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: ipAddr})
        c := pb.NewTransactionsClient(bcPeer.conn)
        c.ReceiveTransaction(ctx, &pb.Transaction{Transaction: "test"})
    }
    return &reply, nil
}

func (s *server) SendTransaction(ctx context.Context, in *pb.Transaction) (*pb.Empty, error) {
    var reply pb.Empty
    fmt.Printf("Send Transaction %v\n", in.Transaction)
    // Send this transaction to all the list of clients we are connected to
    // Need to include the source, so that the peer doesn't send it back to us
    for _, bcPeer := range peerList {
        // Find which one of our IP addresses is in the same network as the peer
        ipAddr, _ := net.ResolveIPAddr("ip", bcPeer.sourceIP)
        // This cast works because ipAddr is a pointer and the pointer to ipAddr does implement 
        // the Addr interface
        ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: ipAddr})
        c := pb.NewTransactionsClient(bcPeer.conn)
        c.ReceiveTransaction(ctx, &pb.Transaction{Transaction: "test"})
    }
    return &reply, nil
}

func (s *server) GetTransactions(in *pb.Empty, stream pb.State_GetTransactionsServer) error {
    var t pb.Transaction
    fmt.Println("Get transactions")
    t.Transaction = "test transaction 1"
    stream.Send(&t)
    return nil
}

func (s *server) Connect(ctx context.Context, in *pb.Hello) (*pb.Ack, error) {
    var reply pb.Ack
    fmt.Println("Peer connect")
    return &reply, nil
}

func connectToPeers(nodeList []string) {
    // Keep trying to connect until you have at least one peer
    for {
        for _, node := range nodeList {
            conn, err := grpc.Dial(strings.Join([]string{node, ":", PORT}, ""), grpc.WithInsecure())
            if err != nil {
                fmt.Printf("Failed to connect to gRPC server: %v", err)
            } else {
                client := pb.NewPeeringClient(conn)
                ctx, _ := context.WithTimeout(context.Background(), 500 * time.Millisecond)
                _, err = client.Connect(ctx, &pb.Hello{})
                if err == nil {
                    // Save that connection, will send new transactions to peers to flood the network
                    fmt.Println("node ", node)
                    peer := BlockchainPeer{conn: conn, peerIP: node, sourceIP: getOutgoingIP(node)}
                    peerList = append(peerList, &peer)
                }
            }
        }
        if len(peerList) > 0 {
            for _, peer := range peerList {
                fmt.Printf("Peer %v source %v\n", peer.peerIP, peer.sourceIP)
            }
            break
        }
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
                    fmt.Println(v.IP)
                    ips = append(ips, *v)
                    for i, val := range nodeList {
                        if val == v.IP.String() {
                            fmt.Println("Remove from nodeList", val)
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
    // TODO: Connect to a seed node to get a list of peers
    // For now start with a hardcoded list of peers
    // Try to connect to each peer in the list, and if successful save that connection 
    // The graph will always be connected because for you to join, you have to peer with an existing node
    // (aside from the very first node). Problem: what you have some network established then one of the nodes restarts
    // and decides to peer with a set of nodes If you restart a node it will connect to nodes it had already connected
    // to before. 
    ips = make([]net.IPNet, 0)
    peerList = make([]*BlockchainPeer, 0)
    nodeList := []string{"172.27.0.2", "172.27.0.3", "172.26.0.2", 
                         "172.26.0.4", "172.25.0.2", "172.25.0.4", 
                         "172.24.0.2", "172.24.0.3"}
    nodeList = removeOurAddress(nodeList)
    go connectToPeers(nodeList)
    startGrpc() 
}
