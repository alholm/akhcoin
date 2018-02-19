package consensus

import (
	"akhcoin/blockchain"
	logging "github.com/ipfs/go-log"
	"github.com/spf13/viper"
	"sort"
	"time"
)

var log = logging.Logger("consensus")

func init() {
	viper.SetDefault("poll.period", int64(10*time.Second))
	viper.SetDefault("poll.epsilon", int64(1*time.Second))
}

type Poll struct {
	votesChan      chan blockchain.Vote
	candidatesChan chan struct {
		id    string
		votes int
	}
	newRoundChan chan struct{}
	votes        map[string]VoterInfo
	top          []Candidate
	maxDelegates int
	maxVotes     int
	freezePeriod time.Duration
	genesisStart int64
	period       int64
}

func (p *Poll) Period() int64 {
	return p.period
}

type Candidate struct {
	id    string
	votes int
}

type VoterInfo struct {
	votes     int
	votedFor  []string
	timeStamp int64
}

//Creates new structure that counts incoming votes and maintains list of maxDelegates top voted candidates.
//maxVotes is number of candidates one is allowed to vote for.
//freezePeriod is time required to elapse before voter can vote again
func NewPoll(maxDelegates int, maxVotes int, freezePeriod time.Duration, genesisStart int64) *Poll {
	log.Debugf("New Poll config: md = %d, mv = %d, fp = %v, p = %d", maxDelegates, maxVotes, freezePeriod, viper.GetInt64("poll.period"))
	votes := make(map[string]VoterInfo)
	top := make([]Candidate, 0, maxDelegates)

	candidatesChan := make(chan struct {
		id    string
		votes int
	}, 2)

	poll := &Poll{
		make(chan blockchain.Vote),
		candidatesChan,
		make(chan struct{}),
		votes, top, maxDelegates, maxVotes, freezePeriod,
		genesisStart, viper.GetInt64("poll.period")}

	go poll.startListening()

	return poll
}

func (p *Poll) processVote(vote blockchain.Vote) {
	voter := vote.Voter
	voterInfo := p.votes[voter]
	if time.Duration(vote.TimeStamp-voterInfo.timeStamp) < p.freezePeriod {
		//TODO extract to upper level and punish voter (DoS prevention)
		return
	}

	candidate := vote.Candidate
	for _, votedFor := range voterInfo.votedFor {
		if votedFor == candidate {
			return
		}
	}

	voterInfo.votedFor = append(voterInfo.votedFor, candidate)

	if len(voterInfo.votedFor) > p.maxVotes {
		p.submitCandidate(voterInfo.votedFor[0], -1)
		voterInfo.votedFor = append(voterInfo.votedFor[:0], voterInfo.votedFor[1:]...)
	}
	voterInfo.timeStamp = vote.TimeStamp
	p.votes[voter] = voterInfo

	p.submitCandidate(candidate, 1)

}

func (p *Poll) submitCandidate(id string, votes int) {
	go func() {
		p.candidatesChan <- struct {
			id    string
			votes int
		}{id, votes}
	}()
}

func (p *Poll) startListening() {
	for {
		select {
		case vote := <-p.votesChan:
			p.processVote(vote)

		case candidate := <-p.candidatesChan:
			candidateInfo := p.votes[candidate.id]

			candidateInfo.votes += candidate.votes
			votesN := candidateInfo.votes
			p.votes[candidate.id] = candidateInfo

			p.updateTop(Candidate{candidate.id, votesN})

			//log.Debugf("-> %s = %d ; %v", candidate, votesN, p.top)

		case <-p.newRoundChan:
			//TODO consider clearing by range deletion to decrease GC load
			p.votes = make(map[string]VoterInfo)
			p.top = make([]Candidate, 0, p.maxDelegates)
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

func (p *Poll) updateTop(newCandidate Candidate) {

	if len(p.top) == p.maxDelegates && newCandidate.votes <= p.top[p.maxDelegates-1].votes {
		return
	}

	insertedPos := getPosition(p.top, newCandidate.id)
	if insertedPos != -1 {
		p.top[insertedPos] = newCandidate
	} else if len(p.top) < p.maxDelegates {
		p.top = append(p.top, newCandidate)
		insertedPos = len(p.top) - 1
	} else {
		insertedPos = p.maxDelegates - 1
		p.top[insertedPos] = newCandidate
	}

	requiredPos := sort.Search(insertedPos, func(j int) bool { return p.top[j].votes < newCandidate.votes })

	if requiredPos != insertedPos {
		temp := p.top[requiredPos]
		p.top[requiredPos] = newCandidate
		p.top[insertedPos] = temp
	}
}

func getPosition(top []Candidate, candidateId string) int {
	position := -1
	for i := 0; i < len(top); i++ {
		if top[i].id == candidateId {
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
	votesN := p.votes[candidate].votes
	if votesN >= p.minVotes() {
		result = true
	}
	return
}

func (p *Poll) GetPosition(candidate string) int {
	if len(p.top) == 0 || !p.IsElected(candidate) {
		return -1
	}
	return getPosition(p.top, candidate)
}

func (p *Poll) SubmitVote(vote blockchain.Vote) (err error) {
	//TODO
	//if no active round err = ...; return
	p.votesChan <- vote
	return
}

func (p *Poll) StartNewRound() {
	p.newRoundChan <- struct{}{}
}

func (p *Poll) GetMaxElected() int {
	return p.maxDelegates
}
