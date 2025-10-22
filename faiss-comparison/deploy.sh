#!/bin/bash
set -e

echo "=================================================="
echo "Deploy and Run Scaling Benchmark on EC2"
echo "=================================================="
echo

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
RED='\033[0;31m'
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

echo -e "${BLUE}Step 1: Building Hippocampus CLI locally${NC}"
cd ..
make build-cli
echo -e "${GREEN}✓ CLI built${NC}"
cd faiss-comparison
echo

echo -e "${BLUE}Step 2: Creating deployment package${NC}"
mkdir -p /tmp/hippo-deploy
cp ../bin/hippocampus /tmp/hippo-deploy/
cp benchmark_scaling.sh /tmp/hippo-deploy/
chmod +x /tmp/hippo-deploy/benchmark_scaling.sh
echo -e "${GREEN}✓ Package created${NC}"
echo

echo -e "${BLUE}Step 3: Uploading to EC2${NC}"
scp -i ~/.ssh/${KEY_NAME}.pem -o StrictHostKeyChecking=no \
    /tmp/hippo-deploy/* \
    ec2-user@$INSTANCE_IP:/home/ec2-user/hippocampus-benchmark/
echo -e "${GREEN}✓ Files uploaded${NC}"
echo

echo -e "${BLUE}Step 4: Running benchmarks on EC2${NC}"
echo -e "${CYAN}  This will take approximately 30-40 minutes...${NC}"
echo

# Run the scaling benchmark
BENCHMARK_CMD="cd /home/ec2-user/hippocampus-benchmark && ./benchmark_scaling.sh"

ssh -i ~/.ssh/${KEY_NAME}.pem -o StrictHostKeyChecking=no \
    ec2-user@$INSTANCE_IP \
    "$BENCHMARK_CMD" | tee /tmp/benchmark-output.txt

echo
echo -e "${BLUE}Step 5: Saving results${NC}"
mkdir -p results
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
cp /tmp/benchmark-output.txt results/benchmark_${TIMESTAMP}.txt
echo -e "${GREEN}✓ Results saved to: results/benchmark_${TIMESTAMP}.txt${NC}"
echo

# Download database files if they exist
echo -e "${BLUE}Step 6: Downloading artifacts${NC}"
scp -i ~/.ssh/${KEY_NAME}.pem -o StrictHostKeyChecking=no \
    ec2-user@$INSTANCE_IP:/home/ec2-user/hippocampus-benchmark/*.bin \
    results/ 2>/dev/null || echo -e "${YELLOW}  No .bin files to download${NC}"
echo -e "${GREEN}✓ Artifacts downloaded${NC}"
echo

echo "=================================================="
echo -e "${GREEN}✓✓✓ Benchmark Complete! ✓✓✓${NC}"
echo "=================================================="
echo
echo -e "${CYAN}Results:${NC}"
echo "  Output: results/benchmark_${TIMESTAMP}.txt"
echo
echo -e "${CYAN}View results:${NC}"
echo "  cat results/benchmark_${TIMESTAMP}.txt"
echo
echo -e "${YELLOW}Don't forget to terminate the instance when done!${NC}"
echo "  aws ec2 terminate-instances --region $REGION --instance-ids $INSTANCE_ID"
