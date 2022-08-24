# Proxy Lambda

A lightweight Gateway API + Lambda-based rewriting proxy and corresponding terraform module.

## Configuring

Example `config.json`:

```json
{
  "skipTLSVerification": false,
  "host": "http://localhost:3000",
  "rules": [{
    "pathMatch": ".*",
    "target": "https://www.somehosthere.org",
    "rewriteRequestBody": true,
    "keepRequestHeaders": ["User-Agent", "Content-Type", "Cookie"],
    "dropResponseHeaders": ["Strict-Transport-Security"]
  }]
}
```

## Configuring the Terraform module

Example `terraform.tf`:

```hcl
module "lambda" {
  source = "./module"

  domain      = "somehosthere.org"
  subdomain   = "proxy-test"
  app_name    = "proxy-lambda"
  source_path = "build/bin/app"
  zip_output  = "build/bin/app.zip"
}
```

## Building and Deploying

```bash
make
terraform init
terraform plan -out planfile
terraform apply planfile
curl https://proxy-test.somehosthere.org
terraform destroy
```