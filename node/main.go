package main

import (
	"flag"
	"fmt"
	"io"
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
	// Args
	port := flag.Int("p", 3000, "The port number for the node to listen on.")
	host := flag.String("H", "127.0.0.1", "The host address for the node to bind to.")
	diff := flag.Int("d", DEFAULT_DIFFICULTY, "Block mining difficulty")
	peer := flag.String("P", "", "Address to connect to peer (e.g., 127.0.0.1:3001)")
	mine := flag.Bool("M", false, "Flag if node is a miner")
	malicious := flag.Bool("m", false, "Flag if node is malicious")
	flag.Parse()

	// Logging
	file, err := os.OpenFile(fmt.Sprintf("node_%d.log", *port), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open log file: %v\n", err)
		os.Exit(1)
	}

	multiWriter := io.MultiWriter(os.Stderr, file)
	handler := tint.NewHandler(multiWriter, &tint.Options{
		Level:      slog.LevelDebug,
		TimeFormat: time.Kitchen,
	})

	logger := slog.New(handler)
	slog.SetDefault(logger)

	// Setup
	var miner *bc.Miner
	if *mine {
		miner = bc.NewMiner(*diff)
		go miner.ListenAndMine()
	}

	state := NewNodeState(*host, *port, miner, *malicious)
	server := NewServer(state, *peer)

	// Run
	go server.Start()
	state.Mine()
}
