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

type MemPool struct {
    transactions map[string]*pb.Transaction
}

func (memPool *MemPool) addTransactionToMemPool(transaction *pb.Transaction) {
    tx := getTransactionHash(transaction)
    memPool.transactions[string(tx[:])] = transaction
    fmt.Printf("Added transaction to mempool:")
    fmt.Println(getTransactionString(transaction))
}


func getPubKeyBytes(key *ecdsa.PrivateKey) []byte {
    buf := new(bytes.Buffer)
    buf.Write(key.PublicKey.X.Bytes())
    buf.Write(key.PublicKey.Y.Bytes())
    return buf.Bytes() 
}

func getSignatureBytes(r *big.Int, s *big.Int) [] byte {
    buf := new(bytes.Buffer)
    buf.Write(r.Bytes())
    buf.Write(s.Bytes())
    return buf.Bytes() 
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
    if len(transaction.SenderPubKey) != 154 {
        // Coinbase transaction
        fmt.Println("Coinbase transaction")
        return true
    }
    fmt.Println("sender: ", len(transaction.SenderPubKey))
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
    fmt.Println("Verify: ", verifystatus)
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
func (s *Server) SendTransaction(ctx context.Context, in *pb.Transaction) (*pb.Empty, error) {
    var reply pb.Empty
    if s.Wallet.key == nil {
        return &reply, errors.New("Need to make an account first") 
    }
    // Find some UTXO we can use to cover the transaction
    // If we cannot, then we have to reject the transactionk
    inputUTXO := s.Blockchain.getUTXO(s.Wallet.key, in.Value)
    if inputUTXO == nil {
        return &reply, errors.New(fmt.Sprintf("Not enough coin, balance is %d", s.Blockchain.getBalance(s.Wallet.key)))
    }
    // Reference to our unspent output being used in this transaction
    in.InputUTXO = getTransactionHash(inputUTXO)
    // Our pub key gets added as part of the signing
    signTransaction(in, s.Wallet.key)
    fmt.Printf("Send transaction %v\n", getTransactionString(in))
    s.MemPool.addTransactionToMemPool(in) 
    // Send this transaction to all the list of clients we are connected to
    // Need to include the source, so that the peer doesn't send it back to us
    for _, myPeer := range s.peerList {
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

func (s *Server) GetTransactions(in *pb.Empty, stream pb.State_GetTransactionsServer) error {
    fmt.Println("Get transactions")
    // Walk the mempool 
    for _, transaction := range s.MemPool.transactions {
        stream.Send(transaction)
    }
    return nil
}


// Need to verify a transaction before propagating. This ensures that invalid transactions
// are dropped at the first node which receives it
func (s *Server) ReceiveTransaction(ctx context.Context, in *pb.Transaction) (*pb.Empty, error) {
    var reply pb.Empty
    senderIP := getSenderIP(ctx)
    if ! verifyTransaction(in)  {
        fmt.Println("Reject transaction, invalid signature ", in.SenderPubKey)
        return &reply, nil
    }
    s.MemPool.addTransactionToMemPool(in) 
    for _, myPeer := range s.peerList {
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

func (s *Server) GetBalance(ctx context.Context, in *pb.Empty) (*pb.Balance, error) {
    var balance pb.Balance
    balance.Balance = s.Blockchain.getBalance(s.Wallet.key)
    return &balance, nil
}
