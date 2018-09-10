package main

import (
	pb "./protos"
	"encoding/hex"
	"errors"
	"golang.org/x/net/context"
	"strings"
	"testing"
	"time"
)

// Returns error if the block was not mined
// in time, note this depends on the difficulty
// and power of the machine
func mineBlockHelper(s *Server, minChainLength int) error {
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	// ticker.C is a channel of Time instants
	for mined := false; !mined; {
		select {
		case <-timeout:
			// If our timeout Time instant appears on that channel
			// it is time to end
			return errors.New("Failed to mine block in time")
		case <-ticker.C:
			// Check if we have mined a block
			// if so we are done
			if len(s.Blockchain.blocks) >= minChainLength {
				mined = true
			}
		}
	}
	return nil
}

// Mine at least minChainLength blocks, fail the
// test if anything bad happens
func mineBlocks(s *Server, t *testing.T, minChainLength int) {
	_, err := s.StartMining(context.Background(), &pb.Empty{})
	if err != nil {
		t.Fail()
	}
	// 2 = genesis block as well as our new block
	if err = mineBlockHelper(s, minChainLength); err != nil {
		t.Log(err)
		t.Fail()
	}
	_, err = s.StopMining(context.Background(), &pb.Empty{})
	if err != nil {
		t.Fail()
	}
}

// Check balance updates upon mining
func TestMineBlock(t *testing.T) {
	s := initServer()
	s.Wallet.createKey()
	// Relax difficulty for this
	target, _ := hex.DecodeString(strings.Join([]string{"e", strings.Repeat("f", 19)}, ""))
	s.Blockchain.setTarget(target)
	mineBlocks(s, t, 3)
	balance := int(s.getBalance(&s.Wallet.key.PublicKey))
	numBlocks := len(s.Blockchain.blocks)
	if balance != (numBlocks-1)*BLOCK_REWARD {
		t.Logf("Balance is %d, should be %d", balance, (numBlocks-1)*BLOCK_REWARD)
		t.Fail()
	}
}
