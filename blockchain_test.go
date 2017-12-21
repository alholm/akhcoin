package main

import (
	"fmt"
)

func ExampleAkhNode() {
	node := NewAkhNode()
	block := node.Produce()
	node.Receive(block)
	fmt.Printf("Chain len = %x\n", len(node.blockchain))
	// Output:
	// Block verified!
	// Chain len = 2
}
