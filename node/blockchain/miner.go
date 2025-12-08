package blockchain

import (
	"fmt"
	"log"
	"strings"
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
	OutBlock chan Block

	Mempool    []Transaction
	prevBlock  *Block
	Difficulty int
}

func NewMiner(difficlty int) *Miner {
	return &Miner{
		InTx:       make(chan Transaction),
		InReset:    make(chan NetPayload),
		OutBlock:   make(chan Block),
		Mempool:    []Transaction{},
		Difficulty: difficlty,
	}
}

func (m *Miner) ListenAndMine() {
	prefix := strings.Repeat("0", m.Difficulty)

	fmt.Println("[Miner] Started mining...")

	for {
		select {
		case newTx := <-m.InTx:
			fmt.Println("[Miner] New transaction recieved")
			m.Mempool = append(m.Mempool, newTx)
		case payload := <-m.InReset:
			m.prevBlock = payload.Block
			m.Mempool = payload.NewTsx
		default:
			if len(m.Mempool) == 0 {
				time.Sleep(100 * time.Microsecond)
				continue
			}

			m.miningLoop(prefix)
		}
	}
}

func (m *Miner) miningLoop(prefix string) (*Block, error) {
	transactions, err := transToStr(m.Mempool)
	if err != nil {
		return nil, err
	}

	block := &Block{
		Index:        m.prevBlock.Index + 1,
		PrevHash:     m.prevBlock.PrevHash,
		Transactions: transactions,
		Timestamp:    time.Now().Unix(),
		Nonce:        0,
	}

	for {
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
