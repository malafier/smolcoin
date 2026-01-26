package blockchain

import (
	"encoding/json"
	"fmt"

	crypto "node/cryptography"
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
		return ErrEmptyHash
	}

	blockJson, err := b.SerializeWithoutHash()
	if err != nil {
		return ErrBlockMarshal
	}

	txs, err := strToTxs(b.Transactions)
	if err != nil {
		return ErrTransactionParse
	}

	for _, tx := range txs {
		if err := tx.Validate(); err != nil {
			return fmt.Errorf("Transaction in block is invalid: %s", err.Error())
		}
	}

	if crypto.HashStr(blockJson) != b.Hash {
		return ErrHashMismatch
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
		return "", ErrBlockMarshal
	}
	return crypto.HashStr(blockJson), nil
}

func (b *Block) GetTransactions() ([]Transaction, error) {
	return strToTxs(b.Transactions)
}

func (b *Block) Corrupt() {
	b.Hash = "0000000000CORRUPTED000BLOCK000"
}

var Genesis = Block{
	Index:        0,
	PrevHash:     "0",
	Timestamp:    1761773051,
	Transactions: "GENESIS",
	Hash:         "41899db27848fc64a0b598d133d42dc9d9069b7c0f7f2af10ff7d1a6c955f53c",
}
