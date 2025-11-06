package blockchain

type Block struct {
	Index     int    `json:"index"`
	PrevHash  string `json:"prev_hash"`
	Timestamp int64  `json:"timestamp"`
	Data      string `json:"data"`
}

var Genesis = Block{
	Index:     0,
	PrevHash:  "0",
	Timestamp: 1761773051,
	Data:      "GENESIS",
}
