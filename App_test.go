package main

import (
	"testing"
	"akhcoin/p2p"
	"os"
	"akhcoin/blockchain"
	"time"
)

func TestApp(t *testing.T) {

	os.Remove(p2p.HostsInfoPath)

	node1 := NewAkhNode(9876)
	node2 := NewAkhNode(9765)
	node3 := NewAkhNode(9654)

	node1.Head = blockchain.NewBlock(node1.Head)
	for i := 0; i < 2; i++ {
		node1.Head = blockchain.NewBlock(node1.Head)
		node2.Head = blockchain.NewBlock(node2.Head)
	}

	node3.testPay()

	time.Sleep(500 * time.Millisecond)

	node3.initialBlockDownload()

	node1.Host.Close()
	node2.Host.Close()
	node3.Host.Close()

}
