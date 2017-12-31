package blockchain

import (
	"crypto/sha256"
	"fmt"
	"log"
)

type BlockData struct {
	Hash         string
	ParentHash   string
	Transactions TreeNode
}

type Block struct {
	BlockData
	Parent *Block
	Next   *Block
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
	return &Block{BlockData{ParentHash: "", Hash: Hash("genesis")}, nil, nil}
}

func NewBlock(parent *Block) *Block {
	rootNode := collectTransactions()

	block := &Block{
		BlockData{
			Hash:         Hash(parent.Hash + "TODO: block contents hash"),
			Transactions: rootNode,
		},
		parent,
		nil,
	}
	parent.Next = block
	log.Printf("New Block hash = %s", block.Hash)

	return block
}

//gathers all published but not confirmed transactions
func collectTransactions() TreeNode {
	//TODO temp
	t := Transaction{"me", "someone", 100}
	t1 := Transaction{"other one", "me", 200}
	return TreeNode{T: t, Right: &TreeNode{T: t1}}

}

func Hash(str string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(str)))
}

func Validate(block *Block, chainHead *Block) bool {
	return true
}
