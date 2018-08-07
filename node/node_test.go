package node

import (
	"github.com/alholm/akhcoin/blockchain"
	"github.com/alholm/akhcoin/consensus"
	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/spf13/viper"
	"testing"
	"time"
)

func TestAkhNode_switchToLongest(t *testing.T) {

	viper.Set("poll.period", int64(50*time.Millisecond))
	viper.Set("poll.epsilon", int64(1*time.Millisecond))
	viper.Set("poll.maxDelegates", 3)

	var nodes [3]*AkhNode

	for i := 0; i < 3; i++ {
		nodes[i] = startRandomNode(9654 + i)
	}

	time.Sleep(100 * time.Millisecond)

	/* scenario 1: 3 producers, 1st forked. Everyone are honest

	chain 0: [2] [0] [_] [_] [0] [V]
	                          |   A - has to switch
	        must not switch - V   |
	chain 1: [2] [_] [1] [2] [X] [1]
	chain 2: [2] [_] [1] [2] [_] [1]
	*/

	//3
	forkStart, _ := nodes[2].Produce()
	nodes[0].attach(forkStart.BlockData)
	nodes[1].attach(forkStart.BlockData)

	//1
	nodes[0].Produce()
	time.Sleep(50 * time.Millisecond)

	//2
	b1, _ := nodes[1].Produce()
	nodes[2].attach(b1.BlockData)
	time.Sleep(50 * time.Millisecond)

	//3
	b2, _ := nodes[2].Produce()
	nodes[1].attach(b2.BlockData)
	time.Sleep(50 * time.Millisecond)

	//1
	b3, _ := nodes[0].Produce()
	time.Sleep(50 * time.Millisecond)

	//attempt to convince others to switch to minor fork
	nodes[1].switchToLongest(b3.BlockData, nodes[0].Host.ID())

	//2
	forkEnd, _ := nodes[1].Produce()
	//nodes[2].attach(forkEnd.BlockData)
	nodes[0].switchToLongest(forkEnd.BlockData, nodes[1].Host.ID())

	f1 := nodes[0].Head
	f2 := nodes[1].Head

	for f1.Hash != forkStart.Hash {
		if f1.Hash != f2.Hash {
			t.Error("did not switch")
			break
		}
		f1 = f1.Parent
		f2 = f2.Parent
	}

	/* scenario 2 - fraud: 3rd delegate forged incorrect fork

	chain 1: [2] [_] [1]   [X]
						    A - must not accept
			                |
	chain 2: [2] [_] [2]   [2] or
	         [2] [_] [2][2][2] (more frequently produced blocks)
	This is the only way to misbehave, dishonest delegate can't forge falsified fork with blocks from other delegates
	because its signatures will be wrong.
	*/

	//3 - in parallel with 2
	nodes[2].Produce()
	time.Sleep(50 * time.Millisecond)
	forkEnd, _ = nodes[2].Produce()
	nodes[1].switchToLongest(forkEnd.BlockData, nodes[2].Host.ID())

	if nodes[1].Head.Hash == forkEnd.Hash {
		t.Error("switched to falsified fork when must not")
	}

	for i := 0; i < 3; i++ {
		nodes[i].Host.Close()
	}
}

func TestInitialBlockDownload(t *testing.T) {
	logging.SetLogLevel("main", "DEBUG")

	period := int64(1000 * time.Millisecond)
	viper.Set("poll.period", period)
	viper.Set("poll.epsilon", int64(10*time.Millisecond))
	viper.Set("poll.maxDelegates", 3)

	var nodes [3]*AkhNode
	for i := 0; i < 3; i++ {
		nodes[i] = startRandomNode(10765 + i)
	}
	time.Sleep(200 * time.Millisecond) //waiting for mdns

	time.Sleep(consensus.UntilNext(period))
	time.Sleep(200 * time.Millisecond)
	nodes[1].Pay(nodes[0].Host.ID().Pretty(), 42)
	nodes[2].Pay(nodes[1].Host.ID().Pretty(), 24)
	time.Sleep(100 * time.Millisecond)

	l := len(nodes[0].transactionsPool)
	if l != 2 {
		t.Errorf("%d transactions in pull, has to be 2", l)
	}

	time.Sleep(consensus.UntilNext(period))
	time.Sleep(30 * time.Millisecond)
	nodes[0].Vote(nodes[1].Host.ID().Pretty())
	time.Sleep(30 * time.Millisecond)

	nodes[1].Vote(nodes[2].Host.ID().Pretty())
	time.Sleep(30 * time.Millisecond)
	nodes[2].Vote(nodes[0].Host.ID().Pretty())
	time.Sleep(30 * time.Millisecond)

	l = len(nodes[0].votesPool)
	if l != 3 {
		t.Fatalf("%d votes in pull, has to be 3", l)
	}

	time.Sleep(consensus.UntilNext(period))

	time.Sleep(time.Duration(2 * period)) //3 blocks produced

	newNode := startRandomNode(10765 + 3)

	time.Sleep(100 * time.Millisecond)
	time.Sleep(consensus.UntilNext(period)) //4th block produced
	time.Sleep(100 * time.Millisecond)

	for i := 0; i < 3; i++ {
		if nodes[i].Head.Hash != newNode.Head.Hash {
			t.Fail()
			break
		}
	}

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
