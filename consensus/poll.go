package consensus

import (
	logging "github.com/ipfs/go-log"
)

var log = logging.Logger("consensus")

type Poll struct {
	votesChan    chan string
	newRoundChan chan struct{}
	votes        map[string]int
	top          []Candidate
	maxElected   int
}

type Candidate struct {
	id    string
	votes int
}

func NewPoll(maxElected int) *Poll {
	votes := make(map[string]int)
	top := make([]Candidate, 0, maxElected)

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
			p.top = make([]Candidate, 0, p.maxElected)
		}
	}
}

//Returns minimal number of votes required to be elected in current round, i.e number of votes for last candidate
func (p *Poll) minVotes() int {
	if len(p.top) == cap(p.top) {
		return p.top[len(p.top)-1].votes
	}
	return 0
}

func (p *Poll) insert(newCandidate Candidate) {

	if len(p.top) == p.maxElected && newCandidate.votes <= p.top[p.maxElected-1].votes {
		return
	}

	i := getPosition(p.top, newCandidate)
	if i != -1 {
		p.top[i] = newCandidate
	} else if len(p.top) < p.maxElected {
		p.top = append(p.top, newCandidate)
		i = len(p.top) - 1
	} else {
		i = p.maxElected - 1
		p.top[i] = newCandidate
	}

	for j := 0; j < i; j++ {
		if p.top[j].votes < newCandidate.votes {
			temp := p.top[j]
			p.top[j] = newCandidate
			p.top[i] = temp
			break
		}
	}

}

//can be optimised with binary search, but probably doesn't worth it as max len(top) usually <= 51
func getPosition(top []Candidate, candidate Candidate) int {
	position := -1
	for i := 0; i < len(top); i++ {
		if top[i].id == candidate.id {
			position = i
			break
		}
	}
	return position
}

func (p *Poll) IsElected(candidate string) (result bool) {
	if len(p.top) == 0 {
		return
	}
	votesN := p.votes[candidate]
	if votesN >= p.minVotes() {
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

func (p *Poll) StartNewRound() {
	p.newRoundChan <- struct{}{}
}
