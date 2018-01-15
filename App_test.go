package main

import (
	"testing"
	"akhcoin/p2p"
	"os"
	"time"
)

func TestApp(t *testing.T) {

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
