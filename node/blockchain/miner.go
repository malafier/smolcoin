package blockchain

import (
	"context"
	"encoding/json"
	"log"
	"slices"
	"strings"
	"sync"
	"time"
)

const DEFAULT_DIFFICULTY int = 5

type Miner struct {
	Mempool      []string
	IsMining     bool
	PrevBlock    *Block
	cancelMining context.CancelFunc
	mutex        sync.Mutex
}

func NewMiner() *Miner {
	return &Miner{
		Mempool:   []string{},
		PrevBlock: &Genesis,
		IsMining:  false,
	}
}

func (m *Miner) AddTransaction(transaction string) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if slices.Contains(m.Mempool, transaction) {
		return false
	}
	m.Mempool = append(m.Mempool, transaction)
	return true
}

// func (m *Miner) DeleteTransactions(data string) {
// 	transactions, err = json.Marshal(data)
// 	if err != nil {
// 		log.Print("[W] Failed to marshal transactions.")
// 	}
//
// 	for _, trans := range transactions {
// 		for i, memEl := range m.Mempool {
// 			if trans == memEl {
// 				m.Mempool = slices.Delete(m.Mempool, i, i)
// 			}
// 		}
// 	}
// }

func (m *Miner) StopMining() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.IsMining {
		m.IsMining = false
		m.cancelMining()
	}
}

// func (m *Miner) GetDifficulty() (int, bool) {
// 	m.mutex.Lock()
// 	defer m.mutex.Unlock()
// 	if len(m.Mempool) == 0 {
// 		return 0, false
// 	}
// 	return m.Mempool[len(m.Mempool)-1].Difficulty, true
// }

func (m *Miner) Mine(difficulty int) *Block {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if len(m.Mempool) == 0 {
		m.IsMining = false
		log.Printf("Nothing to mine.")
		return nil
	}

	var ctx context.Context
	ctx, m.cancelMining = context.WithCancel(context.Background())

	block := miningLoop(ctx, m.Mempool, difficulty, m.PrevBlock)

	return block
}

func miningLoop(ctx context.Context, transactions []string, difficulty int, prevBlock *Block) *Block {
	prefix := strings.Repeat("0", difficulty)
	blockData, err := json.Marshal(transactions)
	if err != nil {
		log.Printf("Failed to marshal transactions.")
	}

	block := &Block{
		Index:     prevBlock.Index + 1,
		PrevHash:  prevBlock.Hash,
		Data:      string(blockData),
		Timestamp: time.Now().Unix(),
		Nonce:     0,
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		blockJson, err := block.SerializeWithoutHash()
		if err != nil {
			log.Printf("Failed to marshal a block.")
		}

		blockHash := hash(blockJson)
		if strings.HasPrefix(blockHash, prefix) {
			block.Hash = blockHash
			return block
		} else {
			block.Nonce++
		}
	}
}
