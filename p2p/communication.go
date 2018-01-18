package p2p

import (
	"context"
	"log"
	"github.com/libp2p/go-libp2p-peer"
	"akhcoin/blockchain"
	"sync"
	"github.com/libp2p/go-libp2p-protocol"
)

const protocolsPrefix = "ip4/akhcoin.org/tcp/"

const (
	BlockProto         protocol.ID = protocolsPrefix + "block/1.0.0"
	TransactionProto               = protocolsPrefix + "transaction/1.0.0"
	BlockAnnounceProto             = protocolsPrefix + "blockAnnounce/1.0.0"
	DiscoverProto                  = protocolsPrefix + "discover/1.0.0"
)

type GetBlockMessage struct {
	Message
}

type AckMessage struct {
	Message
}

type BlockStreamHandler struct {
	Genesis *blockchain.Block
}

func (brp *BlockStreamHandler) protocol() protocol.ID {
	return BlockProto
}

func (brp *BlockStreamHandler) handle(ws *WrappedStream) {
	var msg GetBlockMessage
	err := receiveMessage(&msg, ws)
	if err != nil {
		log.Printf("Failed to decode stream: %s\n", err)
		return
	}

	nextBlock := brp.Genesis.Next
	for nextBlock != nil {
		log.Printf("%s: sending block %s\n", ws.stream.Conn().LocalPeer(), nextBlock.Hash)
		err = sendMessage(nextBlock.BlockData, ws)
		if err != nil {
			log.Printf("%s: Failed to transmit a block: %s\n", ws.stream.Conn().RemotePeer(), err)
			return
		}

		var ackMsg AckMessage
		err := receiveMessage(&ackMsg, ws)
		if err != nil {
			log.Printf("%s: Failed get confirmation on block delivery: %s\n", ws.stream.Conn().RemotePeer(), err)
			return
		}

		nextBlock = nextBlock.Next
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
		log.Printf("Failed to process transaction msg: %s\n", err)
		return
	}

	trp.ProcessResult(t)
}

type AnnouncedBlockStreamHandler struct {
	ProcessResult func(bd blockchain.BlockData)
}

func (abrp *AnnouncedBlockStreamHandler) handle(ws *WrappedStream) {
	var bd blockchain.BlockData
	err := receiveMessage(&bd, ws)
	if err != nil {
		log.Printf("Failed to process block msg: %s\n", err)
		return
	}

	abrp.ProcessResult(bd)
}

func (*AnnouncedBlockStreamHandler) protocol() protocol.ID {
	return BlockAnnounceProto
}

func (h *AkhHost) GetBlock(peerID peer.ID, someFunc func(o interface{})) (*blockchain.Block, error) {
	msg := &GetBlockMessage{}
	ws, err := h.SendMessage(msg, peerID, BlockProto)
	if err != nil {
		return nil, err
	}
	block := &blockchain.Block{}
	firstBlock := block
	for {

		var blockData blockchain.BlockData
		err := receiveMessage(&blockData, ws)
		if err != nil {
			log.Printf("%s: %s stream to %s processing ended: %s", h.ID(), BlockProto, peerID, err)
			break
		}
		log.Printf("%s: BlockData received from %s: %s", h.ID(), ws.stream.Conn().RemotePeer(), blockData.Hash)
		block.BlockData = blockData
		nextBlock := &blockchain.Block{Parent: block}
		block.Next = nextBlock
		block = nextBlock

		someFunc(blockData)

		ackMsg := &AckMessage{}
		sendMessage(ackMsg, ws)
	}

	//TODO may be nil
	block.Parent.Next = nil

	return firstBlock, nil
}

func (h *AkhHost) PublishTransaction(t *blockchain.Transaction) {
	h.publish(t, TransactionProto)
}
func (h *AkhHost) PublishBlock(b *blockchain.Block) {
	h.publish(&b.BlockData, BlockAnnounceProto)
}

//TODO error handling, conditional peers selection
func (h *AkhHost) publish(t interface{}, proto protocol.ID) {
	var wg sync.WaitGroup
	for _, peerID := range h.Peerstore().Peers() {
		wg.Add(1)
		go func(peerID peer.ID) {
			defer wg.Done()
			log.Printf("DEBUG: Txn published to %s - %s \n", peerID.Pretty(), h.Peerstore().Addrs(peerID))
			stream, err := h.NewStream(context.Background(), peerID, proto)
			//defer stream.Close()
			if err != nil {
				log.Printf("Error publishing transaction to %s: %s\n", peerID, err)
				return
			}
			ws := WrapStream(stream)

			sendMessage(t, ws)
			//var ackMsg AckMessage
			//err := ws.dec.Decode(&ackMsg)
			//if err != nil {
			//	log.Println(err)
			//	return
			//}
		}(peerID)
	}
	wg.Wait()
	log.Printf("DEBUG: %T %s sent\n", t, t)
}
