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
	transactionsPool map[*Transaction]bool
	Genesis          *Block
	Head             *Block
}

func (*AkhNode) Vote(sign string, addr string) {
	panic("implement me")
}

func (*AkhNode) Produce(parent *Block) *Block {
	return NewBlock(parent)
}

func (node *AkhNode) Receive(b *Block) {
	Verify(b)
	//n.blockchain = append(n.blockchain, b)
}

func Verify(block *Block) error {
	fmt.Println("Block verified!")
	return nil
}

func (node *AkhNode) ReceiveTransaction(t *Transaction) {
	node.transactionsPool[t] = true
}

func NewAkhNode() *AkhNode {
	return &AkhNode{
		transactionsPool: make(map[*Transaction]bool),
		Genesis:          CreateGenesis(),
	}
}
func initBlockchain() []*Block {
	existingBlocks, size := downloadExistingBlocks()
	if size == 0 {
		existingBlocks = []*Block{CreateGenesis()}
	}
	return existingBlocks

}
func downloadExistingBlocks() ([]*Block, int) {
	return []*Block{}, 0
}

//func main() {
//
//	node := NewAkhNode()
//	block := node.Produce()
//	node.Receive(block)i
//	fmt.Printf("Chain len = %x\n", len(node.blockchain))
//	node.Vote("mySign", "someAddr")
//}

func main() {

	//1st launch, we didn't discover any nodes yet, so we have 3 options: (for more details see https://en.bitcoin.it/wiki/Bitcoin_Core_0.11_(ch_4):_P2P_Network)

	//1) hardcoded nodes, TBD TODO

	//2) DNS seeding: on this stage no domains registered, skipping

	//3) User-specified on the command line

	// Parse some flags
	port := flag.Int("p", 0, "port where to start local host")
	remotePeerAddr := flag.String("a", "", "add peer address (format: <IP:port>)")
	remotePeerID := flag.String("id", "", "add peer ID <format>")
	flag.Parse()

	node := NewAkhNode()

	if *port == 0 {
		rand.Seed(time.Now().UnixNano())
		*port = rand.Intn(1000) + 9000
	}

	host := p2p.StartHost(*port)
	p2p.SetStreamHandler(host, p2p.HandleGetBlockStream, node.Genesis)
	host.DumpHostInfo(host)
	/////

	//chain = downloadUtil.downloadExisting blocks

	node.Head = node.Genesis
	host.DiscoverPeers(*remotePeerAddr, *remotePeerID)

	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter text: ")
		text, _ := reader.ReadString('\n')
		if text == "exit\n" {
			break
		} else if text == "g\n" {
			node.Head = NewBlock(node.Head)
		} else if text == "p\n" {
			node.testPay(host)
		} else {
			node.initialBlockDownload(host)
		}
	}

	//dpos.startMining

	//<-make(chan struct{}) // hang forever

}
func (node *AkhNode) testPay(host p2p.AkhHost) {
	private := host.Peerstore().PrivKey(host.ID())
	log.Println(private)

	rand.Seed(time.Now().UnixNano())
	peerIDs := host.Peerstore().Peers()
	i := rand.Intn(len(peerIDs) - 1)
	s := rand.Uint64()

	t := Pay(private, peerIDs[i], s)

	log.Printf("### Just created txn: %s\n", t)
	host.PublishTransaction(t)
}

func (node *AkhNode) initialBlockDownload(host p2p.AkhHost) {
	for _, peerID := range host.Peerstore().Peers() {
		fmt.Printf("### requesting block from %s\n", peerID.Pretty())
		block, err := host.GetBlock(peerID)
		if err != nil {
			log.Println(err)
			continue
		}
		for block != nil {
			Validate(block, node.Head)
			fmt.Printf("Recieved block, hash: %s\n", block.Hash)
			transaction := block.Transactions.Right.T
			fmt.Printf("%s sent %d to %s\n", transaction.Sender, transaction.Amount, transaction.Recipient)
			block = block.Next
		}
	}
}
