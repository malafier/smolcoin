package blockchain

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"math/big"
)

type Transaction struct {
	Sender     string  `json:"sender"`
	Reciever   string  `json:"reciever"`
	Ammount    float32 `json:"ammount"`
	Timestamp  int     `json:"timestamp"`
	Difficulty int     `json:"difficulty"`
}

type TransactionMessage struct {
	Transaction *Transaction `json:"transaction"`
	Hash        string       `json:"hash"`
	PublicKey   string       `json:"sender_pk"`
	R           *big.Int     `json:"r"`
	S           *big.Int     `json:"s"`
}

func (tm *TransactionMessage) TransactionIsValid() bool {
	data, err := json.Marshal(tm.Transaction)
	if err != nil {
		log.Printf("Failed to mashal transactions.")
		return false
	}
	hashedData := sha256.Sum256(data)

	publicKey, err := pemToPublicKey(tm.PublicKey)
	if err != nil {
		log.Fatal(err)
		return false
	}

	return ecdsa.Verify(publicKey, hashedData[:], tm.R, tm.S)

}

func pemToPublicKey(pemKey string) (*ecdsa.PublicKey, error) {
	block, rest := pem.Decode([]byte(pemKey))
	if block == nil {
		return nil, fmt.Errorf("Failed to decode PEM block, rest: %s", rest)
	}
	if block.Type != "PUBLIC KEY" {
		return nil, errors.New("PEM block type is not 'PUBLIC KEY'")
	}

	genericPubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse DER public key: %v", err)
	}

	ecdsaPubKey, ok := genericPubKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("Key is not an ECDSA public key")
	}

	return ecdsaPubKey, nil
}
