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
	"sync"
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
		log.Printf("%s: Received %s stream from %s", h2.ID(), blockProto, stream.Conn().RemotePeer())
		ws := WrapStream(stream)
		defer stream.Close()
		handler(ws, lastBlock)
		log.Printf("%s: %s stream from %s processing finished", h2.ID(), blockProto, stream.Conn().RemotePeer())
	})

	h2.SetStreamHandler(transactionProto, func(stream inet.Stream) {
		log.Printf("%s: Received %s stream from %s", h2.ID(), transactionProto, stream.Conn().RemotePeer())
		ws := WrapStream(stream)
		defer stream.Close()
		HandleTransactionStream(ws)
		log.Printf("%s: %s stream from %s processing finished", h2.ID(), transactionProto, stream.Conn().RemotePeer())
	})
}

func SendMessage(msg interface{}, ws *WrappedStream) (err error) {
	err = ws.enc.Encode(msg)
	// Because output is buffered with bufio, we need to flush!
	ws.w.Flush()
	return
}

func receiveMessage(ws *WrappedStream) (msg *MyMessage, err error) {
	err = ws.dec.Decode(msg)
	return
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
		log.Printf("Failed to decode stream: %s\n", err)
		return
	}

	nextBlock := genesis.Next
	for nextBlock != nil {
		log.Printf("%s: sending block %s\n", ws.stream.Conn().LocalPeer(), nextBlock.Hash)
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
	//defer stream.Close()
	if err != nil {
		return nil, err
	}
	ws := WrapStream(stream)

	msg := &GetBlockMessage{}
	SendMessage(msg, ws)

	block := &blockchain.Block{}
	firstBlock := block
	for {

		var blockData blockchain.BlockData
		err = ws.dec.Decode(&blockData)
		if err != nil {
			log.Printf("%s: %s stream to %s processing ended: %s", h.ID(), blockProto, stream.Conn().RemotePeer(), err)
			break
		}
		log.Printf("%s: BlockData received from %s: %s", h.ID(), stream.Conn().RemotePeer(), blockData.Hash)
		block.BlockData = blockData
		nextBlock := &blockchain.Block{Parent: block}
		block.Next = nextBlock
		block = nextBlock

		ackMsg := &AckMessage{}
		SendMessage(ackMsg, ws)
	}

	//TODO may be nil
	block.Parent.Next = nil

	//pid := ws.stream.Conn().LocalPeer()

	return firstBlock, nil
}

//TODO error handling
func (h *AkhHost) PublishTransaction(t *blockchain.Transaction) {
	var wg sync.WaitGroup
	for _, peerID := range h.Peerstore().Peers() {
		wg.Add(1)
		go func(peerID peer.ID) {
			defer wg.Done()
			log.Printf("### Txn published to %s - %s \n", peerID.Pretty(), h.Peerstore().Addrs(peerID))
			stream, err := h.NewStream(context.Background(), peerID, transactionProto)
			//defer stream.Close()
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
	wg.Wait()
	log.Printf("### Transaction %s sent\n", t)
}

func HandleTransactionStream(ws *WrappedStream) {

	var t blockchain.Transaction
	err := ws.dec.Decode(&t)
	if err != nil {
		log.Printf("Failed to process transaction msg: %s\n", err)
	}

	verified, _ := t.Verify()
	log.Printf("### Txn received: %s, VERIFIED=%t\n", &t, verified)
	//ackMsg := &AckMessage{}
	//SendMessage(ackMsg, ws)
}
