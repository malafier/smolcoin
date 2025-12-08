package blockchain

import (
	"log"
	"slices"
	"strings"
	"sync"
	"time"
)

const DEFAULT_DIFFICULTY int = 5

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
	}
}

func (m *Miner) ListenAndMine() {
	prefix := strings.Repeat("0", m.difficulty)
	log.Println("[Miner] Started mining...")

	block := &Block{}

	// Mining loop
	for {
		select {
		// New transaction
		case newTx := <-m.InTx:
			log.Println("[Miner] New transaction recieved")
			m.mempool = append(m.mempool, newTx)
			m.isStopped = false

			var err error
			block.Transactions, err = transToStr(m.mempool)
			if err != nil {
				log.Fatal("[Miner] Parsing transactions failed misreably. Miner cannot run anymore")
			}
			block.Timestamp = time.Now().Unix()
			block.Nonce = 0

		// New block added, block is reseting
		case payload := <-m.InReset:
			log.Println("[Miner] New transaction recieved")
			m.prevBlock = payload.Block
			m.mempool = payload.NewTsx
			m.isStopped = false

			var err error
			block.Transactions, err = transToStr(m.mempool)
			if err != nil {
				log.Fatal("[Miner] Parsing transactions failed misreably. Miner cannot run anymore")
			}
			block.Timestamp = time.Now().Unix()
			block.PrevHash = m.prevBlock.Hash
			block.Index = m.prevBlock.Index
			block.Nonce = 0

		// Mining
		default:
			if len(m.mempool) == 0 || len(block.Transactions) == 0 || m.isStopped {
				time.Sleep(100 * time.Microsecond)
				continue
			}

			blockJson, err := block.SerializeWithoutHash()
			if err != nil {
				log.Fatal("[Miner] Marshaling block failed misreably. Miner cannot run anymore")
			}

			blockHash := hash(blockJson)
			if strings.HasPrefix(blockHash, prefix) {
				block.Hash = blockHash

				log.Printf("\nNew Block mined\nId: %d Hash: %s\n\n", block.Index, block.Hash[:10])
				m.mempool = m.mempool[:0]
				m.prevBlock = block
				m.OutBlock <- block
				block = &Block{}
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
		hashes[i], _ = tx.Hash() // This ignores possible error, becouse transaction should be checked muliple times at this point
	}
	return slices.Contains(hashes, txHash)
}

func (m *Miner) GetMempoolAndStop() []Transaction {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.isStopped = true
	return m.mempool
}
