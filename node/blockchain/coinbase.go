package blockchain

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	crypto "node/cryptography"
)

const COINBASE_SK string = "-----BEGIN PRIVATE KEY-----\nMIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgKkNOqe9c7AZHMNq7\n9wocbkYRvKzn5zDAJ8jgayQBWXehRANCAASfT/1IUVW46ai4Ow7isYzPQwa9Vf2U\nDseGuR4CeDMSO/bhYGp+wVz51XdcGdwoR8ypf6l8o8gWUF7lFy8M37g9\n-----END PRIVATE KEY-----\n"
const COINBASE_PK string = "-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEn0/9SFFVuOmouDsO4rGMz0MGvVX9\nlA7HhrkeAngzEjv24WBqfsFc+dV3XBncKEfMqX+pfKPIFlBe5RcvDN+4PQ==\n-----END PUBLIC KEY-----\n"
const COINBASE_LOGIN string = "MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgKkNOqe9c7AZHMNq79wocbkYRvKzn5zDAJ8jgayQBWXehRANCAASfT/1IUVW46ai4Ow7isYzPQwa9Vf2UDseGuR4CeDMSO/bhYGp+wVz51XdcGdwoR8ypf6l8o8gWUF7lFy8M37g9"

const COINS_TO_GIVE float64 = 2.0

func Coinbase(reciever string) (Transaction, error) {
	coinbase := Transaction{
		Sender:    COINBASE_PK,
		Reciever:  reciever,
		Ammount:   COINS_TO_GIVE,
		Timestamp: int(time.Now().Unix()),
	}

	hashed, err := coinbase.HashByte()
	if err != nil {
		return coinbase, errors.New("Failed to hash coinbase ¯\\_(ツ)_/¯")
	}

	privKey, err := crypto.PemToPrivateKey(COINBASE_SK)
	if err != nil {
		return coinbase, fmt.Errorf("Failed to do SK conversion ಠ_ಠ, sorry: %s", err)
	}

	sign, err := ecdsa.SignASN1(rand.Reader, privKey, hashed)
	if err != nil {
		return coinbase, fmt.Errorf("Signing went wrong: %s", err)
	}
	coinbase.Signature = hex.EncodeToString(sign)

	return coinbase, nil
}
