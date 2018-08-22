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

func mineBlockHelper(minChainLength int) error {
    // Returns error if the block was not mined
    // in time, note this depends on the difficulty
    // and power of the machine
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
                if len(blockChain.blocks) >= minChainLength {
                    mined = true
                } else {
                    fmt.Println("blocks ", len(blockChain.blocks))
                }
        }
    }
    return nil
}

func mineBlocks(s server, t *testing.T, minChainLength int) {
    // Mine a block, ensure our balance is updated
    _, err := s.StartMining(context.Background(), &pb.Empty{})
    if err != nil {
        t.Fail()
    }
    // 2 = genesis block as well as our new block
    if err = mineBlockHelper(minChainLength); err != nil {
        t.Log(err)
        t.Fail()
    }
    _, err = s.StopMining(context.Background(), &pb.Empty{})
    if err != nil {
        t.Fail()
    }
}

func TestMineBlock(t *testing.T) {
    initBlockChain()
    key := createKey()
    setKey(key)
    // Relax difficulty for this
    target, _ := hex.DecodeString(strings.Join([]string{"e", strings.Repeat("f", 19)}, ""))
    blockChain.setTarget(target)
    s := server{}
    mineBlocks(s, t, 3)
    if int(getBalance()) != (len(blockChain.blocks) - 1)*BLOCK_REWARD {
        t.Logf("Balance is %d, should be %d", getBalance(), (len(blockChain.blocks) - 1)*BLOCK_REWARD)
        t.Fail()
    }
}

// Mine a block to fill the balance with some coin
// then send a transaction, mine another block to validate
// that transaction and ensure the balance is correct after that
func TestBalanceDecrement(t *testing.T) {
    initBlockChain()
    key := createKey()
    setKey(key)
    // Relax difficulty for this
    target, _ := hex.DecodeString(strings.Join([]string{"e", strings.Repeat("f", 19)}, ""))
    blockChain.setTarget(target)
    s := server{}
    mineBlocks(s, t, 2)
    t.Log(getBalance())
}
