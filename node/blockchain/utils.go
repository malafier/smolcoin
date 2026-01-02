package blockchain

import (
	"encoding/json"
)

func txsToStr(transactions []Transaction) (string, error) {
	byteTrans, err := json.Marshal(transactions)
	if err != nil {
		return "", err
	}
	return string(byteTrans), nil
}

func strToTxs(str string) ([]Transaction, error) {
	var parsed []Transaction
	err := json.Unmarshal([]byte(str), &parsed)
	if err != nil {
		return nil, err
	}
	return parsed, nil
}
