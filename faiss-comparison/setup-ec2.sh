#!/bin/bash

set -e

echo "=================================================="
echo "Hippocampus Benchmark EC2 Setup"
echo "Region: ap-southeast-2"
echo "=================================================="
echo

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
RED='\033[0;31m'
NC='\033[0m'

# Configuration
REGION="ap-southeast-2"
INSTANCE_TYPE="t3.medium"  # 2 vCPU, 4GB RAM - good balance
KEY_NAME="hippocampus-benchmark"
SECURITY_GROUP_NAME="hippocampus-benchmark-sg"
INSTANCE_NAME="hippocampus-benchmark"

echo -e "${BLUE}Step 1: Checking AWS CLI configuration${NC}"
if ! aws sts get-caller-identity > /dev/null 2>&1; then
    echo -e "${RED}✗ AWS CLI not configured. Run 'aws configure' first${NC}"
    exit 1
fi
echo -e "${GREEN}✓ AWS CLI configured${NC}"
echo

echo -e "${BLUE}Step 2: Creating SSH key pair${NC}"
if aws ec2 describe-key-pairs --region $REGION --key-names $KEY_NAME > /dev/null 2>&1; then
    echo -e "${YELLOW}⚠ Key pair already exists: $KEY_NAME${NC}"
else
    aws ec2 create-key-pair \
        --region $REGION \
        --key-name $KEY_NAME \
        --query 'KeyMaterial' \
        --output text > ~/.ssh/${KEY_NAME}.pem
    chmod 400 ~/.ssh/${KEY_NAME}.pem
    echo -e "${GREEN}✓ Key pair created: ~/.ssh/${KEY_NAME}.pem${NC}"
fi
echo

echo -e "${BLUE}Step 3: Creating security group${NC}"
VPC_ID=$(aws ec2 describe-vpcs --region $REGION --filters "Name=isDefault,Values=true" --query 'Vpcs[0].VpcId' --output text)

if aws ec2 describe-security-groups --region $REGION --group-names $SECURITY_GROUP_NAME > /dev/null 2>&1; then
    echo -e "${YELLOW}⚠ Security group already exists${NC}"
    SG_ID=$(aws ec2 describe-security-groups --region $REGION --group-names $SECURITY_GROUP_NAME --query 'SecurityGroups[0].GroupId' --output text)
else
    SG_ID=$(aws ec2 create-security-group \
        --region $REGION \
        --group-name $SECURITY_GROUP_NAME \
        --description "Security group for Hippocampus benchmarks" \
        --vpc-id $VPC_ID \
        --query 'GroupId' \
        --output text)

    # Get current IP
    MY_IP=$(curl -s https://checkip.amazonaws.com)

    # Allow SSH from current IP only
    aws ec2 authorize-security-group-ingress \
        --region $REGION \
        --group-id $SG_ID \
        --protocol tcp \
        --port 22 \
        --cidr ${MY_IP}/32

    echo -e "${GREEN}✓ Security group created: $SG_ID${NC}"
    echo -e "${CYAN}  SSH allowed from: ${MY_IP}${NC}"
fi
echo

echo -e "${BLUE}Step 4: Finding Amazon Linux 2023 AMI${NC}"
AMI_ID=$(aws ec2 describe-images \
    --region $REGION \
    --owners amazon \
    --filters "Name=name,Values=al2023-ami-2023*x86_64" "Name=state,Values=available" \
    --query 'sort_by(Images, &CreationDate)[-1].ImageId' \
    --output text)
echo -e "${GREEN}✓ AMI found: $AMI_ID${NC}"
echo

echo -e "${BLUE}Step 5: Creating user data script${NC}"
cat > /tmp/ec2-userdata.sh << 'EOF'
#!/bin/bash
set -e

# Update system
yum update -y

# Install dependencies
yum install -y \
    git \
    golang \
    python3 \
    python3-pip \
    python3-devel \
    gcc \
    gcc-c++ \
    make \
    bc

# Create working directory
mkdir -p /home/ec2-user/hippocampus-benchmark
chown ec2-user:ec2-user /home/ec2-user/hippocampus-benchmark

# Install Python packages as ec2-user
su - ec2-user -c "
    python3 -m venv /home/ec2-user/venv
    source /home/ec2-user/venv/bin/activate
    pip install --upgrade pip
    pip install boto3 numpy faiss-cpu
"

echo "EC2 instance ready for benchmarking" > /home/ec2-user/setup-complete.txt
EOF
echo -e "${GREEN}✓ User data script created${NC}"
echo

echo -e "${BLUE}Step 6: Creating IAM role for Bedrock access${NC}"
ROLE_NAME="hippocampus-benchmark-role"
INSTANCE_PROFILE_NAME="hippocampus-benchmark-profile"

# Check if role exists
if aws iam get-role --role-name $ROLE_NAME > /dev/null 2>&1; then
    echo -e "${YELLOW}⚠ IAM role already exists${NC}"
else
    # Create trust policy
    cat > /tmp/trust-policy.json << 'TRUST'
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "ec2.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
TRUST

    # Create role
    aws iam create-role \
        --role-name $ROLE_NAME \
        --assume-role-policy-document file:///tmp/trust-policy.json \
        --description "Role for Hippocampus benchmark EC2 instance" > /dev/null

    # Create and attach Bedrock policy
    cat > /tmp/bedrock-policy.json << 'POLICY'
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "bedrock:InvokeModel",
        "bedrock:InvokeModelWithResponseStream"
      ],
      "Resource": "*"
    }
  ]
}
POLICY

    aws iam put-role-policy \
        --role-name $ROLE_NAME \
        --policy-name BedrockAccess \
        --policy-document file:///tmp/bedrock-policy.json

    echo -e "${GREEN}✓ IAM role created${NC}"
fi

# Create instance profile if it doesn't exist
if aws iam get-instance-profile --instance-profile-name $INSTANCE_PROFILE_NAME > /dev/null 2>&1; then
    echo -e "${YELLOW}⚠ Instance profile already exists${NC}"
else
    aws iam create-instance-profile --instance-profile-name $INSTANCE_PROFILE_NAME > /dev/null
    aws iam add-role-to-instance-profile \
        --instance-profile-name $INSTANCE_PROFILE_NAME \
        --role-name $ROLE_NAME
    echo -e "${GREEN}✓ Instance profile created${NC}"
    # Wait a bit for IAM propagation
    echo -e "${CYAN}  Waiting for IAM propagation...${NC}"
    sleep 10
fi
echo

echo -e "${BLUE}Step 7: Launching EC2 instance${NC}"
INSTANCE_ID=$(aws ec2 run-instances \
    --region $REGION \
    --image-id $AMI_ID \
    --instance-type $INSTANCE_TYPE \
    --key-name $KEY_NAME \
    --security-group-ids $SG_ID \
    --user-data file:///tmp/ec2-userdata.sh \
    --block-device-mappings '[{"DeviceName":"/dev/xvda","Ebs":{"VolumeSize":20,"VolumeType":"gp3"}}]' \
    --tag-specifications "ResourceType=instance,Tags=[{Key=Name,Value=$INSTANCE_NAME}]" \
    --iam-instance-profile Name=$INSTANCE_PROFILE_NAME \
    --query 'Instances[0].InstanceId' \
    --output text)

echo -e "${GREEN}✓ Instance launched: $INSTANCE_ID${NC}"
echo

echo -e "${BLUE}Step 8: Waiting for instance to be running${NC}"
aws ec2 wait instance-running --region $REGION --instance-ids $INSTANCE_ID
echo -e "${GREEN}✓ Instance is running${NC}"
echo

# Get instance IP
INSTANCE_IP=$(aws ec2 describe-instances \
    --region $REGION \
    --instance-ids $INSTANCE_ID \
    --query 'Reservations[0].Instances[0].PublicIpAddress' \
    --output text)

echo -e "${BLUE}Step 9: Waiting for initialization to complete (this may take 2-3 minutes)${NC}"
echo -e "${CYAN}  Installing system packages and Python dependencies...${NC}"

# Wait for SSH to be available
sleep 30
for i in {1..20}; do
    if ssh -i ~/.ssh/${KEY_NAME}.pem -o StrictHostKeyChecking=no -o ConnectTimeout=5 ec2-user@$INSTANCE_IP "test -f /home/ec2-user/setup-complete.txt" 2>/dev/null; then
        echo -e "${GREEN}✓ Initialization complete${NC}"
        break
    fi
    echo -n "."
    sleep 10
done
echo

echo "=================================================="
echo -e "${GREEN}✓✓✓ EC2 Instance Ready! ✓✓✓${NC}"
echo "=================================================="
echo
echo -e "${CYAN}Instance Details:${NC}"
echo "  Instance ID:   $INSTANCE_ID"
echo "  Instance Type: $INSTANCE_TYPE"
echo "  Region:        $REGION"
echo "  Public IP:     $INSTANCE_IP"
echo "  SSH Key:       ~/.ssh/${KEY_NAME}.pem"
echo
echo -e "${CYAN}Connect via SSH:${NC}"
echo "  ssh -i ~/.ssh/${KEY_NAME}.pem ec2-user@$INSTANCE_IP"
echo
echo -e "${CYAN}Next Steps:${NC}"
echo "  1. Run './deploy.sh' to deploy code and run benchmarks"
echo "  2. Or manually SSH and run benchmarks"
echo
echo -e "${YELLOW}Cost:${NC} ~\$0.05/hour for t3.medium"
echo -e "${RED}Remember to terminate when done!${NC}"
echo
echo -e "${CYAN}Terminate instance:${NC}"
echo "  aws ec2 terminate-instances --region $REGION --instance-ids $INSTANCE_ID"
echo

# Save instance details
cat > .ec2-instance.txt << DETAILS
INSTANCE_ID=$INSTANCE_ID
INSTANCE_IP=$INSTANCE_IP
REGION=$REGION
KEY_NAME=$KEY_NAME
ROLE_NAME=$ROLE_NAME
INSTANCE_PROFILE_NAME=$INSTANCE_PROFILE_NAME
DETAILS

echo -e "${GREEN}✓ Instance details saved to .ec2-instance.txt${NC}"
echo
echo -e "${YELLOW}Cleanup Resources:${NC}"
echo "  # Terminate instance"
echo "  aws ec2 terminate-instances --region $REGION --instance-ids $INSTANCE_ID"
echo
echo "  # Delete IAM resources (after instance is terminated)"
echo "  aws iam remove-role-from-instance-profile --instance-profile-name $INSTANCE_PROFILE_NAME --role-name $ROLE_NAME"
echo "  aws iam delete-instance-profile --instance-profile-name $INSTANCE_PROFILE_NAME"
echo "  aws iam delete-role-policy --role-name $ROLE_NAME --policy-name BedrockAccess"
echo "  aws iam delete-role --role-name $ROLE_NAME"
