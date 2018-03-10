package blockchain

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/libp2p/go-libp2p-peer"
)

type Transaction struct {
	Unit
	Recipient string
	Amount    uint64
}

func (t *Transaction) GetCorpus() *bytes.Buffer {
	// Gather corpus to Sign.
	corpus := t.Unit.GetCorpus()
	corpus.Write([]byte(t.Recipient))
	amountBytes := make([]byte, 16)
	binary.PutUvarint(amountBytes, t.Amount)
	corpus.Write(amountBytes)
	return corpus

}

func (t *Transaction) String() string {
	return fmt.Sprintf("%d from %s to %s", t.Amount, t.Signer, t.Recipient)
}

func (t *Transaction) Verify() (result bool, err error) {
	if t.Signer == t.Recipient {
		err = fmt.Errorf("self payment")
		return
	}
	return verify(t)
}

func Pay(private crypto.PrivKey, recipient peer.ID, amount uint64) *Transaction {

	sender, _ := peer.IDFromPrivateKey(private)
	public, _ := private.GetPublic().Bytes()

	t := Transaction{Unit: Unit{Signer: sender.Pretty(), PublicKey: public, TimeStamp: GetTimeStamp()}, Recipient: recipient.Pretty(), Amount: amount}
	sign, _ := private.Sign(t.GetCorpus().Bytes())
	t.Sign = sign

	return &t
}
