package balances

import (
	"akhcoin/blockchain"
	"sort"
)

type Balances struct {
	m            *map[string]uint64
	putChan      chan blockchain.Transaction
	getChan      chan string
	responseChan chan uint64
	rewardChan   chan struct {
		string
		uint
	}
}

func NewBalances() *Balances {
	m := make(map[string]uint64, 100) //magic constant
	put := make(chan blockchain.Transaction)
	getChan := make(chan string)
	responseChan := make(chan uint64)
	rewardChan := make(chan struct {
		string
		uint
	})
	b := &Balances{&m, put, getChan, responseChan, rewardChan}
	go func(b *Balances) {
		for {
			select {
			case t := <-b.putChan:
				(*b.m)[t.GetSigner()] -= t.Amount
				(*b.m)[t.Recipient] += t.Amount
			case r := <-b.rewardChan:
				(*b.m)[r.string] += uint64(r.uint)
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

func (b *Balances) SubmitReward(receiver string, amount uint) {
	b.rewardChan <- struct {
		string
		uint
	}{receiver, amount}
}

func (b *Balances) Get(peerID string) uint64 {
	b.getChan <- peerID
	return <-b.responseChan
}

type ByTimestamp []blockchain.Transaction

func (t ByTimestamp) Len() int      { return len(t) }
func (t ByTimestamp) Swap(i, j int) { t[i], t[j] = t[j], t[i] }

func (t ByTimestamp) Less(i, j int) bool { return t[i].GetTimestamp() < t[j].GetTimestamp() }

//TODO far from optimal
func (b *Balances) CollectValidTxns(transactions []blockchain.Transaction, skipInvalid bool) []blockchain.Transaction {
	sort.Sort(ByTimestamp(transactions))

	result := make([]blockchain.Transaction, len(transactions))
	copy(result, transactions)

	tempMap := make(map[string]uint64, len(transactions))
	for _, t := range transactions {
		tempMap[t.GetSigner()] = b.Get(t.GetSigner())
	}

	for i := 0; i < len(result); i++ {
		t := result[i]
		if tempMap[t.GetSigner()] >= t.Amount {
			tempMap[t.GetSigner()] -= t.Amount
			tempMap[t.Recipient] += t.Amount
		} else if skipInvalid {
			result = append(result[:i], result[i+1:]...)
			i--
		} else {
			return []blockchain.Transaction{}
		}
	}

	return result
}
