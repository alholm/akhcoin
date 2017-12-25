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
	"bufio"
	"os"
	"strings"
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
	port := flag.Int("p", 9000, "port where to start local host")
	remotePeerAddr := flag.String("a", "", "add peer address (format: <IP:port>)")
	remotePeerID := flag.String("id", "", "add peer ID <format>")
	flag.Parse()

	fmt.Printf("Peer: %s, port: %d, bye!\n", *remotePeerAddr, *port)

	host := p2p.MakeRandomHost(*port)

	host.Start()
	p2p.SetStreamHandler(host)

	fmt.Printf("%s : %s\n", host.Addrs()[0], host.ID().Pretty())

	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter text: ")
		text, _ := reader.ReadString('\n')
		if text == "exit\n" {
			break
		} else {
			sendSmth(*remotePeerAddr, *remotePeerID, host)
		}
		fmt.Println(text)
	}

	//<-make(chan struct{}) // hang forever

}
func sendSmth(remotePeerAddr string, remotePeerID string, host p2p.MyHost) {
	fmt.Printf("Sending to %s : %s;\n", remotePeerAddr, remotePeerID)
	if len(remotePeerAddr) > 0 && len(remotePeerID) > 0 {

		split := strings.Split(remotePeerAddr, ":")
		addrStr := fmt.Sprintf("/ip4/%s/tcp/%s", split[0], split[1])
		fmt.Printf("Sending to %s;\n", addrStr)
		addr, _ := ma.NewMultiaddr(addrStr) //"/ip4/127.0.0.1/tcp/9000"

		decID, err := peer.IDB58Decode(remotePeerID) //"QmcHWzenJP3B2jrvEDGk9Gdbw964LdrQCTZDDoT4nePbBU"

		if err != nil {
			log.Fatal(err)
		}

		peerInfo := peerstore.PeerInfo{ID: decID, Addrs: []ma.Multiaddr{addr}}

		host.AddPeer(&peerInfo)

		// Create new stream from h1 to h2 and start the conversation
		stream, err := host.NewStream(context.Background(), peerInfo.ID, "/akh/1.0.0")
		if err != nil {
			log.Fatalln(err)
		}
		wrappedStream := p2p.WrapStream(stream)
		// This sends the first message
		p2p.SendMessage(0, wrappedStream)
		// We keep the conversation on the created stream so we launch
		// this to handle any responses
		p2p.HandleStream(wrappedStream)
		// When we are done, close the stream on our side and exit.
		stream.Close()
	}
}
