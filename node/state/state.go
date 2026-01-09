package state

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	bc "node/blockchain"
)

const MAX_SMOLCOINS float64 = 100.0
const SYNC_THRESHOLD int = 6

type Peer struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

func (p *Peer) Addr() string {
	return fmt.Sprintf("%s:%d", p.Host, p.Port)
}

type NodeState struct {
	Host       string
	Port       int
	PeerHeader http.Header
	Difficulty int

	// blockchain
	blockchain []bc.Block
	ledger     map[string]float64
	txHistory  map[string]bool
	chainLock  sync.RWMutex

	// Peers
	Peers     map[string]Peer
	peersLock sync.RWMutex

	miner *bc.Miner
}

func NewNodeState(host string, port int, miner *bc.Miner) *NodeState {
	node := &NodeState{
		Host:       host,
		Port:       port,
		blockchain: []bc.Block{bc.Genesis},
		txHistory:  make(map[string]bool),
		Peers:      make(map[string]Peer),
		ledger:     make(map[string]float64),
		miner:      miner,
	}
	node.PeerHeader = make(http.Header)
	node.PeerHeader.Set("Peer", node.Addr())
	return node
}

func (ns *NodeState) Addr() string {
	return fmt.Sprintf("%s:%d", ns.Host, ns.Port)
}

func (ns *NodeState) AddPeer(host string, port int) error {
	newPeer := Peer{Host: host, Port: port}
	addr := newPeer.Addr()

	if addr == ns.Addr() {
		return errors.New("Cannot add self as peer.")
	}

	ns.peersLock.Lock()
	defer ns.peersLock.Unlock()
	ns.Peers[addr] = newPeer

	slog.Info("[Node] Added new peer", "peer", addr)
	return nil
}

func (ns *NodeState) RemovePeer(peer Peer) {
	ns.peersLock.Lock()
	defer ns.peersLock.Unlock()
	delete(ns.Peers, peer.Addr())
}

func (ns *NodeState) PeersList() []Peer {
	ns.peersLock.RLock()
	defer ns.peersLock.RUnlock()

	peerList := make([]Peer, 0, len(ns.Peers))
	for _, peer := range ns.Peers {
		peerList = append(peerList, peer)
	}

	return peerList
}

func (ns *NodeState) PeerCount() int {
	ns.peersLock.RLock()
	defer ns.peersLock.RUnlock()
	return len(ns.Peers)
}

func (ns *NodeState) AddBlock(block *bc.Block) error {
	if err := block.Validate(); err != nil {
		return fmt.Errorf("Block invalid: %s", err.Error())
	}

	ns.chainLock.Lock()
	if err := ns.doAddBlock(block); err != nil {
		ns.chainLock.Unlock()
		return err
	}
	ns.chainLock.Unlock()
	slog.Debug("Block added to blockchain", "hash", block.Hash[:16])

	// Reseting miner
	if ns.miner != nil {
		mempool := ns.miner.GetMempool()
		ns.miner.Stop()
		txs, _ := block.GetTransactions()
		var newMempool []bc.Transaction
		for _, tx := range mempool {
			if !slices.Contains(txs, tx) {
				newMempool = append(newMempool, tx)
			}
		}

		ns.miner.InReset <- bc.NetPayload{
			Block:  block,
			NewTsx: newMempool,
		}

		slog.Debug("Miner reset with new block", "hash", block.Hash[:16])
	}

	ns.UpdateLedger()
	payload, err := block.Serialize()
	if err != nil {
		return fmt.Errorf("Failed to serialize new block. Unable to broadcast: %s", err.Error())
	}
	go ns.broadcast("block", payload)

	return nil
}

func (ns *NodeState) doAddBlock(block *bc.Block) error {
	// Validation
	lastBlock := ns.blockchain[len(ns.blockchain)-1]
	if block.Hash == lastBlock.Hash {
		return fmt.Errorf("Block is already last in chain")
	}
	if lastBlock.Index+1 != block.Index {
		return errors.New("Indexes mismatch")
	}
	if lastBlock.Hash != block.PrevHash {
		return errors.New("Hash mismatch")
	}
	prefix := strings.Repeat("0", ns.Difficulty)
	if !strings.HasPrefix(block.Hash, prefix) {
		return errors.New("Prefix not long enough")
	}
	slog.Debug("Block validated", "hash", block.Hash)

	txs, _ := block.GetTransactions()
	for _, tx := range txs {
		txHash, err := tx.HashStr()
		if err != nil {
			return errors.New("Transaction in block is invalid")
		}
		inBlock, ok := ns.txHistory[txHash]
		if inBlock && ok {
			return errors.New("Transaction already in block")
		}
		err = tx.Validate()
		if ok && err != nil {
			return fmt.Errorf("Transaction is invalid: %s", err)
		}

		ns.txHistory[txHash] = true
	}
	slog.Debug("All transactions in block are valid", "hash", block.Hash[:16])

	// Appeding
	ns.blockchain = append(ns.blockchain, *block)
	return nil
}

func (ns *NodeState) AddTransaction(tx bc.Transaction) error {
	if err := tx.Validate(); err != nil {
		return fmt.Errorf("Transaction invalid: %s", err)
	}

	hash, err := tx.HashStr()
	if err != nil {
		return errors.New("Something went wrong with transaction. Sorry")
	}

	ns.chainLock.RLock()
	_, ok := ns.txHistory[hash]
	ns.chainLock.RUnlock()
	if ok {
		return errors.New("Transaction already registered")
	}

	ledger := ns.ledgerWithMempool()
	record := ledger[tx.SenderId()]
	if tx.SenderId() == bc.COINBASE_LOGIN && (tx.Amount != bc.COINS_TO_GIVE || record-tx.Amount < -MAX_SMOLCOINS) {
		return errors.New("Max ammount of smolcoins already reached or wrong amount of coins given")
	}
	if tx.SenderId() != bc.COINBASE_LOGIN && record-tx.Amount < 0 {
		return errors.New("Cannot send more coins than what they have")
	}

	if ns.miner != nil {
		contains := ns.miner.MempoolContains(hash)
		if contains {
			return errors.New("Already have this transaction in mempool")
		}

		ns.miner.InTx <- tx
	}

	ns.chainLock.Lock()
	defer ns.chainLock.Unlock()
	ns.txHistory[hash] = false
	payload, _ := tx.Serialize()
	go ns.broadcast("transaction", payload)
	return nil
}

func (ns *NodeState) UpdateLedger() {
	ns.chainLock.Lock()
	defer ns.chainLock.Unlock()
	ns.doUpdateLedger()
}

func (ns *NodeState) doUpdateLedger() {
	ledger := make(map[string]float64)
	for _, block := range ns.blockchain {
		tsx, err := block.ParseTransactions()
		if err != nil && block.Index != 0 {
			slog.Error("Failed to parse transactions for some reason")
			continue
		}

		for _, tx := range tsx {
			ledger[tx.SenderId()] -= tx.Amount
			ledger[tx.ReceiverId()] += tx.Amount
		}
	}
	ns.ledger = ledger
}

func (ns *NodeState) GetIds() []string {
	ns.chainLock.RLock()
	defer ns.chainLock.RUnlock()

	keys := make([]string, 0, len(ns.ledger))
	for key := range ns.ledger {
		keys = append(keys, key)
	}
	return keys
}

func (ns *NodeState) GetLedger() map[string]float64 {
	ns.chainLock.RLock()
	defer ns.chainLock.RUnlock()
	return ns.ledger
}

func (ns *NodeState) AddId(id string) {
	ns.chainLock.Lock()
	defer ns.chainLock.Unlock()
	_, ok := ns.ledger[id]
	if !ok {
		ns.ledger[id] = 0.0
		slog.Info("[Node] Id added to ledger")
	}
}

func (ns *NodeState) Mine() {
	if ns.miner != nil {
		go func() {
			slog.Info("[Node] Waiting for miner...")
			for block := range ns.miner.OutBlock {
				slog.Info("Mined new block", "hash", block.Hash[:16])
				err := ns.AddBlock(block)
				if err != nil {
					slog.Error("[Node] Could not add MY own block", "err", err.Error())
				}
			}
		}()
	}

	for {
		time.Sleep(10 * time.Second)
		ns.sync()
	}
}

func (ns *NodeState) GetChain() ChainNet {
	ns.chainLock.RLock()
	defer ns.chainLock.RUnlock()
	return ChainNet{
		Chain: ns.blockchain,
		Len:   len(ns.blockchain),
	}
}

func (ns *NodeState) sync() {
	ns.peersLock.RLock()
	peers := ns.Peers
	ns.peersLock.RUnlock()

	chains := make([]*ChainNet, 0)

	for _, peer := range peers {
		req, err := http.NewRequest("GET", fmt.Sprintf("http://%s/chain", peer.Addr()), nil)
		if err != nil {
			slog.Warn("Failed to connect to peer for block synchronization")
			continue
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Peer", ns.Addr())

		client := http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			ns.RemovePeer(peer)
			slog.Warn("Failed to get sync responce. Peer removed", "peer", peer.Addr())
			continue
		}

		var chain ChainNet
		if err := json.NewDecoder(resp.Body).Decode(&chain); err != nil {
			slog.Error("Failed to decode chain", "err", err)
			continue
		}
		chains = append(chains, &chain)
		resp.Body.Close()
	}

	ns.chainLock.Lock()
	myChainLen := len(ns.blockchain)
	for i, chain := range chains {
		// Invalid
		if chain.Len != len(chain.Chain) {
			slog.Debug("Sync candidate failed: Invalid len")
			chains[i] = nil
			continue
		}

		// Blockchain is roughly the same - skip
		if chain.Len == myChainLen && chain.Chain[chain.Len-1].Hash == ns.blockchain[myChainLen-1].Hash {
			slog.Debug("Sync candidate failed: Similar chains")
			chains[i] = nil
			continue
		}

		// Blockchain is shorter - skip
		if chain.Len < myChainLen {
			slog.Debug("Sync candidate failed: Incoming chain is shorter")
			chains[i] = nil
			continue
		}

		// Blockchain is longer but not long enough to be trusted - skip for now
		if chain.Len < myChainLen+SYNC_THRESHOLD {
			slog.Debug("Sync candidate failed: Incoming chain not long enough")
			chains[i] = nil
			continue
		}

		// Blockchain invalid
		if err := chain.Validate(); err != nil {
			slog.Debug("Sync candidate failed: Chain is invalid")
			chains[i] = nil
		}
	}
	ns.chainLock.Unlock()

	chains = slices.DeleteFunc(chains, func(c *ChainNet) bool { return c == nil })
	if len(chains) == 0 {
		return
	}

	slices.SortFunc(chains, func(a, b *ChainNet) int { return b.Len - a.Len })

	for j := range chains {
		// Staging chain
		candidate := chains[j]
		candidateChain := candidate.Chain
		if len(candidateChain) == 0 || candidateChain[0].Hash != bc.Genesis.Hash {
			continue
		}

		tempTxHistory := make(map[string]bool)

		prefix := strings.Repeat("0", ns.Difficulty)

		for i, block := range candidateChain {
			if hash, err := block.CreateHash(); err != nil || hash != block.Hash {
				slog.Warn("Sync failed: Block hash mismatch", "index", block.Index)
				continue
			}

			txs, err := block.GetTransactions()
			if err != nil {
				slog.Warn("Sync failed: Could not parse transactions", "index", block.Index)
				continue
			}
			for _, tx := range txs {
				txHash, _ := tx.HashStr()
				if tempTxHistory[txHash] {
					slog.Warn("Sync failed: Duplicate transaction found in new chain", "tx", txHash)
					continue
				}
				if err := tx.Validate(); err != nil {
					slog.Warn("Sync failed: Invalid transaction in chain", "err", err)
					continue
				}
				tempTxHistory[txHash] = true
			}

			if i == 0 {
				continue
			}

			prevBlock := candidateChain[i-1]

			if block.PrevHash != prevBlock.Hash {
				slog.Warn("Sync failed: Broken chain link", "index", block.Index)
				continue
			}
			if block.Index != prevBlock.Index+1 {
				slog.Warn("Sync failed: Index mismatch", "index", block.Index)
				continue
			}
			if !strings.HasPrefix(block.Hash, prefix) {
				slog.Warn("Sync failed: Insufficient difficulty", "index", block.Index)
				continue
			}
		}

		// Final check if current chain is still shorter
		ns.chainLock.RLock()
		if len(ns.blockchain)+SYNC_THRESHOLD >= len(candidateChain) {
			continue
		}
		ns.chainLock.RUnlock()

		// Commit staging
		ns.chainLock.Lock()
		slog.Info("Replacing chain with new longer chain", "old_len", len(ns.blockchain), "new_len", len(candidateChain))

		var mempool []bc.Transaction
		if ns.miner != nil {
			mempool = ns.miner.GetMempool()
			ns.miner.Stop()
		}

		ns.blockchain = candidateChain
		ns.txHistory = tempTxHistory

		ns.doUpdateLedger()

		if ns.miner != nil {
			newMempool := make([]bc.Transaction, 0)
			for _, tx := range mempool {
				hash, _ := tx.HashStr()
				if !ns.txHistory[hash] { // Check against the new history
					newMempool = append(newMempool, tx)
				}
			}

			lastBlock := ns.blockchain[len(ns.blockchain)-1]
			ns.miner.InReset <- bc.NetPayload{Block: &lastBlock, NewTsx: newMempool}
		}
		ns.chainLock.Unlock()

		break
	}

}

func (ns *NodeState) broadcast(uri string, payload []byte) {
	peerList := ns.PeersList()
	slog.Info(fmt.Sprintf("[Node] Broadcasting message to %d peer(s)...", len(peerList)))

	var wg sync.WaitGroup
	for _, peer := range peerList {
		wg.Add(1)
		go func(p Peer) {
			defer wg.Done()
			ns.sendReqToPeer(p, uri, payload)
		}(peer)
	}
	wg.Wait()
	slog.Info("[Node] Broadcast finished.")
}

func (ns *NodeState) sendReqToPeer(peer Peer, uri string, payload []byte) {
	url := fmt.Sprintf("http://%s/%s", peer.Addr(), uri)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		slog.Warn("  - Failed to create request for peer", "peer", peer.Addr(), "err", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Peer", ns.Addr())

	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		ns.RemovePeer(peer)
		slog.Warn("  - Failed to send message for peer", "peer", peer.Addr(), "err", err)
		return
	}
	defer resp.Body.Close()

	slog.Info(fmt.Sprintf("  - Sent to %s (Status: %s)", peer.Addr(), resp.Status))
}

func (ns *NodeState) ledgerWithMempool() map[string]float64 {
	ns.chainLock.RLock()
	defer ns.chainLock.RUnlock()

	ledger := ns.ledger
	if ns.miner != nil {
		mempool := ns.miner.GetMempool()
		for _, tx := range mempool {
			ledger[tx.SenderId()] -= tx.Amount
			ledger[tx.ReceiverId()] += tx.Amount
		}
	}
	return ledger
}
