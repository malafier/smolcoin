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

	blockToHash := struct {
		Index     int    `json:"index"`
		PrevHash  string `json:"prev_hash"`
		Timestamp int64  `json:"timestamp"`
		Data      string `json:"data"`
		Nonce     int    `json:"nonce"`
	}{}

	blockJson, err := json.Marshal(blockToHash)
	if err != nil {
		log.Printf("Failed to mashal a block.")
		return false
	}
	return hash(blockJson) == b.Hash
}

func (b *Block) CreateHash() (string, error) {
	block := b
	if block.Hash != "" {
		block.Hash = ""
	}

	blockJson, err := json.Marshal(block)
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
