package p2p

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"github.com/libp2p/go-libp2p-crypto"
	inet "github.com/libp2p/go-libp2p-net"
	"github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	"github.com/libp2p/go-libp2p-swarm"
	ma "github.com/multiformats/go-multiaddr"

	bhost "github.com/libp2p/go-libp2p/p2p/host/basic"
	"github.com/multiformats/go-multicodec"
	json "github.com/multiformats/go-multicodec/json"
	"github.com/libp2p/go-libp2p-protocol"
)

type AkhHost struct {
	bhost.BasicHost
}

type Message interface {
	fmt.Stringer
	MsgText() string
}

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
	private, public, _ := crypto.GenerateKeyPair(crypto.RSA, 2048)
	pid, _ := peer.IDFromPublicKey(public)

	// /ip4/0.0.0.0 - "any interface" address will be expanded to the known local interfaces.
	listen, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port))
	if err != nil {
		log.Fatal(err)
	}

	ps := pstore.NewPeerstore()
	ps.AddPrivKey(pid, private)
	ps.AddPubKey(pid, public)

	n, err := swarm.NewNetwork(context.Background(), []ma.Multiaddr{listen}, pid, ps, nil)
	if err != nil {
		log.Fatal(err)
	}

	h := AkhHost{*bhost.New(n)}

	//TODO temp, think where it belongs
	drp := &DiscoverStreamHandler{&ps}
	h.AddStreamHandler(drp)
	log.Printf("DEBUG: host %s %s on %v started\n", h.ID().Pretty(), h.ID(), []ma.Multiaddr{listen})

	return h
}

type StreamHandler interface {
	handle(ws *WrappedStream)
	protocol() protocol.ID
}

func (h *AkhHost) AddStreamHandler(handler StreamHandler) {
	h.SetStreamHandler(handler.protocol(), func(stream inet.Stream) {
		log.Printf("%s: Received %s stream from %s", h.ID(), handler.protocol(), stream.Conn().RemotePeer())
		ws := WrapStream(stream)
		defer stream.Close()
		handler.handle(ws)
		log.Printf("%s: %s stream from %s processing finished", h.ID(), handler.protocol(), stream.Conn().RemotePeer())
	})
}

func (h *AkhHost) ask(peerID peer.ID, question Message, proto protocol.ID, answer interface{}) (err error) {
	ws, err := h.SendMessage(&question, peerID, proto)
	if err != nil {
		return
	}
	err = receiveMessage(&answer, ws)
	if err != nil {
		err = fmt.Errorf("%s: %s stream to %s processing ended: %s", h.ID(), proto, peerID, err)

	}
	return
}

func answer(ws *WrappedStream, question Message, getAnswer func() interface{}) (err error) {
	err = receiveMessage(&question, ws)
	if err != nil {
		err = fmt.Errorf("Failed to decode stream: %s\n", err)
		return
	}

	err = sendMessage(getAnswer(), ws)
	if err != nil {
		err = fmt.Errorf("%s: Failed to transmit peer info: %s\n", ws.stream.Conn().RemotePeer(), err)
	}
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

func sendMessage(msg interface{}, ws *WrappedStream) (err error) {
	err = ws.enc.Encode(msg)
	// Because output is buffered with bufio, we need to flush!
	ws.w.Flush()
	return
}

func receiveMessage(msg interface{}, ws *WrappedStream) (err error) {
	err = ws.dec.Decode(msg)
	return
}
