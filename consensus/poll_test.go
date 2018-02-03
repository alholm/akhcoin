package consensus

import (
	"testing"
	"time"
	logging "github.com/ipfs/go-log"
	"sync"
)

func init() {
	logging.SetLogLevel("consensus", "DEBUG")
}

func TestPoll_IsElected(t *testing.T) {
	poll := NewPoll(3)
	var wg sync.WaitGroup
	wg.Add(5)
	go poll.voteForNTimes("winner", 15, &wg)
	go poll.voteForNTimes("second", 13, &wg)
	go poll.voteForNTimes("thirdd", 11, &wg)
	go poll.voteForNTimes("loser1", 1, &wg)
	go poll.voteForNTimes("loser2", 10, &wg)

	wg.Wait()
	time.Sleep(100 * time.Millisecond)

	if !poll.IsElected("winner") {
		t.Fatal("Winner not elected\n", poll.top)
	}

	if !poll.IsElected("second") {
		t.Fatal("Second not elected\n", poll.top)
	}

	if !poll.IsElected("thirdd") {
		t.Fatal("Third not elected\n", poll.top)
	}

	if poll.IsElected("loser1") {
		t.Fatal("loser1 is elected\n", poll.top)
	}

	if poll.IsElected("loser2") {
		t.Fatal("loser2 is elected\n", poll.top)
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
		go p.SubmitVoteFor(candidate)
	}
}
