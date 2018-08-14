package main

import (
    "fmt"
    "crypto/sha256"
    "crypto/ecdsa"
    "encoding/binary"
    "crypto/elliptic"
    "math/big"
)

var PubKeyCurve elliptic.Curve  

type Transaction struct {
    hash []byte // Hash of the whole transaction, to be used in the merkle tree in a block
    signature []byte // Depends on private key associated with the sender, signed using ECC
    sender []byte // Pub key 
    receiver []byte // Pub key
    value uint32 
}

func (transaction *Transaction) Verify() bool {
    // Check that the signature came from the private key associated with the public key of the sender
    // Signature is simply R and S (both 32 byte numbers) concatentated
    // Convert to big ints
    // Pub key will be concatenation of X and Y big ints converted to bytes 
    pubKey := ecdsa.PublicKey{Curve: PubKeyCurve}
    pubKey.X = new(big.Int)
    pubKey.Y = new(big.Int)
    pubKey.X.SetBytes(transaction.sender[:32])
    pubKey.Y.SetBytes(transaction.sender[32:])
    r := new(big.Int)
    r.SetBytes(transaction.signature[:32])
    s := new(big.Int)
    s.SetBytes(transaction.signature[32:])
    verifystatus := ecdsa.Verify(&pubKey, transaction.GetHash(), r, s)
    return verifystatus 
}

func (transaction *Transaction) ToString() string {
    return fmt.Sprintf("%v --> %v $%v", transaction.sender, transaction.receiver, transaction.value)
}

func (transaction *Transaction) GetHash() []byte {
    // SHA hash is 32 bytes
	toHash := make([]byte, 0)
	toHash = append(toHash, []byte(transaction.sender)...)
	toHash = append(toHash, []byte(transaction.receiver)...)
    value := make([]byte, 4)
	binary.LittleEndian.PutUint32(value, transaction.value)
	toHash = append(toHash, value...)
	sum := sha256.Sum256(toHash)
    return sum[:]
}
