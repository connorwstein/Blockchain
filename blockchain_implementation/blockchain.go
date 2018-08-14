package main

import (
    "crypto/sha256"
    "fmt"
	"bytes"
	"encoding/binary"
    "golang.org/x/net/context"
    "google.golang.org/grpc"
    "strings"
    "net"
    pb "./protos"
)

type server struct{}

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
    lis, err := net.Listen("tcp", strings.Join([]string{":", "4000"}, ""))
    if err != nil {
        fmt.Printf("gRPC server failed to start listening: %v", err)
    }
    s := grpc.NewServer()
    pb.RegisterTransactionsServer(s, &server{})
//     // Register reflection service on gRPC server.
//     reflection.Register(s)
    if err := s.Serve(lis); err != nil {
        fmt.Printf("gRPC server failed to start serving: %v", err)
    }
}
func (s *server) ReceiveTransaction(ctx context.Context, in *pb.Transaction) (*pb.Empty, error) {
    var reply pb.Empty
    return &reply, nil
}

func (s *server) SendTransaction(ctx context.Context, in *pb.Transaction) (*pb.Empty, error) {
    var reply pb.Empty
    return &reply, nil
}

func main() {
    startGrpc() 
}
