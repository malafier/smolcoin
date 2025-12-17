package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

const COINBASE_SK string = "-----BEGIN PRIVATE KEY-----\nMIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgKkNOqe9c7AZHMNq7\n9wocbkYRvKzn5zDAJ8jgayQBWXehRANCAASfT/1IUVW46ai4Ow7isYzPQwa9Vf2U\nDseGuR4CeDMSO/bhYGp+wVz51XdcGdwoR8ypf6l8o8gWUF7lFy8M37g9\n-----END PRIVATE KEY-----\n"
const COINBASE_PK string = "-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEn0/9SFFVuOmouDsO4rGMz0MGvVX9\nlA7HhrkeAngzEjv24WBqfsFc+dV3XBncKEfMqX+pfKPIFlBe5RcvDN+4PQ==\n-----END PUBLIC KEY-----\n"

type Transaction struct {
	Sender    string  `json:"sender"`
	Reciever  string  `json:"reciever"`
	Ammount   float64 `json:"ammount"`
	Timestamp int     `json:"timestamp"`
	Signature string  `json:"signature"`
}

// TODO: dodać walidaje, tak aby nie można było dodawać coinbase w nieskończoność
func Coinbase(reciever string) (Transaction, error) {
	coinbase := Transaction{
		Sender:    COINBASE_PK,
		Reciever:  reciever,
		Ammount:   2.0,
		Timestamp: int(time.Now().Unix()),
	}

	hashed, err := coinbase.HashByte()
	if err != nil {
		return coinbase, errors.New("Failed to hash coinbase ¯\\_(ツ)_/¯")
	}

	privKey, err := pemToPrivateKey(COINBASE_SK)
	if err != nil {
		return coinbase, fmt.Errorf("Failed to do SK conversion ಠ_ಠ, sorry: %s", err)
	}

	sign, err := ecdsa.SignASN1(rand.Reader, privKey, hashed)
	if err != nil {
		return coinbase, fmt.Errorf("Signing went wrong: %s", err)
	}
	coinbase.Signature = string(sign)

	return coinbase, nil
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
	return hash(serialized), nil
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
		return errors.New("Cannot send minus coins")
	}

	// Signature verification
	publicKey, err := pemToPublicKey(t.Sender)
	if err != nil {
		return errors.New("Failed to recieve public key from login")
	}

	serializedData, err := t.SerializeWithoutSign()
	if err != nil {
		return errors.New("Serialization failed")
	}
	hashedData := sha256.Sum256(serializedData)
	signByte, err := hex.DecodeString(t.Signature)
	if err != nil {
		return fmt.Errorf("Wrong signature: %s", err.Error())
	}

	if !ecdsa.VerifyASN1(publicKey, hashedData[:], signByte) {
		slog.Error("Validation not good",
			"pk", t.Sender,
			"hash", hex.EncodeToString(hashedData[:]),
			"sing", t.Signature,
			"msg", string(serializedData),
		)
		return errors.New("Signature invalid")
	}
	return nil
}

func (t *Transaction) SenderId() string {
	return keyAsId(t.Sender)
}

func (t *Transaction) RecieverId() string {
	return keyAsId(t.Reciever)
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

func pemToPrivateKey(pemKey string) (*ecdsa.PrivateKey, error) {
	trimedBytes := []byte(strings.TrimSpace(pemKey))
	block, rest := pem.Decode(trimedBytes)
	if block == nil || block.Type != "PRIVATE KEY" {
		return nil, fmt.Errorf("Failed to decode PEM block, rest: %s", rest)
	}

	genericPrvKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse DER public key: %v", err)
	}

	ecdsaPrvKey, ok := genericPrvKey.(*ecdsa.PrivateKey)
	if !ok {
		return nil, errors.New("Key is not an ECDSA private key")
	}

	return ecdsaPrvKey, nil
}
