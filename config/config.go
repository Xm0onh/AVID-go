package config

import (
	"flag"
	"math"
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
	ExpectedChunks = 1
)

// Reed-Solomon parameters
const (
	DataShards   = 30
	ParityShards = 5
)

// Luby-Transform parameters
const (
	LTSourceBlocks = 1
	RandomSeed     = 42
)

var (
	c                   = 1.0
	LTEncodedBlockCount = int(c*math.Sqrt(float64(LTSourceBlocks))) + LTSourceBlocks
)

const (
	RaptorSourceBlocks      = 10
	RaptorEncodedBlockCount = 12
)

var (
	NodeID         string
	CodingMethod   string
	Mode           string
	Nodes          = 2
	BazantineNodes = 3
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
	OriginalLength  = 18876679
)

var (
	K = 1
)
