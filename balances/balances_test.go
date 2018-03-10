package balances

import (
	"akhcoin/blockchain"
	"testing"
)

func TestBalances_CollectValidTxns(t *testing.T) {

	b := NewBalances()

	transaction := blockchain.Transaction{Unit: blockchain.Unit{Signer: "bank", TimeStamp: blockchain.GetTimeStamp()}, Recipient: "me", Amount: 42}

	validTxns := b.CollectValidTxns([]blockchain.Transaction{transaction}, true)

	if len(validTxns) != 0 {
		t.Fatalf("invalid tx not filtered")
	}

	b.SubmitReward("bank", 1000)
	validTxns = b.CollectValidTxns([]blockchain.Transaction{transaction}, false)

	if len(validTxns) == 0 {
		t.Fatalf("valid tx filtered")
	}

	transaction2 := blockchain.Transaction{Unit: blockchain.Unit{Signer: "me", TimeStamp: blockchain.GetTimeStamp()}, Recipient: "him", Amount: 24}

	validTxns = b.CollectValidTxns([]blockchain.Transaction{transaction2, transaction}, true)

	if len(validTxns) < 2 {
		t.Fatalf("Valid transactions filtered")
	}

	transaction2.TimeStamp = transaction.TimeStamp - 10000
	validTxns = b.CollectValidTxns([]blockchain.Transaction{transaction2, transaction}, true)

	if len(validTxns) != 1 && validTxns[0].GetSigner() != "bank" {
		t.Fatalf("Invalid transactions not filtered")
	}

	validTxns = b.CollectValidTxns([]blockchain.Transaction{transaction, transaction2}, false)

	if len(validTxns) != 0 {
		t.Fatalf("Invalid transactions not filtered")
	}
}
