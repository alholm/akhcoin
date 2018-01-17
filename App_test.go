package main

import (
	"testing"
	"akhcoin/p2p"
	"os"
	"time"
)

func TestInitialBlockDownload(t *testing.T) {

	os.Remove(p2p.HostsInfoPath)

	node1 := NewAkhNode(9876)
	node2 := NewAkhNode(9765)
	node3 := NewAkhNode(9654)

	node2.testPay()
	node3.testPay()

	time.Sleep(500 * time.Millisecond)

	l := len(node1.transactionsPool)
	if l != 2 {
		t.Errorf("%d transactions in pull, has to be 2", l)
	}

	node1.Produce()

	for i := 0; i < 2; i++ {
		node1.Produce()
		node2.Produce()
	}

	node3.initialBlockDownload()

	node1.Host.Close()
	node2.Host.Close()
	node3.Host.Close()
}

func TestAkhNode_Announce(t *testing.T) {
	os.Remove(p2p.HostsInfoPath)

	node1 := NewAkhNode(9876)
	node2 := NewAkhNode(9765)
	node3 := NewAkhNode(9654)

	node2.testPay()
	node3.testPay()

	time.Sleep(500 * time.Millisecond)

	block, _ := node1.Produce()
	node1.Announce(block)

	time.Sleep(500 * time.Millisecond)

	if node2.Head.Hash != block.Hash || node3.Head.Hash != block.Hash {
		t.Error("Announced block wasn't consumed")
	}

	l := len(node2.transactionsPool)
	if l > 0 {
		t.Errorf("%d transactions in pull, has to be 0", l)
	}

}
