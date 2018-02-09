package consensus

import (
	"time"
	"akhcoin/blockchain"
)

func StartProduction(poll *Poll, id string, period int) (ttpChan chan struct{}) {
	ttpChan = make(chan struct{})

	go func(ttpChan chan struct{}) {

		timeToStart := period - blockchain.CurrentTime().Second()%period
		sleep(timeToStart)

		for {

			position := poll.GetPosition(id)

			if position == -1 {
				sleep(period)
				continue
			}

			//elected, wait for our turn
			sleep(position * period)

			//start ticking each our turn in round
			for ; poll.IsElected(id); {

				ttpChan <- struct{}{}

				sleep(poll.GetMaxElected() * period)
			}

		}
	}(ttpChan)

	return ttpChan
}

func sleep(seconds int) {
	time.Sleep(time.Duration(seconds) * time.Second)
}
