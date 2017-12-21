package blockchain

import (
	"crypto/sha256"
	"hash"
)

type Block struct {
	Hash         hash.Hash
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

//func (b *Block) Y() hash.Hash {
//	return b.y
//}
//
//func (b *Block) SetY(y hash.Hash) {
//	b.y = y
//}

func CreateGenesis() *Block {
	return &Block{Parent: nil, Hash: sha256.New()}
}

func NewBlock() *Block {
	var rootNode TreeNode
	rootNode = collectTransactions()
	block := Block{Hash: sha256.New(), Transactions: rootNode}
	return &block
}

//gathers all published but not confirmed transactions
func collectTransactions() TreeNode {
	//TODO temp
	t := Transaction{"me", "someone", 100}
	return TreeNode{T: t}

}
