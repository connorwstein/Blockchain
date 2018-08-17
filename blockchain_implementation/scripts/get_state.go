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

func send(amount int, destination string) {
    conn := connect()
    c := pb.NewTransactionsClient(conn)
    fmt.Println(strconv.Itoa(amount))
    _, err := c.SendTransaction(context.Background(), &pb.Transaction{
                                Value: uint64(amount)})
    if err != nil {
        fmt.Println("Error sending transaction")
    }
    conn.Close()
}

func main() {
    if len(os.Args) < 2 {
        fmt.Println("State or send subcommand is required")
        os.Exit(1)
    }
    stateCommand := flag.NewFlagSet("state", flag.ExitOnError)
    sendCommand := flag.NewFlagSet("send", flag.ExitOnError)
    getOp := stateCommand.String("get", "", "what you want to get")
    sendAmount := sendCommand.Int("amount", 0, "how much to send")
    sendDest := sendCommand.String("dest", "", "where to send")
    switch os.Args[1] {
        case "state":
            stateCommand.Parse(os.Args[2:])
            fmt.Printf("get state of %v\n", *getOp)
            getTransactions()
        case "send":
            sendCommand.Parse(os.Args[2:])
            fmt.Printf("send %v to %v\n", *sendAmount, *sendDest)
            send(*sendAmount, *sendDest)
        default:
            flag.PrintDefaults()
            os.Exit(1)
    }
}
