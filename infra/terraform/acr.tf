locals {
  # Use provided name or generate a random one (acr names must be globally unique)
  acr_name = var.acr_name != null ? var.acr_name : "acreventingestor${random_pet.suffix.id}"
}

resource "azurerm_container_registry" "acr" {
  name                = replace(local.acr_name, "-", "") # Remove hyphens as ACR names must be alphanumeric
  resource_group_name = azurerm_resource_group.rg.name
  location            = azurerm_resource_group.rg.location
  sku                 = "Basic"
  admin_enabled       = true # Useful for local debugging, though Identity is preferred for AKS

  tags = {
    Environment = "Production"
  }
}
