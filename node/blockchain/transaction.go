package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	crypto "node/cryptography"
)

type Transaction struct {
	Sender    string  `json:"sender"`
	Reciever  string  `json:"reciever"`
	Ammount   float64 `json:"ammount"`
	Timestamp int     `json:"timestamp"`
	Signature string  `json:"signature"`
}

func (t *Transaction) Serialize() ([]byte, error) {
	return json.Marshal(t)
}

func (t *Transaction) SerializeWithoutSign() ([]byte, error) {
	data := struct {
		Sender    string  `json:"sender"`
		Reciever  string  `json:"reciever"`
		Ammount   float64 `json:"ammount"`
		Timestamp int     `json:"timestamp"`
	}{
		Sender:    t.Sender,
		Reciever:  t.Reciever,
		Ammount:   t.Ammount,
		Timestamp: t.Timestamp,
	}
	return json.Marshal(data)
}

func (t *Transaction) HashByte() ([]byte, error) {
	serialized, err := t.SerializeWithoutSign()
	if err != nil {
		return serialized, err
	}
	hashed := sha256.Sum256(serialized)
	return hashed[:], nil
}

func (t *Transaction) HashStr() (string, error) {
	serialized, err := t.SerializeWithoutSign()
	if err != nil {
		return "", err
	}
	return crypto.HashStr(serialized), nil
}

func (t *Transaction) String() string {
	var out bytes.Buffer
	out.WriteString("From: ")
	out.WriteString(t.SenderId()[:10])
	out.WriteString("  To: ")
	out.WriteString(t.RecieverId()[:10])
	out.WriteString("  Ammount: ")
	out.WriteString(fmt.Sprintf("%f\n", t.Ammount))
	return out.String()
}

func (t *Transaction) Validate() error {
	if t.Ammount < 0.0 {
		return ErrNegativeAmmountGiven
	}

	// Signature verification
	publicKey, err := crypto.PemToPublicKey(t.Sender)
	if err != nil {
		return errors.New("Failed to recieve public key from login")
	}

	serializedData, err := t.SerializeWithoutSign()
	if err != nil {
		return ErrTxFailedSerialization
	}
	hashedData := sha256.Sum256(serializedData)
	signByte, err := hex.DecodeString(t.Signature)
	if err != nil {
		return ErrSignatureInvalid
	}

	if !ecdsa.VerifyASN1(publicKey, hashedData[:], signByte) {
		slog.Error("Validation not good",
			"pk", t.Sender,
			"hash", hex.EncodeToString(hashedData[:]),
			"sing", t.Signature,
			"msg", string(serializedData),
		)
		return ErrSignatureInvalid
	}
	return nil
}

func (t *Transaction) SenderId() string {
	return keyAsId(t.Sender)
}

func (t *Transaction) RecieverId() string {
	return keyAsId(t.Reciever)
}
