package blockchain

import (
	"fmt"
	"github.com/libp2p/go-libp2p-peer"
	"github.com/satori/go.uuid"
)

func ExampleNewBlock() {
	priv, _, _ := NewKeys()
	t := Pay(priv, peer.ID("some"), 42)
	v := NewVote(priv, "other")

	parent := CreateGenesis()
	block := NewBlock(priv, parent, []Transaction{*t}, []Vote{*v})

	verified, _ := Verify(&block.BlockData)
	fmt.Println(verified)

	origNonce := block.Nonce
	block.Nonce, _ = uuid.NewV1()
	verified, _ = Verify(&block.BlockData)
	fmt.Println(verified)

	block.Nonce = origNonce
	block.Transactions[0].T.Amount = 0
	verified, _ = Verify(&block.BlockData)
	fmt.Println(verified)

	block.Transactions[0].T.Amount = 42
	block.Votes[0].Candidate = "third"
	verified, _ = Verify(&block.BlockData)
	fmt.Println(verified)

	// Output:
	// true
	// false
	// false
	// false

}
