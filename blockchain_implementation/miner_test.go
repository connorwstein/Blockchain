package main

import (
    "testing"
    pb "./protos"
    "fmt"
    "golang.org/x/net/context"
//     "crypto/ecdsa"
//     "crypto/rand"
    "time"
    "errors"
    "strings"
    "encoding/hex"
)

var nodeList []string = []string{"172.27.0.2", "172.27.0.3", "172.26.0.2", 
                         "172.26.0.4", "172.25.0.2", "172.25.0.3", 
                         "172.24.0.2", "172.24.0.3"}

// Returns error if the block was not mined
// in time, note this depends on the difficulty
// and power of the machine
func mineBlockHelper(s *Server, minChainLength int) error {
    timeout := time.After(30 * time.Second) 
    ticker := time.NewTicker(10 * time.Millisecond)
    defer ticker.Stop()
    // ticker.C is a channel of Time instants
    for mined := false; !mined; {
        select {
            case <- timeout:
                // If our timeout Time instant appears on that channel
                // it is time to end
                return errors.New("Failed to mine block in time")
            case <- ticker.C:
                // Check if we have mined a block
                // if so we are done
                if len(s.Blockchain.blocks) >= minChainLength {
                    mined = true
                } else {
                    fmt.Println("blocks ", len(s.Blockchain.blocks))
                }
        }
    }
    return nil
}

// Mine at least minChainLength blocks, fail the
// test if anything bad happens
func mineBlocks(s *Server, t *testing.T, minChainLength int) {
    _, err := s.StartMining(context.Background(), &pb.Empty{})
    if err != nil {
        t.Fail()
    }
    // 2 = genesis block as well as our new block
    if err = mineBlockHelper(s, minChainLength); err != nil {
        t.Log(err)
        t.Fail()
    }
    _, err = s.StopMining(context.Background(), &pb.Empty{})
    if err != nil {
        t.Fail()
    }
}

// Check balance updates upon mining
func TestMineBlock(t *testing.T) {
    s := initServer(nodeList)
    s.Wallet.createKey() 
    // Relax difficulty for this
    target, _ := hex.DecodeString(strings.Join([]string{"e", strings.Repeat("f", 19)}, ""))
    s.Blockchain.setTarget(target)
    mineBlocks(s, t, 3)
    balance := int(s.getBalance(s.Wallet.key))
    numBlocks := len(s.Blockchain.blocks)
    if balance != (numBlocks - 1)*BLOCK_REWARD {
        t.Logf("Balance is %d, should be %d", balance, (numBlocks - 1)*BLOCK_REWARD)
        t.Fail()
    }
}

// Mine a block to fill the balance with some coin
// then send a transaction, mine another block to validate
// that transaction and ensure the balance is correct after that
func TestBalanceDecrement(t *testing.T) {
    s := initServer(nodeList)
    s.Wallet.createKey() 
    // Relax difficulty for this
    target, _ := hex.DecodeString(strings.Join([]string{"e", strings.Repeat("f", 19)}, ""))
    s.Blockchain.setTarget(target)
    mineBlocks(s, t, 2)
    t.Log(s.Blockchain.getBalance(s.Wallet.key))
}
