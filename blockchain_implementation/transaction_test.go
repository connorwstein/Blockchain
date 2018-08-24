package main

import (
    "testing"
    pb "./protos"
    "encoding/hex"
    "strings"
    "golang.org/x/net/context"
    "crypto/ecdsa"
    "crypto/rand"
)

func TestVerifyTransaction(t *testing.T) {
    s := initServer()
    s.Wallet.createKey() 
    // As if we had mined some coin earlier
    // Create a signed transaction then ensure that it verifies correctly 
    mint := pb.TXO{ReceiverPubKey: getPubKeyBytes(s.Wallet.key), Value: BLOCK_REWARD}
    var txos []*pb.TXO
    txos = append(txos, &mint)
    trans := pb.Transaction{Vout: txos}
    rInt, sInt, _ := ecdsa.Sign(rand.Reader, s.Wallet.key, getTransactionHash(&trans))
    // Returns two big ints
    trans.Signature = getSignatureBytes(rInt, sInt)
    if !s.Blockchain.verifyTransaction(&trans) {
        t.Fail()
    }
    // Mine a block so we have some money
    target, _ := hex.DecodeString(strings.Join([]string{"2", strings.Repeat("f", 19)}, ""))
    s.Blockchain.setTarget(target)
    mineBlocks(s, t, 2)
    balance := s.Blockchain.getBalance(s.Wallet.key)
    inputUTXO := s.Blockchain.getUTXOsToCoverTransaction(s.Wallet.key, BLOCK_REWARD)
    t.Log(balance)
    t.Log(s.Blockchain.txIndex)
    // Spend a valid amount
    var vout []*pb.TXO
    var vin []*pb.TXI
    var txi pb.TXI
    var txo pb.TXO
    txi.TxID = getTransactionHash(inputUTXO.transaction)
    txi.Index = uint64(inputUTXO.index)
    // Just send all of it back to our selves for simplicity
    txo.ReceiverPubKey = getPubKeyBytes(s.Wallet.key) 
    txo.Value = BLOCK_REWARD 
    vin = append(vin, &txi)
    vout = append(vout, &txo)
    var spend pb.Transaction
    spend.Vin = vin
    spend.Vout = vout
    rInt, sInt, _ = ecdsa.Sign(rand.Reader, s.Wallet.key, getTransactionHash(&spend))
    // Returns two big ints
    spend.Signature = getSignatureBytes(rInt, sInt)
    _, err := s.ReceiveTransaction(context.Background(), &spend)
    // Should acccept this transaction
    if err != nil  {
        t.Log("Should have accepted")
        t.Fail()
    }
    // Should not accept
    spend.Vout[0].Value = uint64(2)*balance
    _, err = s.ReceiveTransaction(context.Background(), &spend)
    if err == nil  {
        t.Log("Should not have accepted")
        t.Fail()
    }
}

// func TestReceive(t *testing.T) {
//     s := initServer()
//     s.Wallet.createKey() 
//     testVal := uint64(100)
//     req := pb.TransactionRequest{ReceiverPubKey: getPubKeyBytes(s.Wallet.key), 
//                           Value: testVal}
//     _, err := s.ReceiveTransaction(context.Background(), &req)
//     if err != nil {
//         t.Errorf("HelloTest(%v) got unexpected error")
//     }
//     // Check that transaction is now in the memPool
//     transHash := string(getTransactionHash(&req))
//     if _, ok := s.MemPool.transactions[transHash]; !ok {
//         t.Log("Did not find transaction in mempool")
//         t.Fail()
//     }
//     // Make sure value is correct
//     if s.MemPool.transactions[transHash].Value != testVal {
//         t.Fail()
//     }
//     // Note balance should still be zero because no block has been mined
//     if s.Blockchain.getBalance(s.Wallet.key) != 0 {
//         t.Fail()
//     }
// }
// 
// // Reject a faulty transaction where someone claims to 
// // send more than they actually have
// func TestReceiveReject(t *testing.T) {
// }
// 
// func TestSend(t *testing.T) {
//     s := initServer()
//     s.Wallet.createKey() 
//     req := pb.TransactionRequest{Value: 100}
//     // Should fail because we have no money
//     _, err := s.SendTransaction(context.Background(), &req)
//     if err == nil {
//         t.Errorf("Send test should have failed, no UTXO can cover that transaction", err)
//     }
// }
