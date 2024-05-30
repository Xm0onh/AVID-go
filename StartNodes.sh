#!/bin/bash

ps aux | grep go | grep -v grep | awk '{print $2}' | xargs kill -9

rm -f DispersalNode.txt
rm -f bootstrap_addrs.txt

rm -rf output
mkdir output

# Number of regular nodes to start
NUM_NODES=41
PORT_BASE=4007
BOOTSTRAP_PORT_BASE=4001
BOOTSTRAP_NODES=4
BOOTSTRAP_READY_TIMEOUT=30
IP_BASE="127.0.0."
CODING_METHOD="RS"


# Function to set up IP aliases
setup_ip_aliases() {
    for i in $(seq 1 $((BOOTSTRAP_NODES + NUM_NODES + 1))); do
        ip_suffix=$((i + 1))
        ip="${IP_BASE}${ip_suffix}"
        if ! ifconfig lo0 | grep -q "$ip"; then
            sudo ifconfig lo0 alias $ip
            echo "Aliased $ip to loopback interface."
        else
            echo "$ip is already aliased to loopback interface."
        fi
    done
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

# Start the bootstrap nodes first
for i in $(seq 0 $BOOTSTRAP_NODES)
do
    ip_suffix=$((i + 1))
    ip="${IP_BASE}${ip_suffix}"
    osascript -e "tell application \"Terminal\" to do script \"cd '$(pwd)' && go run ./cmd/main.go -node=BootstrapNode$i -port=$((BOOTSTRAP_PORT_BASE + i - 1)) -bootstrap=true -ip=$ip -coding=$CODING_METHOD\""
done

# Wait for all bootstrap nodes to be ready
for i in $(seq 0 $BOOTSTRAP_NODES)
do
    ip_suffix=$((i + 1))
    ip="${IP_BASE}${ip_suffix}"
    port=$((BOOTSTRAP_PORT_BASE + i - 1))
    echo "Waiting for BootstrapNode$i on IP $ip and port $port to be ready..."
    if ! check_bootstrap_ready $ip $port; then
        echo "BootstrapNode$i on IP $ip and port $port did not become ready in time. Exiting..."
        exit 1
    fi
    echo "BootstrapNode$i on IP $ip and port $port is ready."
done

echo "All bootstrap nodes are ready. Starting regular nodes in 10 seconds..."
sleep 10

echo "Node 1 can be started manually with the following command:"
echo "cd $(pwd) && go run ./cmd/main.go -node=Node1 -port=4006 -ip=${IP_BASE}7 -coding=$CODING_METHOD"

# Start Node 1 automatically
osascript -e "tell application \"Terminal\" to do script \"cd '$(pwd)' && go run ./cmd/main.go -node=Node1 -port=4006 -ip=${IP_BASE}7 -coding=$CODING_METHOD\""
sleep 5

# Start the rest of the regular nodes
for i in $(seq 2 $((NUM_NODES + 1)))
do
    ip_suffix=$((i + 6))
    ip="${IP_BASE}${ip_suffix}"
    osascript -e "tell application \"Terminal\" to do script \"cd '$(pwd)' && go run ./cmd/main.go -node=Node$i -port=$((PORT_BASE + i - 2)) -ip=$ip -coding=$CODING_METHOD\""
    sleep 2
    echo "Node$i started on port $((PORT_BASE + i - 2)) with IP $ip -coding=$CODING_METHOD"
done

echo "$NUM_NODES nodes started in separate terminal windows, excluding Node 1 which should be started manually."
