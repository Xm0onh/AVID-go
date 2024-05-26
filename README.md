# LT-AVID
Asynchronous verifiable information dispersal with libp2p in golang

Manual running:
```go
go run ./cmd/. -node=Node<ID>
```

Automatic running:
```bash
./StartNodes.sh
```

Stop all the nodes:
```terminal
ps aux | grep go | grep -v grep | awk '{print $2}' | xargs kill -9
```