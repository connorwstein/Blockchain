package main

import (
    "testing"
    pb "./protos"
    "golang.org/x/net/context"
    "crypto/ecdsa"
    "crypto/elliptic"
    "crypto/rand"
)

// func TestTransactionVerify(t *testing.T) {
//     // Create a signed transaction then ensure that it verifies correctly 
//     trans := pb.Transaction{}
//     // Get a keypair
//     // Need the curve for our trapdoor
//     PubKeyCurve := elliptic.P256() 
//     // Allocate memory for a private key
//     privatekey := new(ecdsa.PrivateKey)
//     // Generate the keypair based on the curve
//     privatekey, _ = ecdsa.GenerateKey(PubKeyCurve, rand.Reader)
//     var pubkey ecdsa.PublicKey = privatekey.PublicKey
// 
//     t.Log("Private Key :", privatekey)
//     t.Log("Public Key :", pubkey)
//     trans.SenderPubKey = make([]byte, 0)
//     trans.SenderPubKey = append(trans.SenderPubKey, pubkey.X.Bytes()...)
//     trans.SenderPubKey = append(trans.SenderPubKey,  pubkey.Y.Bytes()...)
//     r, s, _ := ecdsa.Sign(rand.Reader, privatekey, GetHash(trans))
//     // Returns two big ints
//     trans.Signature = make([]byte, 0)
//     trans.Signature = append(trans.Signature, r.Bytes()...)
//     trans.Signature = append(trans.Signature, s.Bytes()...)
//     if !Verify(trans, PubKeyCurve) {
//         t.Fail()
//     }
// }

func TestReceive(t *testing.T) {
    s := server{}
    req := pb.Transaction{Value: 1000}
    _, err := s.ReceiveTransaction(context.Background(), &req)
    if err != nil {
        t.Errorf("HelloTest(%v) got unexpected error")
    }
    // TODO: Simulate receiving a bunch of transactions and accumulating them into a block 
}

func TestSend(t *testing.T) {
    s := server{}
    req := pb.Transaction{Value: 100}
    _, err := s.SendTransaction(context.Background(), &req)
    if err != nil {
        t.Errorf("HelloTest(%v) got unexpected error")
    }
}
