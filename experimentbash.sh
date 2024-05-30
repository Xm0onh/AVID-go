#!/bin/bash

# Clean up previous runs
pkill -f "go run ./cmd/main.go"
rm -f DispersalNode.txt
rm -f bootstrap_addrs.txt
rm -rf output
mkdir output

# Number of regular nodes to start
NUM_NODES=10
PORT_BASE=4007
BOOTSTRAP_PORT_BASE=4001
BOOTSTRAP_NODES=1
CODING_METHOD="RS"
MODE="download"
IP_BASE="127.0.0."

# Function to set up IP aliases
setup_ip_aliases() {
    for i in $(seq 1 $((BOOTSTRAP_NODES + NUM_NODES + 1))); do
        ip_suffix=$((i + 1))
        ip="${IP_BASE}${ip_suffix}"
        if ! ifconfig lo0 | grep -q "$ip"; then
            sudo ifconfig lo0 alias $ip up
            echo "Aliased $ip to loopback interface."
        else
            echo "$ip is already aliased to loopback interface."
        fi
    done
}

setup_ip_aliases

# Function to add network latency
# add_network_latency() {
#     local interface=$1
#     local delay=$2
#     sudo tc qdisc add dev $interface root netem delay $delay
# }

# Function to check if a bootstrap node is ready
check_bootstrap_ready() {
    local ip=$1
    local port=$2
    for i in $(seq 1 30); do
        if nc -z $ip $port; then
            return 0
        fi
        sleep 1
    done
    return 1
}

# Create tmux session for nodes
tmux new-session -d -s nodes

# Start the bootstrap nodes first
for i in $(seq 0 $((BOOTSTRAP_NODES - 1)))
do
    ip_suffix=$((i + 1))
    ip="${IP_BASE}${ip_suffix}"
    port=$((BOOTSTRAP_PORT_BASE + i))
    session="BootstrapNode$i"
    tmux new-window -t nodes -n $session
    tmux send-keys -t nodes:$session "cd '$(pwd)' && cpulimit -l 50 -- go run ./cmd/main.go -node=BootstrapNode$i -port=$port -bootstrap=true -ip=$ip -mode=$MODE -coding=$CODING_METHOD" C-m
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
        tmux kill-session -t nodes
        exit 1
    fi
    echo "BootstrapNode$i on IP $ip and port $port is ready."
done

echo "All bootstrap nodes are ready. Starting regular nodes in 10 seconds..."
sleep 10

# Start Node 1
session="Node1"
ip="${IP_BASE}7"
tmux new-window -t nodes -n $session
tmux send-keys -t nodes:$session "cd '$(pwd)' && cpulimit -l 50 -- go run ./cmd/main.go -node=Node1 -port=4006 -ip=$ip -mode=$MODE -coding=$CODING_METHOD" C-m

# Add network latency to Node 1
# add_network_latency lo0 "50ms"

# Start the rest of the regular nodes
for i in $(seq 2 $((NUM_NODES + 1)))
do
    ip_suffix=$((i + 6))
    ip="${IP_BASE}${ip_suffix}"
    port=$((PORT_BASE + i - 2))
    session="Node$i"
    tmux new-window -t nodes -n $session
    tmux send-keys -t nodes:$session "cd '$(pwd)' && cpulimit -l $NUM_NODES -- go run ./cmd/main.go -node=Node$i -port=$port -ip=$ip -mode=$MODE -coding=$CODING_METHOD" C-m

    # Add network latency to each node
    # add_network_latency lo0 "$((i * 10))ms"

    sleep 2
    echo "Node$i started on port $port with IP $ip -coding=$CODING_METHOD"
done

echo "$NUM_NODES nodes started in separate tmux windows."

# Attach to the tmux session
tmux attach -t nodes
