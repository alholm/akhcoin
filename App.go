package main

import (
	. "akhcoin/blockchain"
	"fmt"
	"akhcoin/p2p"
	"flag"
	console "log"
	"math/rand"
	"time"
	"net/http"
	logging "github.com/ipfs/go-log"
	"github.com/abiosoft/ishell"
	"github.com/libp2p/go-libp2p-crypto"
	"io/ioutil"
)

var log = logging.Logger("main")

const privateKeyFileName = "id_rsa"

func main() {
	//1st launch, we didn't discover any nodes yet, so we have 3 options: (for more details see https://en.bitcoin.it/wiki/Bitcoin_Core_0.11_(ch_4):_P2P_Network)
	//1) hardcoded nodes
	//2) DNS seeding: on this stage no domains registered, skipping
	//3) User-specified on the command line

	logging.LevelError()
	//logging.LevelDebug() //to see libp2p debug messages :
	logging.SetLogLevel("main", "DEBUG")
	logging.SetLogLevel("p2p", "DEBUG")

	port := flag.Int("p", p2p.DefaultPort,
		fmt.Sprintf("port where to start local host, %d will be used by default", p2p.DefaultPort))
	keyPath := flag.String("k", "",
		fmt.Sprintf("path to private key file, will be attempted to read from \"%s\" in current directory by default", privateKeyFileName))

	flag.Parse()

	console.Println("AkhCoin 0.1. Welcome!")

	var keyBytes []byte
	var readKeyErr error

	if len(*keyPath) > 0 {
		keyBytes, readKeyErr = ioutil.ReadFile(*keyPath)
		if readKeyErr != nil {
			log.Error(fmt.Errorf("Failed to read key file: %s\n", readKeyErr))
		}
	}

	if len(*keyPath) == 0 || readKeyErr != nil {

		preShell := ishell.New()

		preShell.Println("No keys provided, type \"help\" to see available commands:")

		preShell.AddCmd(&ishell.Cmd{
			Name: "key",
			Help: "specify private key file path, eg </home/.ssh/private.key>; <./id_rsa> by default",
			Func: func(c *ishell.Context) {
				path := privateKeyFileName
				if len(c.Args) > 0 {
					path = c.Args[0]
				}
				keyBytes, readKeyErr = ioutil.ReadFile(path)
				if readKeyErr == nil {
					c.Println("Private key successfully added")
					c.Stop()
				} else {
					c.Err(fmt.Errorf("Failed to read key file: %s\n", readKeyErr))
				}
			},
		})

		preShell.AddCmd(&ishell.Cmd{
			Name: "gen",
			Help: "generate new key pair and dump to <filename> and <filename>.pub ",
			Func: func(c *ishell.Context) {
				keyBytes, readKeyErr = generateAndDumpKeys()
				if readKeyErr == nil {
					c.Println("Private key successfully added")
					c.Stop()
				} else {
					c.Err(fmt.Errorf("Failed to generate keys: %s\n", readKeyErr))
				}
			},
		})

		preShell.Run()
	}

	node := NewAkhNode(*port, keyBytes)

	startHttpServer(node, port)

	// by default, new shell includes 'exit', 'help' and 'clear' commands.
	shell := ishell.New()

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
			_, err := node.Produce()
			if err != nil {
				err = fmt.Errorf("Failed to produce new block: %s\n", err)
				c.Err(err)
				log.Warning(err)
			}
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

	shell.Print(shell.HelpText())

	shell.Run()

	node.Host.Close()
}

func generateAndDumpKeys() (privateBytes []byte, err error) {
	private, public, _ := crypto.GenerateKeyPair(crypto.RSA, 2048)
	privateBytes, _ = crypto.MarshalPrivateKey(private)
	err = ioutil.WriteFile(privateKeyFileName, privateBytes, 0644)
	if err != nil {
		return
	}
	publicBytes, _ := crypto.MarshalPublicKey(public)
	err = ioutil.WriteFile(privateKeyFileName+".pub", publicBytes, 0644)
	return
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

	rand.Seed(time.Now().UnixNano())
	peerIDs := host.Peerstore().Peers()
	if len(peerIDs) <= 1 {
		log.Debugf("TEMP: no peers")
		return
	}
	i := rand.Intn(len(peerIDs) - 1)
	s := rand.Uint64()

	t := Pay(private, peerIDs[i], s)

	log.Debugf("Just created txn: %s\n", t)
	host.PublishTransaction(t)
}
