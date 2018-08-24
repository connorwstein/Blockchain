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

func getPubKeyBytesFromPublicKey(key *ecdsa.PublicKey) []byte {
    buf := new(bytes.Buffer)
    buf.Write(key.X.Bytes())
    buf.Write(key.Y.Bytes())
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
//     transaction.SenderPubKey = getPubKeyBytes(key)
    r, s, _ := ecdsa.Sign(rand.Reader, key, getTransactionHash(transaction))
    // Returns two big ints which we concatenate as the signature 
    transaction.Signature = getSignatureBytes(r, s)
    return transaction
}

// Check
// 1. Signature came from the private key associated with the public key of the sender  
// 2. The referenced UTXO exists and is not already spent 
// 3. Vin == Vout (value wise)  
func (blockChain Blockchain) verifyTransaction(transaction *pb.Transaction) bool {
    if len(transaction.Signature) != 64 {
        return false
    }
    r := new(big.Int)
    r.SetBytes(transaction.Signature[:32])
    s := new(big.Int)
    s.SetBytes(transaction.Signature[32:])
    fmt.Println(r, s)
    toVerify := ecdsa.PublicKey{Curve: elliptic.P256()}
    toVerify.X = new(big.Int)
    toVerify.Y = new(big.Int)
    if len(transaction.Vin) == 0 {
        // Coinbase transaction, only need to verify that the
        // output receiver pub key matches the signature
        // and the block reward is exact
        if len(transaction.Vout) != 1 {
            // should only be one output to the miner
            return false
        } 
        if transaction.Vout[0].Value != BLOCK_REWARD {
            return false
        }
        toVerify.X.SetBytes(transaction.Vout[0].ReceiverPubKey[:32])
        toVerify.Y.SetBytes(transaction.Vout[0].ReceiverPubKey[32:])
         
        fmt.Println("coin base transaction")
    } else {
        senderPubKeyBytes := blockChain.getSenderPubKey(transaction.Vin[0])
        toVerify.X.SetBytes(senderPubKeyBytes[:32])
        toVerify.Y.SetBytes(senderPubKeyBytes[32:])
        sendersUTXOs := blockChain.getUTXOs(&toVerify)
        totalVinValue := uint64(0)
        // Each one of Vin should be in the set of that senders UTXOs
        for _, txi := range transaction.Vin {
            idx := blockChain.txIndex[string(txi.TxID)]
            value := blockChain.blocks[idx.blockHash].Transactions[idx.index].Vout[txi.Index].Value
            totalVinValue += value
            found := false
            for _, utxo := range sendersUTXOs {
                if bytes.Equal(txi.TxID, getTransactionHash(utxo.transaction)) && txi.Index == uint64(utxo.index) {
                    found = true
                }
            }
            if !found {
                fmt.Println("Referencing an invalid UTXO")
                return false
            }
        }
        // Check whether vout value matches
        totalVoutValue := uint64(0)
        for _, txo := range transaction.Vout {
            totalVoutValue += txo.Value 
        }
        if totalVoutValue != totalVinValue {
            fmt.Printf("Invalid transaction: Vin value %d Vout value %d", totalVinValue, totalVoutValue)
            return false
        }
    }
    verifystatus := ecdsa.Verify(&toVerify, getTransactionHash(transaction), r, s)
    fmt.Println("Verify signature: ", verifystatus)
    return verifystatus 
}

func getTXIString(tx *pb.TXI) string {
    var buf bytes.Buffer
    buf.WriteString("\n  TxID:")
    buf.WriteString(hex.EncodeToString(tx.TxID[:]))
    buf.WriteString("\n  Index:")
    buf.WriteString(strconv.Itoa(int(tx.Index)))
    return buf.String()
}

func getTXOString(tx *pb.TXO) string {
    var buf bytes.Buffer
    buf.WriteString("\n  Receiver:")
    pubKey := ecdsa.PublicKey{Curve: elliptic.P256()}
    pubKey.X = new(big.Int)
    pubKey.Y = new(big.Int)
    pubKey.X.SetBytes(tx.ReceiverPubKey[:32])
    pubKey.Y.SetBytes(tx.ReceiverPubKey[32:])
    buf.WriteString(strings.Join([]string{pubKey.X.String(), pubKey.Y.String()}, ""))
    buf.WriteString("\n  Amount:")
    buf.WriteString(strconv.Itoa(int(tx.Value)))
    return buf.String()
}

func getTransactionString(transaction *pb.Transaction) string {
    var buf bytes.Buffer
    buf.WriteString("\nTransaction Hash: ")
    buf.WriteString(hex.EncodeToString(getTransactionHash(transaction)[:]))
    buf.WriteString("\nVin: ")
    if len(transaction.Vin) == 0 {
        buf.WriteString("Miner reward")
    }
    for _, inputUTXO := range transaction.Vin {
        buf.WriteString("\n TX: ")
        buf.WriteString(getTXIString(inputUTXO))
    }
    buf.WriteString("\nVout: ")
    for _, outputTX := range transaction.Vout {
        buf.WriteString("\n TX: ")
        buf.WriteString(getTXOString(outputTX))
    }
    buf.WriteString("\nHeight: ")
    buf.WriteString(strconv.Itoa(int(transaction.Height)))
    return buf.String()
}

func getTransactionHash(transaction *pb.Transaction) []byte {
    buf := new(bytes.Buffer)
    for _, inputUTXO := range transaction.Vin {
	    buf.Write(inputUTXO.TxID)
        binary.Write(buf, binary.LittleEndian, inputUTXO.Index)
    } 
    for _, outputTX := range transaction.Vout {
	    buf.Write(outputTX.ReceiverPubKey)
        binary.Write(buf, binary.LittleEndian, outputTX.Value)
    } 
    // Super important: Height needed to make coinbase transactions unique
    binary.Write(buf, binary.LittleEndian, transaction.Height)
	sum := sha256.Sum256(buf.Bytes())
    return sum[:]
}

// Note this is an honest node, need to find a way to test a malicious node
func (s *Server) SendTransaction(ctx context.Context, in *pb.TransactionRequest) (*pb.Empty, error) {
    var reply pb.Empty
    if s.Wallet.key == nil {
        return &reply, errors.New("Need to make an account first") 
    }
    // Find some UTXO we can use to cover the transaction
    // If we cannot, then we have to reject the transactionk
    inputUTXO := s.Blockchain.getUTXOsToCoverTransaction(s.Wallet.key, in.Value)
    if inputUTXO == nil {
        return &reply, errors.New(fmt.Sprintf("Not enough coin, balance is %d", s.Blockchain.getBalance(s.Wallet.key)))
    }
    // Reference to that unspent output being used in this transaction
    // Just one input and output right now
    var trans pb.Transaction 
    var input pb.TXI
    input.TxID = getTransactionHash(inputUTXO.transaction)
    input.Index = uint64(inputUTXO.index)
    trans.Vin  = append(trans.Vin, &input) 
    var output pb.TXO
    output.ReceiverPubKey = append(output.ReceiverPubKey, in.ReceiverPubKey...) 
    output.Value = in.Value 
    trans.Vout = append(trans.Vout, &output)
    signTransaction(&trans, s.Wallet.key)
    fmt.Printf("Send transaction %v\n", getTransactionString(&trans))
    s.MemPool.addTransactionToMemPool(&trans) 
    // Send this transaction to all the list of clients we are connected to
    // Need to include the source, so that the peer doesn't send it back to us
    for _, myPeer := range s.peerList {
        // Find which one of our IP addresses is in the same network as the peer
        ipAddr, _ := net.ResolveIPAddr("ip", myPeer.sourceIP)
        // This cast works because ipAddr is a pointer and the pointer to ipAddr does implement 
        // the Addr interface
        ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: ipAddr})
        c := pb.NewTransactionsClient(myPeer.conn)
        c.ReceiveTransaction(ctx, &trans)
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
    if ! s.Blockchain.verifyTransaction(in)  {
        fmt.Println("Reject transaction, invalid signature")
        return &reply, errors.New("Dropping invalid transaction")
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
