package state

import bc "node/blockchain"

type ChainNet struct {
	Chain []bc.Block `json:"chain"`
	Len   int        `json:"length"`
}

func (c *ChainNet) Validate() error {
	for _, block := range c.Chain {
		if err := block.Validate(); err != nil {
			return err
		}
	}
	return nil
}
