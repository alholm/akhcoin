package blockchain

import (
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/libp2p/go-libp2p-peer"
	"fmt"
)

func ExampleTransaction(){
	priv, _, _ := crypto.GenerateKeyPair(crypto.RSA, 2048)
	t := Pay(priv, peer.ID("some"), 42)
	fmt.Println(t.Verify())
	t.Amount++
	fmt.Println(t.Verify())
	// Output:
	// true
	// false

}