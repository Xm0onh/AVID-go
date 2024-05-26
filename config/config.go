package config


import (
	"sync"

	"github.com/libp2p/go-libp2p/core/peer"
)

type NodeData struct {
	OriginalData string
	Chunks       [][]byte
	Received     map[int][]byte
}

var (
	ReceivedChunks  = sync.Map{}
	SentChunks      = sync.Map{}
	NodeMutex       = sync.Mutex{}
	ConnectedPeers  []peer.AddrInfo
	Node1ID         peer.ID 
	ReceivedFrom    = sync.Map{}
	Counter         = 0
	ChunksRecByNode = make([][]byte, 3)
	ReadyCounter    = 0
	ExpectedChunks  = 3
)
