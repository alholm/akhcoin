package main

import (
	. "akhcoin/blockchain"
	"fmt"
	"akhcoin/p2p"
	"flag"
	"log"
	"bufio"
	"os"
	"math/rand"
	"time"
)

type Node interface {
	Vote(sign string, addr string)

	Produce(*Block) *Block

	Receive(*Block)

	ReceiveTransaction(*Transaction)
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
	}
	privateKey := node.Host.Peerstore().PrivKey(node.Host.ID())
	block = NewBlock(privateKey, node.Head, pool)
	node.Head = block

	//TODO ATTENTION! RACE CONDITION, has to be guarded
	node.transactionsPool = pool[:0]

	return
}

func (node *AkhNode) Receive(b *Block) {
	Verify(b)
	//n.blockchain = append(n.blockchain, b)
}

func (node *AkhNode) ReceiveTransaction(t Transaction) {
	verified, _ := Verify(&t)
	log.Printf("### Txn received: %s, VERIFIED=%t\n", &t, verified)
	//TODO ATTENTION! RACE CONDITION
	node.transactionsPool = append(node.transactionsPool, t)
}

func NewAkhNode(port int) (node *AkhNode) {
	genesis := CreateGenesis()
	transactionPool := make([]Transaction, 0, 100) //magic constant

	host := p2p.StartHost(port)

	node = &AkhNode{
		transactionsPool: transactionPool,
		Genesis:          genesis,
		Head:             genesis,
		Host:             host,
	}

	brp := &p2p.BlockStreamHandler{Genesis: genesis}
	p2p.SetStreamHandler(host, brp)

	trp := &p2p.TransactionStreamHandler{ProcessResult: node.ReceiveTransaction}
	p2p.SetStreamHandler(host, trp)
	host.DumpHostInfo()
	host.DiscoverPeers()
	return
}

func main() {

	//1st launch, we didn't discover any nodes yet, so we have 3 options: (for more details see https://en.bitcoin.it/wiki/Bitcoin_Core_0.11_(ch_4):_P2P_Network)
	//1) hardcoded nodes
	//2) DNS seeding: on this stage no domains registered, skipping
	//3) User-specified on the command line

	// Parse some flags
	port := flag.Int("p", 9765, "port where to start local host")
	remotePeerAddr := *flag.String("a", "", "add peer address (format: <IP:port>)")
	remotePeerID := *flag.String("id", "", "add peer ID <format>")
	flag.Parse()

	//TODO temp to test several nodes on single machine
	if *port == 9765 {
		rand.Seed(time.Now().UnixNano())
		*port = rand.Intn(1000) + 9000
	}

	node := NewAkhNode(*port)

	if len(remotePeerAddr) > 0 && len(remotePeerID) > 0 {
		node.Host.AddPeerManually(remotePeerAddr, remotePeerID)
	}

	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter text: ")
		text, _ := reader.ReadString('\n')
		if text == "exit\n" {
			break
		} else if text == "g\n" {
			block, err := node.Produce()
			log.Printf("%s: New Block hash = %s, error: %s\n", node.Host.ID(), block.Hash, err)
		} else if text == "p\n" {
			node.testPay()
		} else {
			node.initialBlockDownload()
		}
	}

	node.Host.Close()

	//dpos.startMining

	//<-make(chan struct{}) // hang forever

}
func (node *AkhNode) testPay() {
	host := node.Host
	private := node.Host.Peerstore().PrivKey(host.ID())
	//log.Println(private)

	rand.Seed(time.Now().UnixNano())
	peerIDs := host.Peerstore().Peers()
	i := rand.Intn(len(peerIDs) - 1)
	s := rand.Uint64()

	t := Pay(private, peerIDs[i], s)

	log.Printf("### Just created txn: %s\n", t)
	host.PublishTransaction(t)
}

func (node *AkhNode) initialBlockDownload() {
	for _, peerID := range node.Host.Peerstore().Peers() {
		log.Printf("%s requesting block from %s\n", node.Host.ID(), peerID)
		block, err := node.Host.GetBlock(peerID, FuncName)
		if err != nil {
			log.Println(err)
			continue
		}
		for block != nil {
			valid := Validate(block, node.Head)
			if !valid {
				//TODO
			}
			block = block.Next
		}
	}
}
func FuncName(blockData interface{}) {
	data, err := blockData.(BlockData)
	log.Printf("##### %T: %v # %v\n", data, data, err)
	//v := reflect.ValueOf(blockData)
	//v.Kind()
	//log.Printf("##### %v # %v\n", v, v.Kind())
}
