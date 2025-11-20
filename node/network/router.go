package network

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	bc "node/blockchain"
)

type Server struct {
	State *NodeState
	Miner *bc.Miner
}

func NewServer(state *NodeState, initialPeer string, ifMiner bool, difficulty int) *Server {
	var miner *bc.Miner
	if ifMiner {
		miner = bc.NewMiner(difficulty)
	}
	server := &Server{State: state, Miner: miner}
	server.connectToInitialPeer(initialPeer)
	return server
}

func (s *Server) checkPeerHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		peerInfo := r.Header.Get("Peer")
		if s.State.PeerCount() > 3 || peerInfo == "" {
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

		s.State.AddPeer(host, port)

		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Node running on %s", s.State.Addr())
}

func (s *Server) handleAddPeer(w http.ResponseWriter, r *http.Request) {
	req := struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	}{}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid data. 'host' and 'port' are required.")
		return
	}
	defer r.Body.Close()

	if err := s.State.AddPeer(req.Host, req.Port); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondWithJSON(w, http.StatusCreated, map[string]any{
		"message": "Peer added successfully.",
	})
}

func (s *Server) handleGetPeers(w http.ResponseWriter, r *http.Request) {
	respondWithJSON(w, http.StatusOK, map[string]any{"peers": s.State.PeersList()})
}

func (s *Server) handleNewTransaction(w http.ResponseWriter, r *http.Request) {
	var req bc.TransactionMessage
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[E] Failed to decode request: %s\n-- Err: %s", r.Body, err)
		respondWithError(w, http.StatusBadRequest, "Invalid data.")
		return
	}
	defer r.Body.Close()

	log.Printf("[I] Redieved new transaction %s\n", req.Transaction)

	if !req.TransactionIsValid() {
		log.Print("[E] Transaction failed verification.\n")
		respondWithError(w, http.StatusBadRequest, "Transaction failed verification.")
		return
	}

	if s.Miner != nil {
		added := s.Miner.AddTransaction(req.Transaction)
		if !added {
			respondWithJSON(w, http.StatusAccepted, map[string]string{"message": "Transaction already recived."})
			return
		}
	}

	message, err := json.Marshal(req)
	if err != nil {
		log.Print("[E] Failed to marshal reqest for broadcast for some reason")
		respondWithError(w, http.StatusBadRequest, "Failed to marshal reqest for broadcast for some reason")
		return
	}

	go s.broadcastMessage("transaction", message)
	if s.Miner != nil && !s.Miner.IsMining {
		go s.Miner.Mine()
	}

	respondWithJSON(w, http.StatusAccepted, map[string]string{"message": "Transaction accepted and broadcasted"})
}

func (s *Server) handleNewBlock(w http.ResponseWriter, r *http.Request) {
	var req *bc.Block
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid data.")
		return
	}
	defer r.Body.Close()

	if !req.IsValid() {
		respondWithError(w, http.StatusBadRequest, "Invalid block hash")
		return
	}

	if err := s.State.AddBlock(req); err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Block declined: %s.", err.Error()))
		return
	}

	// data := req.Data
	// if s.Miner != nil {
	// 	s.Miner.StopMining()
	// 	s.Miner.DeleteTransactions(data)
	//
	// 	s.State.ChainLock.Lock()
	// 	s.Miner.PrevBlock = req
	// 	go s.Miner.Mine(bc.DEFAULT_DIFFICULTY)
	// }

	message, err := req.SerializeWithoutHash()
	if err != nil {
		log.Print("[E] Failed to marshal reqest for broadcast for some reason")
		respondWithError(w, http.StatusBadRequest, "Failed to marshal reqest for broadcast for some reason")
		return
	}
	log.Printf("[I] Redieved new block %s\n", req.Hash[:10])

	go s.broadcastMessage("block", message)

	respondWithJSON(w, http.StatusAccepted, map[string]string{"message": "Block added to chain"})
}

func (s *Server) broadcastMessage(uri string, payload []byte) {
	peerList := s.State.PeersList()
	log.Printf("[I] Broadcasting message to %d peer(s)...\nmessage: %s\n", len(peerList), string(payload))

	var wg sync.WaitGroup
	for _, peer := range peerList {
		wg.Add(1)
		go func(p Peer) {
			defer wg.Done()
			s.sendMessageToPeer(p, uri, payload)
		}(peer)
	}
	wg.Wait()
	log.Println("[I] Broadcast finished.")
}

func (s *Server) sendMessageToPeer(peer Peer, uri string, payload []byte) {
	url := fmt.Sprintf("http://%s/%s", peer.Addr(), uri)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		log.Printf("  - [W] Failed to create request for %s. Error: %v\n", peer.Addr(), err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Peer", s.State.Addr())

	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("  - [W] Failed to send message to %s. Error: %v\n", peer.Addr(), err)
		if checkPeerDead(peer) {
			log.Printf("  - [E] Failed to connect to %s. Peer removed.\n", peer.Addr())
			s.State.RemovePeer(peer)
		}
		return
	}
	defer resp.Body.Close()

	log.Printf("  - [I] Sent to %s (Status: %s)\n", peer.Addr(), resp.Status)
}

func (s *Server) connectToInitialPeer(initialPeer string) {
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

	url := fmt.Sprintf("http://%s:%d/", host, port)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Peer", s.State.Addr())
	client := http.Client{Timeout: 5 * time.Second}
	if _, err := client.Do(req); err == nil {
		s.State.AddPeer(host, port)
		log.Printf("[I] Successfully connected to initial peer %s\n", initialPeer)
	} else {
		log.Printf("[E] Failed to connect to %s. Error: %v\n", initialPeer, err)
	}
}

func (s *Server) StartServer() {
	router := http.NewServeMux()

	router.HandleFunc("GET /", s.handleIndex)
	router.HandleFunc("GET /peers", s.handleGetPeers)
	router.HandleFunc("POST /peer", s.handleAddPeer)
	router.HandleFunc("POST /block", s.handleNewBlock)
	router.HandleFunc("POST /transaction", s.handleNewTransaction)

	handlerWithMiddleware := s.checkPeerHeader(router)

	serverAddr := s.State.Addr()
	log.Printf("[I] Starting s.State on http://%s\n", serverAddr)
	if err := http.ListenAndServe(serverAddr, handlerWithMiddleware); err != nil {
		log.Fatalf("[F] Failed to start server: %v\n", err)
	}
}
