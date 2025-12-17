package blockchain

import (
	"encoding/json"
	"errors"
	"fmt"
)

type Block struct {
	Nonce        int    `json:"nonce"`
	Index        int    `json:"index"`
	PrevHash     string `json:"prev_hash"`
	Timestamp    int64  `json:"timestamp"`
	Transactions string `json:"trans"`
	Hash         string `json:"hash"`
}

func (b *Block) Validate() error {
	if b.Hash == "" {
		return errors.New("Hash is empty")
	}

	blockJson, err := b.SerializeWithoutHash()
	if err != nil {
		return errors.New("Failed to mashal a block.")
	}

	txs, err := strToTxs(b.Transactions)
	if err != nil {
		return errors.New("Failed to parse transactions")
	}

	for _, tx := range txs {
		if err := tx.Validate(); err != nil {
			return fmt.Errorf("Transaction in block is invalid: %s", err.Error())
		}
	}

	if hash(blockJson) != b.Hash {
		return errors.New("Hashes are not matching")
	}

	return nil
}

func (b *Block) ParseTransactions() ([]Transaction, error) {
	var trans []Transaction
	err := json.Unmarshal([]byte(b.Transactions), &trans)
	if err != nil {
		return nil, err
	}
	return trans, nil
}

func (b *Block) Serialize() ([]byte, error) {
	return json.Marshal(b)
}

func (b *Block) SerializeWithoutHash() ([]byte, error) {
	data := struct {
		Index        int    `json:"index"`
		PrevHash     string `json:"prev_hash"`
		Timestamp    int64  `json:"timestamp"`
		Transactions string `json:"trans"`
		Nonce        int    `json:"nonce"`
	}{
		Index:        b.Index,
		PrevHash:     b.PrevHash,
		Timestamp:    b.Timestamp,
		Transactions: b.Transactions,
		Nonce:        b.Nonce,
	}
	return json.Marshal(data)
}

func (b *Block) CreateHash() (string, error) {
	blockJson, err := b.SerializeWithoutHash()
	if err != nil {
		return "", errors.New("Failed to mashal a block.")
	}
	return hash(blockJson), nil
}

func (b *Block) GetTransactions() ([]Transaction, error) {
	return strToTxs(b.Transactions)
}

var Genesis = Block{
	Index:        0,
	PrevHash:     "0",
	Timestamp:    1761773051,
	Transactions: "GENESIS",
	Hash:         "a80c2a115782e2699002fc1104a53507014b0694dd706e2ab1c4172c3ce6d234",
}
