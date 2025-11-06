package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// ## 1. Block Definition (Unchanged)
type Block struct {
	Index     int    `json:"index"`
	PrevHash  string `json:"prev_hash"`
	Timestamp int64  `json:"timestamp"`
	Data      string `json:"data"`
}

var genesis = Block{
	Index:     0,
	PrevHash:  "0",
	Timestamp: 1761773051,
	Data:      "GENESIS",
}

// ## 2. Node Definition (Unchanged)
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
	Blockchain []Block
	Peers      map[string]Peer
	PeersLock  sync.RWMutex
	HttpClient *http.Client
}

func NewNode(host string, port int) *Node {
	node := &Node{
		Host:       host,
		Port:       port,
		Blockchain: []Block{genesis},
		Peers:      make(map[string]Peer),
		HttpClient: &http.Client{Timeout: 2 * time.Second},
	}
	node.PeerHeader = make(http.Header)
	node.PeerHeader.Set("Peer", node.Addr())
	return node
}

func (n *Node) Addr() string {
	return fmt.Sprintf("%s:%d", n.Host, n.Port)
}

// ## 3. JSON Response Helpers (Unchanged)

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshaling JSON: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Internal Server Error"}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

// ## 4. Middleware (Unchanged)
func (n *Node) checkPeerHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		peerInfo := r.Header.Get("Peer")

		n.PeersLock.RLock()
		peerCount := len(n.Peers)
		n.PeersLock.RUnlock()

		if peerCount > 3 || peerInfo == "" {
			next.ServeHTTP(w, r)
			return
		}

		host, portStr, err := net.SplitHostPort(peerInfo)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		port, err := strconv.Atoi(portStr)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		newPeer := Peer{Host: host, Port: port}
		addr := newPeer.Addr()

		if addr != n.Addr() {
			n.PeersLock.Lock()
			if _, exists := n.Peers[addr]; !exists {
				n.Peers[addr] = newPeer
			}
			n.PeersLock.Unlock()
		}
		next.ServeHTTP(w, r)
	})
}

// ## 5. HTTP Handlers (Method checks removed)

// index handler (GET /)
func (n *Node) index(w http.ResponseWriter, r *http.Request) {
	// The 'if r.Method ...' check is no longer needed
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Node running on %s", n.Addr())
}

// addPeer handler (POST /peer)
type AddPeerRequest struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

func (n *Node) addPeer(w http.ResponseWriter, r *http.Request) {
	// The 'if r.Method ...' check is no longer needed
	var req AddPeerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid data. 'host' and 'port' are required.")
		return
	}
	defer r.Body.Close()

	newPeer := Peer{Host: req.Host, Port: req.Port}
	addr := newPeer.Addr()

	if addr == n.Addr() {
		respondWithJSON(w, http.StatusOK, map[string]string{"message": "Cannot add self as peer."})
		return
	}

	n.PeersLock.Lock()
	n.Peers[addr] = newPeer
	n.PeersLock.Unlock()

	log.Printf("[I] Added new peer: %s\n", addr)

	respondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"message": "Peer added successfully.",
		"peers":   n.getPeerList(),
	})
}

// getPeers handler (GET /peers)
func (n *Node) getPeers(w http.ResponseWriter, r *http.Request) {
	// The 'if r.Method ...' check is no longer needed
	respondWithJSON(w, http.StatusOK, map[string]interface{}{"peers": n.getPeerList()})
}

// receiveMessage handler (POST /message)
type MessageRequest struct {
	Message string `json:"message"`
	Sender  string `json:"sender"`
}

func (n *Node) receiveMessage(w http.ResponseWriter, r *http.Request) {
	// The 'if r.Method ...' check is no longer needed
	var req MessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid data. 'message' is required.")
		return
	}
	defer r.Body.Close()

	if req.Sender == "" {
		req.Sender = "Unknown"
	}

	log.Printf("\n[Message Received from %s]: %s\n\n", req.Sender, req.Message)
	respondWithJSON(w, http.StatusOK, map[string]string{"message": "Message received."})
}

// broadcastMessageRoute handler (POST /broadcast)
type BroadcastRequest struct {
	Message string `json:"message"`
}

func (n *Node) broadcastMessageRoute(w http.ResponseWriter, r *http.Request) {
	// The 'if r.Method ...' check is no longer needed
	var req BroadcastRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid data. 'message' is required.")
		return
	}
	defer r.Body.Close()

	go n.broadcastMessage(req.Message)

	respondWithJSON(w, http.StatusOK, map[string]string{"message": "Broadcast initiated."})
}

// ## 6. Node Logic (Unchanged)

func (n *Node) broadcastMessage(message string) {
	peerList := n.getPeerList()
	log.Printf("Broadcasting message to %d peer(s)...\n", len(peerList))

	payload := MessageRequest{
		Message: message,
		Sender:  n.Addr(),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[E] Failed to marshal broadcast payload: %v\n", err)
		return
	}

	var wg sync.WaitGroup
	for _, peer := range peerList {
		wg.Add(1)
		go func(p Peer) {
			defer wg.Done()
			n.sendMessageToPeer(p, payloadBytes)
		}(peer)
	}
	wg.Wait()
	log.Println("Broadcast finished.")
}

func (n *Node) sendMessageToPeer(peer Peer, payload []byte) {
	url := fmt.Sprintf("http://%s/message", peer.Addr())

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		log.Printf("  - [W] Failed to create request for %s. Error: %v\n", peer.Addr(), err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Peer", n.Addr())

	resp, err := n.HttpClient.Do(req)
	if err != nil {
		log.Printf("  - [W] Failed to send message to %s. Error: %v\n", peer.Addr(), err)
		if n.checkPeerDead(peer) {
			log.Printf("  - [E] Failed to connect to %s. Peer removed.\n", peer.Addr())
			n.removePeer(peer)
		}
		return
	}
	defer resp.Body.Close()

	log.Printf("  - [I] Sent to %s (Status: %s)\n", peer.Addr(), resp.Status)
}

func (n *Node) checkPeerDead(peer Peer) bool {
	checkClient := http.Client{Timeout: 1 * time.Second}
	_, err := checkClient.Get(fmt.Sprintf("http://%s/", peer.Addr()))
	return err != nil
}

func (n *Node) removePeer(peer Peer) {
	n.PeersLock.Lock()
	defer n.PeersLock.Unlock()
	delete(n.Peers, peer.Addr())
}

func (n *Node) getPeerList() []Peer {
	n.PeersLock.RLock()
	defer n.PeersLock.RUnlock()
	peerList := make([]Peer, 0, len(n.Peers))
	for _, peer := range n.Peers {
		peerList = append(peerList, peer)
	}
	return peerList
}

func (n *Node) connectToInitialPeer(initialPeer string) {
	if initialPeer == "" {
		return
	}
	host, portStr, err := net.SplitHostPort(initialPeer)
	if err != nil {
		log.Printf("[E] Invalid initial peer address: %v\n", err)
		return
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		log.Printf("[E] Invalid initial peer port: %v\n", err)
		return
	}
	peerAddr := (Peer{host, port}).Addr()
	url := fmt.Sprintf("http://%s/", peerAddr)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Peer", n.Addr())
	client := http.Client{Timeout: 5 * time.Second}
	if _, err := client.Do(req); err == nil {
		log.Printf("[I] Successfully connected to initial peer %s\n", initialPeer)
		n.PeersLock.Lock()
		n.Peers[initialPeer] = Peer{Host: host, Port: port}
		n.PeersLock.Unlock()
	} else {
		log.Printf("[E] Failed to connect to %s. Error: %v\n", initialPeer, err)
	}
}

// ## 7. Server Function (Updated for Go 1.22+)
func startServer(node *Node) {
	// http.NewServeMux() now returns the enhanced router
	router := http.NewServeMux()

	// Register routes with method-specific patterns
	router.HandleFunc("GET /", node.index)
	router.HandleFunc("POST /peer", node.addPeer)
	router.HandleFunc("GET /peers", node.getPeers)
	router.HandleFunc("POST /message", node.receiveMessage)
	router.HandleFunc("POST /broadcast", node.broadcastMessageRoute)

	// Wrap the router with middleware
	handlerWithMiddleware := node.checkPeerHeader(router)

	// Start the server
	serverAddr := node.Addr()
	log.Printf("[I] Starting node on http://%s\n", serverAddr)
	if err := http.ListenAndServe(serverAddr, handlerWithMiddleware); err != nil {
		log.Fatalf("[F] Failed to start server: %v\n", err)
	}
}

// ## 8. Main Function (Unchanged)
func main() {
	port := flag.Int("p", 3000, "The port number for the node to listen on.")
	host := flag.String("H", "127.0.0.1", "The host address for the node to bind to.")
	peer := flag.String("P", "", "Address to connect to peer (e.g., 127.0.0.1:3001)")
	flag.Parse()

	node := NewNode(*host, *port)
	node.connectToInitialPeer(*peer)
	startServer(node)
}
