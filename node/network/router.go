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
)

func respondWithJSON(w http.ResponseWriter, code int, payload any) {
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

func checkPeerHeader(next http.Handler, node *Node) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		peerInfo := r.Header.Get("Peer")

		if node.PeerCount() > 3 || peerInfo == "" {
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

		node.AddPeer(host, port)

		next.ServeHTTP(w, r)
	})
}

func index(w http.ResponseWriter, r *http.Request, node *Node) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Node running on %s", node.Addr())
}

func addPeer(w http.ResponseWriter, r *http.Request, node *Node) {
	req := struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	}{}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid data. 'host' and 'port' are required.")
		return
	}
	defer r.Body.Close()

	if err := node.AddPeer(req.Host, req.Port); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondWithJSON(w, http.StatusCreated, map[string]any{
		"message": "Peer added successfully.",
	})
}

func getPeers(w http.ResponseWriter, r *http.Request, node *Node) {
	respondWithJSON(w, http.StatusOK, map[string]interface{}{"peers": node.PeersList()})
}

type MessageRequest struct {
	Message string `json:"message"`
	Sender  string `json:"sender"`
}

func receiveMessage(w http.ResponseWriter, r *http.Request) {
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

func broadcastMessageRoute(w http.ResponseWriter, r *http.Request, node *Node) {
	req := struct {
		Message string `json:"message"`
	}{}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid data. 'message' is required.")
		return
	}
	defer r.Body.Close()

	go broadcastMessage(req.Message, node)

	respondWithJSON(w, http.StatusOK, map[string]string{"message": "Broadcasted succesfully"})
}

func broadcastMessage(message string, node *Node) {
	peerList := node.PeersList()
	log.Printf("Broadcasting message to %d peer(s)...\n", len(peerList))

	payload := MessageRequest{
		Message: message,
		Sender:  node.Addr(),
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
			sendMessageToPeer(p, payloadBytes, node)
		}(peer)
	}
	wg.Wait()
	log.Println("Broadcast finished.")
}

func sendMessageToPeer(peer Peer, payload []byte, node *Node) {
	url := fmt.Sprintf("http://%s/message", peer.Addr())

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		log.Printf("  - [W] Failed to create request for %s. Error: %v\n", peer.Addr(), err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Peer", node.Addr())

	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("  - [W] Failed to send message to %s. Error: %v\n", peer.Addr(), err)
		if checkPeerDead(peer) {
			log.Printf("  - [E] Failed to connect to %s. Peer removed.\n", peer.Addr())
			node.RemovePeer(peer)
		}
		return
	}
	defer resp.Body.Close()

	log.Printf("  - [I] Sent to %s (Status: %s)\n", peer.Addr(), resp.Status)
}

func checkPeerDead(peer Peer) bool {
	checkClient := http.Client{Timeout: 1 * time.Second}
	_, err := checkClient.Get(fmt.Sprintf("http://%s/", peer.Addr()))
	return err != nil
}

func ConnectToInitialPeer(initialPeer string, node *Node) {
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
	req.Header.Set("Peer", node.Addr())
	client := http.Client{Timeout: 5 * time.Second}
	if _, err := client.Do(req); err == nil {
		node.AddPeer(host, port)
		log.Printf("[I] Successfully connected to initial peer %s\n", initialPeer)
	} else {
		log.Printf("[E] Failed to connect to %s. Error: %v\n", initialPeer, err)
	}
}

func StartServer(node *Node) {
	router := http.NewServeMux()

	router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) { index(w, r, node) })
	router.HandleFunc("POST /peer", func(w http.ResponseWriter, r *http.Request) { addPeer(w, r, node) })
	router.HandleFunc("GET /peers", func(w http.ResponseWriter, r *http.Request) { getPeers(w, r, node) })
	router.HandleFunc("POST /message", receiveMessage)
	router.HandleFunc("POST /broadcast", func(w http.ResponseWriter, r *http.Request) { broadcastMessageRoute(w, r, node) })

	handlerWithMiddleware := checkPeerHeader(router, node)

	serverAddr := node.Addr()
	log.Printf("[I] Starting node on http://%s\n", serverAddr)
	if err := http.ListenAndServe(serverAddr, handlerWithMiddleware); err != nil {
		log.Fatalf("[F] Failed to start server: %v\n", err)
	}
}
