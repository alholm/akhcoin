package blockchain

import (
	"github.com/libp2p/go-libp2p-peer"
	"github.com/libp2p/go-libp2p-crypto"
	"bytes"
	"encoding/binary"
)

type Transaction struct {
	Sender    string
	Recipient string
	Amount    uint64
	Sign      []byte
	PublicKey crypto.PubKey
}

func (t *Transaction) getCorpus() []byte {
	// Gather corpus to sign.
	corpus := new(bytes.Buffer)
	corpus.Write([]byte(t.Sender))
	corpus.Write([]byte(t.Recipient))
	amountBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(amountBytes, t.Amount)
	corpus.Write(amountBytes)
	return corpus.Bytes()

}

func (t *Transaction) Verify() bool {
	id, _ := peer.IDB58Decode(t.Sender)

	if id.MatchesPublicKey(t.PublicKey) {
		result, _ := t.PublicKey.Verify(t.getCorpus(), t.Sign)
		return result
	}
	return false
}

func Pay(privKey crypto.PrivKey, recipient peer.ID, amount uint64) *Transaction {

	sender, _ := peer.IDFromPrivateKey(privKey)
	t := Transaction{Sender: sender.Pretty(), Recipient: recipient.Pretty(), Amount: amount, PublicKey: privKey.GetPublic()}
	sign, _ := privKey.Sign(t.getCorpus())
	t.Sign = sign

	return &t
}
