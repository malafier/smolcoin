package network

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"slices"
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

func checkPeerDead(peer Peer) bool {
	checkClient := http.Client{Timeout: 1 * time.Second}
	_, err := checkClient.Get(fmt.Sprintf("http://%s/", peer.Addr()))
	return err != nil
}

func sliceIntersection(s1, s2 []string) []string {
	set := make(map[string]bool)
	for _, item := range s1 {
		set[item] = false
	}

	var intersection []string
	for _, item := range s2 {
		_, found := set[item]
		if found && slices.Contains(intersection, item) {
			intersection = append(intersection, item)
		}
	}

	return intersection
}
