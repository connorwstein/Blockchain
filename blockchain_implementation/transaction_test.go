package main

import (
//     "testing"
//     pb "./protos"
//     "golang.org/x/net/context"
//     "crypto/ecdsa"
//     "crypto/rand"
)
// 
// 
// func TestVerifyTransaction(t *testing.T) {
//     initBlockChain()
//     // As if we had mined some coin earlier
//     key := createKey()
//     // Create a signed transaction then ensure that it verifies correctly 
//     req := pb.Transaction{ReceiverPubKey: getPubKeyBytes(key), 
//                           Value: 100}
//     r, s, _ := ecdsa.Sign(rand.Reader, key, getTransactionHash(&req))
//     // Returns two big ints
//     req.Signature = getSignatureBytes(r, s)
//     if !verifyTransaction(&req) {
//         t.Fail()
//     }
// }
// 
// func TestReceive(t *testing.T) {
//     s := server{}
//     initBlockChain()
//     // As if we had mined some coin earlier
//     key := createKey()
//     setKey(key)
//     testVal := uint64(100)
//     req := pb.Transaction{ReceiverPubKey: getPubKeyBytes(key), 
//                           Value: testVal}
//     _, err := s.ReceiveTransaction(context.Background(), &req)
//     if err != nil {
//         t.Errorf("HelloTest(%v) got unexpected error")
//     }
//     // Check that transaction is now in the memPool
//     transHash := string(getTransactionHash(&req))
//     if _, ok := memPool[transHash]; !ok {
//         t.Log("Did not find transaction in mempool")
//         t.Fail()
//     }
//     // Make sure value is correct
//     if memPool[transHash].Value != testVal {
//         t.Fail()
//     }
//     // Note balance should still be zero because no block has been mined
//     if getBalance() != 0 {
//         t.Fail()
//     }
// }
// 
// func TestSend(t *testing.T) {
//     s := server{}
//     key := createKey()
//     setKey(key)
//     req := pb.Transaction{Value: 100}
//     // Should fail because we have no money
//     _, err := s.SendTransaction(context.Background(), &req)
//     if err == nil {
//         t.Errorf("Send test should have failed, no UTXO can cover that transaction", err)
//     }
// }
