package p2p

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"github.com/libp2p/go-libp2p-crypto"
	inet "github.com/libp2p/go-libp2p-net"
	"github.com/libp2p/go-libp2p-peer"
	ps "github.com/libp2p/go-libp2p-peerstore"
	"github.com/libp2p/go-libp2p-swarm"
	ma "github.com/multiformats/go-multiaddr"

	bhost "github.com/libp2p/go-libp2p/p2p/host/basic"
	"github.com/multiformats/go-multicodec"
	json "github.com/multiformats/go-multicodec/json"
	"akhcoin/blockchain"
)

const protocolsPrefix = "ip4/akhcoin.org/tcp/"
const blockProto = protocolsPrefix + "block/1.0.0"
const transactionProto = protocolsPrefix + "transaction/1.0.0"

type AkhHost struct {
	bhost.BasicHost
	network *swarm.Network
}

type Message interface {
	fmt.Stringer
	MsgText() string
}

// MyMessage is a serializable/encodable object that we will send
// on a Stream.
type MyMessage struct {
	Msg    string
	Index  int
	HangUp bool
}

func (m *MyMessage) String() string {
	return m.Msg
}

func (m *MyMessage) MsgText() string {
	return m.Msg
}

type WrappedStream struct {
	stream inet.Stream
	enc    multicodec.Encoder
	dec    multicodec.Decoder
	w      *bufio.Writer
	r      *bufio.Reader
}

func WrapStream(s inet.Stream) *WrappedStream {
	reader := bufio.NewReader(s)
	writer := bufio.NewWriter(s)
	// TODO use binary or protobuf
	// See https://godoc.org/github.com/multiformats/go-multicodec/json
	dec := json.Multicodec(false).Decoder(reader)
	enc := json.Multicodec(false).Encoder(writer)
	return &WrappedStream{
		stream: s,
		r:      reader,
		w:      writer,
		enc:    enc,
		dec:    dec,
	}
}

func StartHost(port int) AkhHost {
	// Ignoring most errors for brevity
	// See echo example for more details and better implementation
	private, public, _ := crypto.GenerateKeyPair(crypto.RSA, 2048)
	pid, _ := peer.IDFromPublicKey(public)
	listen, _ := ma.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", port))
	ps := ps.NewPeerstore()
	ps.AddPrivKey(pid, private)
	ps.AddPubKey(pid, public)
	n, _ := swarm.NewNetwork(context.Background(),
		[]ma.Multiaddr{listen}, pid, ps, nil)
	myHost := AkhHost{*bhost.New(n), n}
	return myHost
}

func (h *AkhHost) Start() {
	h.BasicHost = *bhost.New(h.network)
}

func SetStreamHandler(h2 AkhHost, handler func(ws *WrappedStream, genesis *blockchain.Block), lastBlock *blockchain.Block) {
	h2.SetStreamHandler(blockProto, func(stream inet.Stream) {
		log.Printf("%s: Received %s stream", h2.ID(), blockProto)
		ws := WrapStream(stream)
		defer stream.Close()
		handler(ws, lastBlock)
	})

	h2.SetStreamHandler(transactionProto, func(stream inet.Stream) {
		log.Printf("%s: Received %s stream", stream.Conn().RemotePeer(), transactionProto)
		ws := WrapStream(stream)
		defer stream.Close()
		HandleTransactionStream(ws)
	})
}

func SendMessage(msg interface{}, ws *WrappedStream) error {
	err := ws.enc.Encode(msg)
	// Because output is buffered with bufio, we need to flush!
	ws.w.Flush()
	return err
}

func receiveMessage(ws *WrappedStream) (*MyMessage, error) {
	var msg MyMessage
	err := ws.dec.Decode(&msg)

	if err != nil {
		return nil, err
	}
	return &msg, nil
}

type GetBlockMessage struct {
	Message
}

type AckMessage struct {
	Message
}

func HandleGetBlockStream(ws *WrappedStream, genesis *blockchain.Block) {
	var msg GetBlockMessage
	err := ws.dec.Decode(&msg)
	if err != nil {
		log.Printf("Couldn't decode incoming 'Get Block' stream: %s\n", err)
		return
	}
	log.Printf("##### %T message recieved\n", msg)

	nextBlock := genesis.Next
	for nextBlock != nil {
		log.Printf("##### Sending block %s\n", nextBlock.Hash)
		err = SendMessage(nextBlock.BlockData, ws)
		if err != nil {
			log.Fatalln(err)
		}

		var ackMsg AckMessage
		err := ws.dec.Decode(&ackMsg)
		if err != nil {
			log.Println(err)
			return
		}

		nextBlock = nextBlock.Next
	}
}

func (h *AkhHost) GetBlock(peerID peer.ID) (*blockchain.Block, error) {
	// Create new stream from h1 to h2 and start the conversation
	stream, err := h.NewStream(context.Background(), peerID, blockProto)
	if err != nil {
		return nil, err
	}
	ws := WrapStream(stream)

	msg := &GetBlockMessage{}
	SendMessage(msg, ws)

	log.Println("##### GetBlockMessage sent")

	block := &blockchain.Block{}
	firstBlock := block
	for {

		var blockData blockchain.BlockData
		err = ws.dec.Decode(&blockData)
		if err != nil {
			log.Printf("##### stream processing ended: %x", err)
			break
		}
		log.Printf("##### BlockData received %s", blockData.Hash)
		block.BlockData = blockData
		nextBlock := &blockchain.Block{Parent: block}
		block.Next = nextBlock
		block = nextBlock

		ackMsg := &AckMessage{}
		SendMessage(ackMsg, ws)
	}

	block.Parent.Next = nil

	pid := ws.stream.Conn().LocalPeer()
	log.Printf("##### %s says: %t %s\n", pid, *msg, block.Parent.Hash)

	stream.Close()

	return firstBlock, nil
}

//TODO error handling
func (h *AkhHost) PublishTransaction(t *blockchain.Transaction) {

	for _, peerID := range h.Peerstore().Peers() {
		go func(peerID peer.ID) {
			log.Printf("### to _ %s _ %s \n", peerID.Pretty(), h.Peerstore().Addrs(peerID))
			stream, err := h.NewStream(context.Background(), peerID, transactionProto)
			if err != nil {
				log.Printf("Error publishing transaction to %s: %s\n", peerID, err)
				return
			}
			ws := WrapStream(stream)

			SendMessage(*t, ws)
			//var ackMsg AckMessage
			//err := ws.dec.Decode(&ackMsg)
			//if err != nil {
			//	log.Println(err)
			//	return
			//}
		}(peerID)
	}
	defer log.Printf("### Transaction %s sent\n", t)
}

func HandleTransactionStream(ws *WrappedStream) {

	var t blockchain.Transaction
	err := ws.dec.Decode(&t)
	if err != nil {
		log.Printf("Failed to process transaction msg: %s\n", err)
	}

	log.Printf("### Txn received: %s, VERIFIED=%t\n", &t, t.Verify())
	//ackMsg := &AckMessage{}
	//SendMessage(ackMsg, ws)
}
