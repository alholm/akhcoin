package main

import (
	. "akhcoin/blockchain"
	"fmt"
	"akhcoin/p2p"
	"bytes"
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
	Genesis          *Block
	Head             *Block
}

func (*AkhNode) Vote(sign string, addr string) {
	panic("implement me")
}

func (node *AkhNode) Produce() (block *Block, err error) {
	pool := node.transactionsPool
	if len(pool) == 0 {
		err = fmt.Errorf("no transactions in pool, no block needed")
		return
	}
	privateKey := node.Host.Peerstore().PrivKey(node.Host.ID())
	block = NewBlock(privateKey, node.Head, pool)
	node.Head = block

	//TODO ATTENTION! RACE CONDITION, has to be guarded
	node.transactionsPool = pool[:0]

	log.Infof("%s: New Block hash = %s, error: %s\n", node.Host.ID().Pretty(), block.Hash, err)

	return
}

func (node *AkhNode) Announce(block *Block) (err error) {
	node.Host.PublishBlock(block)
	return nil
}

func (node *AkhNode) ReceiveTransaction(t Transaction) {
	verified, _ := t.Verify()
	log.Debugf("Txn received: %s, VERIFIED=%t\n", &t, verified)
	//TODO ATTENTION! RACE CONDITION
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

func NewAkhNode(port int, privateKey []byte) (node *AkhNode) {
	genesis := CreateGenesis()
	transactionPool := make([]Transaction, 0, 100) //magic constant

	host := p2p.StartHost(port, privateKey)

	node = &AkhNode{
		transactionsPool: transactionPool,
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

	//host.DumpHostInfo()
	host.DiscoverPeers()
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
