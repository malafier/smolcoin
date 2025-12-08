package network

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"slices"
	"strconv"
	"time"

	bc "node/blockchain"
	state "node/state"
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

func respondWithMessage(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"message": message})
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func checkPeerDead(peer state.Peer) bool {
	checkClient := http.Client{Timeout: 1 * time.Second}
	_, err := checkClient.Get(fmt.Sprintf("http://%s/", peer.Addr()))
	return err != nil
}

func sliceIntersection(t1, t2 []bc.Transaction) []bc.Transaction {
	set := make(map[bc.Transaction]bool)
	for _, item := range t1 {
		set[item] = false
	}

	var intersection []bc.Transaction
	for _, item := range t2 {
		_, found := set[item]
		if found && slices.Contains(intersection, item) {
			intersection = append(intersection, item)
		}
	}

	return intersection
}

func parsePeer(addr string) (string, int, error) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return "", 0, err
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return "", 0, err
	}
	return host, port, nil
}
