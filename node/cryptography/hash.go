package cryptography

import (
	"crypto/sha256"
	"encoding/hex"
)

func HashStr(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func HashBytes(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}
