package main

import (
	"flag"
	"fmt"
	"github.com/alholm/akhcoin/p2p"
	"io/ioutil"
	console "log"
	"net/http"
	"os"

	"github.com/abiosoft/ishell"
	"github.com/alholm/akhcoin/blockchain"
	"github.com/alholm/akhcoin/node"
	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-crypto"
	"strconv"
)

var log = logging.Logger("main")

const privateKeyFileName = "id_rsa"

func main() {
	//1st launch, we didn't discover any nodes yet, so we have 3 options: (for more details see https://en.bitcoin.it/wiki/Bitcoin_Core_0.11_(ch_4):_P2P_Network)
	//1) hardcoded nodes
	//2) DNS seeding: on this stage no domains registered, skipping
	//3) User-specified on the command line

	logging.LevelError()
	//logging.LevelDebug() //to see all libp2p debug messages :
	logging.SetLogLevel("main", "DEBUG")
	logging.SetLogLevel("p2p", "DEBUG")
	logging.SetLogLevel("consensus", "DEBUG")
	// logging.SetLogLevel("mdns", "DEBUG")

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
			log.Error(fmt.Errorf("failed to read key file: %s", readKeyErr))
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
					c.Err(fmt.Errorf("failed to read key file: %s", readKeyErr))
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
					c.Err(fmt.Errorf("failed to generate keys: %s", readKeyErr))
				}
			},
		})

		preShell.AddCmd(&ishell.Cmd{
			Name: "exit",
			Help: "exit the program",
			Func: func(c *ishell.Context) {
				c.Stop()
				os.Exit(0)
			},
		})

		preShell.Run()
	}

	akhNode := node.NewAkhNode(*port, keyBytes)

	startHttpServer(akhNode, port)

	// by default, new shell includes 'exit', 'help' and 'clear' commands.
	shell := ishell.New()

	shell.AddCmd(&ishell.Cmd{
		Name: "pay",
		Help: "pay user, format: pay <Peer ID> <amount>",
		Func: func(c *ishell.Context) {
			if len(c.Args) < 2 {
				c.Err(fmt.Errorf("not enough arguments"))
				return
			}
			peerId := c.Args[0]
			amountStr := c.Args[1]
			amount, err := strconv.ParseUint(amountStr, 0, 64)
			if err != nil {
				c.Err(err)
				return
			}

			err = akhNode.Pay(peerId, amount)
			if err != nil {
				c.Err(err)
			}

		},
	})

	shell.AddCmd(&ishell.Cmd{
		Name: "v",
		Help: "vote, format: vote <Peer ID>",
		Func: func(c *ishell.Context) {
			peerId := ""
			if len(c.Args) == 0 {
				c.Err(fmt.Errorf("not enough arguments"))
				return
			}
			err := akhNode.Vote(peerId)
			if err != nil {
				c.Err(err)
			}
		},
	})

	shell.AddCmd(&ishell.Cmd{
		Name: "-ap",
		Help: "add peer, format: -ap <IP>[:port] <peer ID>",
		Func: func(c *ishell.Context) {
			if len(c.Args) < 2 {
				c.Err(fmt.Errorf("not enough arguments, see help"))
				return
			}
			err := akhNode.Host.AddPeerManually(c.Args[0], c.Args[1])
			if err != nil {
				c.Err(err)
			}
		},
	})

	shell.Print(shell.HelpText())

	shell.Run()

	akhNode.Host.Close()
}

func generateAndDumpKeys() (privateBytes []byte, err error) {
	private, public, _ := blockchain.NewKeys()
	privateBytes, _ = crypto.MarshalPrivateKey(private)
	err = ioutil.WriteFile(privateKeyFileName, privateBytes, 0644)
	if err != nil {
		return
	}
	publicBytes, _ := crypto.MarshalPublicKey(public)
	err = ioutil.WriteFile(privateKeyFileName+".pub", publicBytes, 0644)
	return
}

func startHttpServer(akhNode *node.AkhNode, port *int) {
	viewHandler := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "<h1>%s</h1>", akhNode.Head.Hash)
	}
	http.HandleFunc("/", viewHandler)
	go http.ListenAndServe(fmt.Sprintf(":%d", *port-1000), nil)
}
