package consensus

import (
	"akhcoin/blockchain"
	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/libp2p/go-libp2p-peer"
	"sync"
	"testing"
	"time"
)

var winners, losers map[string]int

func init() {
	logging.SetLogLevel("consensus", "DEBUG")
	winners = map[string]int{"winner": 20, "second": 19, "thirdd": 17, "forthh": 11, "fifthh": 10}
	losers = map[string]int{"loser1": 1, "loser2": 9, "loser3": 5}

}

func doElection() *Poll {
	poll := NewPoll(5, 1, 3*time.Second, getTestStartTime())
	var wg sync.WaitGroup
	wg.Add(8)
	for candidate, votes := range winners {
		go poll.voteForNTimes(candidate, votes, &wg)
	}
	for candidate, votes := range losers {
		go poll.voteForNTimes(candidate, votes, &wg)
	}
	wg.Wait()
	time.Sleep(100 * time.Millisecond)
	return poll
}

func TestPoll_IsElected(t *testing.T) {
	poll := doElection()

	for candidate := range winners {
		if !poll.IsElected(candidate) {
			t.Fatalf("%s not elected: %v\n", candidate, poll.top)
		}
	}

	for candidate := range losers {
		if poll.IsElected(candidate) {
			t.Fatalf("%s is elected: %v\n", candidate, poll.top)
		}
	}

	poll.StartNewRound()

	time.Sleep(100 * time.Millisecond)

	votesLen := len(poll.votes)
	topLen := len(poll.top)
	if votesLen != 0 && topLen != 0 {
		t.Fatalf("new round not started: votes len = %d, top len = %d", votesLen, topLen)
	}

}

func (p *Poll) voteForNTimes(candidate string, n int, wg *sync.WaitGroup) {
	defer wg.Done()
	for i := 0; i < n; i++ {
		p.submitCandidate(candidate, 1)
	}
}

func TestPoll_ProcessVote(t *testing.T) {
	poll := NewPoll(2, 2, 1*time.Second, 0)
	privates := make([]crypto.PrivKey, 3)
	peerIds := make([]peer.ID, 3)

	for i := 0; i < 3; i++ {
		private, public, _ := blockchain.NewKeys()
		peerId, _ := peer.IDFromPublicKey(public)
		privates[i] = private
		peerIds[i] = peerId
	}

	poll.SubmitVote(*blockchain.NewVote(privates[0], peerIds[1]))
	poll.SubmitVote(*blockchain.NewVote(privates[1], peerIds[2]))
	poll.SubmitVote(*blockchain.NewVote(privates[2], peerIds[0]))
	time.Sleep(10 * time.Millisecond)
	if poll.votes[peerIds[0].Pretty()].votes == 0 ||
		poll.votes[peerIds[1].Pretty()].votes == 0 ||
		poll.votes[peerIds[2].Pretty()].votes == 0 {
		t.Fatal("poll.votes filled incorrectly")
	}
	poll.SubmitVote(*blockchain.NewVote(privates[1], peerIds[0]))
	time.Sleep(10 * time.Millisecond)
	if poll.votes[peerIds[0].Pretty()].votes != 1 {
		t.Fatalf("freezePeriod ignored: %d", poll.votes[peerIds[0].Pretty()].votes)
	}
	time.Sleep(1010 * time.Millisecond)
	poll.SubmitVote(*blockchain.NewVote(privates[1], peerIds[0]))
	time.Sleep(10 * time.Millisecond)
	if poll.votes[peerIds[0].Pretty()].votes != 2 {
		t.Fatalf("wrong freezePeriod handling: %d", poll.votes[peerIds[0].Pretty()].votes)
	}

	if len(poll.votes[peerIds[1].Pretty()].votedFor) != 2 {
		t.Fatalf("voted for filled incorrectly: %v", poll.votes[peerIds[1].Pretty()].votedFor)
	}

	time.Sleep(1010 * time.Millisecond)
	vote := *blockchain.NewVote(privates[1], peerIds[1])
	poll.SubmitVote(vote) //self voting should be prevented on the upper level
	time.Sleep(10 * time.Millisecond)
	votedFor := poll.votes[peerIds[1].Pretty()].votedFor
	if len(votedFor) != 2 && votedFor[0] != peerIds[0].Pretty() && votedFor[1] != peerIds[1].Pretty() {
		t.Fatalf("voted for changed incorrectly: %v", poll.votes[peerIds[1].Pretty()].votedFor)
	}

	if poll.votes[peerIds[2].Pretty()].votes != 0 {
		t.Fatal("Vote changing didn't reflect first voted candidate")
	}
}

func getTestStartTime() int64 {
	return time.Now().UTC().UnixNano() - int64(42742*time.Millisecond)
}

