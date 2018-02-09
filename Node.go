package main

import (
	. "akhcoin/blockchain"
	"akhcoin/p2p"
	"bytes"
	"math/rand"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-crypto"
	"github.com/libp2p/go-libp2p-peer"
	"akhcoin/consensus"
)

type Node interface {
	Vote(sign string, addr string)

	Produce() (*Block, error)

	Receive(*Block)

	ReceiveTransaction(t Transaction)
}

type AkhNode struct {
	Host             p2p.AkhHost
	transactionsPool []Transaction //TODO avoid duplication (can't just use map of T as T has byte arrays which don't define equity
	poll             *consensus.Poll
	Genesis          *Block
	Head             *Block
	sync.Mutex
}

func (node *AkhNode) Vote(peerIdStr string) {
	var peerId peer.ID
	if len(peerIdStr) == 0 {
		peerId = node.getRandomPeer()
		if &peerId == nil {
			return
		}
	} else {
		var err error
		peerId, err = peer.IDB58Decode(peerIdStr)
		if err != nil {
			log.Error(err)
			return
		}
	}

	vote := NewVote(node.GetPrivate(), peerId)
	node.Host.PublishVote(vote)
}

func (node *AkhNode) Produce() (block *Block, err error) {
	pool := node.transactionsPool
	privateKey := node.Host.Peerstore().PrivKey(node.Host.ID())
	block = NewBlock(privateKey, node.Head, pool)
	node.Head = block

	//TODO ATTENTION! RACE CONDITION, has to be guarded
	node.transactionsPool = pool[:0]

	log.Infof("%s: New Block hash = %s\n", node.Host.ID().Pretty(), block.Hash)

	return
}

func (node *AkhNode) Announce(block *Block) (err error) {
	node.Host.PublishBlock(block)
	return nil
}

func (node *AkhNode) ReceiveTransaction(t Transaction) {
	verified, err := t.Verify()
	log.Debugf("Txn received: %s, VERIFIED=%t\n", &t, verified)
	if err != nil {
		log.Warningf("Invalid transaction received: %s\n", err)
		return
	}

	node.addTransactionToPool(t)
}

//Synchronous operation, consider using channels
func (node *AkhNode) addTransactionToPool(t Transaction) {
	node.Lock()
	defer node.Unlock()
	node.transactionsPool = append(node.transactionsPool, t)
}

//TODO think of reaction to invalid block
func (node *AkhNode) Receive(bd BlockData) {
	block := &Block{BlockData: bd, Parent: node.Head}
	verified, err := Validate(block, node.Head)
	log.Debugf("Block received: %s, VERIFIED=%t\n", bd.Hash, verified)

	if err != nil {
		log.Error(err)
		return
	}
	node.Attach(block)

	node.Lock()
	defer node.Unlock()
	for _, t := range bd.Transactions {
		for j, y := range node.transactionsPool {
			if bytes.Equal(y.Sign, t.T.Sign) {
				//delete
				node.transactionsPool = append(node.transactionsPool[:j], node.transactionsPool[j+1:]...)
			}
		}
	}
}

func (node *AkhNode) Attach(b *Block) {
	node.Head.Next = b
	node.Head = b
}

func (node *AkhNode) ReceiveVote(v Vote) {
	verified, err := v.Verify()
	log.Debugf("Vote received: %s, VERIFIED=%t\n", &v, verified)
	if err != nil {
		log.Warningf("Invalid vote received: %s\n", err)
		return
	}

	err = node.poll.SubmitVoteFor(v.Candidate)
	if err != nil {
		log.Errorf("Failed to submit vote: %s\n", err)
	}
}

func (node *AkhNode) GetPrivate() crypto.PrivKey {
	return node.Host.Peerstore().PrivKey(node.Host.ID())
}

func NewAkhNode(port int, privateKey []byte) (node *AkhNode) {
	genesis := CreateGenesis()
	transactionPool := make([]Transaction, 0, 100) //magic constant

	host := p2p.StartHost(port, privateKey)

	node = &AkhNode{
		transactionsPool: transactionPool,
		poll:             consensus.NewPoll(3),
		Genesis:          genesis,
		Head:             genesis,
		Host:             host,
	}

	brp := &p2p.BlockStreamHandler{Genesis: genesis}
	host.AddStreamHandler(brp)

	trp := &p2p.TransactionStreamHandler{ProcessResult: node.ReceiveTransaction}
	host.AddStreamHandler(trp)

	abrp := &p2p.AnnouncedBlockStreamHandler{ProcessResult: node.Receive}
	host.AddStreamHandler(abrp)

	vrp := &p2p.VoteStreamHandler{ProcessResult: node.ReceiveVote}
	host.AddStreamHandler(vrp)

	//host.DumpHostInfo()
	host.DiscoverPeers()

	ttpChan := consensus.StartProduction(node.poll, node.Host.ID().Pretty(), 10)

	go func() {
		for range ttpChan {
			block, err := node.Produce()
			if err != nil {
				// log.Error(err)
				continue
			}
			err = node.Announce(block)
			if err != nil {
				log.Error(err)
			}
		}
	}()

	return
}

func (node *AkhNode) initialBlockDownload() {
	for _, peerID := range node.Host.Peerstore().Peers() {
		log.Debugf("%s requesting block from %s\n", node.Host.ID().Pretty(), peerID.Pretty())
		block, err := node.Host.GetBlock(peerID, singleBlockCallback)
		if err != nil {
			log.Error(err)
			continue
		}
		for block != nil {
			valid, err := Validate(block, node.Head)
			if !valid {
				log.Warning(err)
				//TODO
			}
			node.Attach(block)
			block = block.Next
		}
	}
}

func singleBlockCallback(blockData interface{}) {
}

func (node *AkhNode) testPay() {
	peer := node.getRandomPeer()

	if peer == "" {
		return
	}
	s := rand.Uint64()

	private := node.GetPrivate()
	t := Pay(private, peer, s)

	log.Debugf("Just created txn: %s\n", t)
	node.Host.PublishTransaction(t)
}

func (node *AkhNode) getRandomPeer() peer.ID {
	peerIDs := node.Host.Peerstore().Peers()
	if len(peerIDs) <= 1 {
		log.Debugf("TEMP: no peers")
		return ""
	}
	rand.Seed(time.Now().UnixNano())
	i := rand.Intn(len(peerIDs) - 1)

	return peerIDs[i]
}
