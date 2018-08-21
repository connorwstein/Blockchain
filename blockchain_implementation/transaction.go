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
    "encoding/hex"
    "strconv"
    "bytes"
    "strings"
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
// 1. Signature came from the private key associated with the public key of the sender - DONE
// 2. The referenced UTXO exists and is not already spent - TODO
// 3. Sum of the input UTXO is larger than the output - TODO
func TransactionVerify(transaction *pb.Transaction) bool {
    // Check that the signature came from the private key associated with the public key of the sender
    // Signature is simply R and S (both 32 byte numbers) concatentated
    // Convert to big ints
    // Pub key will be concatenation of X and Y big ints converted to bytes 
    if len(transaction.SenderPubKey) <= 4 {
        // Coinbase transaction
        return true
    }
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

func getTransactionString(transaction *pb.Transaction) string {
    var buf bytes.Buffer
    buf.WriteString("\nTransaction Hash: ")
    buf.WriteString(hex.EncodeToString(GetHash(transaction)[:]))
    buf.WriteString("\nInput UTXO: ")
    buf.WriteString(hex.EncodeToString(transaction.InputUTXO[:]))
    buf.WriteString("\nSender: ")
    // Two 32 byte integers concated 
    if len(transaction.SenderPubKey) > 4 {
    pubKey := ecdsa.PublicKey{Curve: elliptic.P256()}
    pubKey.X = new(big.Int)
    pubKey.Y = new(big.Int)
    pubKey.X.SetBytes(transaction.SenderPubKey[:32])
    pubKey.Y.SetBytes(transaction.SenderPubKey[32:])
    buf.WriteString(strings.Join([]string{pubKey.X.String(), pubKey.Y.String()}, ""))
    } else {
        buf.WriteString("Miner reward")
    }
    buf.WriteString("\nReceiver: ")
    pubKey2 := ecdsa.PublicKey{Curve: elliptic.P256()}
    pubKey2.X = new(big.Int)
    pubKey2.Y = new(big.Int)
    pubKey2.X.SetBytes(transaction.ReceiverPubKey[:32])
    pubKey2.Y.SetBytes(transaction.ReceiverPubKey[32:])
    buf.WriteString(strings.Join([]string{pubKey2.X.String(), pubKey2.Y.String()}, ""))
    buf.WriteString("\nValue: ")
    buf.WriteString(strconv.Itoa(int(transaction.Value)))
    return buf.String()
}


func GetHash(transaction *pb.Transaction) []byte {
    // SHA hash is 32 bytes
    // TODO: use a writer here
	toHash := make([]byte, 0)
	toHash = append(toHash, []byte(transaction.InputUTXO)...)
	toHash = append(toHash, []byte(transaction.ReceiverPubKey)...)
	toHash = append(toHash, []byte(transaction.SenderPubKey)...)
    value := make([]byte, 8)
	binary.LittleEndian.PutUint64(value, transaction.Value)
	toHash = append(toHash, value...)
	sum := sha256.Sum256(toHash)
    return sum[:]
}
