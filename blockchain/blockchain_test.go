package blockchain

import (
	"fmt"
	"github.com/libp2p/go-libp2p-peer"
	"github.com/spf13/viper"
)

func ExampleNewBlock() {
	viper.Set("reward", 50)

	priv, _, _ := NewKeys()
	t := Pay(priv, peer.ID("some"), 42)
	v := NewVote(priv, "other")

	parent := CreateGenesis()
	block := NewBlock(priv, parent, []Transaction{*t}, []Vote{*v})

	verified, _ := Verify(&block.BlockData, &parent.BlockData)
	fmt.Println(verified)

	block.Reward = 100
	verified, _ = Verify(&block.BlockData, &parent.BlockData)
	fmt.Println(verified)

	block.Reward = 50
	block.Transactions[0].Amount = 0
	verified, _ = Verify(&block.BlockData, &parent.BlockData)
	fmt.Println(verified)

	block.Transactions[0].Amount = 42
	block.Votes[0].Candidate = "third"
	verified, _ = Verify(&block.BlockData, &parent.BlockData)
	fmt.Println(verified)

	// Output:
	// true
	// false
	// false
	// false

}
