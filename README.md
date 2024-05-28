# AVID With LT - RS erasure coding algorithm
Asynchronous verifiable information dispersal with libp2p in golang

Manual running - Bootstrap Nodes:
```bash
go run ./cmd/main.go -node=<BootstrapNode$id> -bootstrap=true -port=<PORT> -ip=<IP> -coding=<RS/LT>
```

Manual running - Dispersal Node:
```bash
go run ./cmd/main.go -node=Node1 -port=<PORT> -ip=<IP> -coding=<RS/LT>
```

Manual running - Retriever Node:
```bash
go run ./cmd/main.go -node=<Node$id> -port=<PORT> -ip=<IP> -coding=<RS/LT>
```

Automatic running (Prefered):
```bash
./StartNodes.sh
```

Stop all the nodes:
```terminal
ps aux | grep go | grep -v grep | awk '{print $2}' | xargs kill -9
```


Config:

```go
const (
	ExpectedChunks = 6
)

// Reed-Solomon parameters
const (
	DataShards   = 5
	ParityShards = 3
)

// Luby-Transform parameters
const (
	LTSourceBlocks      = 5
	LTEncodedBlockCount = 7
	RandomSeed          = 42
)

var (
	NodeID         string
	CodingMethod   string
	Nodes          = 10
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
	ChunksRecByNode = make([][]byte, DataShards+ParityShards)
	ReadyCounter    = 0
	StartTime       time.Time
	OriginalLength  = 29283680
)
```