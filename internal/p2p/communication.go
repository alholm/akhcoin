package p2p

import (
	"context"
	"github.com/alholm/akhcoin/pkg/blockchain"
	"io"
	"sync"

	"fmt"
	"github.com/libp2p/go-libp2p-peer"
	"github.com/libp2p/go-libp2p-protocol"
)

const protocolsPrefix = "ip4/akhcoin.org/tcp/"

const (
	BlockProto         protocol.ID = protocolsPrefix + "block/1.0.0"
	TransactionProto               = protocolsPrefix + "transaction/1.0.0"
	BlockAnnounceProto             = protocolsPrefix + "blockAnnounce/1.0.0"
	DiscoverProto                  = protocolsPrefix + "discover/1.0.0"
	VoteAnnounceProto              = protocolsPrefix + "vote/1.0.0"
)

type GetBlockMessage struct {
	Message
	BlockHash string
}

type BlockStreamHandler struct {
	Head **blockchain.Block
}

func (brp *BlockStreamHandler) protocol() protocol.ID {
	return BlockProto
}

func (brp *BlockStreamHandler) handle(ws *WrappedStream) {
	var msg GetBlockMessage
	err := receiveMessage(&msg, ws)
	if err != nil {
		log.Warningf("Failed to decode stream: %s\n", err)
		return
	}

	nextBlock := *brp.Head
	for nextBlock != nil {

		if nextBlock.Hash == msg.BlockHash {
			log.Debugf("%s: sending block %s\n", ws.stream.Conn().LocalPeer().Pretty(), nextBlock.Hash)
			err = sendMessage(nextBlock.BlockData, ws)
			if err != nil {
				log.Warningf("%s: Failed to transmit a block: %s\n", ws.stream.Conn().RemotePeer().Pretty(), err)
				break
			}
		}

		nextBlock = nextBlock.Parent
	}

}

type TransactionStreamHandler struct {
	ProcessResult func(t blockchain.Transaction)
}

func (trp *TransactionStreamHandler) protocol() protocol.ID {
	return TransactionProto
}

func (trp *TransactionStreamHandler) handle(ws *WrappedStream) {
	var t blockchain.Transaction
	err := receiveMessage(&t, ws)
	if err != nil {
		log.Warningf("Failed to process transaction msg: %s\n", err)
		return
	}

	trp.ProcessResult(t)
}

type AnnouncedBlockStreamHandler struct {
	ProcessResult func(bd blockchain.BlockData, peerId peer.ID)
}

func (abrp *AnnouncedBlockStreamHandler) handle(ws *WrappedStream) {
	var bd blockchain.BlockData
	err := receiveMessage(&bd, ws)
	if err != nil {
		log.Warningf("Failed to process block msg: %s\n", err)
		return
	}

	abrp.ProcessResult(bd, ws.stream.Conn().RemotePeer())
}

func (*AnnouncedBlockStreamHandler) protocol() protocol.ID {
	return BlockAnnounceProto
}

type VoteStreamHandler struct {
	ProcessResult func(v blockchain.Vote)
}

func (vrp *VoteStreamHandler) handle(ws *WrappedStream) {
	var v blockchain.Vote
	err := receiveMessage(&v, ws)
	if err != nil {
		log.Warningf("Failed to process Vote msg: %s\n", err)
		return
	}

	vrp.ProcessResult(v)
}

func (*VoteStreamHandler) protocol() protocol.ID {
	return VoteAnnounceProto
}

func (h *AkhHost) GetBlock(peerID peer.ID, blockHash string) (bd blockchain.BlockData, err error) {
	msg := &GetBlockMessage{BlockHash: blockHash}
	ws, err := h.SendMessage(msg, peerID, BlockProto)
	if err != nil {
		return
	}

	err = receiveMessage(&bd, ws)
	if err != nil {
		if err != io.EOF {
			log.Warningf("%s: %s stream to %s processing ended: %s", h.ID(), BlockProto, peerID.Pretty(), err)
			err = fmt.Errorf("failed to receive block %s : %s", blockHash, err)
		} else {
			err = fmt.Errorf("block %s wasn't received", blockHash)
		}
		return

	}
	log.Debugf("%s: BlockData received from %s: %s", h.ID(), ws.stream.Conn().RemotePeer().Pretty(), bd.Hash)

	return
}

func (h *AkhHost) PublishTransaction(t *blockchain.Transaction) {
	h.publish(t, TransactionProto)
}
func (h *AkhHost) PublishBlock(b *blockchain.Block) {
	h.publish(&b.BlockData, BlockAnnounceProto)
}
func (h *AkhHost) PublishVote(v *blockchain.Vote) {
	h.publish(v, VoteAnnounceProto)
}

//TODO error handling, conditional peers selection
func (h *AkhHost) publish(t interface{}, proto protocol.ID) {
	var wg sync.WaitGroup
	for _, peerID := range h.Peerstore().Peers() {
		wg.Add(1)
		go func(peerID peer.ID) {
			defer wg.Done()
			log.Debugf("%T published to %s - %s \n", t, peerID.Pretty(), h.Peerstore().Addrs(peerID))
			stream, err := h.NewStream(context.Background(), peerID, proto)
			//defer stream.Close()
			if err != nil {
				log.Warningf("Error publishing %T to %s: %s\n", t, peerID.Pretty(), err)
				return
			}
			ws := WrapStream(stream)

			sendMessage(t, ws)
		}(peerID)
	}
	wg.Wait()
	log.Debugf("%T %s sent\n", t, t)
}
