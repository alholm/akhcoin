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

const protocol = "/akh/1.0.0"

type MyHost struct {
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

// streamWrap wraps a libp2p stream. We encode/decode whenever we
// write/read from a stream, so we can just carry the encoders
// and bufios with us
type WrappedStream struct {
	stream inet.Stream
	enc    multicodec.Encoder
	dec    multicodec.Decoder
	w      *bufio.Writer
	r      *bufio.Reader
}

// wrapStream takes a stream and complements it with r/w bufios and
// decoder/encoder. In order to write raw data to the stream we can use
// wrap.w.Write(). To encode something into it we can wrap.enc.Encode().
// Finally, we should wrap.w.Flush() to actually send the data. Handling
// incoming data works similarly with wrap.r.Read() for raw-reading and
// wrap.dec.Decode() to decode.
func WrapStream(s inet.Stream) *WrappedStream {
	reader := bufio.NewReader(s)
	writer := bufio.NewWriter(s)
	// This is where we pick our specific multicodec. In order to change the
	// codec, we only need to change this place.
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

func StartHost(port int) MyHost {
	// Ignoring most errors for brevity
	// See echo example for more details and better implementation
	priv, pub, _ := crypto.GenerateKeyPair(crypto.RSA, 2048)
	pid, _ := peer.IDFromPublicKey(pub)
	listen, _ := ma.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", port))
	ps := ps.NewPeerstore()
	ps.AddPrivKey(pid, priv)
	ps.AddPubKey(pid, pub)
	n, _ := swarm.NewNetwork(context.Background(),
		[]ma.Multiaddr{listen}, pid, ps, nil)
	myHost := MyHost{*bhost.New(n), n}
	return myHost
}

func (h *MyHost) Start() {
	h.BasicHost = *bhost.New(h.network)
}

func (h *MyHost) AddPeer(peerInfo *ps.PeerInfo) {
	h.Peerstore().AddAddrs(peerInfo.ID, peerInfo.Addrs, ps.PermanentAddrTTL)
}

func SetStreamHandler(h2 MyHost, handler func(ws *WrappedStream)) {
	// Define a stream handler for host number 2
	h2.SetStreamHandler(protocol, func(stream inet.Stream) {
		log.Printf("%s: Received a stream", h2.ID())
		wrappedStream := WrapStream(stream)
		defer stream.Close()
		handler(wrappedStream)
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

func HandleGetBlockStream(ws *WrappedStream) {
	var msg GetBlockMessage
	err := ws.dec.Decode(&msg)
	if err != nil {
		log.Fatalln(err)
		return
	}
	log.Printf("%T message recieved", msg)

	hash := blockchain.Hash("New Genesis")
	log.Printf("Hash generated: %s", hash)
	block := blockchain.Block{Hash: hash}
	err = SendMessage(block, ws)
	if err != nil {
		log.Fatalln(err)
	}
}

func (h *MyHost) GetBlock(peerID peer.ID) *blockchain.Block {
	// Create new stream from h1 to h2 and start the conversation
	stream, err := h.NewStream(context.Background(), peerID, "/akh/1.0.0")
	if err != nil {
		log.Fatalln(err)
	}
	wrappedStream := WrapStream(stream)
	// This sends the first message
	msg := &GetBlockMessage{}
	SendMessage(msg, wrappedStream)
	// We keep the conversation on the created stream so we launch
	// this to handle any responses
	var block blockchain.Block
	err = wrappedStream.dec.Decode(&block)
	if err != nil {
		log.Fatalln(err)
		return nil
	}
	pid := wrappedStream.stream.Conn().LocalPeer()
	log.Printf("%s says: %t %s\n", pid, msg, block.Hash)
	// When we are done, close the stream on our side and exit.
	stream.Close()

	return &block
}
