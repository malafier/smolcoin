package blockchain

type Transaction struct {
	Sender     string  `json:"sender"`
	Reciever   string  `json:"reciever"`
	Ammount    float32 `json:"ammount"`
	Timestamp  int     `json:"timestamp"`
	Difficulty int     `json:"difficulty"`
}
