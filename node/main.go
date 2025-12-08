package main

import (
	"flag"

	bc "node/blockchain"
	. "node/network"
	. "node/state"
)

func main() {
	port := flag.Int("p", 3000, "The port number for the node to listen on.")
	host := flag.String("H", "127.0.0.1", "The host address for the node to bind to.")
	peer := flag.String("P", "", "Address to connect to peer (e.g., 127.0.0.1:3001)")
	mine := flag.Bool("M", false, "Flag if node is a miner")
	flag.Parse()

	var miner *bc.Miner
	if *mine {
		miner = bc.NewMiner(bc.DEFAULT_DIFFICULTY)
		miner.ListenAndMine()
	}

	state := NewNodeState(*host, *port)

	server := NewServer(state, miner, *peer)
	server.Start()
}
