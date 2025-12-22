package blockchain

import "errors"

var (
	ErrEmptyHash             = errors.New("Hash is empty")
	ErrBlockMarshal          = errors.New("Failed to mashal a block.")
	ErrTransactionParse      = errors.New("Failed to parse transactions")
	ErrHashMismatch          = errors.New("Hashes are not matching")
	ErrNegativeAmmountGiven  = errors.New("Cannot send negative ammount of coins")
	ErrTxFailedSerialization = errors.New("Failed to serialize transaction")
	ErrSignatureInvalid      = errors.New("Signature is invalid")
)
