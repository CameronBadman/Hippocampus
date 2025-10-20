terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.aws_region
}

data "archive_file" "lambda_zip" {
  type        = "zip"
  source_file = "${path.module}/bootstrap"
  output_path = "${path.module}/lambda.zip"
}

resource "aws_vpc" "main" {
  cidr_block           = "10.0.0.0/16"
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = {
    Name = "hippocampus-vpc"
  }
}

resource "aws_subnet" "public_a" {
  vpc_id                  = aws_vpc.main.id
  cidr_block              = "10.0.1.0/24"
  availability_zone       = "${var.aws_region}a"
  map_public_ip_on_launch = true

  tags = {
    Name = "hippocampus-public-a"
  }
}

resource "aws_subnet" "public_b" {
  vpc_id                  = aws_vpc.main.id
  cidr_block              = "10.0.2.0/24"
  availability_zone       = "${var.aws_region}b"
  map_public_ip_on_launch = true

  tags = {
    Name = "hippocampus-public-b"
  }
}

resource "aws_subnet" "private_a" {
  vpc_id            = aws_vpc.main.id
  cidr_block        = "10.0.3.0/24"
  availability_zone = "${var.aws_region}a"

  tags = {
    Name = "hippocampus-private-a"
  }
}

resource "aws_subnet" "private_b" {
  vpc_id            = aws_vpc.main.id
  cidr_block        = "10.0.4.0/24"
  availability_zone = "${var.aws_region}b"

  tags = {
    Name = "hippocampus-private-b"
  }
}

resource "aws_internet_gateway" "main" {
  vpc_id = aws_vpc.main.id

  tags = {
    Name = "hippocampus-igw"
  }
}

resource "aws_eip" "nat" {
  domain = "vpc"

  tags = {
    Name = "hippocampus-nat-eip"
  }

  depends_on = [aws_internet_gateway.main]
}

resource "aws_nat_gateway" "main" {
  allocation_id = aws_eip.nat.id
  subnet_id     = aws_subnet.public_a.id

  tags = {
    Name = "hippocampus-nat"
  }

  depends_on = [aws_internet_gateway.main]
}

resource "aws_route_table" "public" {
  vpc_id = aws_vpc.main.id

  tags = {
    Name = "hippocampus-public-rt"
  }
}

resource "aws_route" "public_internet" {
  route_table_id         = aws_route_table.public.id
  destination_cidr_block = "0.0.0.0/0"
  gateway_id             = aws_internet_gateway.main.id
}

resource "aws_route_table_association" "public_a" {
  subnet_id      = aws_subnet.public_a.id
  route_table_id = aws_route_table.public.id
}

resource "aws_route_table_association" "public_b" {
  subnet_id      = aws_subnet.public_b.id
  route_table_id = aws_route_table.public.id
}

resource "aws_route_table" "private" {
  vpc_id = aws_vpc.main.id

  tags = {
    Name = "hippocampus-private-rt"
  }
}

resource "aws_route" "private_nat" {
  route_table_id         = aws_route_table.private.id
  destination_cidr_block = "0.0.0.0/0"
  nat_gateway_id         = aws_nat_gateway.main.id
}

resource "aws_route_table_association" "private_a" {
  subnet_id      = aws_subnet.private_a.id
  route_table_id = aws_route_table.private.id
}

resource "aws_route_table_association" "private_b" {
  subnet_id      = aws_subnet.private_b.id
  route_table_id = aws_route_table.private.id
}

resource "aws_security_group" "lambda_sg" {
  name        = "hippocampus-lambda-sg"
  description = "Security group for Lambda function"
  vpc_id      = aws_vpc.main.id

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "hippocampus-lambda-sg"
  }
}

resource "aws_security_group" "efs_sg" {
  name        = "hippocampus-efs-sg"
  description = "Security group for EFS"
  vpc_id      = aws_vpc.main.id

  ingress {
    from_port       = 2049
    to_port         = 2049
    protocol        = "tcp"
    security_groups = [aws_security_group.lambda_sg.id]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "hippocampus-efs-sg"
  }
}

resource "aws_security_group" "memcached_sg" {
  name        = "hippocampus-memcached-sg"
  description = "Security group for Memcached cluster"
  vpc_id      = aws_vpc.main.id

  ingress {
    from_port       = 11211
    to_port         = 11211
    protocol        = "tcp"
    security_groups = [aws_security_group.lambda_sg.id]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "hippocampus-memcached-sg"
  }
}

resource "aws_efs_file_system" "agents" {
  creation_token = "hippocampus-agents"
  
  lifecycle_policy {
    transition_to_ia = "AFTER_30_DAYS"
  }

  tags = {
    Name = "hippocampus-agents"
  }
}

resource "aws_efs_mount_target" "agents_a" {
  file_system_id  = aws_efs_file_system.agents.id
  subnet_id       = aws_subnet.private_a.id
  security_groups = [aws_security_group.efs_sg.id]
}

resource "aws_efs_mount_target" "agents_b" {
  file_system_id  = aws_efs_file_system.agents.id
  subnet_id       = aws_subnet.private_b.id
  security_groups = [aws_security_group.efs_sg.id]
}

resource "aws_efs_access_point" "agents" {
  file_system_id = aws_efs_file_system.agents.id

  posix_user {
    gid = 1000
    uid = 1000
  }

  root_directory {
    path = "/agents"
    creation_info {
      owner_gid   = 1000
      owner_uid   = 1000
      permissions = "755"
    }
  }

  tags = {
    Name = "hippocampus-agents-access-point"
  }
}

resource "aws_elasticache_subnet_group" "memcached" {
  name       = "hippocampus-memcached-subnet"
  subnet_ids = [aws_subnet.private_a.id, aws_subnet.private_b.id]
}

resource "aws_elasticache_cluster" "memcached" {
  cluster_id           = "hippocampus-cache"
  engine               = "memcached"
  node_type            = var.memcached_node_type
  num_cache_nodes      = 1
  parameter_group_name = "default.memcached1.6"
  port                 = 11211
  subnet_group_name    = aws_elasticache_subnet_group.memcached.name
  security_group_ids   = [aws_security_group.memcached_sg.id]

  tags = {
    Name = "hippocampus-memcached"
  }
}

resource "aws_iam_role" "lambda_role" {
  name = "hippocampus_lambda_role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "lambda.amazonaws.com"
      }
    }]
  })
}

resource "aws_iam_role_policy_attachment" "lambda_basic" {
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
  role       = aws_iam_role.lambda_role.name
}

resource "aws_iam_role_policy_attachment" "lambda_vpc" {
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaVPCAccessExecutionRole"
  role       = aws_iam_role.lambda_role.name
}

resource "aws_iam_role_policy" "lambda_s3_policy" {
  name = "hippocampus_lambda_s3_policy"
  role = aws_iam_role.lambda_role.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "s3:GetObject",
          "s3:PutObject",
          "s3:DeleteObject"
        ]
        Resource = "${aws_s3_bucket.hippocampus_data.arn}/*"
      },
      {
        Effect = "Allow"
        Action = [
          "s3:ListBucket"
        ]
        Resource = aws_s3_bucket.hippocampus_data.arn
      }
    ]
  })
}

resource "aws_iam_role_policy" "lambda_bedrock_policy" {
  name = "hippocampus_lambda_bedrock_policy"
  role = aws_iam_role.lambda_role.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "bedrock:InvokeModel",
          "bedrock:InvokeModelWithResponseStream"
        ]
        Resource = "*"
      }
    ]
  })
}

resource "aws_iam_role_policy" "lambda_efs_policy" {
  name = "hippocampus_lambda_efs_policy"
  role = aws_iam_role.lambda_role.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "elasticfilesystem:ClientMount",
          "elasticfilesystem:ClientWrite",
          "elasticfilesystem:DescribeMountTargets"
        ]
        Resource = aws_efs_file_system.agents.arn
      }
    ]
  })
}

resource "aws_s3_bucket" "hippocampus_data" {
  bucket = var.s3_bucket_name
}

resource "aws_s3_bucket_versioning" "hippocampus_data" {
  bucket = aws_s3_bucket.hippocampus_data.id

  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_lambda_function" "hippocampus" {
  filename         = data.archive_file.lambda_zip.output_path
  function_name    = "hippocampus"
  role            = aws_iam_role.lambda_role.arn
  handler         = "bootstrap"
  source_code_hash = data.archive_file.lambda_zip.output_base64sha256
  runtime         = "provided.al2023"
  timeout         = 60
  memory_size     = 1024

  vpc_config {
    subnet_ids         = [aws_subnet.private_a.id, aws_subnet.private_b.id]
    security_group_ids = [aws_security_group.lambda_sg.id]
  }

  file_system_config {
    arn              = aws_efs_access_point.agents.arn
    local_mount_path = "/mnt/efs"
  }

  ephemeral_storage {
    size = 2048
  }

  environment {
    variables = {
      S3_BUCKET          = aws_s3_bucket.hippocampus_data.bucket
      EFS_PATH           = "/mnt/efs/agents"
      MEMCACHED_ENDPOINT = "${aws_elasticache_cluster.memcached.cache_nodes[0].address}:${aws_elasticache_cluster.memcached.cache_nodes[0].port}"
    }
  }

  depends_on = [
    aws_efs_mount_target.agents_a,
    aws_efs_mount_target.agents_b,
    aws_nat_gateway.main
  ]
}

resource "aws_apigatewayv2_api" "hippocampus_api" {
  name          = "hippocampus-api"
  protocol_type = "HTTP"
}

resource "aws_apigatewayv2_stage" "default" {
  api_id      = aws_apigatewayv2_api.hippocampus_api.id
  name        = "$default"
  auto_deploy = true
}

resource "aws_apigatewayv2_integration" "lambda_integration" {
  api_id             = aws_apigatewayv2_api.hippocampus_api.id
  integration_type   = "AWS_PROXY"
  integration_uri    = aws_lambda_function.hippocampus.invoke_arn
  integration_method = "POST"
}

resource "aws_apigatewayv2_route" "insert" {
  api_id    = aws_apigatewayv2_api.hippocampus_api.id
  route_key = "POST /insert"
  target    = "integrations/${aws_apigatewayv2_integration.lambda_integration.id}"
}

resource "aws_apigatewayv2_route" "search" {
  api_id    = aws_apigatewayv2_api.hippocampus_api.id
  route_key = "POST /search"
  target    = "integrations/${aws_apigatewayv2_integration.lambda_integration.id}"
}

resource "aws_apigatewayv2_route" "insert_csv" {
  api_id    = aws_apigatewayv2_api.hippocampus_api.id
  route_key = "POST /insert-csv"
  target    = "integrations/${aws_apigatewayv2_integration.lambda_integration.id}"
}

resource "aws_apigatewayv2_route" "agent_curate" {
  api_id    = aws_apigatewayv2_api.hippocampus_api.id
  route_key = "POST /agent-curate"
  target    = "integrations/${aws_apigatewayv2_integration.lambda_integration.id}"
}

resource "aws_lambda_permission" "api_gateway" {
  statement_id  = "AllowAPIGatewayInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.hippocampus.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.hippocampus_api.execution_arn}/*/*"
}

output "api_endpoint" {
  value       = aws_apigatewayv2_api.hippocampus_api.api_endpoint
  description = "HTTP API endpoint for testing"
}

output "s3_bucket" {
  value       = aws_s3_bucket.hippocampus_data.bucket
  description = "S3 bucket for cold storage"
}

output "efs_id" {
  value       = aws_efs_file_system.agents.id
  description = "EFS file system ID"
}

output "memcached_endpoint" {
  value       = "${aws_elasticache_cluster.memcached.cache_nodes[0].address}:${aws_elasticache_cluster.memcached.cache_nodes[0].port}"
  description = "Memcached endpoint"
}
