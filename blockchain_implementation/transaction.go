package main

import (
    "fmt"
    "crypto/sha256"
//     "crypto/ecdsa"
    "encoding/binary"
    "crypto/elliptic"
//     "math/big"
    pb "./protos"
)

// Check
// 1. Signature came from the private key associated with the public key of the sender
// 2. The referenced UTXO exists and is not already spent
// 3. Sum of the input UTXO is larger than the output
func Verify(transaction pb.Transaction, curve elliptic.Curve) bool {
    // Check that the signature came from the private key associated with the public key of the sender
    // Signature is simply R and S (both 32 byte numbers) concatentated
    // Convert to big ints
    // Pub key will be concatenation of X and Y big ints converted to bytes 
//     pubKey := ecdsa.PublicKey{Curve: curve}
//     pubKey.X = new(big.Int)
//     pubKey.Y = new(big.Int)
//     pubKey.X.SetBytes(transaction.SenderPubKey[:32])
//     pubKey.Y.SetBytes(transaction.SenderPubKey[32:])
//     r := new(big.Int)
//     r.SetBytes(transaction.Signature[:32])
//     s := new(big.Int)
//     s.SetBytes(transaction.Signature[32:])
//     fmt.Println(r, s)
//     verifystatus := ecdsa.Verify(&pubKey, GetHash(transaction), r, s)
//     return verifystatus 
    return true
}

func TransactionToString(transaction pb.Transaction) string {
    return fmt.Sprintf("%v --> %v $%v", transaction.InputUTXO, transaction.ReceiverPubKey, transaction.Value)
}

func GetHash(transaction pb.Transaction) []byte {
    // SHA hash is 32 bytes
    // TODO: use a writer here
	toHash := make([]byte, 0)
	toHash = append(toHash, []byte(transaction.InputUTXO)...)
	toHash = append(toHash, []byte(transaction.ReceiverPubKey)...)
    value := make([]byte, 8)
	binary.LittleEndian.PutUint64(value, transaction.Value)
	toHash = append(toHash, value...)
	sum := sha256.Sum256(toHash)
    return sum[:]
}
