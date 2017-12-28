package blockchain

import (
	"crypto/sha256"
	"fmt"
)

type Block struct {
	Hash         string
	Parent       *Block
	Transactions TreeNode
}
type Transaction struct {
	Sender    string
	Recipient string
	Amount    int64
}

type TreeNode struct {
	T           Transaction
	Left, Right *TreeNode
}

func CreateGenesis() *Block {
	return &Block{Parent: nil, Hash: Hash("genesis")}
}

func NewBlock() *Block {
	var rootNode TreeNode
	rootNode = collectTransactions()
	block := Block{Hash: Hash("TODO: block contents hash"), Transactions: rootNode}
	return &block
}

//gathers all published but not confirmed transactions
func collectTransactions() TreeNode {
	//TODO temp
	t := Transaction{"me", "someone", 100}
	return TreeNode{T: t}

}

func Hash(str string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(str)))
}
