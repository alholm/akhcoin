package blockchain

import (
	"github.com/libp2p/go-libp2p-peer"
	"github.com/libp2p/go-libp2p-crypto"
	"bytes"
	"encoding/binary"
	"fmt"
)

type Transaction struct {
	Sender    string
	Recipient string
	Amount    uint64
	Sign      []byte
	PublicKey []byte
}

func (t *Transaction) String() string {
	return fmt.Sprintf("%d from %s to %s", t.Amount, t.Sender, t.Recipient)
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

func (t *Transaction) Verify() (result bool, err error){
	result = false
	id, err := peer.IDB58Decode(t.Sender)
	if err != nil {
		return
	}
	public, err := crypto.UnmarshalPublicKey(t.PublicKey)
	if err != nil {
		return
	}

	if id.MatchesPublicKey(public) {
		result, err = public.Verify(t.getCorpus(), t.Sign)
	}
	return
}

func Pay(private crypto.PrivKey, recipient peer.ID, amount uint64) *Transaction {

	sender, _ := peer.IDFromPrivateKey(private)
	public, _ := private.GetPublic().Bytes()

	t := Transaction{Sender: sender.Pretty(), Recipient: recipient.Pretty(), Amount: amount, PublicKey: public}
	sign, _ := private.Sign(t.getCorpus())
	t.Sign = sign

	return &t
}
