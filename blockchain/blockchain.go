package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p-crypto"
	"github.com/libp2p/go-libp2p-peer"
	"github.com/satori/go.uuid"
)

type BlockData struct {
	Hash         string
	ParentHash   string
	Transactions Transactions
	Votes        Votes
	Nonce        uuid.UUID
	Sign         []byte
	Signer       string
	PublicKey    []byte
	TimeStamp    int64
}

type Transactions []TxWrapper

type Votes []Vote

func (b *BlockData) GetSigner() string {
	return b.Signer
}

func (b *BlockData) GetPublicKey() []byte {
	return b.PublicKey
}

func (b *BlockData) GetCorpus() []byte {
	// Gather corpus to Sign.
	corpus := new(bytes.Buffer)
	corpus.Write([]byte(b.ParentHash))
	corpus.Write(b.Nonce.Bytes())
	for _, t := range b.Transactions {
		corpus.Write(t.T.Sign)
	}
	for _, v := range b.Votes {
		corpus.Write(v.Sign)
	}
	timeStampBytes := make([]byte, 16)
	binary.PutVarint(timeStampBytes, b.TimeStamp)
	corpus.Write(timeStampBytes)

	return corpus.Bytes()
}

func (b *BlockData) GetSign() []byte {
	return b.Sign
}

func (b *BlockData) String() string {
	return fmt.Sprintf("%s, %s", b.Hash, b.Nonce.String())
}

type Block struct {
	BlockData
	Parent *Block
	Next   *Block
}

type TxWrapper struct {
	T          *Transaction
	Prev, Next *Transaction
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
	txnWrappers := collectTransactions(parent.lastTransaction(), txnsPool)

	nonce, _ := uuid.NewV1()

	block := &Block{
		BlockData{
			Transactions: txnWrappers,
			Votes:        votesPool,
			Nonce:        nonce,
			ParentHash:   parent.Hash,
		},
		parent,
		nil,
	}
	block.Hash = Hash(block.GetCorpus())
	parent.Next = block
	block.TimeStamp = GetTimeStamp()

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

func (b *Block) lastTransaction() (t *Transaction) {
	l := len(b.Transactions)
	if l > 0 {
		t = b.Transactions[l-1].T
	}
	return
}

//gathers all published but not confirmed transactions
func collectTransactions(prevBlockTxn *Transaction, transactions []Transaction) (txwr []TxWrapper) {

	txwr = make([]TxWrapper, len(transactions))
	prev := prevBlockTxn
	for i, t := range transactions {
		wr := TxWrapper{T: &t, Prev: prev}
		txwr[i] = wr
		if i > 0 {
			txwr[i-1].Next = &t
		}
		prev = &t
	}

	return
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
func Verify(block *BlockData) (valid bool, err error) {
	valid, err = verify(block)
	if !valid {
		err = fmt.Errorf("invalid block: %s: %s", block.Hash, err)
		return
	}

	for _, t := range block.Transactions {
		transaction := t.T
		valid, err = verify(transaction)
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
