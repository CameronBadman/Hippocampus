variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "ap-southeast-2"
}

variable "s3_bucket_name" {
  description = "S3 bucket name for storing agent memories"
  type        = string
}

variable "memcached_node_type" {
  description = "ElastiCache Memcached node type"
  type        = string
  default     = "cache.t3.micro"
}
