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

func (p Peer) Addr() string {
	return fmt.Sprintf("%s:%d", p.Host, p.Port)
}

type NodeState struct {
	Host       string
	Port       int
	PeerHeader http.Header
	Blockchain []bc.Block
	Peers      map[string]Peer
	PeersLock  sync.RWMutex
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
