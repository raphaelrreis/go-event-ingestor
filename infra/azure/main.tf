resource "random_pet" "suffix" {
  length = 1
}

resource "azurerm_resource_group" "rg" {
  name     = var.resource_group_name
  location = var.location
  tags = {
    Environment = "Production"
    Project     = "GoEventIngestor"
    ManagedBy   = "Terraform"
  }
}
