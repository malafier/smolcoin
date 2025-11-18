package blockchain

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
)

// type Transaction struct {
// 	Sender     string  `json:"sender"`
// 	Reciever   string  `json:"reciever"`
// 	Ammount    float32 `json:"ammount"`
// 	Timestamp  int     `json:"timestamp"`
// 	Difficulty int     `json:"difficulty"`
// }

type TransactionMessage struct {
	Transaction string `json:"transaction"`
	Signature   string `json:"signature"`
	PublicKey   string `json:"sender_pk"`
}

func (tm *TransactionMessage) TransactionIsValid() bool {
	publicKey, err := pemToPublicKey(tm.PublicKey)
	if err != nil {
		log.Fatal(err)
		return false
	}

	hashedData := sha256.Sum256([]byte(tm.Transaction))
	return ecdsa.VerifyASN1(publicKey, hashedData[:], []byte(tm.Signature))

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
