# LT-AVID
Asynchronous verifiable information dispersal with libp2p in golang

```go
go run . -node=Node<ID>
```

Run the nodes:
```bash
./StartNodes.sh
```

Stop the all the nodes:
```terminal
ps aux | grep go | grep -v grep | awk '{print $2}' | xargs kill -9
```