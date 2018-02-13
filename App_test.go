package main

import (
	"akhcoin/blockchain"
	"akhcoin/p2p"
	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-crypto"
	"os"
	"testing"
	"time"
)

func init() {
	logging.SetLogLevel("main", "DEBUG")
	logging.SetLogLevel("p2p", "DEBUG")
	//logging.SetLogLevel("mdns", "DEBUG")
}

func TestInitialBlockDownload(t *testing.T) {

	var nodes [3]*AkhNode
	for i := 0; i < 3; i++ {
		nodes[i] = startRandomNode(10765 + i)
	}
	time.Sleep(1100 * time.Millisecond) //waiting for mdns

	nodes[1].testPay()
	nodes[2].testPay()

	time.Sleep(500 * time.Millisecond)

	l := len(nodes[0].transactionsPool)
	if l != 2 {
		t.Errorf("%d transactions in pull, has to be 2", l)
	}

	nodes[0].Produce()

	for i := 0; i < 2; i++ {
		nodes[0].Produce()
		nodes[1].Produce()
	}

	nodes[2].initialBlockDownload()

	for i := 0; i < 3; i++ {
		nodes[i].Host.Close()
	}
}
func startRandomNode(p int) *AkhNode {
	private, _, _ := blockchain.NewKeys()
	privateBytes, _ := crypto.MarshalPrivateKey(private)
	node := NewAkhNode(p, privateBytes)
	return node
}

func TestAkhNode_Announce(t *testing.T) {
	os.Remove(p2p.HostsInfoPath)

	var nodes [3]*AkhNode
	for i := 0; i < 3; i++ {
		nodes[i] = startRandomNode(9654 + i)
	}

	time.Sleep(1100 * time.Millisecond) //waiting for mdns

	nodes[1].testPay()
	nodes[2].testPay()

	time.Sleep(500 * time.Millisecond)

	block, err := nodes[0].Produce()
	if err != nil {
		t.Error(err)
	}
	nodes[0].Announce(block)

	time.Sleep(500 * time.Millisecond)

	if nodes[1].Head.Hash != block.Hash || nodes[2].Head.Hash != block.Hash {
		t.Error("Announced block wasn't consumed")
	}

	l := len(nodes[1].transactionsPool)
	if l > 0 {
		t.Errorf("%d transactions in pull, has to be 0", l)
	}

}
