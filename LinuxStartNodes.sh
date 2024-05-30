#!/bin/bash

# Kill any running Go processes
ps aux | grep go | grep -v grep | awk '{print $2}' | xargs kill -9

# Remove previous files and create output directory
rm -f DispersalNode.txt
rm -f bootstrap_addrs.txt
rm -rf output
mkdir output

# Configuration
NUM_NODES=21
PORT_BASE=4007
BOOTSTRAP_PORT_BASE=4001
BOOTSTRAP_NODES=3
BOOTSTRAP_READY_TIMEOUT=30
IP_BASE="127.0.0."
CODING_METHOD="RS"
MODE="download"

# Function to set up IP aliases
setup_ip_aliases() {
    for i in $(seq 1 $((BOOTSTRAP_NODES + NUM_NODES + 1))); do
        ip_suffix=$((i + 1))
        ip="${IP_BASE}${ip_suffix}"
        if ! ip addr show lo | grep -q "$ip"; then
            sudo ip addr add $ip dev lo
            echo "Aliased $ip to loopback interface."
        else
            echo "$ip is already aliased to loopback interface."
        fi
    done
    sudo ip link set dev lo up
}

# Function to check if a bootstrap node is ready
check_bootstrap_ready() {
    local ip=$1
    local port=$2
    for i in $(seq 1 $BOOTSTRAP_READY_TIMEOUT); do
        nc -z $ip $port && return 0
        sleep 1
    done
    return 1
}

# Set up IP aliases
setup_ip_aliases

# Start tmux session
tmux new-session -d -s mynodes -n "BootstrapNode0"

# Start the bootstrap nodes first
for i in $(seq 0 $((BOOTSTRAP_NODES - 1)))
do
    ip_suffix=$((i + 1))
    ip="${IP_BASE}${ip_suffix}"
    if [ $i -eq 0 ]; then
        tmux send-keys "go run ./cmd/main.go -node=BootstrapNode$i -port=$((BOOTSTRAP_PORT_BASE + i)) -bootstrap=true -ip=$ip -mode=$MODE -coding=$CODING_METHOD" C-m
    else
        tmux new-window -t mynodes: -n "BootstrapNode$i" "go run ./cmd/main.go -node=BootstrapNode$i -port=$((BOOTSTRAP_PORT_BASE + i)) -bootstrap=true -ip=$ip -mode=$MODE -coding=$CODING_METHOD; read"
    fi
done

# Wait for all bootstrap nodes to be ready
for i in $(seq 0 $((BOOTSTRAP_NODES - 1)))
do
    ip_suffix=$((i + 1))
    ip="${IP_BASE}${ip_suffix}"
    port=$((BOOTSTRAP_PORT_BASE + i))
    echo "Waiting for BootstrapNode$i on IP $ip and port $port to be ready..."
    if ! check_bootstrap_ready $ip $port; then
        echo "BootstrapNode$i on IP $ip and port $port did not become ready in time. Exiting..."
        exit 1
    fi
    echo "BootstrapNode$i on IP $ip and port $port is ready."
done

echo "All bootstrap nodes are ready. Starting regular nodes in 10 seconds..."
sleep 10

# Start Node 1 in the current tmux window
tmux new-window -t mynodes: -n "Node1"
tmux send-keys -t mynodes:Node1 "go run ./cmd/main.go -node=Node1 -port=4006 -ip=${IP_BASE}7 -mode=$MODE -coding=$CODING_METHOD" C-m

# Start the rest of the regular nodes in tmux windows
for i in $(seq 2 $((NUM_NODES + 1)))
do
    ip_suffix=$((i + 6))
    ip="${IP_BASE}${ip_suffix}"
    tmux new-window -t mynodes: -n "Node$i" "go run ./cmd/main.go -node=Node$i -port=$((PORT_BASE + i - 2)) -ip=$ip -mode=$MODE -coding=$CODING_METHOD; read"
    sleep 2
    echo "Node$i started on port $((PORT_BASE + i - 2)) with IP $ip -coding=$CODING_METHOD"
done

echo "$NUM_NODES nodes started in separate tmux windows. Use 'tmux attach -t mynodes' to view them."
