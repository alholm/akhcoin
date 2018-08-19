package blockchain

import (
	"fmt"
	"github.com/libp2p/go-libp2p-peer"
)

func ExampleTransaction() {
	priv, _, _ := NewKeys()
	t := Pay(priv, peer.ID("some"), 42)
	verified, _ := verify(t)
	fmt.Println(verified)
	t.Amount++
	verified, _ = verify(t)
	fmt.Println(verified)
	// Output:
	// true
	// false

}
