package main

import (
    "fmt"
    "os"
    "flag"
    pb "../protos"
    "google.golang.org/grpc"
//     "strings"
    "golang.org/x/net/context"
    "io"
    "strconv"
)

func connect() *grpc.ClientConn {
    conn, err := grpc.Dial("localhost:8333", grpc.WithInsecure())
    if err != nil {
        fmt.Printf("Failed to connect to gRPC server: %v", err)
    }
    return conn
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
            fmt.Printf("%v", feature)
        }
        fmt.Println(feature)
    }
    conn.Close()
}

func getBlocks() {
    conn := connect()
    c := pb.NewStateClient(conn)
    stream, err := c.GetBlocks(context.Background(), &pb.Empty{})
    if err != nil {
        fmt.Printf("Unable to get state: %v", err)
    }
    for {
        feature, err := stream.Recv()
        if err == io.EOF {
            break
        }
        if err != nil {
            fmt.Printf("%v", feature)
        }
        fmt.Println(feature)
    }
    conn.Close()
}

// Destination should be the pubkey of someone else
// The daemon performs the wallet functionality i.e. 
// determines which utxo you reference. 
func send(amount int, destination string) {
    conn := connect()
    c := pb.NewTransactionsClient(conn)
    fmt.Println(strconv.Itoa(amount))
    var trans pb.Transaction
    trans.Value = uint64(amount)
    trans.ReceiverPubKey = []byte(destination)
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
    _, err := c.NewAccount(context.Background(), &pb.Account{Name: name})
    if err != nil {
        fmt.Println("Error sending transaction", err)
    }
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

func getBalance() {
    // Need to make a new key pair associated with this account
    conn := connect()
    c := pb.NewWalletClient(conn)
    balance, err := c.GetBalance(context.Background(), &pb.Empty{})
    if err != nil {
        fmt.Println("Error sending transaction", err)
    }
    fmt.Printf("Balance %d\n", balance.Balance)
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
                default:
                    fmt.Println("Unknown get op")
            }
        case "mine":
            mineCommand.Parse(os.Args[2:])
            switch *mineAction {
                case "start":
                    startMining()
                default:
                    fmt.Println("Unknown mine action")
            }
        default:
            flag.PrintDefaults()
            os.Exit(1)
    }
}
