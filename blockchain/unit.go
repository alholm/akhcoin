package blockchain

import (
	"bytes"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/libp2p/go-libp2p-peer"
)

type Signable interface {
	GetSigner() string //getters have not conventional form as would interfere with fields names, which can not be private, or would require a lot of setters
	GetPublicKey() []byte
	GetCorpus() *bytes.Buffer
	GetSign() []byte
	GetTimestamp() int64
}

//Basic Unit that can be presented in blockchain
//All fields should be public to be (de)serialize
type Unit struct {
	Signable
	Hash      string
	Signer    string
	Sign      []byte
	PublicKey []byte
	TimeStamp int64
}

func (u *Unit) GetSigner() string {
	return u.Signer
}

func (u *Unit) GetPublicKey() []byte {
	return u.PublicKey
}

func (u *Unit) GetCorpus() *bytes.Buffer {
	corpus := new(bytes.Buffer)
	corpus.Write([]byte(u.Signer))
	corpus.Write(getBytes(u.TimeStamp))
	return corpus
}

func (u *Unit) GetSign() []byte {
	return u.Sign
}

func (u *Unit) GetTimestamp() int64 {
	return u.TimeStamp
}

func verify(s Signable) (result bool, err error) {
	result = false
	id, err := peer.IDB58Decode(s.GetSigner())
	if err != nil {
		return
	}
	public, err := crypto.UnmarshalPublicKey(s.GetPublicKey())
	if err != nil {
		return
	}

	if id.MatchesPublicKey(public) {
		result, err = public.Verify(s.GetCorpus().Bytes(), s.GetSign())
	}
	return
}
