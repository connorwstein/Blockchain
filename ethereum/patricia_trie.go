package main

import (
	"crypto/sha256"
	"encoding/hex"
    "fmt"
	"bytes"
    "log"
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
}

// Given a char return the associated index in path
// i.e. 'c' --> 0x0c --> 12
func getIndex(char string) int {
	// this is string representing a hexademical char
	indexBytes, _ := hex.DecodeString(strings.Join([]string{"0", char}, ""))
	index := int(indexBytes[0])
	return index	
}

func (mpt *MPT) updateHelper(nodeHash string, path string, pathIndex int, value string) string {
	var newNode Node
	if pathIndex == len(path) {
		// If path is empty we have found the node
		// need to create an empty node to insert
		node, ok := mpt.db[nodeHash]
		if ok {
			// Node already here just update value 
			log.Printf("updating node from %s to %s\n", node.value, value)
			newNode = node 
			newNode.value = value
		} else {
			// Brand new node
			newNode = Node{value: value}
		}
	} else {
		// Insert a new node (value) and its hash (key) 
		index := getIndex(string(path[pathIndex]))
		oldNode, ok := mpt.db[nodeHash]
		if !ok {
			newNode = Node{} 
		} else {
			newNode = oldNode
		}
		nextHash := mpt.updateHelper(oldNode.path[index], path, pathIndex + 1, value)
		newNode.path[index] = nextHash	
	}
	// insert the node
	hashNew := getHash(newNode)
	mpt.db[hashNew] = newNode
    if len(mpt.db) == 1 {
        mpt.rootHash = hashNew
    }
	log.Printf("Adding node %v at hash %v\n", hashNew, newNode)
	return hashNew 
}

func (mpt MPT) String() string {	
	buf := bytes.NewBufferString("Dump tree: \n")
	buf.Write([]byte(fmt.Sprintf("  Root hash %v\n", mpt.rootHash)))
	for k, v := range mpt.db {
		buf.Write([]byte(fmt.Sprintf("key: %v value: %v\n", k, v)))
	}
	return buf.String()
}

func (mpt *MPT) update(key string, value string) {
	// Get the hex representation of the bytes in key (leave value as string for now)
	log.Printf("Inserting %v\n", getHexString([]byte(key)))
	hash := mpt.updateHelper(mpt.rootHash, getHexString([]byte(key)), 0, value)
	log.Printf("Inserted key %s (%v) value %s at %v\n", key, getHexString([]byte(key)), value, hash) 
}

func (mpt MPT) get(key string) string {	
	// Start from the rootHash and keep walking	until we either run out of chars in the hex string or hit a null 
	// (null meaning no such item)
	// leaves are denoted by an empty path
	keyString := getHexString([]byte(key))
	result := ""
	currentNode := mpt.db[mpt.rootHash]
	for _, c := range keyString {
		// Should be a hash here
		nextNodeHash := currentNode.path[getIndex(string(c))]
		if nextNodeHash == "" {
			// we have hit the end
			result = currentNode.value 
			break
		} 
		currentNode = mpt.db[nextNodeHash]
	}
	return result
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
