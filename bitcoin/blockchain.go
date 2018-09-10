package main

import (
	pb "./protos"
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"
)

type Blockchain struct {
	blocks map[string]*pb.Block
	// Could be extended to handle temporary forks
	tipsOfChains []*pb.Block
	// Would be the pool of orphan blocks
	//     orphanBlocks []*pb.Block
	nextBlockNum int
	target       []byte // difficulty for mining
	// This an index to lookup a block hash by transaction hash,
	// the real bitcoin implementation has something similar but heavily cached/optimized
	// see bitcoin/src/index/txindex.h
	txIndex map[string]TxIndex
}

// Block and index of transaction
// to quickly look up a transaction
type TxIndex struct {
	blockHash string
	index     int
}

// Wrapper with details about the location
// of a specific TXI / TXO
type UTXO struct {
	transaction *pb.Transaction // transaction which has the UTXO
	index       int             // vout index
}

func (b *Blockchain) setTarget(inputTarget []byte) {
	b.target = inputTarget
}

func (b *Blockchain) addGenesisBlock() {
	var genesis pb.Block
	var genesisHeader pb.BlockHeader
	// Block # 1
	genesisHeader.Height = 1
	genesisHeader.PrevBlockHash = make([]byte, 32)
	genesisHeader.MerkleRoot = make([]byte, 32)
	genesis.Header = &genesisHeader
	b.nextBlockNum = 2 // Next block num
	b.blocks[string(getBlockHash(&genesis))] = &genesis
	// Currently the longest chain is this block to build on
	// top of
	b.tipsOfChains = append(b.tipsOfChains, &genesis)
}

// Determine a set of UTXOs which can cover the transaction amount
// return nil if it is not possible
func (blockChain Blockchain) getUTXOsToCoverTransaction(key *ecdsa.PrivateKey, desiredAmount uint64) []*UTXO {
	var currentAmount uint64
	var results []*UTXO
	for _, utxo := range blockChain.getUTXOs(&key.PublicKey) {
		if currentAmount >= desiredAmount {
			return results
		} else {
			currentAmount += blockChain.getValueUTXO(utxo)
			results = append(results, utxo)
		}
	}
	// No such utxo
	return results
}

func (blockChain Blockchain) getBalance(key *ecdsa.PublicKey) uint64 {
	var balance uint64
	for _, utxo := range blockChain.getUTXOs(key) {
		balance += blockChain.getValueUTXO(utxo)
	}
	return balance
}

func (blockChain Blockchain) getTransaction(hash []byte) *pb.Transaction {
	idx, ok := blockChain.txIndex[string(hash)]
	if !ok {
		fmt.Println("Transaction not indexed")
		return nil
	}
	return blockChain.blocks[idx.blockHash].Transactions[idx.index]
}

func (blockChain Blockchain) getValueUTXO(utxo *UTXO) uint64 {
	txIndex := blockChain.txIndex[string(getTransactionHash(utxo.transaction))]
	block := blockChain.blocks[txIndex.blockHash]
	return block.Transactions[txIndex.index].Vout[utxo.index].Value
}

func (blockChain Blockchain) getTXO(utxo *UTXO) *pb.TXO {
	txIndex := blockChain.txIndex[string(getTransactionHash(utxo.transaction))]
	block := blockChain.blocks[txIndex.blockHash]
	return block.Transactions[txIndex.index].Vout[utxo.index]
}

// Given a transaction input, lookup the pubkey of the transaction hash
// that input references
func (blockChain Blockchain) getSenderPubKey(txi *pb.TXI) []byte {
	txIndex := blockChain.txIndex[string(txi.TxID)]
	block := blockChain.blocks[txIndex.blockHash]
	trans := block.Transactions[txIndex.index]
	txo := trans.Vout[txi.Index]
	return txo.ReceiverPubKey
}

func (blockChain Blockchain) getUTXOs(key *ecdsa.PublicKey) []*UTXO {
	sent := make([]*UTXO, 0)
	received := make([]*UTXO, 0)
	utxos := make([]*UTXO, 0)
	// Make two lists --> inputs from our pubkey and outputs to our pubkey
	// Then walk the outputs looking to see if that output transaction is referenced
	// anywhere in an input, then the utxo was spent
	for _, block := range blockChain.blocks {
		for _, transaction := range block.Transactions {
			for _, inputUTXO := range transaction.Vin {
				// If the transaction hash and index in this vin references an output which has our pub key
				// that means we spent that index
				if bytes.Equal(blockChain.getSenderPubKey(inputUTXO), getPubKeyBytesFromPublicKey(key)) {
					fmt.Printf("\nSpent %s", getTXIString(inputUTXO))
					sent = append(sent, &UTXO{transaction: blockChain.getTransaction(inputUTXO.TxID),
						index: int(inputUTXO.Index)})
				}
			}
			for i, outputTX := range transaction.Vout {
				if bytes.Equal(outputTX.ReceiverPubKey, getPubKeyBytesFromPublicKey(key)) {
					fmt.Printf("\nReceived %s", getTXOString(outputTX))
					received = append(received, &UTXO{transaction: transaction,
						index: i})
				}
			}
		}
	}
	spent := false
	// Walk through all transactions which reference us as an output (all received)
	// check if each one is spent, if not it is a UTXO
	for _, candidateUTXO := range received {
		spent = false
		// Loop over the Vin's with our pubkey as the sender (spent)
		// If the transaction hash of the candidateUTXO matches, we cannot use it
		for _, spentTX := range sent {
			if bytes.Equal(getTransactionHash(spentTX.transaction), getTransactionHash(candidateUTXO.transaction)) && spentTX.index == candidateUTXO.index {
				spent = true
			}
		}
		if !spent {
			utxos = append(utxos, candidateUTXO)
		}
	}
	return utxos
}

func getBlockString(block *pb.Block) string {
	var buf bytes.Buffer
	buf.WriteString("\nBlock Hash: ")
	buf.WriteString(hex.EncodeToString(getBlockHash(block)))
	buf.WriteString("\nBlock Header: ")
	buf.WriteString("\n  prevBlockHash: ")
	buf.WriteString(hex.EncodeToString(block.Header.PrevBlockHash[:]))
	buf.WriteString("\n  timestamp: ")
	buf.WriteString(time.Unix(0, int64(block.Header.TimeStamp)).String())
	buf.WriteString("\n  nonce: ")
	buf.WriteString(strconv.Itoa(int(block.Header.Nonce)))
	buf.WriteString("\n  height: ")
	buf.WriteString(strconv.Itoa(int(block.Header.Height)))
	buf.WriteString("\nTransactions:\n\n")
	for i := range block.Transactions {
		buf.WriteString("\n")
		buf.WriteString(getTransactionString(block.Transactions[i]))
		buf.WriteString("\n")
	}
	return buf.String()
}

func (blockChain Blockchain) blockIsValid(target []byte, block *pb.Block) bool {
	// Check whether the block is mined, its previous block is
	// mined and all transactions are valid
	if !checkHashMined(target, getBlockHash(block)) {
		fmt.Println("invalid block hash not mined, target:", target)
		return false
	}
	for _, trans := range block.Transactions {
		if !blockChain.verifyTransaction(trans) {
			fmt.Println("transaction invalid in block")
			return false
		}
	}
	return true
}

func getBlockHash(block *pb.Block) []byte {
	buf := new(bytes.Buffer)
	buf.Write(block.Header.PrevBlockHash)
	buf.Write(block.Header.MerkleRoot)
	binary.Write(buf, binary.LittleEndian, block.Header.TimeStamp)
	binary.Write(buf, binary.LittleEndian, block.Header.Height)
	binary.Write(buf, binary.LittleEndian, block.Header.DifficultyTarget)
	binary.Write(buf, binary.LittleEndian, block.Header.Nonce)
	for _, trans := range block.Transactions {
		buf.Write(getTransactionHash(trans))
	}
	sum := sha256.Sum256(buf.Bytes())
	return sum[:]
}

// Note having the merkle root in the block header allows one to
// hash only the block header and obtain a unique hash for that whole block
// because any transaction change in the block will alter the merkle root
func getMerkleRoot(input []*pb.Transaction) []byte {
	numTransactions := len(input)
	if numTransactions == 1 {
		return getTransactionHash(input[0])
	}
	if numTransactions%2 != 0 {
		// Odd number of transactions need to double the last input,
		// could only happen on the first recursive call
		input = append(input, input[numTransactions-1])
		numTransactions += 1
	}
	m1 := getMerkleRoot(input[:numTransactions/2])
	m2 := getMerkleRoot(input[numTransactions/2:])
	buf := new(bytes.Buffer)
	buf.Write(m1)
	buf.Write(m2)
	sum := sha256.Sum256(buf.Bytes())
	return sum[:]
}
