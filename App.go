package main

import (
	. "akhcoin/blockchain"
	"fmt"
	"akhcoin/p2p"
	"flag"
	"log"
	"math/rand"
	"time"
	"bytes"
	"net/http"
	logging "github.com/ipfs/go-log"
	"github.com/abiosoft/ishell"
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
	}
	privateKey := node.Host.Peerstore().PrivKey(node.Host.ID())
	block = NewBlock(privateKey, node.Head, pool)
	node.Head = block

	//TODO ATTENTION! RACE CONDITION, has to be guarded
	node.transactionsPool = pool[:0]

	return
}

func (node *AkhNode) Announce(block *Block) (err error) {
	node.Host.PublishBlock(block)
	return nil
}

func (node *AkhNode) ReceiveTransaction(t Transaction) {
	verified, _ := t.Verify()
	log.Printf("DEBUG: Txn received: %s, VERIFIED=%t\n", &t, verified)
	//TODO ATTENTION! RACE CONDITION
	node.transactionsPool = append(node.transactionsPool, t)
}

//TODO think of reaction to invalid block
func (node *AkhNode) Receive(bd BlockData) {
	block := &Block{BlockData: bd, Parent: node.Head}
	verified, err := Validate(block, node.Head)
	log.Printf("DEBUG: Block received: %s, VERIFIED=%t\n", bd.Hash, verified)

	if err != nil {
		log.Println(err)
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
	host.AddStreamHandler(brp)

	trp := &p2p.TransactionStreamHandler{ProcessResult: node.ReceiveTransaction}
	host.AddStreamHandler(trp)

	abrp := &p2p.AnnouncedBlockStreamHandler{ProcessResult: node.Receive}
	host.AddStreamHandler(abrp)

	host.DumpHostInfo()
	host.DiscoverPeers()
	return
}

func main() {

	//1st launch, we didn't discover any nodes yet, so we have 3 options: (for more details see https://en.bitcoin.it/wiki/Bitcoin_Core_0.11_(ch_4):_P2P_Network)
	//1) hardcoded nodes
	//2) DNS seeding: on this stage no domains registered, skipping
	//3) User-specified on the command line

	logging.LevelError() //logging.LevelDebug()

	port := flag.Int("p", p2p.DefaultPort, "port where to start local host")
	flag.Parse()

	node := NewAkhNode(*port)

	startHttpServer(node, port)

	// by default, new shell includes 'exit', 'help' and 'clear' commands.
	shell := ishell.New()

	shell.Println("Type \"help\" to see available commands:")

	shell.AddCmd(&ishell.Cmd{
		Name: "p",
		Help: "pay user",
		Func: func(c *ishell.Context) {
			node.testPay()
		},
	})

	shell.AddCmd(&ishell.Cmd{
		Name: "g",
		Help: "generate block",
		Func: func(c *ishell.Context) {
			block, err := node.Produce()
			c.Printf("%s: New Block hash = %s, error: %s\n", node.Host.ID(), block.Hash, err)
			node.testPay()
		},
	})

	shell.AddCmd(&ishell.Cmd{
		Name: "d",
		Help: "initial blocks download",
		Func: func(c *ishell.Context) {
			node.initialBlockDownload()
		},
	})

	shell.AddCmd(&ishell.Cmd{
		Name: "-ap",
		Help: "add peer, format: -ap <IP>[:port] <peer ID>",
		Func: func(c *ishell.Context) {
			if len(c.Args) < 2 {
				c.Println("Not enough arguments, see help")
				return
			}
			err := node.Host.AddPeerManually(c.Args[0], c.Args[1])
			if err != nil {
				c.Err(err)
			}
		},
	})

	shell.Run()

	node.Host.Close()
}

func startHttpServer(node *AkhNode, port *int) {
	viewHandler := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "<h1>%s</h1><div>%s</div>", node.Head.Nonce, node.Head.Hash)
	}
	http.HandleFunc("/", viewHandler)
	go http.ListenAndServe(fmt.Sprintf(":%d", *port-1000), nil)
}

func (node *AkhNode) testPay() {
	host := node.Host
	private := node.Host.Peerstore().PrivKey(host.ID())
	//log.Println(private)

	rand.Seed(time.Now().UnixNano())
	peerIDs := host.Peerstore().Peers()
	if len(peerIDs) <= 1 {
		log.Println("TEMP: no peers")
		return
	}
	i := rand.Intn(len(peerIDs) - 1)
	s := rand.Uint64()

	t := Pay(private, peerIDs[i], s)

	log.Printf("### Just created txn: %s\n", t)
	host.PublishTransaction(t)
}

func (node *AkhNode) initialBlockDownload() {
	for _, peerID := range node.Host.Peerstore().Peers() {
		log.Printf("%s requesting block from %s\n", node.Host.ID(), peerID)
		block, err := node.Host.GetBlock(peerID, singleBlockCallback)
		if err != nil {
			log.Println(err)
			continue
		}
		for block != nil {
			valid, err := Validate(block, node.Head)
			if !valid {
				log.Println(err)
				//TODO
			}
			node.Attach(block)
			block = block.Next
		}
	}
}
func singleBlockCallback(blockData interface{}) {
}
