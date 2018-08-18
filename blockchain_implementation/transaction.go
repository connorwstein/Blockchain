package main

import (
    "fmt"
    "crypto/sha256"
//     "crypto/ecdsa"
    "encoding/binary"
    "crypto/elliptic"
    "math/big"
    "crypto/ecdsa"
    "crypto/rand"
    pb "./protos"
)

func signTransaction(transaction *pb.Transaction) *pb.Transaction {
    // Use the private key to create a signature associated with this transaction
    // Note we treat public keys as just the concatenation of the x,y points on the elliptic curve
    transaction.SenderPubKey = make([]byte, 0)
    transaction.SenderPubKey = append(transaction.SenderPubKey, key.PublicKey.X.Bytes()...)
    transaction.SenderPubKey = append(transaction.SenderPubKey,  key.PublicKey.Y.Bytes()...)
    r, s, _ := ecdsa.Sign(rand.Reader, key, GetHash(transaction))
    // Returns two big ints which we concatenate as the signature 
    transaction.Signature = make([]byte, 0)
    transaction.Signature = append(transaction.Signature, r.Bytes()...)
    transaction.Signature = append(transaction.Signature, s.Bytes()...)
    return transaction
}

// Check
// 1. Signature came from the private key associated with the public key of the sender
// 2. The referenced UTXO exists and is not already spent
// 3. Sum of the input UTXO is larger than the output
func TransactionVerify(transaction *pb.Transaction) bool {
    // Check that the signature came from the private key associated with the public key of the sender
    // Signature is simply R and S (both 32 byte numbers) concatentated
    // Convert to big ints
    // Pub key will be concatenation of X and Y big ints converted to bytes 
    pubKey := ecdsa.PublicKey{Curve: elliptic.P256()}
    pubKey.X = new(big.Int)
    pubKey.Y = new(big.Int)
    pubKey.X.SetBytes(transaction.SenderPubKey[:32])
    pubKey.Y.SetBytes(transaction.SenderPubKey[32:])
    r := new(big.Int)
    r.SetBytes(transaction.Signature[:32])
    s := new(big.Int)
    s.SetBytes(transaction.Signature[32:])
    fmt.Println(r, s)
    verifystatus := ecdsa.Verify(&pubKey, GetHash(transaction), r, s)
    return verifystatus 
}

func TransactionToString(transaction pb.Transaction) string {
    return fmt.Sprintf("%v --> %v $%v", transaction.InputUTXO, transaction.ReceiverPubKey, transaction.Value)
}

func GetHash(transaction *pb.Transaction) []byte {
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
