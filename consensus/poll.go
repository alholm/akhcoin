package consensus

import (
	logging "github.com/ipfs/go-log"
	"sort"
)

var log = logging.Logger("consensus")

type Poll struct {
	votesChan    chan string
	newRoundChan chan struct{}
	votes        map[string]int
	top          sorted
	maxElected   int
}

type Candidate struct {
	id    string
	votes int
}

func NewPoll(maxElected int) *Poll {
	votes := make(map[string]int)
	top := make(sorted, 0, maxElected)

	poll := &Poll{votesChan: make(chan string), newRoundChan: make(chan struct{}), votes: votes, top: top, maxElected: maxElected}

	go poll.startListening()

	return poll
}

func (p *Poll) startListening() {
	for {
		select {
		case candidate := <-p.votesChan:
			p.votes[candidate]++
			votesN := p.votes[candidate]

			p.insert(Candidate{candidate, votesN})

			log.Debugf("-> %s = %d ; %v", candidate, votesN, p.top)

		case <-p.newRoundChan:
			//TODO consider clearing by range deletion to decrease GC load
			p.votes = make(map[string]int)
			p.top = make(sorted, 0, p.maxElected)
		}
	}
}

type sorted []Candidate

func (s sorted) minVotes() int {
	if len(s) == cap(s) {
		return s[len(s)-1].votes
	}
	return 0
}

func (s sorted) Len() int {
	return len(s)
}

func (s sorted) Less(i, j int) bool {
	return s[i].votes > s[j].votes
}

func (s sorted) Swap(i, j int) {
	t := s[i]
	s[i] = s[j]
	s[j] = t

}

func (p *Poll) insert(newCandidate Candidate) {
	defer sort.Sort(p.top)
	if len(p.top) == 0 {
		p.top = append(p.top, newCandidate)
	} else {
		if len(p.top) < p.maxElected || newCandidate.votes > p.top[p.maxElected-1].votes {
			for i := 0; i < len(p.top); i++ {
				if p.top[i].id == newCandidate.id {
					p.top[i] = newCandidate
					return
				}
			}
			if len(p.top) < p.maxElected {
				p.top = append(p.top, newCandidate)
			} else {
				p.top[p.maxElected-1] = newCandidate
			}
		}
	}
}

func (p *Poll) IsElected(candidate string) (result bool) {
	if len(p.top) == 0 {
		return
	}
	votesN := p.votes[candidate]
	lastCandidateVotes := p.top.minVotes()

	if votesN >= lastCandidateVotes {
		result = true
	}
	return
}

func (p *Poll) SubmitVoteFor(candidate string) (err error) {
	//TODO
	//if no active round err = ...; return
	p.votesChan <- candidate
	return
}

//func (p *Poll) Size() int {
//	return len(p.votes)
//}

func (p *Poll) StartNewRound() {
	p.newRoundChan <- struct{}{}
}
