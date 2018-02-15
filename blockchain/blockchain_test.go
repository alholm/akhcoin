package blockchain

import (
	"fmt"
	"github.com/libp2p/go-libp2p-peer"
	"github.com/satori/go.uuid"
)

func ExampleNewBlock() {
	priv, _, _ := NewKeys()
	t := Pay(priv, peer.ID("some"), 42)

	parent := CreateGenesis()
	block := NewBlock(priv, parent, []Transaction{*t})
	verified, _ := verify(block)
	fmt.Println(verified)
	block.Nonce, _ = uuid.NewV1()
	verified, _ = verify(block)
	fmt.Println(verified)
	// Output:
	// true
	// false

}
