package config

import (
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

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
	Nodes           = 20
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
