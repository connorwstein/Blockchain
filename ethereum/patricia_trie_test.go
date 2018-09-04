package main

import (
    "testing"
    "fmt"
)

func TestInsert(t *testing.T) {
    mpt := MPT{}  
    mpt.initMPT()
    mpt.insert("hello", "world")
    fmt.Println(mpt)
}
