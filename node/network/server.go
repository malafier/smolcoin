package network

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	bc "node/blockchain"
	state "node/state"
)

type Server struct {
	State *state.NodeState
}

func NewServer(state *state.NodeState, initialPeer string) *Server {
	server := &Server{State: state}
	server.connectToInitialPeer(initialPeer)
	return server
}

func (s *Server) checkHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		peerInfo := r.Header.Get("Peer")
		clientInfo := r.Header.Get("Client")

		switch {
		case peerInfo != "":
			if s.State.PeerCount() > 3 {
				next.ServeHTTP(w, r)
				return
			}

			host, port, err := parsePeer(peerInfo)
			if err != nil {
				http.Error(w, "Wrong peer address", http.StatusBadRequest)
				return
			}
			s.State.AddPeer(host, port)

		case clientInfo != "":
			// TODO: handle client pub key

		default:
			http.Error(w, "Request must contain at least one valid header", http.StatusBadRequest)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Node running on %s", s.State.Addr())
}

func (s *Server) handleGetClients(w http.ResponseWriter, r *http.Request) {
	respondWithJSON(w, http.StatusOK, map[string][]string{"clients": s.State.GetClients()})
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

	respondWithMessage(w, http.StatusCreated, "Peer added successfully.")
}

func (s *Server) handleGetPeers(w http.ResponseWriter, r *http.Request) {
	respondWithJSON(w, http.StatusOK, map[string]any{"peers": s.State.PeersList()})
}

func (s *Server) handleNewTransaction(w http.ResponseWriter, r *http.Request) {
	var req bc.Transaction
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[E] Failed to decode request: %s\n-- Err: %s", r.Body, err)
		respondWithError(w, http.StatusBadRequest, "Invalid data.")
		return
	}
	defer r.Body.Close()

	log.Printf("[I] Recieved new transaction: %s\n", req.String())

	err := s.State.AddTransaction(req)
	if err != nil {
		log.Printf("[Node] Block declined: %s", err.Error())
		respondWithMessage(w, http.StatusAccepted, "Transaction already recived.")
		return
	}

	// message, err := json.Marshal(req)
	// if err != nil {
	// 	log.Print("[E] Failed to marshal reqest for broadcast for some reason")
	// 	respondWithError(w, http.StatusBadRequest, "Failed to marshal reqest for broadcast for some reason")
	// 	return
	// }

	go s.broadcast("transaction", []byte(r.Body.Close().Error()))

	respondWithMessage(w, http.StatusAccepted, "Transaction accepted and broadcasted")
}

func (s *Server) handleNewBlock(w http.ResponseWriter, r *http.Request) {
	var req *bc.Block
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid data.")
		return
	}
	defer r.Body.Close()

	if err := s.State.AddBlock(req); err != nil {
		log.Printf("[Node] Block declined: %s", err.Error())
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Block declined: %s.", err.Error()))
		return
	}

	log.Printf("[Node] Recieved new block %s\n", req.Hash[:10])

	// message, err := req.SerializeWithoutHash()
	// if err != nil {
	// 	log.Print("[E] Failed to marshal reqest for broadcast for some reason")
	// 	respondWithError(w, http.StatusBadRequest, "Failed to marshal reqest for broadcast for some reason")
	// 	return
	// }

	go s.broadcast("block", []byte(r.Body.Close().Error()))

	respondWithMessage(w, http.StatusAccepted, "Block added to chain")
}

func (s *Server) broadcast(uri string, payload []byte) {
	peerList := s.State.PeersList()
	log.Printf("[Node] Broadcasting message to %d peer(s)...\nmessage: %s\n", len(peerList), string(payload))

	var wg sync.WaitGroup
	for _, peer := range peerList {
		wg.Add(1)
		go func(p state.Peer) {
			defer wg.Done()
			s.sendReqToPeer(p, uri, payload)
		}(peer)
	}
	wg.Wait()
	log.Println("[Node] Broadcast finished.")
}

func (s *Server) sendReqToPeer(peer state.Peer, uri string, payload []byte) {
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
	host, port, err := parsePeer(initialPeer)
	if err != nil {
		log.Printf("[E] Invalid initial peer address: %v\n", err)
		return
	}

	url := fmt.Sprintf("http://%s:%d/", host, port)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Peer", s.State.Addr())
	client := http.Client{Timeout: 5 * time.Second}
	if _, err := client.Do(req); err == nil {
		s.State.AddPeer(host, port)
		log.Printf("[Node] Successfully connected to initial peer %s\n", initialPeer)
	} else {
		log.Printf("[E] Failed to connect to %s. Error: %v\n", initialPeer, err)
	}
}

// TODO: make this non blocking
func (s *Server) Start() {
	router := http.NewServeMux()

	router.HandleFunc("GET /", s.handleIndex)
	router.HandleFunc("GET /users", s.handleGetClients)
	router.HandleFunc("GET /peers", s.handleGetPeers)
	router.HandleFunc("POST /peer", s.handleAddPeer)
	router.HandleFunc("POST /block", s.handleNewBlock)
	router.HandleFunc("POST /transaction", s.handleNewTransaction)

	handlerWithMiddleware := s.checkHeaders(router)

	serverAddr := s.State.Addr()
	log.Printf("[Node] Starting node on http://%s\n", serverAddr)
	if err := http.ListenAndServe(serverAddr, handlerWithMiddleware); err != nil {
		log.Fatalf("[F] Failed to start server: %v\n", err)
	}
}
