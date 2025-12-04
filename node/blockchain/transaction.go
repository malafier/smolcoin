package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"strings"
)

type Transaction struct {
	Sender     string  `json:"sender"`
	Reciever   string  `json:"reciever"`
	Ammount    float32 `json:"ammount"`
	Timestamp  int     `json:"timestamp"`
	Difficulty int     `json:"difficulty"`
	PublicKey  string  `json:"pub_key"`
	Signature  string  `json:"signature"`
}

func (t *Transaction) Serialize() ([]byte, error) {
	return json.Marshal(t)
}

func (t *Transaction) SerializeWithoutSign() ([]byte, error) {
	data := struct {
		Sender     string  `json:"sender"`
		Reciever   string  `json:"reciever"`
		Ammount    float32 `json:"ammount"`
		Timestamp  int     `json:"timestamp"`
		Difficulty int     `json:"difficulty"`
	}{
		Sender:     t.Sender,
		Reciever:   t.Reciever,
		Ammount:    t.Ammount,
		Timestamp:  t.Timestamp,
		Difficulty: t.Difficulty,
	}
	return json.Marshal(data)
}

func (t *Transaction) Hash() (string, error) {
	serialized, err := t.SerializeWithoutSign()
	if err != nil {
		return "", err
	}
	return hash(serialized), err
}

func (t *Transaction) String() string {
	var out bytes.Buffer
	out.WriteString("From: ")
	out.WriteString(t.Sender[:10])
	out.WriteString("  To: ")
	out.WriteString(t.Reciever[:10])
	out.WriteString("  Ammount: ")
	out.WriteString(fmt.Sprintf("%f\n", t.Ammount))
	return out.String()
}

func (t *Transaction) TransactionIsValid() bool {
	publicKey, err := pemToPublicKey(t.PublicKey)
	if err != nil {
		log.Printf("%s\n", err)
		return false
	}

	signature, err := base64.StdEncoding.DecodeString(t.Signature)
	if err != nil {
		log.Printf("%s\n", err)
		return false
	}

	serializedData, err := t.SerializeWithoutSign()
	if err != nil {
		return false
	}
	hashedData := sha256.Sum256(serializedData)
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
