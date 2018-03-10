package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p-crypto"
	"github.com/libp2p/go-libp2p-peer"
	"github.com/spf13/viper"
)

//All fields should be public to be able to deserialize
type BlockData struct {
	Hash         string
	ParentHash   string
	Transactions Transactions
	Votes        Votes
	Sign         []byte
	Signer       string
	PublicKey    []byte
	TimeStamp    int64
	Reward       uint
}

type Transactions []Transaction

type Votes []Vote

func (b *BlockData) GetSigner() string {
	return b.Signer
}

func (b *BlockData) GetPublicKey() []byte {
	return b.PublicKey
}

func getBasicCorpus(s Signable) *bytes.Buffer {
	corpus := new(bytes.Buffer)
	corpus.Write([]byte(s.GetSigner()))
	corpus.Write(getBytes(s.GetTimestamp()))
	return corpus
}

func (b *BlockData) GetCorpus() []byte {
	// Gather corpus to Sign.
	corpus := getBasicCorpus(b)
	corpus.Write([]byte(b.ParentHash))
	for _, t := range b.Transactions {
		corpus.Write(t.Sign)
	}
	for _, v := range b.Votes {
		corpus.Write(v.Sign)
	}
	corpus.Write(getBytes(int64(b.Reward)))

	return corpus.Bytes()
}

func getBytes(n int64) []byte {
	bytes := make([]byte, 16)
	binary.PutVarint(bytes, n)
	return bytes
}

func (b *BlockData) GetSign() []byte {
	return b.Sign
}

func (b *BlockData) GetTimestamp() int64 {
	return b.TimeStamp
}

func (b *BlockData) String() string {
	return fmt.Sprintf("%s", b.Hash)
}

type Block struct {
	BlockData
	Parent *Block
	Next   *Block
}

func NewKeys() (crypto.PrivKey, crypto.PubKey, error) {
	private, public, err := crypto.GenerateKeyPair(crypto.RSA, 2048)
	return private, public, err
}

func CreateGenesis() *Block {
	return &Block{Parent: nil, Next: nil, BlockData: BlockData{ParentHash: "", Hash: HashStr("genesis"),
		TimeStamp: time.Date(2018, 02, 13, 06, 00, 00, 00, time.UTC).UnixNano()}}
}

func NewBlock(privateKey crypto.PrivKey, parent *Block, txnsPool []Transaction, votesPool []Vote) *Block {
	block := &Block{
		BlockData{
			Transactions: txnsPool,
			Votes:        votesPool,
			ParentHash:   parent.Hash,
		},
		parent,
		nil,
	}
	block.Hash = Hash(block.GetCorpus())
	parent.Next = block
	block.TimeStamp = GetTimeStamp()
	block.Reward = uint(viper.GetInt("reward"))

	//TODO error handling
	id, _ := peer.IDFromPrivateKey(privateKey)
	block.Signer = id.Pretty()
	block.PublicKey, _ = privateKey.GetPublic().Bytes()
	block.Sign, _ = privateKey.Sign(block.GetCorpus())

	return block
}

//GetTimeStamp returns current timestamp
func GetTimeStamp() int64 {
	return CurrentTime().UnixNano()
}

//TODO implement network time adjustment
func CurrentTime() time.Time {
	return time.Now().UTC()
}

func HashStr(str string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(str)))
}

func Hash(bytes []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(bytes))
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
		result, err = public.Verify(s.GetCorpus(), s.GetSign())
	}
	return
}

//Block cryptographic verification
func Verify(block *BlockData, parent *BlockData) (valid bool, err error) {
	if block.ParentHash != parent.Hash {
		err = fmt.Errorf("block %s has %s ParentHash, %s required", block.Hash, block.ParentHash, parent.Hash)
		return
	}

	requiredReward := uint(viper.GetInt("reward"))
	if block.Reward != requiredReward {
		err = fmt.Errorf("block %s has incorrect reward = %d, required: %d", block.Hash, block.Reward, requiredReward)
		return
	}

	valid, err = verify(block)
	if !valid {
		err = fmt.Errorf("invalid block: %s: %s", block.Hash, err)
		return
	}

	lastTS := parent.GetTimestamp()
	for _, t := range block.Transactions {
		transaction := t
		//Transactions must be crated only within block production time frame
		if t.GetTimestamp() < lastTS || t.GetTimestamp() > block.TimeStamp {
			err = fmt.Errorf("block %s transaction: %s has wrong timestamp", block.Hash, t)
			return
		}
		lastTS = t.GetTimestamp()

		valid, err = verify(&transaction)
		if !valid {
			err = fmt.Errorf("invalid transaction in block: %s sent %d to %s: %s", transaction.GetSigner(),
				transaction.Amount, transaction.Recipient, err)
			return
		}
	}

	for _, v := range block.Votes {
		valid, err = verify(&v)
		if !valid {
			err = fmt.Errorf("invalid vote in block: %s voted for %s: %s", v.GetSigner(), v.Candidate, err)
			return
		}
	}

	return
}
