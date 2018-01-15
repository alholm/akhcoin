package blockchain

import (
	"crypto/sha256"
	"fmt"
	"bytes"
	"github.com/satori/go.uuid"
	"log"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/libp2p/go-libp2p-peer"
)

type BlockData struct {
	Hash         string
	ParentHash   string
	Transactions []TxWrapper
	Nonce        uuid.UUID
	Sign         []byte
	Signer       string
	PublicKey    []byte
}

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
	return corpus.Bytes()
}

func (b *BlockData) GetSign() []byte {
	return b.Sign
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

func CreateGenesis() *Block {
	return &Block{BlockData{ParentHash: "", Hash: HashStr("genesis")}, nil, nil}
}

func NewBlock(privateKey crypto.PrivKey, parent *Block, txnsPool []Transaction) *Block {
	txnWrappers := collectTransactions(parent.lastTransaction(), txnsPool)

	nonce := uuid.NewV1()

	block := &Block{
		BlockData{
			Transactions: txnWrappers,
			Nonce:        nonce,
			ParentHash:   parent.Hash,
		},
		parent,
		nil,
	}
	block.Hash = Hash(block.GetCorpus())
	parent.Next = block

	//TODO error handling
	id, _ := peer.IDFromPrivateKey(privateKey)
	block.Signer = id.Pretty()
	block.PublicKey, _ = privateKey.GetPublic().Bytes()
	block.Sign, _ = privateKey.Sign(block.GetCorpus())

	return block
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

//TODO think of reaction to invalid block
func Validate(block *Block, chainHead *Block) (valid bool) {
	valid, err := Verify(block)
	if !valid {
		log.Printf("INVALID BLOCK: %s: %s\n", block.Hash, err)
		return
	}

	for _, t := range block.Transactions {
		transaction := t.T
		valid, err := Verify(transaction)
		if !valid {
			log.Printf("INVALID BLOCK TRANSACTION: %s sent %d to %s: %s\n", transaction.GetSigner(), transaction.Amount, transaction.Recipient, err)
			break
		}

		log.Printf("%s sent %d to %s\n", transaction.GetSigner(), transaction.Amount, transaction.Recipient)
	}
	return
}
