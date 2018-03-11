package blockchain

import (
	"fmt"

	"bytes"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/libp2p/go-libp2p-peer"
)

type Vote struct {
	Unit
	Candidate string
}

func (v *Vote) GetCorpus() *bytes.Buffer {
	corpus := v.Unit.GetCorpus()
	corpus.Write([]byte(v.Candidate))
	return corpus
}

func (v *Vote) Verify() (result bool, err error) {
	if v.Signer == v.Candidate {
		err = fmt.Errorf("self voting")
		return
	}
	return verify(v)
}

func (v *Vote) String() string {
	return fmt.Sprintf("%s voted for %s", v.Signer, v.Candidate)
}

func NewVote(private crypto.PrivKey, candidate peer.ID) *Vote {

	sender, _ := peer.IDFromPrivateKey(private)
	public, _ := private.GetPublic().Bytes()

	v := Vote{Unit: Unit{Signer: sender.Pretty(), PublicKey: public, TimeStamp: GetTimeStamp()}, Candidate: candidate.Pretty()}
	sign, _ := private.Sign(v.GetCorpus().Bytes())
	v.Sign = sign

	return &v
}
