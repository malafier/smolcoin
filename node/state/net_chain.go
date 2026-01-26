package state

import (
	"errors"
	bc "node/blockchain"
)

type ChainNet struct {
	Chain []bc.Block `json:"chain"`
	Len   int        `json:"length"`
}

func (c *ChainNet) Validate() error {
	if c.Chain[0] != bc.Genesis {
		return errors.New("Genesis doesn't match")
	}
	if c.Len <= 1 {
		return nil
	}
	for _, block := range c.Chain[1:] {
		if err := block.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (c *ChainNet) TransactionsCount() int {
	counter := 0
	for _, block := range c.Chain {
		txs, err := block.ParseTransactions()
		if err != nil {
			return -1
		}
		counter += len(txs)
	}
	return counter
}
