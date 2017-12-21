package main

import (
	. "akhcoin/blockchain"
	"fmt"
)

type Node interface {
	Vote(sign string, addr string)

	Produce() *Block

	Receive(*Block)

	ReceiveTransaction(*Transaction)
}

type AkhNode struct {
	transactionsPool map[*Transaction]bool
	blockchain       []*Block
	sign             string
}

func (*AkhNode) Vote(sign string, addr string) {
	panic("implement me")
}

func (*AkhNode) Produce() *Block {
	return NewBlock()
}

func (n *AkhNode) Receive(b *Block) {
	Verify(b)
	n.blockchain = append(n.blockchain, b)
}

func Verify(block *Block) error {
	fmt.Println("Block verified!")
	return nil
}

func (n *AkhNode) ReceiveTransaction(t *Transaction) {
	n.transactionsPool[t] = true
}

func NewAkhNode() *AkhNode {
	node := AkhNode{
		transactionsPool: make(map[*Transaction]bool),
		blockchain:       initBlockchain(),
		sign:             "",
	}
	return &node
}
func initBlockchain() []*Block {
	existingBlocks, size := downloadExistingBlocks()
	if size == 0 {
		existingBlocks = []*Block{CreateGenesis()}
	}
	return existingBlocks

}
func downloadExistingBlocks() ([]*Block, int) {
	return []*Block{}, 0
}

func main() {

	node := NewAkhNode()
	block := node.Produce()
	node.Receive(block)
	fmt.Printf("Chain len = %x\n", len(node.blockchain))
	node.Vote("mySign", "someAddr")
}
