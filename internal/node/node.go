package node

import (
	. "github.com/alholm/akhcoin/pkg/blockchain"
	"github.com/alholm/akhcoin/internal/p2p"
	"sync"
	"time"

	"fmt"
	"github.com/alholm/akhcoin/pkg/balances"
	"github.com/alholm/akhcoin/pkg/consensus"
	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/libp2p/go-libp2p-peer"
	"github.com/spf13/viper"
)

var log = logging.Logger("main")

type AkhNode struct {
	Host             p2p.AkhHost
	transactionsPool []Transaction //TODO avoid duplication (can't just use map of T as T has byte arrays which don't define equity
	votesPool        []Vote
	poll             *consensus.Poll
	Genesis          *Block
	Head             *Block
	balances         *balances.Balances
	sync.Mutex
}

func NewAkhNode(port int, privateKey []byte) (node *AkhNode) {
	genesis := CreateGenesis()
	transactionPool := make([]Transaction, 0, 100) //magic constant
	votesPool := make([]Vote, 0, 100)              //magic constant

	host := p2p.StartHost(port, privateKey, true)

	node = &AkhNode{
		transactionsPool: transactionPool,
		votesPool:        votesPool,
		poll: consensus.NewPoll(viper.GetInt("poll.MaxDelegates"), viper.GetInt("poll.MaxVotes"),
			viper.GetDuration("poll.freezePeriod")*time.Second, genesis.GetTimestamp()),
		Genesis:  genesis,
		Head:     genesis,
		balances: balances.NewBalances(),
		Host:     host,
	}

	brp := &p2p.BlockStreamHandler{Head: &node.Head}
	host.AddStreamHandler(brp)

	trp := &p2p.TransactionStreamHandler{ProcessResult: node.ReceiveTransaction}
	host.AddStreamHandler(trp)

	abrp := &p2p.AnnouncedBlockStreamHandler{ProcessResult: node.Receive}
	host.AddStreamHandler(abrp)

	vrp := &p2p.VoteStreamHandler{ProcessResult: node.ReceiveVote}
	host.AddStreamHandler(vrp)

	host.DiscoverPeers()

	ttpChan := consensus.StartProduction(node.poll, node.Host.ID().Pretty())

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

//Check whether transaction was created during current production period
//TODO Txns created at the end of production period may get lost
func (node *AkhNode) timeValid(s Signable) bool {
	currentTimeStamp := GetTimeStamp()
	currentSlotStart := node.poll.GetCurrentSlotStart(currentTimeStamp)
	return s.GetTimestamp() > currentSlotStart && s.GetTimestamp() < currentTimeStamp
}

func (node *AkhNode) ReceiveTransaction(t Transaction) {
	verified, err := t.Verify()
	timeValid := node.timeValid(&t)

	log.Debugf("Txn received: %s, Verified=%t, time valid: %t\n", &t, verified, timeValid)
	if err != nil {
		log.Warningf("Invalid transaction received: %s\n", err)
		return
	}
	if !timeValid {
		log.Warningf("Transaction received at wrong time")
		return
	}

	node.addTransactionToPool(t)
}

//TODO synchronize
func (node *AkhNode) addTransactionToPool(t Transaction) {
	node.transactionsPool = append(node.transactionsPool, t)
}

//TODO synchronize
func (node *AkhNode) addVoteToPool(v Vote) {
	node.votesPool = append(node.votesPool, v)
}

//TODO think of reaction to invalid block
//TODO retransmit valid block
func (node *AkhNode) Receive(bd BlockData, peerId peer.ID) {
	node.Lock()
	defer node.Unlock()

	if bd.Hash == node.Head.Hash {
		return
	}

	//in case we've just joined network we have no option but to trust first block we received is valid
	//and download whole chain from peer sent it.
	//Known attack here is fraud producer will change number of votes in the block, so that poll state will differ from
	//correct one and node will decline true blocks. But in this case the chain block is on may consist only of blocks
	//produced by that single misbehaved producer, which will become visible soon.

	if node.Head != node.Genesis {
		//filter outdated and misproduced blocks
		valid, err := node.poll.IsValid(&bd, GetTimeStamp())

		log.Debugf("Block received: %s, valid: %v\n", bd.Hash, valid)
		if !valid {
			log.Error(err)
			return
		}
	}
	if bd.ParentHash == node.Head.Hash {
		node.attach(bd)
	} else {
		//switch to the longest chain if there is one, decline otherwise
		node.switchToLongest(bd, peerId)
	}
}

//See Node_test for scenarios handled
func (node *AkhNode) switchToLongest(forkTip BlockData, peerId peer.ID) {
	myForkLen := 0
	hisForkLen := 0

	myBlock := node.Head
	hisBlock := &Block{BlockData: forkTip}

	for {

		for hisBlock.GetTimestamp() > myBlock.GetTimestamp() {
			var err error
			hisBlock, err = node.getParent(hisBlock, peerId)
			if err != nil {
				log.Error(err)
				return
			}
			_, err = node.isValidForkElement(hisBlock, forkTip)
			if err != nil {
				log.Error(err)
				return
			}

			hisForkLen++
		}

		for myBlock.GetTimestamp() > hisBlock.GetTimestamp() && myBlock != node.Genesis {
			myBlock = myBlock.Parent
			//TODO revert block transactions
			myForkLen++
		}

		//myBlock and hisBlock hashes can not be different at this point, as all blocks were verified
		//timestamps of the block fork started from are _exactly_ the same as blocks are identical
		if hisBlock.GetTimestamp() == myBlock.GetTimestamp() {
			break
		}
	}
	if myForkLen >= hisForkLen { //we are on the longest chain
		return
	}

	originalHead := node.Head
	node.Head = myBlock
	//TODO reconstruct poll and accounts state at this block
	for hisBlock.Next != nil {
		err := node.attach(hisBlock.Next.BlockData)
		if err != nil {
			log.Errorf("Couldn't switch to fork with tip %s: block %s invalid: %s", forkTip.Hash, hisBlock.Next.BlockData.Hash, err)
			node.Head = originalHead
			//TODO reconstruct poll and balances
			return
		}
		hisBlock = hisBlock.Next
	}
	node.adjustPools(forkTip)
}

func (node *AkhNode) getParent(block *Block, peerId peer.ID) (parent *Block, err error) {
	bd, err := node.Host.GetBlock(peerId, block.ParentHash)
	if err != nil {
		return
	}
	parent = &Block{BlockData: bd, Next: block}
	block.Parent = parent
	return
}

func (node *AkhNode) isValidForkElement(block *Block, forkTip BlockData) (valid bool, err error) {
	if block.Next.ParentHash != block.Hash || block.Next.GetTimestamp()-block.GetTimestamp() < node.poll.Period()-consensus.Epsilon {
		err = fmt.Errorf("invalid parent in incoming fork, block: %s", block.Hash)
		return
	}

	if block.Signer == forkTip.Signer {
		roundDuration := node.poll.Period() * int64(node.poll.GetMaxElected())
		if forkTip.GetTimestamp()-block.GetTimestamp() < roundDuration-consensus.Epsilon {
			err = fmt.Errorf("potential fraud: fork received from %s with block produced not in order", block.Signer)
			return
		}
	}

	return
}

func (node *AkhNode) attach(bd BlockData) (err error) {
	verified, err := bd.Verify(&node.Head.BlockData)

	log.Debugf("Block received: %s, verified: %v\n", bd.Hash, verified)
	if !verified {
		log.Error(err)
		return
	}

	err = node.updateBalances(bd)
	if err != nil {
		return err
	}
	block := &Block{BlockData: bd, Parent: node.Head}
	node.Head.Next = block
	node.Head = block

	node.adjustPools(bd)
	return
}

func (node *AkhNode) updateBalances(bd BlockData) (err error) {
	txns := make([]Transaction, 0, len(bd.Transactions))
	for _, t := range bd.Transactions {
		txns = append(txns, t)
	}

	validTxns := node.balances.CollectValidTxns(txns, false)
	if len(txns) != len(validTxns) {
		return fmt.Errorf("block %s contains incorrect transactions from balances perspective", bd.Hash)
	}

	for _, t := range bd.Transactions {
		err := node.balances.Submit(t)
		if err != nil {
			return fmt.Errorf("invalid block transaction: %s: %s", &t, err)
		}
	}

	node.balances.SubmitReward(bd.Signer, bd.Reward)
	return
}

func (node *AkhNode) adjustPools(bd BlockData) {
	//node.Lock()
	//defer node.Unlock()
	//we received new block, meaning that all transaction in pool are outdated by timeValid definition
	//for _, t := range bd.Transactions {
	//	for j, y := range node.transactionsPool {
	//		if bytes.Equal(y.Sign, t.T.Sign) {
	//			//delete
	//			node.transactionsPool = append(node.transactionsPool[:j], node.transactionsPool[j+1:]...)
	//			break
	//		}
	//	}
	//}
	node.transactionsPool = node.transactionsPool[:0]
	node.votesPool = node.votesPool[:0]
}

func (node *AkhNode) ReceiveVote(v Vote) {
	verified, err := v.Verify()
	timeValid := node.timeValid(&v)

	log.Debugf("Vote received: %s, Verified=%t, time valid: %t\n", &v, verified, timeValid)
	if err != nil {
		log.Warningf("Invalid vote received: %s\n", err)
		return
	}

	if !timeValid {
		log.Warningf("Vote received at wrong time")
		return
	}

	err = node.poll.SubmitVote(v)
	if err != nil {
		log.Errorf("Failed to submit vote: %s\n", err)
	}

	node.addVoteToPool(v)
}

func (node *AkhNode) Produce() (block *Block, err error) {
	node.Lock()
	defer node.Unlock()
	txnsPool := node.balances.CollectValidTxns(node.transactionsPool, true)
	votesPool := node.votesPool
	privateKey := node.Host.Peerstore().PrivKey(node.Host.ID())
	block = NewBlock(privateKey, node.Head, txnsPool, votesPool)
	//TODO ineffective: excess verification
	node.attach(block.BlockData)

	log.Infof("%s: New Block hash = %s\n", node.Host.ID().Pretty(), block.Hash)

	return
}

func (node *AkhNode) Announce(block *Block) (err error) {
	node.Host.PublishBlock(block)
	return nil
}

func (node *AkhNode) GetPrivate() crypto.PrivKey {
	return node.Host.Peerstore().PrivKey(node.Host.ID())
}

func (node *AkhNode) Pay(peerIdStr string, amount uint64) error {
	peerId, err := peer.IDB58Decode(peerIdStr)

	if err != nil {
		return err
	}

	private := node.GetPrivate()
	t := Pay(private, peerId, amount)

	node.Host.PublishTransaction(t)
	node.ReceiveTransaction(*t)

	return nil
}

func (node *AkhNode) Vote(peerIdStr string) error {

	peerId, err := peer.IDB58Decode(peerIdStr)
	if err != nil {
		return err
	}

	vote := NewVote(node.GetPrivate(), peerId)

	node.Host.PublishVote(vote)
	node.ReceiveVote(*vote)

	return nil
}
