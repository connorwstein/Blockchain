package main

import (
    "fmt"
//     "os"
    pb "../protos"
    "google.golang.org/grpc"
//     "strings"
    "golang.org/x/net/context"
)

func main() {
    conn, err := grpc.Dial("172.24.0.2:4000", grpc.WithInsecure())
    if err != nil {
        fmt.Printf("Failed to connect to gRPC server: %v", err)
    }
    defer conn.Close()

    c := pb.NewTransactionsClient(conn)
    _, err = c.SendTransaction(context.Background(), &pb.Transaction{Transaction: "test"})
    if err != nil {
        fmt.Printf("Unable to get state: %v", err)
    }
}
