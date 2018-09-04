package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"bytes"
// 	"strconv"
	"strings"
)


type Node struct {
	// Simple case first, inefficient merkle patricia tree
	// just have 17 items	
	// hashes of nodes, if you have a hash in path[0] that indicates there is a 0 in the path,
	// similarly for path[15] indicates an f in the path
	path [16]string 
	value string
}

func (n Node) String() string {
	return fmt.Sprintf("0:%s 1:%s 2:%s 3:%s 4:%s 5:%s 6:%s 7:%s 8:%s 9:%s A:%s B:%s C:%s D:%s E:%s F:%s Value: %s", 
				n.path[0], n.path[1], n.path[2], n.path[3], n.path[4], n.path[5], n.path[6], n.path[7],
				n.path[8], n.path[9], n.path[10], n.path[11], n.path[12], n.path[13], n.path[14], n.path[15], n.value)
}

type MPT struct {
	db map[string]Node // key is a hash
	rootHash string	
}

func (mpt *MPT) initMPT() {
	mpt.db = make(map[string]Node)
	rootNode := Node{}
	rootHash := getHash(rootNode)
	mpt.db[rootHash] = rootNode
	mpt.rootHash = rootHash
}

func (mpt *MPT) insertHelper(nodeHash string, path string, pathIndex int, value string) string {
	var newNode Node
	if pathIndex == len(path) {
		// If path is empty we have found the node
		// need to create an empty node to insert
		node, ok := mpt.db[nodeHash]
		if ok {
			// Node already here just update value 
			fmt.Printf("updating node from %s to %s\n", node.value, value)
			newNode = node 
			newNode.value = value
		} else {
			// Brand new node
			newNode = Node{value: value}
		}
	} else {
		// need to continue our descent, increment path index
		index := path[pathIndex] // index in which we need to insert hash, 
		// this is string representing a hexademical char
		a, _ := hex.DecodeString(strings.Join([]string{"0", string(index)}, ""))
		v := int(a[0])
		fmt.Printf("index in path %d %d\n", index, v)
		oldNode, ok := mpt.db[nodeHash]
		if !ok {
			newNode = Node{} 
		} else {
			newNode = oldNode
		}
		// hash to be inserted at newNode.path[index]
		fmt.Printf("old node %v\n", oldNode.path[v])
		nextHash := mpt.insertHelper(oldNode.path[v], path, pathIndex + 1, value)
		newNode.path[v] = nextHash	
	}
	// insert the node
	hashNew := getHash(newNode)
	mpt.db[hashNew] = newNode
	fmt.Printf("Adding node %v at hash %v\n", hashNew, newNode)
	return hashNew 
}

func (mpt MPT) String() string {	
	buf := bytes.NewBufferString("Dump tree: \n")
	for k, v := range mpt.db {
		buf.Write([]byte(fmt.Sprintf("key: %v value: %v\n", k, v)))
	}
	return buf.String()
}

func (mpt *MPT) insert(key string, value string) {
	// Get the hex representation of the bytes in key (leave value as string for now)
	fmt.Printf("Inserting %v\n", getHexString([]byte(key)))
	hash := mpt.insertHelper(mpt.rootHash, getHexString([]byte(key)), 0, value)
	fmt.Printf("Inserted key %s (%v) value %s at %v\n", key, getHexString([]byte(key)), value, hash) 
}

func getHash(node Node) string {
	// values should be hashed
    h := sha256.New()
	for i := range node.path {
    	h.Write([]byte(node.path[i]))
	}
	h.Write([]byte(node.value))
    hashBytes := h.Sum(nil)
	return getHexString(hashBytes)
}

func getHexString(src []byte) string {
	dst := make([]byte, hex.EncodedLen(len(src)))
	hex.Encode(dst, src)
	return string(dst)
}
