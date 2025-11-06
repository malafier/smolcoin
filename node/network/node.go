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

type Node struct {
	Host       string
	Port       int
	PeerHeader http.Header
	Blockchain []bc.Block
	Peers      map[string]Peer
	PeersLock  sync.RWMutex
	HttpClient *http.Client
}

func NewNode(host string, port int) *Node {
	node := &Node{
		Host:       host,
		Port:       port,
		Blockchain: []bc.Block{bc.Genesis},
		Peers:      make(map[string]Peer),
	}
	node.PeerHeader = make(http.Header)
	node.PeerHeader.Set("Peer", node.Addr())
	return node
}

func (n *Node) Addr() string {
	return fmt.Sprintf("%s:%d", n.Host, n.Port)
}

func (n *Node) AddPeer(host string, port int) error {
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

func (n *Node) RemovePeer(peer Peer) {
	n.PeersLock.Lock()
	defer n.PeersLock.Unlock()
	delete(n.Peers, peer.Addr())
}

func (n *Node) PeersList() []Peer {
	n.PeersLock.RLock()
	defer n.PeersLock.RUnlock()

	peerList := make([]Peer, 0, len(n.Peers))
	for _, peer := range n.Peers {
		peerList = append(peerList, peer)
	}

	return peerList
}

func (n *Node) PeerCount() int {
	n.PeersLock.RLock()
	defer n.PeersLock.RUnlock()
	return len(n.Peers)

}
