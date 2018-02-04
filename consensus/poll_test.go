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
	poll := NewPoll(5)

	winners := map[string]int{"winner": 20, "second": 19, "thirdd": 17, "forthh": 11, "fifthh": 10}
	losers := map[string]int{"loser1": 1, "loser2": 9, "loser3": 5}

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
		go p.SubmitVoteFor(candidate)
	}
}
