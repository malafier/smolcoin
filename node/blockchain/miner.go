package blockchain

import (
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"
)

type NetPayload struct {
	Block  *Block
	NewTsx []Transaction
}

type Miner struct {
	InTx     chan Transaction
	InReset  chan NetPayload
	OutBlock chan *Block

	mempool    []Transaction
	prevBlock  *Block
	difficulty int
	isStopped  bool
	mu         sync.RWMutex
}

func NewMiner(difficlty int) *Miner {
	return &Miner{
		InTx:       make(chan Transaction),
		InReset:    make(chan NetPayload),
		OutBlock:   make(chan *Block),
		mempool:    []Transaction{},
		difficulty: difficlty,
		prevBlock:  &Genesis,
	}
}

func (m *Miner) ListenAndMine() {
	prefix := strings.Repeat("0", m.difficulty)
	block := &Block{
		Index:    1,
		PrevHash: m.prevBlock.Hash,
	}
	slog.Debug("[Miner] Started mining...")

	// Mining loop
	for {
		if block == nil {
			block = &Block{}
		}

		select {
		// Add transaction
		case newTx := <-m.InTx:
			slog.Info("[Miner] New transaction recieved")
			m.mempool = append(m.mempool, newTx)
			m.isStopped = false
			if block == nil {
				continue
			}

			if len(m.mempool) == 1 {
				coinbase, err := Coinbase(newTx.Sender)
				if err != nil {
					slog.Error("[Miner] Failed to create coinbase", "err", err)
					continue
				}
				m.mempool = append(m.mempool, coinbase)
			}

			var err error
			block.Transactions, err = txsToStr(m.mempool)
			if err != nil {
				slog.Error("[Miner] Parsing transactions failed misreably. Miner cannot run anymore")
				panic(-3)
			}
			block.Timestamp = time.Now().Unix()
			block.Nonce = 0

		// New block added, block is reseting
		case payload := <-m.InReset:
			slog.Info("[Miner] New transaction pool and block recieved")
			m.prevBlock = payload.Block
			m.mempool = payload.NewTsx
			m.isStopped = false

			var err error
			block.Transactions, err = txsToStr(m.mempool)
			if err != nil {
				slog.Error("[Miner] Parsing transactions failed misreably. Miner cannot run anymore")
				panic(-3)
			}
			block.Timestamp = time.Now().Unix()
			block.PrevHash = m.prevBlock.Hash
			block.Index = m.prevBlock.Index + 1
			block.Nonce = 0

		// Mining
		default:
			notMemorySafe := block == nil || m.prevBlock == nil
			if notMemorySafe || len(m.mempool) == 0 || len(block.Transactions) == 0 || m.isStopped {
				time.Sleep(100 * time.Microsecond)
				continue
			}

			blockJson, err := block.SerializeWithoutHash()
			if err != nil {
				slog.Error("[Miner] Marshaling block failed misreably. Miner cannot run anymore")
				panic(-3)
			}

			blockHash := hash(blockJson)
			if strings.HasPrefix(blockHash, prefix) {
				block.Hash = blockHash

				slog.Info("New blck mined", "id", block.Index, "hash", block.Hash[:16])
				m.OutBlock <- block
				m.mempool = m.mempool[:0]
				m.prevBlock = nil
				block = nil
				continue
			}
			block.Nonce++
		}
	}
}

func (m *Miner) MempoolContains(txHash string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	hashes := make([]string, len(m.mempool))
	for i, tx := range m.mempool {
		hashes[i], _ = tx.HashStr() // This ignores possible error, becouse transaction should be checked muliple times at this point
	}
	return slices.Contains(hashes, txHash)
}

func (m *Miner) GetMempool() []Transaction {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.mempool
}

func (m *Miner) Stop() {
	m.isStopped = true
}
