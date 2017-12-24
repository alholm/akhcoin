package main

import (
	. "akhcoin/blockchain"
	"fmt"
	"akhcoin/p2p"
	"flag"
	"log"
	"github.com/libp2p/go-libp2p-peerstore"
	"context"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/libp2p/go-libp2p-peer"
)

type Node interface {
	Vote(sign string, addr string)

	Produce() *Block

	Receive(*Block)

	ReceiveTransaction(*Transaction)
}

type AkhNode struct {
	transactionsPool map[*Transaction]bool
	blockchain       []*Block
	sign             string
}

func (*AkhNode) Vote(sign string, addr string) {
	panic("implement me")
}

func (*AkhNode) Produce() *Block {
	return NewBlock()
}

func (n *AkhNode) Receive(b *Block) {
	Verify(b)
	n.blockchain = append(n.blockchain, b)
}

func Verify(block *Block) error {
	fmt.Println("Block verified!")
	return nil
}

func (n *AkhNode) ReceiveTransaction(t *Transaction) {
	n.transactionsPool[t] = true
}

func NewAkhNode() *AkhNode {
	node := AkhNode{
		transactionsPool: make(map[*Transaction]bool),
		blockchain:       initBlockchain(),
		sign:             "",
	}
	return &node
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
	remotePeer := *flag.String("a", "", "add peers - comma-separated list of peers addresses TODO <format>")
	port := *flag.Int("p", 9000, "port where to start local host")
	flag.Parse()
	fmt.Printf("Peer: %s, port: %d, bye!\n", remotePeer, port)

	//host := p2p.MakeRandomHost(port)
	//fmt.Println(host.Addrs()[0])

	//host.Start()
	//fmt.Printf("%s : %s\n", host.Addrs()[0], host.ID().Pretty())
	//p2p.SetStreamHandler(host)
	//defer host.Close()
	//<-make(chan struct{}) // hang forever

	host2 := p2p.MakeRandomHost(11000)
	//fmt.Println(host.Addrs()[0])

	host2.Start()
	fmt.Printf("%s : %s\n", host2.Addrs()[0], host2.ID().Pretty())


	addr, _ := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/9000")

	decID, err := peer.IDB58Decode("QmVSzfqX9CLUkvA3z1h4HwvX1Ud9ipxbar3QE8MgReShpA")
	//decID, err := peer.IDB58Decode(host.ID().Pretty())

	if err != nil {
		log.Fatal(err)
	}

	peerInfo := peerstore.PeerInfo{ID: decID, Addrs: []ma.Multiaddr{addr}}

	host2.AddPeer(&peerInfo)

	// Create new stream from h1 to h2 and start the conversation
	stream, err := host2.NewStream(context.Background(), peerInfo.ID, "/akh/1.0.0")
	if err != nil {
		log.Fatal(err)
	}
	wrappedStream := p2p.WrapStream(stream)
	// This sends the first message
	p2p.SendMessage(0, wrappedStream)
	// We keep the conversation on the created stream so we launch
	// this to handle any responses
	p2p.HandleStream(wrappedStream)
	// When we are done, close the stream on our side and exit.
	stream.Close()
	if err != nil {
		log.Fatalln(err)
	}

	if len(remotePeer) == 0 {
		log.Fatalln("Please provide at least one remotePeer to bind on with -a")
	}

}
