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