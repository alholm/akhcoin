package main

import (
	. "akhcoin/blockchain"
	"akhcoin/p2p"
	"bytes"
	"math/rand"
	"sync"
	"time"

	"akhcoin/consensus"
	"fmt"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/libp2p/go-libp2p-peer"
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

func NewAkhNode(port int, privateKey []byte) (node *AkhNode) {
	genesis := CreateGenesis()
	transactionPool := make([]Transaction, 0, 100) //magic constant

	host := p2p.StartHost(port, privateKey, true)

	node = &AkhNode{
		transactionsPool: transactionPool,
		poll:             consensus.NewPoll(3, 1, 20*time.Second, genesis.TimeStamp),
		Genesis:          genesis,
		Head:             genesis,
		Host:             host,
	}

	brp := &p2p.BlockStreamHandler{Head: node.Head}
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
//TODO retransmit valid block
func (node *AkhNode) Receive(bd BlockData) {
	if bd.Hash == node.Head.Hash {
		return
	}

	//filter outdated blocks
	valid, err := node.poll.IsValid(&bd, GetTimeStamp())

	log.Debugf("Block received: %s, valid: %v\n", bd.Hash, valid)
	if !valid {
		log.Error(err)
		return
	}

	verified, err := Verify(&bd)

	log.Debugf("Block received: %s, verified: %v\n", bd.Hash, verified)
	if !verified {
		log.Error(err)
		return
	}

	if bd.ParentHash == node.Head.Hash {
		node.Attach(bd)
		node.adjustPool(bd)
	} else {
		//switch to the longest chain if there is one, decline otherwise
		node.switchToLongest(bd)
		//TODO
		//node.adjustPool(new fork)
	}
}

//TODO is it safe not to validate whether block was produced by elected at that time delegate?
//TODO use blockData pointers to avoid excess stack usage
func (node *AkhNode) switchToLongest(bd BlockData) {
	myForkLen := 0
	hisForkLen := 0

	myBlock := node.Head
	hisBlock := &Block{BlockData: bd}

	for {

		for hisBlock.TimeStamp > myBlock.TimeStamp {
			var err error
			hisBlock, err = node.getParent(hisBlock)
			if err != nil {
				log.Error(err)
				return
			}
			_, err = node.isValidParent(hisBlock)
			if err != nil {
				log.Error(err)
				return
			}

			hisForkLen++
		}

		for myBlock.TimeStamp > hisBlock.TimeStamp && myBlock != node.Genesis {
			myBlock = myBlock.Parent
			myForkLen++
		}

		//myBlock and hisBlock hashes can not be different at this point, as all blocks were verified
		//timestamps of the block fork started from are exactly the same as blocks are identical
		if hisBlock.TimeStamp == myBlock.TimeStamp {
			break
		}
	}
	if myForkLen >= hisForkLen { //we are on the longest chain
		return
	}

	node.Head = myBlock
	for hisBlock != nil {
		node.Attach(hisBlock.BlockData)
		hisBlock = hisBlock.Next
	}
}

func (node *AkhNode) getParent(block *Block) (parent *Block, err error) {
	bd, err := node.Host.GetBlock(node.getRandomPeer(), block.ParentHash)
	if err != nil {
		return
	}
	parent = &Block{BlockData: *bd, Next: block}
	block.Parent = parent
	return
}

//TODO timestamps safe comparison
func (node *AkhNode) isValidParent(block *Block) (valid bool, err error) {
	if block.Next.ParentHash != block.Hash || block.Next.TimeStamp-block.TimeStamp < node.poll.Period() /*±1000*/ {
		err = fmt.Errorf("invalid fork")
		return
	}
	valid, err = Verify(&block.BlockData)
	return
}

func (node *AkhNode) Attach(bd BlockData) {
	block := &Block{BlockData: bd, Parent: node.Head}
	node.Head.Next = block
	node.Head = block
}

func (node *AkhNode) adjustPool(bd BlockData) {
	node.Lock()
	defer node.Unlock()
	for _, t := range bd.Transactions {
		for j, y := range node.transactionsPool {
			if bytes.Equal(y.Sign, t.T.Sign) {
				//delete
				node.transactionsPool = append(node.transactionsPool[:j], node.transactionsPool[j+1:]...)
				break
			}
		}
	}
}

func (node *AkhNode) ReceiveVote(v Vote) {
	verified, err := v.Verify()
	log.Debugf("Vote received: %s, VERIFIED=%t\n", &v, verified)
	if err != nil {
		log.Warningf("Invalid vote received: %s\n", err)
		return
	}

	err = node.poll.SubmitVote(v)
	if err != nil {
		log.Errorf("Failed to submit vote: %s\n", err)
	}
}

func (node *AkhNode) Produce() (block *Block, err error) {
	node.Lock()
	defer node.Unlock()
	pool := node.transactionsPool
	privateKey := node.Host.Peerstore().PrivKey(node.Host.ID())
	block = NewBlock(privateKey, node.Head, pool)
	node.Head = block

	node.transactionsPool = pool[:0]

	log.Infof("%s: New Block hash = %s\n", node.Host.ID().Pretty(), block.Hash)

	return
}

func (node *AkhNode) Announce(block *Block) (err error) {
	node.Host.PublishBlock(block)
	return nil
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

func (node *AkhNode) GetPrivate() crypto.PrivKey {
	return node.Host.Peerstore().PrivKey(node.Host.ID())
}

func (node *AkhNode) initialBlockDownload() {
	for _, peerID := range node.Host.Peerstore().Peers() {
		log.Debugf("%s requesting block from %s\n", node.Host.ID().Pretty(), peerID.Pretty())
		//block, err := node.Host.GetBlock(peerID)
		//if err != nil {
		//	log.Error(err)
		//	continue
		//}
		//for block != nil {
		//	valid, err := Verify(block)
		//	if !valid {
		//		log.Warning(err)
		//		//TODO
		//	}
		//	node.Attach(block)
		//	block = block.Next
		//}
	}
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
