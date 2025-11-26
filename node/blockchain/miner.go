package blockchain

import (
	"context"
	"log"
	"slices"
	"strings"
	"sync"
	"time"
)

const DEFAULT_DIFFICULTY int = 5

type Miner struct {
	Mempool      []string
	PrevBlock    *Block
	IsMining     bool
	Difficulty   int
	cancelMining context.CancelFunc
	mutex        sync.Mutex
}

func NewMiner(difficlty int) *Miner {
	return &Miner{
		Difficulty: difficlty,
		Mempool:    []string{},
		PrevBlock:  &Genesis,
		IsMining:   false,
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

func (m *Miner) StopMining() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.IsMining {
		m.IsMining = false
		m.cancelMining()
	}
}

func (m *Miner) Mine() *Block {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if len(m.Mempool) == 0 {
		m.IsMining = false
		log.Printf("Nothing to mine.")
		return nil
	}

	var ctx context.Context
	ctx, m.cancelMining = context.WithCancel(context.Background())

	block, err := m.miningLoop(ctx)
	if err != nil {
		log.Printf("%s\n", err.Error())
	}

	return block
}

func (m *Miner) miningLoop(ctx context.Context) (*Block, error) {
	prefix := strings.Repeat("0", m.Difficulty)

	block := &Block{
		Index:     m.PrevBlock.Index + 1,
		PrevHash:  m.PrevBlock.Hash,
		Data:      m.Mempool,
		Timestamp: time.Now().Unix(),
		Nonce:     0,
	}

	for {
		select {
		case <-ctx.Done():
			return nil, nil
		default:
		}

		blockJson, err := block.SerializeWithoutHash()
		if err != nil {
			log.Printf("Failed to marshal a block.")
			continue
		}

		blockHash := hash(blockJson)
		if strings.HasPrefix(blockHash, prefix) {
			block.Hash = blockHash
			return block, nil
		}
		block.Nonce++
	}
}
