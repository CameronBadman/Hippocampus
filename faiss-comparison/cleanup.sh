#!/bin/bash

set -e

echo "=================================================="
echo "Cleanup EC2 Benchmark Resources"
echo "=================================================="
echo

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# Check for instance details
if [ ! -f ".ec2-instance.txt" ]; then
    echo -e "${RED}✗ No EC2 instance configuration found${NC}"
    echo "Nothing to clean up"
    exit 0
fi

source .ec2-instance.txt

echo -e "${CYAN}Resources to clean:${NC}"
echo "  Instance ID:      $INSTANCE_ID"
echo "  Region:           $REGION"
echo "  Role:             $ROLE_NAME"
echo "  Instance Profile: $INSTANCE_PROFILE_NAME"
echo

read -p "Are you sure you want to delete these resources? (yes/no): " confirm
if [ "$confirm" != "yes" ]; then
    echo -e "${YELLOW}Cleanup cancelled${NC}"
    exit 0
fi

echo
echo -e "${BLUE}Step 1: Terminating EC2 instance${NC}"
aws ec2 terminate-instances --region $REGION --instance-ids $INSTANCE_ID > /dev/null
echo -e "${GREEN}✓ Termination initiated${NC}"

echo -e "${CYAN}Waiting for instance to terminate...${NC}"
aws ec2 wait instance-terminated --region $REGION --instance-ids $INSTANCE_ID
echo -e "${GREEN}✓ Instance terminated${NC}"
echo

echo -e "${BLUE}Step 2: Removing IAM instance profile${NC}"
aws iam remove-role-from-instance-profile \
    --instance-profile-name $INSTANCE_PROFILE_NAME \
    --role-name $ROLE_NAME 2>/dev/null || true
echo -e "${GREEN}✓ Role removed from instance profile${NC}"

aws iam delete-instance-profile \
    --instance-profile-name $INSTANCE_PROFILE_NAME 2>/dev/null || true
echo -e "${GREEN}✓ Instance profile deleted${NC}"
echo

echo -e "${BLUE}Step 3: Deleting IAM role${NC}"
aws iam delete-role-policy \
    --role-name $ROLE_NAME \
    --policy-name BedrockAccess 2>/dev/null || true
echo -e "${GREEN}✓ Role policy deleted${NC}"

aws iam delete-role \
    --role-name $ROLE_NAME 2>/dev/null || true
echo -e "${GREEN}✓ IAM role deleted${NC}"
echo

echo -e "${BLUE}Step 4: Cleaning up local files${NC}"
rm -f .ec2-instance.txt
echo -e "${GREEN}✓ Configuration file removed${NC}"
echo

echo "=================================================="
echo -e "${GREEN}✓✓✓ Cleanup Complete! ✓✓✓${NC}"
echo "=================================================="
echo
echo -e "${YELLOW}Note: The following resources were NOT deleted:${NC}"
echo "  - SSH key pair: ~/.ssh/$KEY_NAME.pem"
echo "  - Security group: $SECURITY_GROUP_NAME"
echo "  - Benchmark results: results/"
echo
echo "To delete these manually:"
echo "  aws ec2 delete-key-pair --region $REGION --key-name $KEY_NAME"
echo "  rm ~/.ssh/$KEY_NAME.pem"
echo "  aws ec2 delete-security-group --region $REGION --group-name $SECURITY_GROUP_NAME"
echo "  rm -rf results/"
