package main

import (
	pb "./protos"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"golang.org/x/net/context"
	"strings"
	"testing"
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
	balance := s.Blockchain.getBalance(&s.Wallet.key.PublicKey)
	inputUTXOs := s.Blockchain.getUTXOsToCoverTransaction(s.Wallet.key, BLOCK_REWARD)
	//     t.Log(balance)
	//     t.Log(s.Blockchain.txIndex)
	// Spend a valid amount
	var vout []*pb.TXO
	var vin []*pb.TXI
	var txi pb.TXI
	var txo pb.TXO
	txi.TxID = getTransactionHash(inputUTXOs[0].transaction)
	txi.Index = uint64(inputUTXOs[0].index)
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
	if err != nil {
		t.Log("Should have accepted")
		t.Fail()
	}
	// Should not accept
	spend.Vout[0].Value = uint64(2) * balance
	_, err = s.ReceiveTransaction(context.Background(), &spend)
	if err == nil {
		t.Log("Should not have accepted")
		t.Fail()
	}
}

func TestSend(t *testing.T) {
	s := initServer()
	s.Wallet.createKey()
	req := pb.TransactionRequest{Value: 100}
	// Should fail because we have no money
	_, err := s.SendTransaction(context.Background(), &req)
	if err == nil {
		t.Errorf("Send test should have failed, no UTXO can cover that transaction", err)
	}
}
