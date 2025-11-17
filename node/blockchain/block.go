package blockchain

import (
	"encoding/json"
	"errors"
	"log"
)

type Block struct {
	Index     int    `json:"index"`
	PrevHash  string `json:"prev_hash"`
	Timestamp int64  `json:"timestamp"`
	Data      string `json:"data"`
	Nonce     int    `json:"nonce"`
	Hash      string `json:"hash,omitempty"`
}

func (b *Block) IsValid() bool {
	if b.Hash != "" {
		return false
	}
	blockJson, err := b.SerializeWithoutHash()
	if err != nil {
		log.Printf("Failed to mashal a block.")
		return false
	}
	return hash(blockJson) == b.Hash
}

func (b *Block) Serialize() ([]byte, error) {
	return json.Marshal(b)
}

func (b *Block) SerializeWithoutHash() ([]byte, error) {
	data := struct {
		Index     int    `json:"index"`
		PrevHash  string `json:"prev_hash"`
		Timestamp int64  `json:"timestamp"`
		Data      string `json:"data"`
		Nonce     int    `json:"nonce"`
	}{
		Index:     b.Index,
		PrevHash:  b.PrevHash,
		Timestamp: b.Timestamp,
		Data:      b.Data,
		Nonce:     b.Nonce,
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

var Genesis = Block{
	Index:     0,
	PrevHash:  "0",
	Timestamp: 1761773051,
	Data:      "GENESIS",
}
