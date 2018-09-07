package evm

import (
    "testing"
)

func TestParse(t *testing.T) {
    instructions := parse("6010602001")
    t.Log(instructions)
}

func TestExecute(t *testing.T) {
    instructions := parse("6010602001")
    t.Log(instructions)
	process(instructions)
}
