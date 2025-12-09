package state

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"slices"
	"strings"
	"sync"

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

	// Blockchain
	Blockchain []bc.Block
	Ledger     map[string]float32
	ChainLock  sync.RWMutex

	// Peers
	Peers     map[string]Peer
	PeersLock sync.RWMutex

	miner *bc.Miner
}

func NewNodeState(host string, port int, miner *bc.Miner) *NodeState {
	node := &NodeState{
		Host:       host,
		Port:       port,
		Blockchain: []bc.Block{bc.Genesis},
		Peers:      make(map[string]Peer),
		Ledger:     make(map[string]float32),
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

func (ns *NodeState) AddBlock(block *bc.Block) error {
	if !block.IsValid() {
		return errors.New("Given block is not valid")
	}

	ns.ChainLock.Lock()
	defer ns.ChainLock.Unlock()

	lastBlock := ns.Blockchain[len(ns.Blockchain)-1]
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

	ns.Blockchain = append(ns.Blockchain, *block)

	// Reseting miner
	if ns.miner != nil {
		mempool := ns.miner.GetMempoolAndStop()
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
	return nil
}

func (ns *NodeState) AddTransaction(tx bc.Transaction) error {
	if !tx.TransactionIsValid() {
		return errors.New("Transaction failed verification.")
	}

	hash, err := tx.Hash()
	if err != nil {
		return errors.New("Something went wrong with transaction. Sorry")
	}

	if ns.miner != nil {
		contains := ns.miner.MempoolContains(hash)
		if contains {
			return errors.New("Already have this transaction in mempool")
		}

		ns.miner.InTx <- tx
	}

	return nil
}

func (ns *NodeState) UpdateLedger() {
	ns.ChainLock.Lock()
	defer ns.ChainLock.Unlock()

	ledger := make(map[string]float32)
	for _, block := range ns.Blockchain {
		transactions, err := block.ParseTransactions()
		if err != nil {
			log.Print("Failed to paarse transactions for some reason")
			continue
		}

		for _, trans := range transactions {
			ledger[trans.Sender] -= trans.Ammount
			ledger[trans.Reciever] += trans.Ammount
		}
	}
	ns.Ledger = ledger
}

func (ns *NodeState) GetIds() []string {
	ns.ChainLock.RLock()
	defer ns.ChainLock.RUnlock()

	keys := make([]string, 0, len(ns.Ledger))
	for key := range ns.Ledger {
		keys = append(keys, key)
	}
	return keys
}

func (ns *NodeState) GetLedger() map[string]float32 {
	ns.ChainLock.RLock()
	defer ns.ChainLock.RUnlock()
	return ns.Ledger
}

func (ns *NodeState) AddId(id string) {
	ns.ChainLock.Lock()
	defer ns.ChainLock.Unlock()
	_, ok := ns.Ledger[id]
	if !ok {
		ns.Ledger[id] = 0.0
		log.Print("[Node] Id added to ledger")
	}
}
