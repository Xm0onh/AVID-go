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

// Must change respected to the codign method
const (
	ExpectedChunks = 18
)

// Reed-Solomon parameters
const (
	DataShards   = 5
	ParityShards = 3
)

// Luby-Transform parameters
const (
	LTSourceBlocks      = 10
	LTEncodedBlockCount = 25
	RandomSeed          = 42
)

var (
	CodingMethod   string
	Nodes          = 25
	ReceivedChunks = sync.Map{}
	SentChunks     = sync.Map{}
	NodeMutex      = sync.Mutex{}
	ConnectedPeers []peer.AddrInfo
	Node1ID        peer.ID
	ReceivedFrom   = sync.Map{}
	Counter        = 0
	// Must be changed to the coding method
	// if LT then it should be LTEncodedBlockCount
	// if RS then it should be DataShards + ParityShards
	ChunksRecByNode = make([][]byte, LTEncodedBlockCount)
	ReadyCounter    = 0
	StartTime       time.Time
	OriginalLength  = 14641840
)
