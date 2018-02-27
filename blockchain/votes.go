package blockchain

import (
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
	corpus := getBasicCorpus(v)
	corpus.Write([]byte(v.Candidate))
	return corpus.Bytes()
}

func (v *Vote) GetSign() []byte {
	return v.Sign
}

func (v *Vote) GetTimestamp() int64 {
	return v.TimeStamp
}

func (v *Vote) Verify() (result bool, err error) {
	if v.Voter == v.Candidate {
		err = fmt.Errorf("self voting")
		return
	}
	return verify(v)
}

func (v *Vote) String() string {
	return fmt.Sprintf("%s voted for %s", v.Voter, v.Candidate)
}

func NewVote(private crypto.PrivKey, candidate peer.ID) *Vote {

	sender, _ := peer.IDFromPrivateKey(private)
	public, _ := private.GetPublic().Bytes()

	v := Vote{Voter: sender.Pretty(), Candidate: candidate.Pretty(), PublicKey: public, TimeStamp: GetTimeStamp()}
	sign, _ := private.Sign(v.GetCorpus())
	v.Sign = sign

	return &v
}
