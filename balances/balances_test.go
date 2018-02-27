package balances

import (
	"testing"
	"akhcoin/blockchain"
)

func TestBalances_CollectValidTxns(t *testing.T) {

	b := NewBalances()

	transaction := blockchain.Transaction{Sender: "bank", Recipient: "me", Amount: 42, TimeStamp: blockchain.GetTimeStamp()}

	validTxns := b.CollectValidTxns([]blockchain.Transaction{transaction}, true)

	if len(validTxns) != 0 {
		t.Fatalf("invalid tx not filtered")
	}

	b.SubmitReward("bank", 1000)
	validTxns = b.CollectValidTxns([]blockchain.Transaction{transaction}, false)

	if len(validTxns) == 0  {
		t.Fatalf("valid tx filtered")
	}

	transaction2 := blockchain.Transaction{Sender: "me", Recipient: "him", Amount: 24, TimeStamp: blockchain.GetTimeStamp()}

	validTxns = b.CollectValidTxns([]blockchain.Transaction{transaction2, transaction}, true)

	if len(validTxns) < 2 {
		t.Fatalf("Valid transactions filtered")
	}

	transaction2.TimeStamp = transaction.TimeStamp - 10000
	validTxns = b.CollectValidTxns([]blockchain.Transaction{transaction2, transaction}, true)

	if len(validTxns) != 1 && validTxns[0].Sender != "bank" {
		t.Fatalf("Invalid transactions not filtered")
	}

	validTxns = b.CollectValidTxns([]blockchain.Transaction{transaction, transaction2}, false)

	if len(validTxns) != 0 {
		t.Fatalf("Invalid transactions not filtered")
	}
}