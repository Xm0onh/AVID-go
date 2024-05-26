package config

import (
	"flag"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

var RendezvousString = flag.String("rendezvous", "libp2p-mdns", "Rendezvous string")

type NodeData struct {
	OriginalData string
	Chunks       [][]byte
	Received     map[int][]byte
}

const (
	DataShards     = 14
	ParityShards   = 6
	ExpectedChunks = 14
)

var (
	Nodes           = 15
	ReceivedChunks  = sync.Map{}
	SentChunks      = sync.Map{}
	NodeMutex       = sync.Mutex{}
	ConnectedPeers  []peer.AddrInfo
	Node1ID         peer.ID
	ReceivedFrom    = sync.Map{}
	Counter         = 0
	ChunksRecByNode = make([][]byte, DataShards+ParityShards)
	ReadyCounter    = 0
	StartTime       time.Time
)
