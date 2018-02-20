package main

import (
	"akhcoin/blockchain"
	"akhcoin/consensus"
	"fmt"
	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/spf13/viper"
	"testing"
	"time"
)

func init() {
	logging.SetLogLevel("consensus", "DEBUG")
	logging.SetLogLevel("p2p", "DEBUG")

	viper.Set("poll.maxDelegates", 3)
}

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
}

func TestInitialBlockDownload(t *testing.T) {
	logging.SetLogLevel("main", "DEBUG")
	logging.SetLogLevel("p2p", "DEBUG")
	period := int64(500 * time.Millisecond)
	viper.Set("poll.period", period)
	viper.Set("poll.epsilon", int64(10*time.Millisecond))

	var nodes [3]*AkhNode
	for i := 0; i < 3; i++ {
		nodes[i] = startRandomNode(10765 + i)
	}
	time.Sleep(100 * time.Millisecond) //waiting for mdns

	nodes[1].testPay()
	nodes[2].testPay()

	time.Sleep(100 * time.Millisecond)

	l := len(nodes[0].transactionsPool)
	if l != 2 {
		t.Errorf("%d transactions in pull, has to be 2", l)
	}
	//TODO has to receive own vote
	vote1 := blockchain.NewVote(nodes[0].GetPrivate(), nodes[1].Host.ID())
	nodes[0].Host.PublishVote(vote1)
	nodes[0].poll.SubmitVote(*vote1)
	fmt.Printf("%s voted for %s\n", vote1.Voter, vote1.Candidate)

	vote2 := blockchain.NewVote(nodes[1].GetPrivate(), nodes[2].Host.ID())
	nodes[1].Host.PublishVote(vote2)
	nodes[1].poll.SubmitVote(*vote2)
	fmt.Printf("%s voted for %s\n", vote2.Voter, vote2.Candidate)

	vote3 := blockchain.NewVote(nodes[2].GetPrivate(), nodes[0].Host.ID())
	nodes[2].Host.PublishVote(vote3)
	nodes[2].poll.SubmitVote(*vote3)
	fmt.Printf("%s voted for %s\n", vote3.Voter, vote1.Candidate)

	time.Sleep(100 * time.Millisecond)

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
