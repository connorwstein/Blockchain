package main

import (
    "testing"
)

func TestInsert(t *testing.T) {
    mpt := MPT{}  
    mpt.initMPT()
    testKey, testValue := "hello", "world"
    mpt.update(testKey, testValue)
    res := mpt.get(testKey)
    if res != testValue {
        t.Fail()
    }
}
