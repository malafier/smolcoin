package blockchain

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"strings"
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
	PublicKey   string `json:"pub_key"`
}

func (tm *TransactionMessage) TransactionIsValid() bool {
	publicKey, err := pemToPublicKey(tm.PublicKey)
	if err != nil {
		log.Printf("%s\n", err)
		return false
	}

	signature, err := base64.StdEncoding.DecodeString(tm.Signature)
	if err != nil {
		log.Printf("%s\n", err)
		return false
	}

	hashedData := sha256.Sum256([]byte(tm.Transaction))
	return ecdsa.VerifyASN1(publicKey, hashedData[:], signature)
}

func pemToPublicKey(pemKey string) (*ecdsa.PublicKey, error) {
	trimedBytes := []byte(strings.TrimSpace(pemKey))
	block, rest := pem.Decode(trimedBytes)
	if block == nil || block.Type != "PUBLIC KEY" {
		return nil, fmt.Errorf("Failed to decode PEM block, rest: %s", rest)
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
