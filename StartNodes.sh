#!/bin/bash

# Number of nodes to start
NUM_NODES=4

for i in $(seq 1 $NUM_NODES)
do
    osascript -e 'tell application "Terminal" to do script "cd '`pwd`' && go run . -node=Node'$i'"'
done

echo "$NUM_NODES nodes started in separate terminal windows."
