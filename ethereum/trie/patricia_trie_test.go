package trie

import (
	"testing"
)

func TestInsert(t *testing.T) {
	mpt := MPT{}
	mpt.initMPT()
	testKey, testValue := "key", "value"
	mpt.update(testKey, testValue)
	t.Log(mpt)
	res := mpt.get(testKey)
	if res != testValue {
		t.Logf("Expected %v got %v", testValue, res)
		t.Fail()
	}
	firstRootHash := mpt.rootHash
	testValue = "value2"
	// If you update again those changes should propagate up creating new hashes all the way up
	mpt.update(testKey, testValue)
	res = mpt.get(testKey)
	if res != testValue {
		t.Logf("Expected %v got %v", testValue, res)
		t.Fail()
	}
	secondRootHash := mpt.rootHash
	if firstRootHash == secondRootHash {
		t.Fail()
	}
}
