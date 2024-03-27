terraform {
  backend "s3" {
    endpoint                    = "sfo3.digitaloceanspaces.com"
    key                         = "pi-sensor/terraform.tfstate"
    bucket                      = "pi-sensor"
    region                      = "us-west-1"
    skip_credentials_validation = true
    skip_metadata_api_check     = true
  }
}
