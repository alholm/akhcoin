package balances

import "akhcoin/blockchain"

type Balances struct {
	m            *map[string]uint64
	putChan      chan blockchain.Transaction
	getChan      chan string
	responseChan chan uint64
}

func NewBalances() *Balances {
	m := make(map[string]uint64, 100) //magic constant
	put := make(chan blockchain.Transaction)
	getChan := make(chan string)
	responseChan := make(chan uint64)
	b := &Balances{&m, put, getChan, responseChan}
	go func(b *Balances) {
		for {
			select {
			case t := <-b.putChan:
				(*b.m)[t.Sender] -= t.Amount
				(*b.m)[t.Recipient] += t.Amount

			case a := <-b.getChan:
				responseChan <- (*b.m)[a]
			}
		}
	}(b)
	return b
}

func (b *Balances) Submit(t blockchain.Transaction) (err error) {
	b.putChan <- t
	return
}

func (b *Balances) Get(peerID string) uint64 {
	b.getChan <- peerID
	return <-b.responseChan
}
