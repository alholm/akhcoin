package blockchain

import (
	"encoding/binary"
	"fmt"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/libp2p/go-libp2p-peer"
)

type Signable interface {
	GetSigner() string
	GetPublicKey() []byte
	GetCorpus() []byte
	GetSign() []byte
	GetTimestamp() int64
}

type Transaction struct {
	Sender    string
	Recipient string
	Amount    uint64
	Sign      []byte
	PublicKey []byte
	TimeStamp int64
}

func (t *Transaction) GetSigner() string {
	return t.Sender
}

func (t *Transaction) GetPublicKey() []byte {
	return t.PublicKey
}

func (t *Transaction) GetCorpus() []byte {
	// Gather corpus to Sign.
	corpus := getBasicCorpus(t)
	corpus.Write([]byte(t.Recipient))
	amountBytes := make([]byte, 16)
	binary.PutUvarint(amountBytes, t.Amount)
	corpus.Write(amountBytes)
	return corpus.Bytes()

}

func (t *Transaction) GetSign() []byte {
	return t.Sign
}

func (t *Transaction) GetTimestamp() int64 {
	return t.TimeStamp
}

func (t *Transaction) String() string {
	return fmt.Sprintf("%d from %s to %s", t.Amount, t.Sender, t.Recipient)
}

func (t *Transaction) Verify() (result bool, err error) {
	if t.Sender == t.Recipient {
		err = fmt.Errorf("self payment")
		return
	}
	return verify(t)
}

func Pay(private crypto.PrivKey, recipient peer.ID, amount uint64) *Transaction {

	sender, _ := peer.IDFromPrivateKey(private)
	public, _ := private.GetPublic().Bytes()

	t := Transaction{Sender: sender.Pretty(), Recipient: recipient.Pretty(), Amount: amount, PublicKey: public, TimeStamp: GetTimeStamp()}
	sign, _ := private.Sign(t.GetCorpus())
	t.Sign = sign

	return &t
}
