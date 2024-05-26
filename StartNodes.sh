#!/bin/bash

# Number of nodes to start
NUM_NODES=20

# Start Node 1 in the background
# cd $(pwd)/cmd && go run . -node=Node1 &

# Start the rest of the nodes in separate terminal windows
for i in $(seq 2 $NUM_NODES)
do
    osascript -e 'tell application "Terminal" to do script "cd '$(pwd)' && cd ./cmd && go run . -node=Node'$i'"'
done

echo "$NUM_NODES nodes started. Node 1 in the background, others in separate terminal windows."

