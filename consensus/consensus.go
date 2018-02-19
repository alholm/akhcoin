package consensus

import (
	"akhcoin/blockchain"
	"fmt"
	"time"
)

var Epsilon = int64(10 * time.Millisecond) //viper.GetInt64("poll.epsilon")

//period - time between blocks production in seconds, ttp - time to produce ticker
func StartProduction(poll *Poll, id string) (ttpChan chan struct{}) {
	ttpChan = make(chan struct{})

	go func(ttpChan chan struct{}) {

		for {
			time.Sleep(UntilNext(poll.period))
			if myTurn(id, poll) {
				ttpChan <- struct{}{}
			}
		}

	}(ttpChan)

	return ttpChan
}

func UntilNext(period int64) time.Duration {
	return time.Duration(period - blockchain.GetTimeStamp()%period)
}

func myTurn(myId string, poll *Poll) bool {
	position := poll.GetPosition(myId)

	if position == -1 {
		return false
	}

	_, currentSlot := poll.getSlotAt(blockchain.GetTimeStamp())

	return currentSlot == position
}

func (p *Poll) getSlotAt(timeStamp int64) (requiredTS int64, slot int) {
	sinceGenesis := timeStamp - p.genesisStart
	sinceNewRound := sinceGenesis % (p.period * int64(p.maxDelegates))
	slot = int(sinceNewRound / p.period)
	requiredTS = timeStamp - timeStamp%p.period
	return
}

//Block validation from DPoS perspective: was block produced by right candidate at the right time?
func (p *Poll) IsValid(block *blockchain.BlockData, receivedAt int64) (valid bool, err error) {
	requiredTS, slot := p.getSlotAt(receivedAt)

	position := p.GetPosition(block.Signer)
	if position != slot {
		return false, fmt.Errorf("not in required slot: %d, producer position = %d", slot, position)
	}
	diff := requiredTS - block.TimeStamp
	if diff < 0 {
		diff = -diff
	}
	valid = diff < Epsilon
	if !valid {
		err = fmt.Errorf("incorrect timestamp: %d. Required: %d ± %v", block.TimeStamp, requiredTS, Epsilon)
	}
	return
}
