package main

import (
	logging "github.com/ipfs/go-log"
	"github.com/spf13/viper"
	"testing"
	"time"
)

func init() {
	logging.SetLogLevel("consensus", "DEBUG")
	logging.SetLogLevel("p2p", "DEBUG")
}

func TestAkhNode_switchToLongest(t *testing.T) {

	viper.Set("poll.period", 50000)
	viper.Set("poll.epsilon", 100)

	var nodes [3]*AkhNode

	for i := 0; i < 3; i++ {
		nodes[i] = startRandomNode(9654 + i)
	}

	time.Sleep(100 * time.Millisecond)

	/* scenario 1: 3 producers, 1st forked

	chain 0: [2] [0] [_] [_] [0]  S - has to switch
	chain 1: [2] [_] [1] [2] [_] [1]
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
	nodes[0].Produce()
	time.Sleep(50 * time.Millisecond)

	//2
	forkEnd, _ := nodes[1].Produce()
	//nodes[2].attach(forkEnd.BlockData)
	nodes[0].switchToLongest(forkEnd.BlockData, nodes[1].Host.ID())

	time.Sleep(100 * time.Millisecond)

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

}
