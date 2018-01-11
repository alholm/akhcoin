package blockchain

import (
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/libp2p/go-libp2p-peer"
	"fmt"
)

func ExampleTransaction() {
	priv, _, _ := crypto.GenerateKeyPair(crypto.RSA, 2048)
	t := Pay(priv, peer.ID("some"), 42)
	verified, _ := t.Verify()
	fmt.Println(verified)
	t.Amount++
	verified, _ = t.Verify()
	fmt.Println(verified)
	// Output:
	// true
	// false

}
