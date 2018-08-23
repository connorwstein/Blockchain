package main

import (
    "testing"
    pb "./protos"
    "golang.org/x/net/context"
    "crypto/ecdsa"
    "crypto/rand"
)

func TestVerifyTransaction(t *testing.T) {
    s := initServer()
    s.Wallet.createKey() 
    // As if we had mined some coin earlier
    // Create a signed transaction then ensure that it verifies correctly 
    req := pb.Transaction{ReceiverPubKey: getPubKeyBytes(s.Wallet.key), 
                          Value: 100}
    rInt, sInt, _ := ecdsa.Sign(rand.Reader, s.Wallet.key, getTransactionHash(&req))
    // Returns two big ints
    req.Signature = getSignatureBytes(rInt, sInt)
    if !verifyTransaction(&req) {
        t.Fail()
    }
}

func TestReceive(t *testing.T) {
    s := initServer()
    s.Wallet.createKey() 
    testVal := uint64(100)
    req := pb.Transaction{ReceiverPubKey: getPubKeyBytes(s.Wallet.key), 
                          Value: testVal}
    _, err := s.ReceiveTransaction(context.Background(), &req)
    if err != nil {
        t.Errorf("HelloTest(%v) got unexpected error")
    }
    // Check that transaction is now in the memPool
    transHash := string(getTransactionHash(&req))
    if _, ok := s.MemPool.transactions[transHash]; !ok {
        t.Log("Did not find transaction in mempool")
        t.Fail()
    }
    // Make sure value is correct
    if s.MemPool.transactions[transHash].Value != testVal {
        t.Fail()
    }
    // Note balance should still be zero because no block has been mined
    if s.Blockchain.getBalance(s.Wallet.key) != 0 {
        t.Fail()
    }
}

// Reject a faulty transaction where someone claims to 
// send more than they actually have
func TestReceiveReject(t *testing.T) {
}

func TestSend(t *testing.T) {
    s := initServer()
    s.Wallet.createKey() 
    req := pb.Transaction{Value: 100}
    // Should fail because we have no money
    _, err := s.SendTransaction(context.Background(), &req)
    if err == nil {
        t.Errorf("Send test should have failed, no UTXO can cover that transaction", err)
    }
}
