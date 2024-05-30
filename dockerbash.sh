#!/bin/bash

# Clean up previous runs
docker kill $(docker ps -q)
docker rm $(docker ps -aq)
rm -f DispersalNode.txt
rm -f bootstrap_addrs.txt
rm -rf output
mkdir output

# Number of regular nodes to start
NUM_NODES=21
PORT_BASE=4007
BOOTSTRAP_PORT_BASE=4001
BOOTSTRAP_NODES=1
BOOTSTRAP_READY_TIMEOUT=30
CODING_METHOD="RS"
MODE="download"
IP_BASE="127.0.0."

# Function to check if a bootstrap node is ready
check_bootstrap_ready() {
    local container_name=$1
    local port=$2
    for i in $(seq 1 $BOOTSTRAP_READY_TIMEOUT); do
        if docker exec $container_name nc -z localhost $port; then
            return 0
        fi
        sleep 1
    done
    return 1
}

# Create Docker network if it doesn't already exist
if ! docker network ls | grep -q mynetwork; then
    docker network create mynetwork
else
    echo "Network 'mynetwork' already exists"
fi

# Start the bootstrap nodes first
for i in $(seq 0 $((BOOTSTRAP_NODES - 1)))
do
    ip_suffix=$((i + 1))
    ip="${IP_BASE}${ip_suffix}"
    container_name="bootstrap_node_$i"
    docker run -d --name $container_name --network mynetwork \
        my-go-app \
        -node=BootstrapNode$i \
        -port=$((BOOTSTRAP_PORT_BASE + i)) \
        -bootstrap=true \
        -ip=$ip \
        -mode=$MODE \
        -coding=$CODING_METHOD

    # Check if container started successfully
    if [ $(docker inspect -f '{{.State.Running}}' $container_name) != "true" ]; then
        echo "Error: Container $container_name failed to start."
        docker logs $container_name
        exit 1
    fi
done

# Wait for all bootstrap nodes to be ready
for i in $(seq 0 $((BOOTSTRAP_NODES - 1)))
do
    container_name="bootstrap_node_$i"
    port=$((BOOTSTRAP_PORT_BASE + i))
    echo "Waiting for BootstrapNode$i on port $port to be ready..."
    if ! check_bootstrap_ready $container_name $port; then
        echo "BootstrapNode$i on port $port did not become ready in time. Exiting..."
        exit 1
    fi
    echo "BootstrapNode$i on port $port is ready."
done

echo "All bootstrap nodes are ready. Starting regular nodes in 10 seconds..."
sleep 10

# Start Node 1
docker run -d --name node_1 --network mynetwork \
    my-go-app \
    -node=Node1 \
    -port=4006 \
    -ip="${IP_BASE}7" \
    -mode=$MODE \
    -coding=$CODING_METHOD

# Check if Node 1 started successfully
if [ $(docker inspect -f '{{.State.Running}}' node_1) != "true" ]; then
    echo "Error: Node 1 failed to start."
    docker logs node_1
    exit 1
fi

echo "Node 1 started on port 4006"

# Start the rest of the regular nodes
for i in $(seq 2 $((NUM_NODES + 1)))
do
    ip_suffix=$((i + 6))
    ip="${IP_BASE}${ip_suffix}"
    container_name="node_$i"
    docker run -d --name $container_name --network mynetwork \
        my-go-app \
        -node=Node$i \
        -port=$((PORT_BASE + i - 2)) \
        -ip=$ip \
        -mode=$MODE \
        -coding=$CODING_METHOD

    # Check if container started successfully
    if [ $(docker inspect -f '{{.State.Running}}' $container_name) != "true" ]; then
        echo "Error: Container $container_name failed to start."
        docker logs $container_name
        exit 1
    fi

    sleep 2
    echo "Node$i started on port $((PORT_BASE + i - 2))"
done

echo "$NUM_NODES nodes started in Docker containers."
