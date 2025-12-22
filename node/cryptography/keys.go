package cryptography

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
)

func PemToPublicKey(pemKey string) (*ecdsa.PublicKey, error) {
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

func PemToPrivateKey(pemKey string) (*ecdsa.PrivateKey, error) {
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
