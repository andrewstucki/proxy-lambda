variable "domain" {
  description = "Deployment domain"
}

variable "subdomain" {
  description = "Deployment subdomain"
  default     = ""
}

variable "app_name" {
  description = "Application name"
}

variable "source_path" {
  description = "Source path"
}

variable "zip_output" {
  description = "Zip output"
}

# Attach role to Managed Policy
variable "iam_policy_arn" {
  description = "IAM Policy to be attached to role"
  type        = list(string)

  default = [
    "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
  ]
}

resource "random_id" "unique_suffix" {
  byte_length = 2
}

locals {
  app_id = "${lower(var.app_name)}-${random_id.unique_suffix.hex}"
  domain = var.subdomain != "" ? "${var.subdomain}.${var.domain}" : var.domain
}