package consensus

import (
	"akhcoin/blockchain"
	"fmt"
	"time"
)

//period - time between blocks production in seconds, ttp - time to produce ticker
func StartProduction(poll *Poll, id string, period int) (ttpChan chan struct{}) {
	poll.period = int64(period) * int64(time.Second)
	ttpChan = make(chan struct{})

	go func(ttpChan chan struct{}) {

		for {
			time.Sleep(untilNext(poll.period))
			if myTurn(id, poll) {
				ttpChan <- struct{}{}
			}
		}

	}(ttpChan)

	return ttpChan
}

func untilNext(period int64) time.Duration {
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

func (p *Poll) IsValid(block *blockchain.Block, receivedAt int64) (valid bool, err error) {
	requiredTS, slot := p.getSlotAt(receivedAt)

	position := p.GetPosition(block.Signer)
	if position != slot {
		//TODO punishment
		return false, fmt.Errorf("not in required slot: %d, producer position = %d", slot, position)
	}
	diff := requiredTS - block.TimeStamp
	if diff < 0 {
		diff = -diff
	}
	valid = diff < int64(time.Second)
	if !valid {
		err = fmt.Errorf("incorrect timestamp: %d. Required: %d ± 1 second", block.TimeStamp, requiredTS)
	}
	return
}
