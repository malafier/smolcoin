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
	"log"
	"strings"
	"time"
)

const COINBASE_SK string = "-----BEGIN PRIVATE KEY-----\nMIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgKkNOqe9c7AZHMNq7\n9wocbkYRvKzn5zDAJ8jgayQBWXehRANCAASfT/1IUVW46ai4Ow7isYzPQwa9Vf2U\nDseGuR4CeDMSO/bhYGp+wVz51XdcGdwoR8ypf6l8o8gWUF7lFy8M37g9\n-----END PRIVATE KEY-----\n"
const COINBASE_PK string = "-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEn0/9SFFVuOmouDsO4rGMz0MGvVX9\nlA7HhrkeAngzEjv24WBqfsFc+dV3XBncKEfMqX+pfKPIFlBe5RcvDN+4PQ==\n-----END PUBLIC KEY-----\n"

type Transaction struct {
	Sender    string  `json:"sender"`
	Reciever  string  `json:"reciever"`
	Ammount   float32 `json:"ammount"`
	Timestamp int     `json:"timestamp"`
	Signature string  `json:"signature"`
}

// TODO: dodać walidaje, tak aby nie można było dodawać coinbase w nieskończoność
func Coinbase(reciever string) Transaction {
	coinbase := Transaction{
		Sender:    COINBASE_PK,
		Reciever:  reciever,
		Ammount:   10.0,
		Timestamp: int(time.Now().Unix()),
	}
	hashed, _ := coinbase.Hash()
	privKey, _ := pemToPrivateKey(COINBASE_SK)
	sign, _ := ecdsa.SignASN1(rand.Reader, privKey, []byte(hashed))
	coinbase.Signature = string(sign)
	return coinbase
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
		Sender:    t.Sender,
		Reciever:  t.Reciever,
		Ammount:   t.Ammount,
		Timestamp: t.Timestamp,
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
		return errors.New("Wrong signature")
	}

	if !ecdsa.VerifyASN1(publicKey, hashedData[:], signByte) {
		log.Printf("[I] Validation not good\npk: %s\nh: %s\ns: %s\ntxt: %s", t.Sender, hex.EncodeToString(hashedData[:]), t.Signature, string(serializedData))
		return errors.New("Signature invalid")
	}
	return nil
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

	privKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse DER public key: %v", err)
	}

	return privKey, nil
}
