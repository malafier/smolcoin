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

type Miner struct {
	mempool      []*Transaction
	isMining     bool
	cancelMining context.CancelFunc
	mutex        sync.Mutex
}

func NewMiner() *Miner {
	return &Miner{
		mempool:  []*Transaction{},
		isMining: false,
	}
}

func (m *Miner) AddTransaction(transaction *Transaction) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.mempool = append(m.mempool, transaction)
}

func (m *Miner) StopMining() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.isMining {
		m.isMining = false
		m.cancelMining()
	}
}

func (m *Miner) GetDifficulty() (int, bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if len(m.mempool) == 0 {
		return 0, false
	}
	return m.mempool[len(m.mempool)-1].Difficulty, true
}

func (m *Miner) Mine(prevBlock *Block, difficulty int) *Block {
	m.mutex.Lock()
	if len(m.mempool) == 0 {
		m.isMining = false
		log.Printf("Nothing to mine.")
		m.mutex.Unlock()
		return nil
	}

	var transactions []*Transaction
	for i, t := range m.mempool {
		if t.Difficulty == difficulty {
			transactions = append(transactions, t)
			m.mempool = slices.Delete(m.mempool, i, i)
		}
	}
	defer m.mutex.Unlock()

	var ctx context.Context
	ctx, m.cancelMining = context.WithCancel(context.Background())

	block := miningLoop(ctx, transactions, difficulty, prevBlock)

	// m.mutex.Lock()
	// defer m.mutex.Unlock()
	if block == nil {
		m.mempool = append(m.mempool, transactions...)
	}
	return block
}

func miningLoop(ctx context.Context, transactions []*Transaction, difficulty int, prevBlock *Block) *Block {
	prefix := strings.Repeat("0", difficulty)
	blockData, err := json.Marshal(transactions)
	if err != nil {
		log.Printf("Failed to mashal transactions.")
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

		blockJson, err := json.Marshal(block)
		if err != nil {
			log.Printf("Failed to mashal a block.")
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
