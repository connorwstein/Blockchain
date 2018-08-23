package main

import (
    "testing"
    pb "./protos"
//     "fmt"
    "golang.org/x/net/context"
    "crypto/ecdsa"
    "crypto/rand"
    "crypto/elliptic"
//     "time"
//     "errors"
    "strings"
    "encoding/hex"
)

// Mine a block to fill the balance with some coin
// then send a transaction, mine another block to validate
// that transaction and ensure the balance is correct after that
func TestBalanceDecrement(t *testing.T) {
    s := initServer()
    s.Wallet.createKey() 
    // Relax difficulty for this
    target, _ := hex.DecodeString(strings.Join([]string{"e", strings.Repeat("f", 19)}, ""))
    s.Blockchain.setTarget(target)
    mineBlocks(s, t, 2)
    // Fake receiver
    curve := elliptic.P256()
    receiverKey := new(ecdsa.PrivateKey)
    // Generate the keypair based on the curve
    receiverKey, _ = ecdsa.GenerateKey(curve, rand.Reader)
    before := s.Blockchain.getBalance(s.Wallet.key)
    req := pb.Transaction{Value: 10, ReceiverPubKey: getPubKeyBytes(receiverKey)}
    // Should succeed because we have money
    _, err := s.SendTransaction(context.Background(), &req)
    if err != nil {
        t.Fail()
    }
    after := s.Blockchain.getBalance(s.Wallet.key)
    // Should be no immediate change to balance until mined
    t.Log(before, after) 
    if before != after {
        t.Fail()
    }
    // Mine the block
    mineBlocks(s, t, 4)
    // Confirm transaction is now in the blockchain
    // Means that our balance should be 10 less than the number
    // of blocks (excluding the genesis block)
    // Remember as we mine we get block rewards as well
    numBlocks := len(s.Blockchain.blocks)
    balance := s.Blockchain.getBalance(s.Wallet.key)
    if int(balance) != (BLOCK_REWARD*(numBlocks - 1) - 10) {
        t.Logf("Balance is %d should be %d", 
                balance, (BLOCK_REWARD*(numBlocks - 1) - 10))
        t.Fail()
    }
}

// Send a partial amount of money
// so a UTXO must be partially spent with the remaining change 
// created a new transaction back to ourselves
func TestMakeChange(t *testing.T) {
//     s := initServer()
//     s.Wallet.createKey() 
//     // Relax difficulty for this
//     target, _ := hex.DecodeString(strings.Join([]string{"e", strings.Repeat("f", 19)}, ""))
//     s.Blockchain.setTarget(target)
//     mineBlocks(s, t, 2)
//     // Fake receiver
//     curve := elliptic.P256()
//     receiverKey := new(ecdsa.PrivateKey)
//     // Generate the keypair based on the curve
//     receiverKey, _ = ecdsa.GenerateKey(curve, rand.Reader)
//     before := s.Blockchain.getBalance(s.Wallet.key)
//     req := pb.Transaction{Value: 8, ReceiverPubKey: getPubKeyBytes(receiverKey)}
//     // Should succeed because we have money
//     _, err := s.SendTransaction(context.Background(), &req)
//     if err != nil {
//         t.Fail()
//     }
}

// How to test a fork situation:
// two blocks mined close to each other so some node receive
// v1 for block X and some other nodes receive v2 for block X.
// we need to store both, wait for an update then adjust accordingly
func TestTemporaryFork(t *testing.T) {
}

// Receive some block that doesn't have a parent, put it in the orphan pool
// until parent received, then add both parent and orphan to the chain and
// remove the orphan from the chain
func TestOrphanBlock(t *testing.T) {
}
