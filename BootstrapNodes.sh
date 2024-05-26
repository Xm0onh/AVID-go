#!/bin/bash

# Start Bootstrap Nodes
for i in {1..5}
do
    osascript -e 'tell application "Terminal" to do script "cd '$(pwd)' && go run cmd/main.go -node=BootstrapNode'$i' -bootstrap -port=400'$i'"'
done
