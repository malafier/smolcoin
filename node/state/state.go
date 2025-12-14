package state

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	bc "node/blockchain"
)

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
	ledger     map[string]float32
	chainLock  sync.RWMutex
	txHistory  map[string]bool

	// Peers
	Peers     map[string]Peer
	PeersLock sync.RWMutex

	miner *bc.Miner
}

func NewNodeState(host string, port int, miner *bc.Miner) *NodeState {
	node := &NodeState{
		Host:       host,
		Port:       port,
		blockchain: []bc.Block{bc.Genesis},
		txHistory:  make(map[string]bool),
		Peers:      make(map[string]Peer),
		ledger:     make(map[string]float32),
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

	ns.PeersLock.Lock()
	ns.Peers[addr] = newPeer
	ns.PeersLock.Unlock()

	log.Printf("[Node] Added new peer: %s\n", addr)
	return nil
}

func (ns *NodeState) RemovePeer(peer Peer) {
	ns.PeersLock.Lock()
	defer ns.PeersLock.Unlock()
	delete(ns.Peers, peer.Addr())
}

func (ns *NodeState) PeersList() []Peer {
	ns.PeersLock.RLock()
	defer ns.PeersLock.RUnlock()

	peerList := make([]Peer, 0, len(ns.Peers))
	for _, peer := range ns.Peers {
		peerList = append(peerList, peer)
	}

	return peerList
}

func (ns *NodeState) PeerCount() int {
	ns.PeersLock.RLock()
	defer ns.PeersLock.RUnlock()
	return len(ns.Peers)
}

// TODO: dodać walidacje hashy transakcji w bloku -- czy się nie powtarzają
// dodać walidacje wszystkich transakcji
func (ns *NodeState) AddBlock(block *bc.Block) error {
	if !block.IsValid() {
		return errors.New("Given block is not valid")
	}

	ns.chainLock.Lock()
	defer ns.chainLock.Unlock()

	// Validation
	lastBlock := ns.blockchain[len(ns.blockchain)-1]
	if lastBlock.Index+1 != block.Index {
		return errors.New("Indexes mismatch")
	}
	if lastBlock.Hash != block.PrevHash {
		return errors.New("Hash mismatch")
	}
	prefix := strings.Repeat("0", ns.Difficulty)
	if strings.HasPrefix(block.Hash, prefix) {
		return errors.New("Prefix not long enough")
	}

	txs, _ := block.GetTransactions()
	for _, tx := range txs {
		txHash, err := tx.Hash()
		if err != nil {
			return errors.New("Transaction invalid")
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

	// Appeding
	ns.blockchain = append(ns.blockchain, *block)

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
	}

	ns.UpdateLedger()
	payload, _ := block.Serialize()
	go ns.broadcast("block", payload)
	return nil
}

func (ns *NodeState) AddTransaction(tx bc.Transaction) error {
	err := tx.Validate()
	if err != nil {
		return fmt.Errorf("Transaction invalid: %s", err)
	}

	hash, err := tx.Hash()
	if err != nil {
		return errors.New("Something went wrong with transaction. Sorry")
	}
	_, ok := ns.txHistory[hash]
	if ok {
		return errors.New("Transaction already registered")
	}
	ledger := ns.ledgerWithMempool()
	record := ledger[tx.SenderId()]
	if record-tx.Ammount < 0.0 {
		return errors.New("Cannot send more coins than what you have")
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

	ledger := make(map[string]float32)
	for _, block := range ns.blockchain {
		tsx, err := block.ParseTransactions()
		if err != nil {
			log.Print("Failed to parse transactions for some reason")
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

func (ns *NodeState) GetLedger() map[string]float32 {
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
		log.Print("[Node] Id added to ledger")
	}
}

func (ns *NodeState) Mine() {
	if ns.miner == nil {
		for {
		}
	}

	log.Printf("[Node] Waiting for miner...")
	for block := range ns.miner.OutBlock {
		log.Printf("Mined new block: %s\n", block.Hash[:16])
		ns.AddBlock(block)
	}
}

func (ns *NodeState) broadcast(uri string, payload []byte) {
	peerList := ns.PeersList()
	log.Printf("[Node] Broadcasting message to %d peer(s)...\nmessage: %s\n", len(peerList), string(payload))

	var wg sync.WaitGroup
	for _, peer := range peerList {
		wg.Add(1)
		go func(p Peer) {
			defer wg.Done()
			ns.sendReqToPeer(p, uri, payload)
		}(peer)
	}
	wg.Wait()
	log.Println("[Node] Broadcast finished.")
}

func (ns *NodeState) sendReqToPeer(peer Peer, uri string, payload []byte) {
	url := fmt.Sprintf("http://%s/%s", peer.Addr(), uri)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		log.Printf("  - [W] Failed to create request for %s. Error: %v\n", peer.Addr(), err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Peer", ns.Addr())

	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		ns.RemovePeer(peer)
		log.Printf("  - [W] Failed to send message to %s. Error: %v\n", peer.Addr(), err)
		return
	}
	defer resp.Body.Close()

	log.Printf("  - [I] Sent to %s (Status: %s)\n", peer.Addr(), resp.Status)
}

func (ns *NodeState) ledgerWithMempool() map[string]float32 {
	ns.chainLock.RLock()
	defer ns.chainLock.RUnlock()

	ledger := ns.ledger
	mempool := ns.miner.GetMempool()
	for _, tx := range mempool {
		ledger[tx.SenderId()] -= tx.Ammount
		ledger[tx.RecieverId()] += tx.Ammount
	}
	return ledger
}
