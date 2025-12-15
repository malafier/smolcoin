package main

import (
	"flag"
	"log/slog"
	"os"
	"time"

	bc "node/blockchain"
	. "node/network"
	. "node/state"

	tint "github.com/lmittmann/tint"
)

const DEFAULT_DIFFICULTY int = 5

func main() {
	// Logging
	handler := tint.NewHandler(os.Stderr, &tint.Options{
		Level:      slog.LevelDebug,
		TimeFormat: time.Kitchen,
	})

	logger := slog.New(handler)
	slog.SetDefault(logger)

	// Args
	port := flag.Int("p", 3000, "The port number for the node to listen on.")
	host := flag.String("H", "127.0.0.1", "The host address for the node to bind to.")
	diff := flag.Int("d", DEFAULT_DIFFICULTY, "Block mining difficulty")
	peer := flag.String("P", "", "Address to connect to peer (e.g., 127.0.0.1:3001)")
	mine := flag.Bool("M", false, "Flag if node is a miner")
	flag.Parse()

	var miner *bc.Miner
	if *mine {
		miner = bc.NewMiner(*diff)
		go miner.ListenAndMine()
	}

	state := NewNodeState(*host, *port, miner)

	server := NewServer(state, *peer)
	go server.Start()

	state.Mine()
}

// TODO:
// Wielu górników
// Obsługa forków oraz orphan block
// Eksperyment: jak sieć zachowa się, gdy złośliwy węzeł będzie wymuszać tworzenie forków?
