package main

import (
    "fmt"
    "crypto/sha256"
//     "crypto/ecdsa"
    "errors"
    "net"
    "google.golang.org/grpc/peer"
    "golang.org/x/net/context"
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

func getPubKeyBytes(key *ecdsa.PrivateKey) []byte {
    pubKeyBytes := make([]byte, 64)
    pubKeyBytes = append(pubKeyBytes, key.PublicKey.X.Bytes()...)
    pubKeyBytes = append(pubKeyBytes,  key.PublicKey.Y.Bytes()...)
    return pubKeyBytes
}

func getSignatureBytes(r *big.Int, s *big.Int) [] byte {
    signatureBytes := make([]byte, 64)
    signatureBytes = append(signatureBytes, r.Bytes()...)
    signatureBytes = append(signatureBytes,  s.Bytes()...)
    return signatureBytes
}

func signTransaction(transaction *pb.Transaction, key *ecdsa.PrivateKey) *pb.Transaction {
    // Use the private key to create a signature associated with this transaction
    // Note we treat public keys as just the concatenation of the x,y points on the elliptic curve
    transaction.SenderPubKey = getPubKeyBytes(key)
    r, s, _ := ecdsa.Sign(rand.Reader, key, getTransactionHash(transaction))
    // Returns two big ints which we concatenate as the signature 
    transaction.Signature = getSignatureBytes(r, s)
    return transaction
}

// Check
// 1. Signature came from the private key associated with the public key of the sender - DONE
// 2. The referenced UTXO exists and is not already spent - TODO
// 3. Sum of the input UTXO is larger than the output - TODO
func verifyTransaction(transaction *pb.Transaction) bool {
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
    verifystatus := ecdsa.Verify(&pubKey, getTransactionHash(transaction), r, s)
    return verifystatus 
}

func getTransactionString(transaction *pb.Transaction) string {
    var buf bytes.Buffer
    buf.WriteString("\nTransaction Hash: ")
    buf.WriteString(hex.EncodeToString(getTransactionHash(transaction)[:]))
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


func getTransactionHash(transaction *pb.Transaction) []byte {
    // SHA hash is 32 bytes
    // TODO: use a writer here
	toHash := make([]byte, 0)
	toHash = append(toHash, []byte(transaction.InputUTXO)...)
	toHash = append(toHash, []byte(transaction.ReceiverPubKey)...)
	toHash = append(toHash, []byte(transaction.SenderPubKey)...)
    value := make([]byte, 8)
	binary.LittleEndian.PutUint64(value, transaction.Value)
	toHash = append(toHash, value...)
	binary.LittleEndian.PutUint64(value, transaction.Height)
	toHash = append(toHash, value...)
	sum := sha256.Sum256(toHash)
    return sum[:]
}

// Note this is an honest node, need to find a way to test a malicious node
func (s *server) SendTransaction(ctx context.Context, in *pb.Transaction) (*pb.Empty, error) {
    var reply pb.Empty
    if key == nil {
        return &reply, errors.New("Need to make an account first") 
    }
    // Find some UTXO we can use to cover the transaction
    // If we cannot, then we have to reject the transactionk
    inputUTXO := getUTXO(in.Value)
    if inputUTXO == nil {
        return &reply, errors.New(fmt.Sprintf("Not enough coin, balance is %d", getBalance()))
    }
    // Reference to our unspent output being used in this transaction
    in.InputUTXO = getTransactionHash(inputUTXO)
    // Our pub key gets added as part of the signing
    signTransaction(in, key)
    fmt.Printf("Send transaction %v\n", getTransactionString(in))
    addTransactionToMemPool(in) 
    // Send this transaction to all the list of clients we are connected to
    // Need to include the source, so that the peer doesn't send it back to us
    for _, myPeer := range peerList {
        // Find which one of our IP addresses is in the same network as the peer
        ipAddr, _ := net.ResolveIPAddr("ip", myPeer.sourceIP)
        // This cast works because ipAddr is a pointer and the pointer to ipAddr does implement 
        // the Addr interface
        ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: ipAddr})
        c := pb.NewTransactionsClient(myPeer.conn)
        c.ReceiveTransaction(ctx, in)
    }
    return &reply, nil
}

func (s *server) GetTransactions(in *pb.Empty, stream pb.State_GetTransactionsServer) error {
    fmt.Println("Get transactions")
    // Walk the mempool 
    for _, transaction := range memPool {
        stream.Send(transaction)
    }
    return nil
}


// Need to verify a transaction before propagating. This ensures that invalid transactions
// are dropped at the first node which receives it
func (s *server) ReceiveTransaction(ctx context.Context, in *pb.Transaction) (*pb.Empty, error) {
    var reply pb.Empty
    senderIP := getSenderIP(ctx)
    if !verifyTransaction(in)  {
        fmt.Println("Reject transaction, invalid signature")
        return &reply, nil
    }
    addTransactionToMemPool(in) 
    for _, myPeer := range peerList {
        if senderIP == "" || myPeer.peerIP == senderIP {
            // Don't send back to the receiver
            continue
        }
        ipAddr, _ := net.ResolveIPAddr("ip", myPeer.sourceIP)
        ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: ipAddr})
        c := pb.NewTransactionsClient(myPeer.conn)
        c.ReceiveTransaction(ctx, in)
    }
    return &reply, nil
}

func addTransactionToMemPool(transaction *pb.Transaction) {
    tx := getTransactionHash(transaction)
    memPool[string(tx[:])] = transaction
    fmt.Printf("Added transaction to mempool:")
    fmt.Println(getTransactionString(transaction))
}

// Walk the blockchain looking for references to our key 
// Maybe the wallet software normally just caches the utxos
// associated with our keys?
func getUTXOs() []*pb.Transaction {
    // Kind of wasteful on space, surely a better way
    sent := make([]*pb.Transaction, 0)
    received := make([]*pb.Transaction, 0)
    utxos := make([]*pb.Transaction, 0)
    // Right now we walk every block and transaction
    // Maybe there is a way to use the merkle root here?
    // Make two lists --> inputs from our pubkey and outputs to our pubkey
    // Then walk the outputs looking to see if that output transaction is referenced
    // anywhere in an input, then the utxo was spent
    for _, block := range blockChain {
        for _, transaction := range block.Transactions {
            if bytes.Equal(transaction.ReceiverPubKey, getPubKey()) {
                received = append(received, transaction) 
            }
            if bytes.Equal(transaction.SenderPubKey, getPubKey()) {
                sent = append(sent, transaction) 
            }
        } 
    }
    spent := false
    for _, candidateUTXO := range received {
        fmt.Println(getTransactionString(candidateUTXO))
        spent = false
        for _, spentTX := range sent {
            if bytes.Equal(spentTX.InputUTXO, getTransactionHash(candidateUTXO)) {
                spent = true 
                fmt.Println("spent ^")
            }
        }
        if !spent {
            utxos = append(utxos, candidateUTXO)
        }
    }
    return utxos 
}

// Find a specific UTXO of ours to reference in a new transaction
// needs to be > desiredAmount.
func getUTXO(desiredAmount uint64) *pb.Transaction {
    for _, utxo := range getUTXOs() {
        if utxo.Value >= desiredAmount {
            return utxo
        } 
    }
    // No such utxo
    return nil
}

func getBalance() uint64 {
    var balance uint64
    for _, utxo := range getUTXOs() {
        balance += utxo.Value
    }
    return balance
}

func (s *server) GetBalance(ctx context.Context, in *pb.Empty) (*pb.Balance, error) {
    var balance pb.Balance
    balance.Balance = getBalance()
    return &balance, nil
}
