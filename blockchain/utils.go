package blockchain

import (
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p-crypto"
)

func NewKeys() (crypto.PrivKey, crypto.PubKey, error) {
	private, public, err := crypto.GenerateKeyPair(crypto.RSA, 2048)
	return private, public, err
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
