package network

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
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

func (s *Server) Start() {
	router := http.NewServeMux()
	router.HandleFunc("GET /", s.handleIndex)
	router.HandleFunc("GET /users", s.handleGetIds)
	router.HandleFunc("GET /ledger", s.handleGetLedger)
	router.HandleFunc("GET /peers", s.handleGetPeers)
	router.HandleFunc("POST /peer", s.handleAddPeer)
	router.HandleFunc("POST /block", s.handleNewBlock)
	router.HandleFunc("POST /transaction", s.handleNewTransaction)

	adminRouter := http.NewServeMux()
	adminRouter.HandleFunc("POST /connect", s.handleConnect)
	router.Handle("POST /admin/", http.StripPrefix("/admin", s.checkAdmin(adminRouter)))

	server := http.Server{
		Addr:    s.State.Addr(),
		Handler: s.checkPeerHeader(router),
	}

	slog.Debug(fmt.Sprintf("[Node] Starting node on http://%s", server.Addr))
	if err := server.ListenAndServe(); err != nil {
		slog.Error("Failed to start server", "err", err)
	}
}

func (s *Server) checkPeerHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		peerInfo := r.Header.Get("Peer")
		if peerInfo == "" || s.State.PeerCount() > 3 {
			next.ServeHTTP(w, r)
			return
		}

		host, port, err := parsePeer(peerInfo)
		if err != nil {
			http.Error(w, "Wrong peer address", http.StatusBadRequest)
			return
		}
		s.State.AddPeer(host, port)

		next.ServeHTTP(w, r)
	})
}

func (s *Server) checkAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("X-Admin-Key")
		if key != "secret" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Node running on %s", s.State.Addr())
}

func (s *Server) handleGetIds(w http.ResponseWriter, r *http.Request) {
	slog.Info("[Node] Sent clients")
	respondWithJSON(w, http.StatusOK, s.State.GetIds())
}

func (s *Server) handleGetLedger(w http.ResponseWriter, r *http.Request) {
	req := struct {
		Id string `json:"id"`
	}{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	defer r.Body.Close()
	s.State.AddId(req.Id)
	slog.Info("[Node] Sent ledger")
	respondWithJSON(w, http.StatusOK, s.State.GetLedger())
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
		slog.Error("[Node] Failed to decode request", "err", err)
		respondWithError(w, http.StatusBadRequest, "Invalid data.")
		return
	}
	defer r.Body.Close()

	slog.Info("[Node] Recieved new transactionn", "tx", req.String())

	if err := s.State.AddTransaction(req); err != nil {
		slog.Info("[Node] Transaction declined", "err", err.Error())
		respondWithMessage(w, http.StatusAccepted, err.Error())
		return
	}
	respondWithMessage(w, http.StatusAccepted, "Transaction accepted and broadcasted")
}

func (s *Server) handleNewBlock(w http.ResponseWriter, r *http.Request) {
	var req *bc.Block
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid data.")
		return
	}
	defer r.Body.Close()

	slog.Info("[Node] Recieved new block", "block", req.Hash[:16])

	if err := s.State.AddBlock(req); err != nil {
		slog.Info("[Node] Block declined", "err", err.Error())
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Block declined: %s.", err.Error()))
		return
	}

	slog.Info("[Node] New block added to chain", "hash", req.Hash[:16])
	respondWithMessage(w, http.StatusAccepted, "Block added to chain")
}

func (s *Server) handleGetChain(w http.ResponseWriter, r *http.Request) {
	chain := s.State.GetChain()
	msg, err := json.Marshal(chain)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		slog.Error("Failed to marshall own block chain")
		return
	}
	slog.Info("[Node] Block chain sent")
	respondWithJSON(w, http.StatusOK, msg)
}

func (s *Server) handleConnect(w http.ResponseWriter, r *http.Request) {
	req := struct {
		Peer string `json:"peer"`
	}{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid data.")
		return
	}
	defer r.Body.Close()

	host, port, err := parsePeer(req.Peer)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid data.")
		return
	}
	s.State.AddPeer(host, port)
}

func (s *Server) connectToInitialPeer(initialPeer string) {
	if initialPeer == "" {
		return
	}
	host, port, err := parsePeer(initialPeer)
	if err != nil {
		slog.Error("[Node] Invalid initial peer address", "err", err.Error())
		return
	}

	url := fmt.Sprintf("http://%s:%d/", host, port)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Peer", s.State.Addr())
	client := http.Client{Timeout: 5 * time.Second}
	if _, err := client.Do(req); err == nil {
		s.State.AddPeer(host, port)
	} else {
		slog.Error("[Node] Failed to connect to peer", "peer", initialPeer, "err", err)
	}
}
