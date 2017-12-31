package main

import (
	. "akhcoin/blockchain"
	"fmt"
	"akhcoin/p2p"
	"flag"
	"log"
	"github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/libp2p/go-libp2p-peer"
	"bufio"
	"os"
	"strings"
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

func (n *AkhNode) Receive(b *Block) {
	Verify(b)
	//n.blockchain = append(n.blockchain, b)
}

func Verify(block *Block) error {
	fmt.Println("Block verified!")
	return nil
}

func (n *AkhNode) ReceiveTransaction(t *Transaction) {
	n.transactionsPool[t] = true
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
	port := flag.Int("p", 9000, "port where to start local host")
	remotePeerAddr := flag.String("a", "", "add peer address (format: <IP:port>)")
	remotePeerID := flag.String("id", "", "add peer ID <format>")
	flag.Parse()

	fmt.Printf("Peer: %s, port: %d, bye!\n", *remotePeerAddr, *port)

	node := NewAkhNode()
	host := p2p.StartHost(*port)
	p2p.SetStreamHandler(host, p2p.HandleGetBlockStream, node.Genesis)
	fmt.Printf("%s : %s\n", host.Addrs()[0], host.ID().Pretty())
	/////

	//chain = downloadUtil.downloadExisting blocks


	node.Head = node.Genesis

	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter text: ")
		text, _ := reader.ReadString('\n')
		if text == "exit\n" {
			break
		} else if text == "p\n" {
			node.Head = NewBlock(node.Head)
		} else {
			peers := discoverPeers(*remotePeerAddr, *remotePeerID)
			for _, peerInfo := range peers {
				host.AddPeer(&peerInfo)
				block := host.GetBlock(peers[0].ID)
				for block != nil {
					Validate(block, node.Head)
					fmt.Printf("Recieved block, hash: %s\n", block.Hash)
					transaction := block.Transactions.Right.T
					fmt.Printf("%s sent %d to %s\n", transaction.Sender, transaction.Amount, transaction.Recipient)
					block = block.Next
				}
			}
		}
	}

	//dpos.startMining

	//<-make(chan struct{}) // hang forever

}
func discoverPeers(remotePeerAddr string, remotePeerID string) []peerstore.PeerInfo {
	fmt.Printf("Sending to %s : %s;\n", remotePeerAddr, remotePeerID)
	peers := make([]peerstore.PeerInfo, 0, 10) //magic constant
	if len(remotePeerAddr) > 0 && len(remotePeerID) > 0 {
		split := strings.Split(remotePeerAddr, ":")
		addrStr := fmt.Sprintf("/ip4/%s/tcp/%s", split[0], split[1])
		fmt.Printf("Sending to %s;\n", addrStr)
		addr, _ := ma.NewMultiaddr(addrStr)
		decID, err := peer.IDB58Decode(remotePeerID)
		if err != nil {
			log.Fatal(err)
		}
		peerInfo := peerstore.PeerInfo{ID: decID, Addrs: []ma.Multiaddr{addr}}
		peers = append(peers, peerInfo)
	}
	return peers
}
