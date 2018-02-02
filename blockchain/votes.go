package blockchain

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/libp2p/go-libp2p-crypto"
	"github.com/libp2p/go-libp2p-peer"
)

type Vote struct {
	Voter     string
	Candidate string
	Sign      []byte
	PublicKey []byte
	TimeStamp int64
}

func (v *Vote) GetSigner() string {
	return v.Voter
}

func (v *Vote) GetPublicKey() []byte {
	return v.PublicKey
}

func (v *Vote) GetCorpus() []byte {
	// Gather corpus to Sign.
	corpus := new(bytes.Buffer)
	corpus.Write([]byte(v.Voter))
	corpus.Write([]byte(v.Candidate))
	timeStampBytes := make([]byte, 8)
	binary.PutVarint(timeStampBytes, v.TimeStamp)
	corpus.Write(timeStampBytes)
	return corpus.Bytes()

}

func (v *Vote) GetSign() []byte {
	return v.Sign
}

func (v *Vote) Verify() (result bool, err error) {
	return verify(v)
}

func (v *Vote) String() string {
	return fmt.Sprintf("%s voted for to %s", v.Voter, v.Candidate)
}

func NewVote(private crypto.PrivKey, candidate peer.ID) *Vote {

	sender, _ := peer.IDFromPrivateKey(private)
	public, _ := private.GetPublic().Bytes()

	v := Vote{Voter: sender.Pretty(), Candidate: candidate.Pretty(), PublicKey: public}
	sign, _ := private.Sign(v.GetCorpus())
	v.Sign = sign

	return &v
}
