package main

import (
	pb "../protos"
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"io"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"
)

// TODO: use interactive cli library
// so we can reuse the connection

func connect() *grpc.ClientConn {
	conn, err := grpc.Dial("localhost:8333", grpc.WithInsecure())
	if err != nil {
		fmt.Printf("Failed to connect to gRPC server: %v", err)
	}
	return conn
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

func getTransactionHash(transaction *pb.Transaction) []byte {
	buf := new(bytes.Buffer)
	for _, inputUTXO := range transaction.Vin {
		buf.Write(inputUTXO.TxID)
		binary.Write(buf, binary.LittleEndian, inputUTXO.Index)
	}
	for _, outputTX := range transaction.Vout {
		buf.Write(outputTX.ReceiverPubKey)
		binary.Write(buf, binary.LittleEndian, outputTX.Value)
	}
	// Height needed to make coinbase transactions unique
	sum := sha256.Sum256(buf.Bytes())
	return sum[:]
}

func getTXIString(tx *pb.TXI) string {
	var buf bytes.Buffer
	buf.WriteString("\n  TxID:")
	buf.WriteString(hex.EncodeToString(tx.TxID[:]))
	buf.WriteString("\n  Index:")
	buf.WriteString(strconv.Itoa(int(tx.Index)))
	return buf.String()
}

func getTXOString(tx *pb.TXO) string {
	var buf bytes.Buffer
	buf.WriteString("\n  Receiver:")
	pubKey := ecdsa.PublicKey{Curve: elliptic.P256()}
	pubKey.X = new(big.Int)
	pubKey.Y = new(big.Int)
	pubKey.X.SetBytes(tx.ReceiverPubKey[:32])
	pubKey.Y.SetBytes(tx.ReceiverPubKey[32:])
	buf.WriteString(strings.Join([]string{pubKey.X.String(), pubKey.Y.String()}, ""))
	buf.WriteString("\n  Amount:")
	buf.WriteString(strconv.Itoa(int(tx.Value)))
	return buf.String()
}

func getTransactionString(transaction *pb.Transaction) string {
	var buf bytes.Buffer
	buf.WriteString("\nTransaction Hash: ")
	buf.WriteString(hex.EncodeToString(getTransactionHash(transaction)[:]))
	buf.WriteString("\nVin: ")
	if len(transaction.Vin) == 0 {
		buf.WriteString("Miner reward")
	}
	for _, inputUTXO := range transaction.Vin {
		buf.WriteString("\n TX: ")
		buf.WriteString(getTXIString(inputUTXO))
	}
	buf.WriteString("\nVout: ")
	for _, outputTX := range transaction.Vout {
		buf.WriteString("\n TX: ")
		buf.WriteString(getTXOString(outputTX))
	}
	return buf.String()
}

func getTransactions() {
	conn := connect()
	c := pb.NewStateClient(conn)
	stream, err := c.GetTransactions(context.Background(), &pb.Empty{})
	if err != nil {
		fmt.Printf("Unable to get state: %v", err)
	}
	for {
		feature, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println(getTransactionString(feature))
		//         fmt.Println(feature)
	}
	conn.Close()
}

func getBlocks() {
	conn := connect()
	defer conn.Close()
	c := pb.NewStateClient(conn)
	stream, err := c.GetBlocks(context.Background(), &pb.Empty{})
	if err != nil {
		fmt.Printf("Unable to get state: %v", err)
	}
	for {
		block, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(getBlockString(block))
	}
}

// Destination should be the pubkey of someone else
// The daemon performs the wallet functionality i.e.
// determines which utxo you reference.
func send(amount int, destination string) {
	conn := connect()
	c := pb.NewTransactionsClient(conn)
	fmt.Println(strconv.Itoa(amount))
	var trans pb.TransactionRequest
	trans.Value = uint64(amount)
	// Destination string is two 32 byte integers concatenated
	x := new(big.Int)
	y := new(big.Int)
	x.SetString(destination[:77], 10)
	y.SetString(destination[77:], 10)
	var recv []byte
	recv = append(recv, x.Bytes()...)
	recv = append(recv, y.Bytes()...)
	trans.ReceiverPubKey = recv
	_, err := c.SendTransaction(context.Background(), &trans)
	if err != nil {
		fmt.Println("Error sending transaction", err)
	}
	conn.Close()
}

func newAccount(name string) {
	// Need to make a new key pair associated with this account
	conn := connect()
	c := pb.NewWalletClient(conn)
	addr, err := c.NewAccount(context.Background(), &pb.Account{Name: name})
	if err != nil {
		fmt.Println("Error sending transaction", err)
	}
	fmt.Println(addr.Address)
	conn.Close()
}

func startMining() {
	conn := connect()
	c := pb.NewMinerClient(conn)
	_, err := c.StartMining(context.Background(), &pb.Empty{})
	if err != nil {
		fmt.Println("Error sending transaction", err)
	}
	conn.Close()
}

func stopMining() {
	conn := connect()
	c := pb.NewMinerClient(conn)
	_, err := c.StopMining(context.Background(), &pb.Empty{})
	if err != nil {
		fmt.Println("Error sending transaction", err)
	}
	conn.Close()
}

func getBalance() {
	// Need to make a new key pair associated with this account
	conn := connect()
	c := pb.NewWalletClient(conn)
	balance, err := c.GetBalance(context.Background(), &pb.Empty{})
	if err != nil {
		fmt.Println("Error sending transaction", err)
	}
	fmt.Printf("%d\n", balance.Balance)
	conn.Close()
}

func getAddress() {
	// Need to make a new key pair associated with this account
	conn := connect()
	c := pb.NewWalletClient(conn)
	address, err := c.GetAddress(context.Background(), &pb.Empty{})
	if err != nil {
		fmt.Println("Error sending transaction", err)
	}
	fmt.Printf("%s\n", address.Address)
	conn.Close()
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("State or send subcommand is required")
		os.Exit(1)
	}
	stateCommand := flag.NewFlagSet("state", flag.ExitOnError)
	sendCommand := flag.NewFlagSet("send", flag.ExitOnError)
	newCommand := flag.NewFlagSet("new", flag.ExitOnError)
	walletCommand := flag.NewFlagSet("wallet", flag.ExitOnError)
	mineCommand := flag.NewFlagSet("mine", flag.ExitOnError)

	getOp := stateCommand.String("get", "", "what you want to get")
	sendAmount := sendCommand.Int("amount", 0, "how much to send")
	sendDest := sendCommand.String("dest", "", "where to send")
	newName := newCommand.String("name", "", "name of account")
	walletGet := walletCommand.String("get", "", "get balance, pubkey etc.")
	mineAction := mineCommand.String("action", "", "start/stop mining")

	switch os.Args[1] {
	case "state":
		stateCommand.Parse(os.Args[2:])
		fmt.Printf("get state of %v\n", *getOp)
		switch *getOp {
		case "transactions":
			getTransactions()
		case "blocks":
			getBlocks()
		default:
			fmt.Println("Unknown get op")
		}
	case "send":
		sendCommand.Parse(os.Args[2:])
		fmt.Printf("send %v to %v\n", *sendAmount, *sendDest)
		send(*sendAmount, *sendDest)
	case "new":
		// Create a new key pair
		newCommand.Parse(os.Args[2:])
		fmt.Println("New account:", *newName)
		newAccount(*newName)
	case "wallet":
		walletCommand.Parse(os.Args[2:])
		switch *walletGet {
		case "balance":
			getBalance()
		case "address":
			getAddress()
		default:
			fmt.Println("Unknown get op")
		}
	case "mine":
		mineCommand.Parse(os.Args[2:])
		switch *mineAction {
		case "start":
			startMining()
		case "stop":
			stopMining()
		default:
			fmt.Println("Unknown mine action")
		}
	default:
		flag.PrintDefaults()
		os.Exit(1)
	}
}
