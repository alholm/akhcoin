package main

import (
	"akhcoin/blockchain"
)

func ExampleAkhNode() {
	node := NewAkhNode()
	block := node.Produce(blockchain.CreateGenesis())
	node.Receive(block)
	// Output:
	// Block verified!
}
