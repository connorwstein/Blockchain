package main

import (
	pb "./protos"
	"bufio"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
	"net"
	"os"
	"strings"
	"time"
)

const (
	PORT         = "8333"
	PEER_CHECK   = 2000
	BLOCK_REWARD = 10
	MINE_SPEED   = 20 // Milliseconds between nonce increments
)

type Server struct {
	peerList map[string]BlockchainPeer
	ips      []net.IPNet // Set of our IP addresses
	Blockchain
	MemPool // Has unconfirmed transactions
	Wallet
	stopMining chan struct{}
	isMining   bool
}

type BlockchainPeer struct {
	conn     *grpc.ClientConn
	peerIP   string
	sourceIP string
}

// In theory you can have many keys, not currently supported
type Wallet struct {
	key   *ecdsa.PrivateKey
	curve elliptic.Curve
}

func (wallet *Wallet) createKey() error {
	// Create a key pair
	// Get a keypair
	// Need the curve for our trapdoor
	// Allocate memory for a private key
	curve := elliptic.P256()
	ellipticKey := new(ecdsa.PrivateKey)
	// Generate the keypair based on the curve
	ellipticKey, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		return err
	}
	wallet.key = ellipticKey
	wallet.curve = curve
	return nil
}

func startServer(server *Server, port string) {
	lis, err := net.Listen("tcp", strings.Join([]string{":", port}, ""))
	if err != nil {
		fmt.Printf("gRPC server failed to start listening: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterTransactionsServer(s, server)
	pb.RegisterPeeringServer(s, server)
	pb.RegisterStateServer(s, server)
	pb.RegisterWalletServer(s, server)
	pb.RegisterMinerServer(s, server)
	pb.RegisterBlocksServer(s, server)
	// Blocking call
	if err := s.Serve(lis); err != nil {
		fmt.Printf("gRPC server failed to start serving: %v", err)
	}
}

func (server Server) getOutgoingIP(peerIP string) (string, error) {
	// Determine which one of our IPs is in the same network as the peer
	ipPeer := net.ParseIP(peerIP)
	for _, ip := range server.ips {
		if ip.Contains(ipPeer) {
			return ip.IP.String(), nil
		}
	}
	return "", errors.New("Can't find outgoing IP for peer")
}

func getSenderIP(ctx context.Context) string {
	var result string
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

func (s *Server) ReceiveBlock(ctx context.Context, in *pb.Block) (*pb.Empty, error) {
	var reply pb.Empty
	senderIP := getSenderIP(ctx)
	fmt.Println("Receive block %s", getBlockString(in))
	// Add this block to our chain after verifying it. Since
	// the majority of the nodes are honest and doing this validation
	// miners are incentivized to be honest otherwise the block with their reward won't actually be included in the longest chain and is
	// thus unusable
	// Verify: block is actually mined and transactions are valid
	if !s.Blockchain.blockIsValid(s.Blockchain.target, in) {
		fmt.Println("Block is invalid")
		fmt.Println(getBlockString(in))
		return &reply, nil
	}
	blockHash := string(getBlockHash(in))
	if _, ok := s.Blockchain.blocks[blockHash]; ok {
		fmt.Printf("Already have block %v", blockHash)
		return &reply, nil
	}
	// If the block number the next block we were looking for update it
	if int(in.Header.Height) == s.Blockchain.nextBlockNum {
		s.Blockchain.nextBlockNum = int(in.Header.Height) + 1
	}
	fmt.Println("Received valid new block adding to local chain\n",
		getBlockString(in))
	// Index all the transactions in this block
	for i := range in.Transactions {
		s.Blockchain.txIndex[string(getTransactionHash(in.Transactions[i]))] = TxIndex{blockHash: blockHash,
			index: i}
	}
	s.Blockchain.blocks[blockHash] = in
	s.Blockchain.tipsOfChains[0] = in
	// Forward this new block along
	for _, myPeer := range s.peerList {
		if senderIP == "" || myPeer.peerIP == senderIP {
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

func (s *Server) GetAddress(ctx context.Context, in *pb.Empty) (*pb.AccountCreated, error) {
	var account pb.AccountCreated
	addr := strings.Join([]string{s.Wallet.key.X.String(), s.Wallet.key.Y.String()}, "")
	fmt.Println(addr)
	account.Address = addr
	return &account, nil
}

func (s *Server) GetBlocks(in *pb.Empty, stream pb.State_GetBlocksServer) error {
	fmt.Println("Get blocks ", len(s.Blockchain.blocks))
	// Walk the mempool
	// This is the only slow part, is building a sorted list
	orderedBlocks := make([]*pb.Block, len(s.Blockchain.blocks))
	for _, block := range s.Blockchain.blocks {
		orderedBlocks[block.Header.Height-1] = block
	}
	for _, block := range orderedBlocks {
		if block != nil {
			fmt.Println("Sending: ", getBlockString(block))
			stream.Send(block)
		}
	}
	return nil
}

func (s *Server) Connect(ctx context.Context, in *pb.Hello) (*pb.Ack, error) {
	var reply pb.Ack
	fmt.Println("Peer connect")
	return &reply, nil
}

func (s *Server) NewAccount(ctx context.Context, in *pb.Account) (*pb.AccountCreated, error) {
	var reply pb.AccountCreated
	fmt.Println("New Account for: ", in.Name)
	err := s.Wallet.createKey()
	if err != nil {
		return &reply, errors.New("Unknown error creating account")
	}
	addr := strings.Join([]string{s.Wallet.key.X.String(), s.Wallet.key.Y.String()}, "")
	fmt.Println(addr)
	reply.Address = addr
	return &reply, nil
}

func (s *Server) GetBalance(ctx context.Context, in *pb.Empty) (*pb.Balance, error) {
	var balance pb.Balance
	balance.Balance = s.Blockchain.getBalance(&s.Wallet.key.PublicKey)
	return &balance, nil
}

func (s Server) tryToConnectToPeers(nodeList []string) {
	for _, node := range nodeList {
		if _, ok := s.peerList[node]; ok {
			continue
		}
		conn, err := grpc.Dial(strings.Join([]string{node, ":", PORT}, ""), grpc.WithInsecure())
		if err != nil {
			fmt.Printf("Failed to connect to gRPC server: %v", err)
		} else {
			client := pb.NewPeeringClient(conn)
			ctx, _ := context.WithTimeout(context.Background(), 500*time.Millisecond)
			_, err = client.Connect(ctx, &pb.Hello{})
			if err == nil {
				// Save that connection, will send new transactions to peers to flood the network
				fmt.Printf("New peer %v!\n", node)
				outgoingIP, _ := s.getOutgoingIP(node)
				s.peerList[node] = BlockchainPeer{conn: conn, peerIP: node, sourceIP: outgoingIP}
			}
		}
	}
	fmt.Println("My peer list: ")
	for _, myPeer := range s.peerList {
		fmt.Printf("Peer %v outgoing interface %v\n", myPeer.peerIP, myPeer.sourceIP)
	}
}

// Always look for new peers in a separate goroutine
// polling at regular intervals
func (s Server) connectToPeers(nodeList []string) {
	ticker := time.NewTicker(PEER_CHECK * time.Millisecond)
	go func() {
		for _ = range ticker.C {
			s.tryToConnectToPeers(nodeList)
		}
	}()
}

func (s *Server) setIPs() {
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
					s.ips = append(s.ips, *v)
				}
			}
		}
	}
}

func removeOurIPs(ourIPs []net.IPNet, otherIPs []string) []string {
	var result []string
	for i := range otherIPs {
		ours := false
		for j := range ourIPs {
			if ourIPs[j].IP.String() == otherIPs[i] {
				ours = true
			}
		}
		if !ours {
			result = append(result, otherIPs[i])
		}
	}
	return result
}

func initServer() *Server {
	// Don't need to initialize the wallet
	var server Server = Server{
		ips:      make([]net.IPNet, 0),
		peerList: make(map[string]BlockchainPeer),
		MemPool:  MemPool{transactions: make(map[string]*pb.Transaction)},
		Blockchain: Blockchain{blocks: make(map[string]*pb.Block),
			txIndex:      make(map[string]TxIndex),
			tipsOfChains: make([]*pb.Block, 0),
			nextBlockNum: 1},
		stopMining: make(chan struct{})}
	server.setIPs()
	target, err := hex.DecodeString(strings.Join([]string{"00", strings.Repeat("f", 18)}, ""))
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("target:", target)
	server.Blockchain.setTarget(target)
	server.Blockchain.addGenesisBlock()
	return &server
}

func getNodeList() ([]string, error) {
	file, err := os.Open("networks.txt")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func main() {
	fmt.Println("Listening")
	// Depends on the IPs of your network
	nodeList, err := getNodeList()
	if err != nil {
		fmt.Println("Error getting nodes ", err)
		return
	}
	server := initServer()
	nodeList = removeOurIPs(server.ips, nodeList)
	server.connectToPeers(nodeList)
	startServer(server, PORT)
}
