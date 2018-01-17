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
	"github.com/libp2p/go-libp2p-protocol"
)

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

type StreamHandler interface {
	handle(ws *WrappedStream)
	protocol() protocol.ID
}

func AddStreamHandler(h2 AkhHost, handler StreamHandler) {
	h2.SetStreamHandler(handler.protocol(), func(stream inet.Stream) {
		log.Printf("%s: Received %s stream from %s", h2.ID(), handler.protocol(), stream.Conn().RemotePeer())
		ws := WrapStream(stream)
		defer stream.Close()
		handler.handle(ws)
		log.Printf("%s: %s stream from %s processing finished", h2.ID(), handler.protocol(), stream.Conn().RemotePeer())
	})
}


func sendMessage(msg interface{}, ws *WrappedStream) (err error) {
	err = ws.enc.Encode(msg)
	// Because output is buffered with bufio, we need to flush!
	ws.w.Flush()
	return
}

func (h *AkhHost) SendMessage(msg interface{}, peerID peer.ID, proto protocol.ID) (ws *WrappedStream, err error) {
	stream, err := h.NewStream(context.Background(), peerID, proto)
	if err != nil {
		return
	}
	ws = WrapStream(stream)
	sendMessage(msg, ws)
	return
}

func receiveMessage(msg interface{}, ws *WrappedStream) (err error) {
	err = ws.dec.Decode(msg)
	return
}
