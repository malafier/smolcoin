package network

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
	ChainLock  sync.Mutex

	// Mempool
	Mempool  []string
	PoolLock sync.Mutex

	// Peers
	Peers     map[string]Peer
	PeersLock sync.RWMutex
}

func NewNodeState(host string, port int) *NodeState {
	node := &NodeState{
		Host:       host,
		Port:       port,
		Blockchain: []bc.Block{bc.Genesis},
		Peers:      make(map[string]Peer),
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

	log.Printf("[I] Added new peer: %s\n", addr)
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
	return nil
}

func (ns *NodeState) AddTransaction(transaction string) bool {
	ns.PoolLock.Lock()
	defer ns.PoolLock.Unlock()
	if slices.Contains(ns.Mempool, transaction) {
		return false
	}

	ns.ChainLock.Lock()
	defer ns.ChainLock.Unlock()
	for _, block := range ns.Blockchain {
		if slices.Contains(block.Data, transaction) {
			return false
		}
	}
	ns.Mempool = append(ns.Mempool, transaction)
	return true
}
