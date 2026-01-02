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
	slog.Debug("Transactions in block validated", "hash", block.Hash[:16])

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
	_, ok := ns.txHistory[hash]
	if ok {
		return errors.New("Transaction already registered")
	}

	ledger := ns.ledgerWithMempool()
	record := ledger[tx.SenderId()]
	if tx.SenderId() == bc.COINBASE_LOGIN && (tx.Ammount != bc.COINS_TO_GIVE || record-tx.Ammount < -MAX_SMOLCOINS) {
		return errors.New("Max ammount of smolcoins already reached or wrong amount of coins given")
	}
	if tx.SenderId() != bc.COINBASE_LOGIN && record-tx.Ammount < 0 {
		return errors.New("Cannot send more coins than what they have")
	}

	if ns.miner != nil {
		contains := ns.miner.MempoolContains(hash)
		if contains {
			return errors.New("Already have this transaction in mempool")
		}

		ns.miner.InTx <- tx
	}

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
			ledger[tx.SenderId()] -= tx.Ammount
			ledger[tx.RecieverId()] += tx.Ammount
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

// TODO: sort things out with ids and PKs
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
	if ns.miner == nil {
		for {
		}
	}

	slog.Info("[Node] Waiting for miner...")
	for block := range ns.miner.OutBlock {
		slog.Info("Mined new block", "hash", block.Hash[:16])
		err := ns.AddBlock(block)
		if err != nil {
			slog.Error("[Node] Could not add MY own block", "err", err.Error())
		}
	}
}

func (ns *NodeState) GetChain() []bc.Block {
	ns.chainLock.RLock()
	defer ns.chainLock.RUnlock()
	return ns.blockchain
}

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

// TODO: use it
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

		defer resp.Body.Close()
		var chain ChainNet
		if err := json.NewDecoder(resp.Body).Decode(&chain); err != nil {
			slog.Error("Failed to decode chain", "err", err)
			continue
		}

		chains = append(chains, &chain)
	}

	ns.chainLock.Lock()
	defer ns.chainLock.Unlock()
	myChainLen := len(ns.blockchain)

	for i, chain := range chains {
		// Blockchain is roughly the same - skip
		if chain.Len == myChainLen && chain.Chain[chain.Len-1].Hash == ns.blockchain[myChainLen].Hash {
			chains[i] = nil
			continue
		}

		// Invalid
		if chain.Len != len(chain.Chain) {
			chains[i] = nil
			continue
		}

		// Blockchain is shorter - skip
		if chain.Len < myChainLen {
			chains[i] = nil
			continue
		}

		// Blockchain is longer but not long enough to be trusted - skip for now
		if chain.Len < myChainLen+6 {
			chains[i] = nil
			continue
		}

		// Blockchain invalid
		if err := chain.Validate(); err != nil {
			chains[i] = nil
		}
	}

	chains = slices.DeleteFunc(chains, func(c *ChainNet) bool { return c == nil })
	if len(chains) == 0 {
		return
	}

	// TODO: add fallback options with rest of chains
	slices.SortFunc(chains, func(a, b *ChainNet) int { return b.Len - a.Len })
	newChain := chains[0]
	var mempool []bc.Transaction
	if ns.miner != nil {
		mempool = ns.miner.GetMempool()
		ns.miner.Stop()
	}

	// Genesis must be set
	if newChain.Chain[0] != bc.Genesis {
		return
	}

	clear(ns.txHistory)
	ns.blockchain = ns.blockchain[:1]
	for _, block := range ns.blockchain[1:] {
		ns.doAddBlock(&block)
	}

	if ns.miner != nil {
		txsHash := make([]string, len(ns.txHistory))
		i := 0
		for hash := range ns.txHistory {
			txsHash[i] = hash
			i++
		}

		newMempool := make([]bc.Transaction, 0)
		for _, tx := range mempool {
			hash, _ := tx.HashStr()
			if !slices.Contains(txsHash, hash) {
				newMempool = append(newMempool, tx)
			}
		}
		lastBlock := ns.blockchain[len(ns.blockchain)]

		ns.miner.InReset <- bc.NetPayload{Block: &lastBlock, NewTsx: newMempool}
	}
	ns.doUpdateLedger()
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
	url := fmt.Sprintf("http://%s/node/%s", peer.Addr(), uri)

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
			ledger[tx.SenderId()] -= tx.Ammount
			ledger[tx.RecieverId()] += tx.Ammount
		}
	}
	return ledger
}
