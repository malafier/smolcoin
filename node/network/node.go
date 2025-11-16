package network

import (
	"errors"
	"fmt"
	"log"
	"net/http"
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

	// Blockchain
	Blockchain []bc.Block
	ChainLock  sync.Mutex

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

func (n *NodeState) Addr() string {
	return fmt.Sprintf("%s:%d", n.Host, n.Port)
}

func (n *NodeState) AddPeer(host string, port int) error {
	newPeer := Peer{Host: host, Port: port}
	addr := newPeer.Addr()

	if addr == n.Addr() {
		return errors.New("Cannot add self as peer.")
	}

	n.PeersLock.Lock()
	n.Peers[addr] = newPeer
	n.PeersLock.Unlock()

	log.Printf("[I] Added new peer: %s\n", addr)
	return nil
}

func (n *NodeState) RemovePeer(peer Peer) {
	n.PeersLock.Lock()
	defer n.PeersLock.Unlock()
	delete(n.Peers, peer.Addr())
}

func (n *NodeState) PeersList() []Peer {
	n.PeersLock.RLock()
	defer n.PeersLock.RUnlock()

	peerList := make([]Peer, 0, len(n.Peers))
	for _, peer := range n.Peers {
		peerList = append(peerList, peer)
	}

	return peerList
}

func (n *NodeState) PeerCount() int {
	n.PeersLock.RLock()
	defer n.PeersLock.RUnlock()
	return len(n.Peers)
}

func (n *NodeState) AddBlock(block *bc.Block) error {
	n.ChainLock.Lock()
	defer n.ChainLock.Unlock()

	lastBlock := n.Blockchain[len(n.Blockchain)-1]
	if lastBlock.Index+1 != block.Index {
		return errors.New("Indexes mismatch")
	}
	if lastBlock.Hash != block.PrevHash {
		return errors.New("Hash mismatch")
	}

	n.Blockchain = append(n.Blockchain, *block)
	return nil
}
