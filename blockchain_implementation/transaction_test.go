package main

import (
    "testing"
    pb "./protos"
    "golang.org/x/net/context"
    "crypto/ecdsa"
    "crypto/elliptic"
    "crypto/rand"
)

func TestTransactionVerify(t *testing.T) {
    // Create a signed transaction then ensure that it verifies correctly 
    trans := Transaction{}
    // Get a keypair
    // Need the curve for our trapdoor
    PubKeyCurve = elliptic.P256() 
    // Allocate memory for a private key
    privatekey := new(ecdsa.PrivateKey)
    // Generate the keypair based on the curve
    privatekey, _ = ecdsa.GenerateKey(PubKeyCurve, rand.Reader)
    var pubkey ecdsa.PublicKey
    pubkey = privatekey.PublicKey

    t.Log("Private Key :", privatekey)
    t.Log("Public Key :", pubkey)
    trans.sender = make([]byte, 0)
    trans.sender = append(trans.sender, pubkey.X.Bytes()...)
    trans.sender = append(trans.sender,  pubkey.Y.Bytes()...)
    r, s, _ := ecdsa.Sign(rand.Reader, privatekey, trans.GetHash())
    // Returns two big ints
    trans.signature = make([]byte, 0)
    trans.signature = append(trans.signature, r.Bytes()...)
    trans.signature = append(trans.signature, s.Bytes()...)
    if !trans.Verify() {
        t.Fail()
    }
}

func TestReceive(t *testing.T) {
    s := server{}
    req := pb.Transaction{Transaction: "helloworld"}
    _, err := s.ReceiveTransaction(context.Background(), &req)
    if err != nil {
        t.Errorf("HelloTest(%v) got unexpected error")
    }
    // TODO: Simulate receiving a bunch of transactions and accumulating them into a block 
}
