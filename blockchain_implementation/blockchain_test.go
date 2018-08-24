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
    target, _ := hex.DecodeString(strings.Join([]string{"2", strings.Repeat("f", 19)}, ""))
    s.Blockchain.setTarget(target)
    mineBlocks(s, t, 2)
    // Fake receiver
    curve := elliptic.P256()
    receiverKey := new(ecdsa.PrivateKey)
    // Generate the keypair based on the curve
    receiverKey, _ = ecdsa.GenerateKey(curve, rand.Reader)
    before := s.Blockchain.getBalance(&s.Wallet.key.PublicKey)
    req := pb.TransactionRequest{Value: BLOCK_REWARD, ReceiverPubKey: getPubKeyBytes(receiverKey)}
    // Should succeed because we have money
    _, err := s.SendTransaction(context.Background(), &req)
    if err != nil {
        t.Fail()
    }
    after := s.Blockchain.getBalance(&s.Wallet.key.PublicKey)
    // Should be no immediate change to balance until mined
    t.Log(before, after) 
    if before != after {
        t.Fail()
    }
    // Mine the block
    mineBlocks(s, t, s.Blockchain.nextBlockNum)
    // Confirm transaction is now in the blockchain
    // Means that our balance should be 10 less than the number
    // of blocks (excluding the genesis block)
    // Remember as we mine we get block rewards as well
    numBlocks := len(s.Blockchain.blocks)
    balance := s.Blockchain.getBalance(&s.Wallet.key.PublicKey)
    if int(balance) != (BLOCK_REWARD*(numBlocks - 1) - BLOCK_REWARD) {
        t.Logf("Balance is %d should be %d", 
                balance, (BLOCK_REWARD*(numBlocks - 1) - BLOCK_REWARD))
        t.Fail()
    }
}

// Send a partial amount of money
// so a UTXO must be partially spent with the remaining change 
// created a new transaction back to ourselves
func TestMakeChange(t *testing.T) {
    s := initServer()
    s.Wallet.createKey() 
    // Relax difficulty for this
    target, _ := hex.DecodeString(strings.Join([]string{"2", strings.Repeat("f", 19)}, ""))
    s.Blockchain.setTarget(target)
    mineBlocks(s, t, 2)  // mine at least 1 block
    // Fake receiver
    curve := elliptic.P256()
    receiverKey := new(ecdsa.PrivateKey)
    // Generate the keypair based on the curve
    receiverKey, _ = ecdsa.GenerateKey(curve, rand.Reader)
    req := pb.TransactionRequest{Value: 8, ReceiverPubKey: getPubKeyBytes(receiverKey)}
    // Should succeed because we have money
    _, err := s.SendTransaction(context.Background(), &req)
    if err != nil {
        t.Fail()
    }
    mineBlocks(s, t, s.Blockchain.nextBlockNum) // 
    desiredBalance := BLOCK_REWARD*(len(s.Blockchain.blocks) - 1) - 8
    after := s.Blockchain.getBalance(&s.Wallet.key.PublicKey) 
    if desiredBalance != int(after) {
        t.Logf("Make change failed, balance is %d should be %d", after, desiredBalance)
        t.Fail()
    }
    // Receiver should have exactly 8
    recv := s.Blockchain.getBalance(&receiverKey.PublicKey)
    if recv != uint64(8) {
        t.Logf("Make change failed, receiver balance is %d should be %d", recv, 8)
        t.Fail()
    }
    // Send another random amount
    req = pb.TransactionRequest{Value: 4, ReceiverPubKey: getPubKeyBytes(receiverKey)}
    // Should succeed because we have money
    _, err = s.SendTransaction(context.Background(), &req)
    if err != nil {
        t.Fail()
    }
    mineBlocks(s, t, s.Blockchain.nextBlockNum)
    desiredBalance = BLOCK_REWARD*(len(s.Blockchain.blocks) - 1) - 12
    after = s.Blockchain.getBalance(&s.Wallet.key.PublicKey) 
    if desiredBalance != int(after) {
        t.Logf("Make change failed, balance is %d should be %d", after, desiredBalance)
        t.Fail()
    }
    // Receiver should have exactly 12
    recv = s.Blockchain.getBalance(&receiverKey.PublicKey)
    if recv != uint64(12) {
        t.Logf("Make change failed, receiver balance is %d should be %d", recv, 12)
        t.Fail()
    }
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
