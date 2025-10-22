#!/bin/bash

set -e

echo "=================================================="
echo "Hippocampus vs FAISS Benchmark Monitor"
echo "=================================================="
echo

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
RED='\033[0;31m'
BOLD='\033[1m'
NC='\033[0m'

# Check for instance details
if [ ! -f ".ec2-instance.txt" ]; then
    echo -e "${RED}✗ No EC2 instance found. Run './setup-ec2.sh' first${NC}"
    exit 1
fi

source .ec2-instance.txt

echo -e "${CYAN}Target Instance:${NC}"
echo "  Instance ID: $INSTANCE_ID"
echo "  IP Address:  $INSTANCE_IP"
echo "  Region:      $REGION"
echo

# Check if the benchmark is running
echo -e "${BLUE}Checking if benchmark is running...${NC}"

if ! ssh -i ~/.ssh/${KEY_NAME}.pem -o StrictHostKeyChecking=no -o ConnectTimeout=5 \
    ec2-user@$INSTANCE_IP "pgrep -f benchmark_scaling.sh" > /dev/null; then
    echo -e "${YELLOW}⚠ No benchmark process found. Either it hasn't started or it's finished.${NC}"
    
    # Check for result file
    if ssh -i ~/.ssh/${KEY_NAME}.pem -o StrictHostKeyChecking=no -o ConnectTimeout=5 \
        ec2-user@$INSTANCE_IP "ls /home/ec2-user/hippocampus-benchmark/benchmark_hippo_*.bin" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Found benchmark files. The benchmark might have completed.${NC}"
        echo -e "   Run './deploy.sh' to download results."
    else
        echo -e "${RED}✗ No benchmark files found. The benchmark may not have started yet.${NC}"
        echo -e "   Run './deploy.sh' to start the benchmark."
    fi
    
    exit 0
fi

echo -e "${GREEN}✓ Benchmark is running!${NC}"
echo

echo -e "${BLUE}Starting live monitoring...${NC}"
echo -e "${YELLOW}Press Ctrl+C to exit monitoring (benchmark will continue running)${NC}"
echo

# Header for the progress display
echo -e "${BOLD}Scale | Status               | Progress           | Timing Details${NC}"
echo -e "------+----------------------+--------------------+-----------------"

# Define regex patterns to match various stages
INSERT_PATTERN="Inserting ([0-9]+) nodes"
SEARCH_PATTERN="Searching ([0-9]+) queries"
PROGRESS_PATTERN="Progress: ([0-9]+)/([0-9]+)"
TIMING_PATTERN="Timing breakdown: Embed=([0-9.]+)ms, Load=([0-9.]+)ms, Search=([0-9.]+)ms"
SCALE_PATTERN="Testing with ([0-9]+) nodes"
FAISS_PATTERN="\[2/2\] FAISS at ([0-9]+) nodes"
HIPPO_PATTERN="\[1/2\] Hippocampus at ([0-9]+) nodes"
FAISS_PURE_PATTERN="Pure FAISS search: ([0-9.]+)ms"

# Initialize variables
current_scale=""
current_phase=""
progress=""
timing=""

# Monitor function for continuous updates
monitor_benchmark() {
    # Use tail -f to continuously watch the log file
    ssh -i ~/.ssh/${KEY_NAME}.pem -o StrictHostKeyChecking=no \
        ec2-user@$INSTANCE_IP "tail -f /tmp/benchmark.log" 2>/dev/null | \
    while read line; do
        # Extract current scale
        if [[ $line =~ $SCALE_PATTERN ]]; then
            current_scale="${BASH_REMATCH[1]}"
            current_phase="Starting..."
            progress=""
            timing=""
            echo -e "\r${CYAN}$current_scale${NC}   | ${YELLOW}$current_phase${NC}           |                    |                 "
        fi
        
        # Extract Hippocampus phase
        if [[ $line =~ $HIPPO_PATTERN ]]; then
            current_phase="Hippocampus"
            progress=""
            timing=""
            echo -e "\r${CYAN}$current_scale${NC}   | ${GREEN}$current_phase${NC}           |                    |                 "
        fi
        
        # Extract FAISS phase
        if [[ $line =~ $FAISS_PATTERN ]]; then
            current_phase="FAISS"
            progress=""
            timing=""
            echo -e "\r${CYAN}$current_scale${NC}   | ${BLUE}$current_phase${NC}                 |                    |                 "
        fi
        
        # Extract insert phase
        if [[ $line =~ $INSERT_PATTERN ]]; then
            current_phase+=" Insert"
            target="${BASH_REMATCH[1]}"
            progress="0/$target"
            echo -e "\r${CYAN}$current_scale${NC}   | ${YELLOW}$current_phase${NC}       | ${progress}        |                 "
        fi
        
        # Extract search phase
        if [[ $line =~ $SEARCH_PATTERN ]]; then
            current_phase="${current_phase/Insert/Search}"
            target="${BASH_REMATCH[1]}"
            progress="0/$target"
            echo -e "\r${CYAN}$current_scale${NC}   | ${YELLOW}$current_phase${NC}      | ${progress}          |                 "
        fi
        
        # Update progress
        if [[ $line =~ $PROGRESS_PATTERN ]]; then
            current="${BASH_REMATCH[1]}"
            total="${BASH_REMATCH[2]}"
            progress="$current/$total"
            percent=$((current * 100 / total))
            echo -e "\r${CYAN}$current_scale${NC}   | ${YELLOW}$current_phase${NC}      | ${progress} (${percent}%) |                 "
        fi
        
        # Extract timing details
        if [[ $line =~ $TIMING_PATTERN ]]; then
            embed="${BASH_REMATCH[1]}"
            load="${BASH_REMATCH[2]}"
            search="${BASH_REMATCH[3]}"
            timing="E:${embed}ms S:${search}ms"
            echo -e "\r${CYAN}$current_scale${NC}   | ${GREEN}$current_phase${NC}      | ${progress}          | ${timing}     "
        fi
        
        # Extract FAISS pure search timing
        if [[ $line =~ $FAISS_PURE_PATTERN ]]; then
            search="${BASH_REMATCH[1]}"
            timing="Search: ${search}ms"
            echo -e "\r${CYAN}$current_scale${NC}   | ${BLUE}$current_phase${NC}               | ${progress}          | ${timing}     "
        fi
        
        # Check for completion
        if [[ $line == *"✓ Benchmark complete!"* ]]; then
            echo -e "\n${GREEN}✓✓✓ Benchmark completed successfully! ✓✓✓${NC}"
            echo -e "\nRun './deploy.sh' to download the results."
            return 0
        fi
    done
}

# Setup monitoring by creating a log file and starting a background process
echo -e "${BLUE}Setting up remote logging...${NC}"

# Start logging on the remote machine
ssh -i ~/.ssh/${KEY_NAME}.pem -o StrictHostKeyChecking=no \
    ec2-user@$INSTANCE_IP "cd /home/ec2-user/hippocampus-benchmark && ps -ef | grep -v grep | grep benchmark_scaling.sh | awk '{print \$2}' > /tmp/benchmark.pid && tail -f -n +1 \$(ps -p \$(cat /tmp/benchmark.pid) -o args= | grep -o '> .*' | cut -c 3-) > /tmp/benchmark.log 2>&1 &" || {
    
    # Alternative approach if the first one fails
    echo -e "${YELLOW}Using alternative monitoring approach...${NC}"
    ssh -i ~/.ssh/${KEY_NAME}.pem -o StrictHostKeyChecking=no \
        ec2-user@$INSTANCE_IP "cd /home/ec2-user/hippocampus-benchmark && dmesg -w | grep benchmark_scaling > /tmp/benchmark.log 2>&1 &"
}

echo -e "${GREEN}✓ Remote logging set up${NC}"
echo

# Start the monitor
monitor_benchmark

# Clean up
echo -e "\n${BLUE}Cleaning up...${NC}"
ssh -i ~/.ssh/${KEY_NAME}.pem -o StrictHostKeyChecking=no \
    ec2-user@$INSTANCE_IP "rm -f /tmp/benchmark.log /tmp/benchmark.pid" 2>/dev/null || true

echo -e "${GREEN}✓ Monitor session ended${NC}"
