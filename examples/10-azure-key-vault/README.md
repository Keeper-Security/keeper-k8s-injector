# Azure Key Vault Authentication

Use Azure Workload Identity to fetch Keeper configuration from Azure Key Vault instead of Kubernetes Secrets.

**Time to complete: ~25 minutes**

## Prerequisites

1. AKS cluster with Workload Identity enabled
2. Keeper K8s Injector installed
3. Azure CLI configured
4. Keeper Secrets Manager application with base64 config

## Quick Start

### Step 1: Store KSM Config in Azure Key Vault

```bash
# Get your KSM config (base64 string from Keeper)

# Set variables
RESOURCE_GROUP="mygroup"
VAULT_NAME="mykeyvault"

# Create Key Vault (if needed)
az keyvault create \
  --name $VAULT_NAME \
  --resource-group $RESOURCE_GROUP \
  --location eastus

# Store KSM config
az keyvault secret set \
  --vault-name $VAULT_NAME \
  --name ksm-config \
  --value '<your-base64-ksm-config>'

# Verify
az keyvault secret show --vault-name $VAULT_NAME --name ksm-config
```

### Step 2: Create Managed Identity

```bash
# Create user-assigned managed identity
az identity create \
  --name keeper-secrets-access \
  --resource-group $RESOURCE_GROUP \
  --location eastus

# Get identity details
IDENTITY_CLIENT_ID=$(az identity show \
  --name keeper-secrets-access \
  --resource-group $RESOURCE_GROUP \
  --query clientId -o tsv)

IDENTITY_PRINCIPAL_ID=$(az identity show \
  --name keeper-secrets-access \
  --resource-group $RESOURCE_GROUP \
  --query principalId -o tsv)

echo "Client ID: $IDENTITY_CLIENT_ID"
echo "Principal ID: $IDENTITY_PRINCIPAL_ID"

# Grant Key Vault access
az keyvault set-policy \
  --name $VAULT_NAME \
  --object-id $IDENTITY_PRINCIPAL_ID \
  --secret-permissions get
```

### Step 3: Create Federated Credential

```bash
# Get AKS OIDC issuer
CLUSTER_NAME="mycluster"

OIDC_ISSUER=$(az aks show \
  --name $CLUSTER_NAME \
  --resource-group $RESOURCE_GROUP \
  --query oidcIssuerProfile.issuerUrl -o tsv)

echo "OIDC Issuer: $OIDC_ISSUER"

# Create federated credential
az identity federated-credential create \
  --name keeper-k8s-federation \
  --identity-name keeper-secrets-access \
  --resource-group $RESOURCE_GROUP \
  --issuer "$OIDC_ISSUER" \
  --subject system:serviceaccount:default:myapp-sa \
  --audience api://AzureADTokenExchange
```

### Step 4: Deploy Kubernetes Resources

```bash
kubectl apply -f https://raw.githubusercontent.com/Keeper-Security/keeper-k8s-injector/main/examples/10-azure-key-vault/azure-key-vault.yaml
```

**Note:** Remember to replace `CLIENT_ID` in the ServiceAccount annotation and `mykeyvault` in the deployment spec with your values before applying.

### Step 5: Verify

```bash
# Check logs
kubectl logs -l app=azure-secrets-example -c keeper-secrets-sidecar | grep "Azure"

# Expected:
# {"level":"info","msg":"fetching KSM config from Azure Key Vault"}
# {"level":"info","msg":"successfully fetched KSM config from Azure Key Vault"}

# Verify secrets injected
kubectl exec deployment/azure-secrets-example -- ls -la /keeper/secrets/
```

## Troubleshooting

### Workload Identity not enabled

```bash
# Check if enabled
az aks show --name $CLUSTER_NAME --resource-group $RESOURCE_GROUP \
  --query oidcIssuerProfile.enabled

# Enable if needed
az aks update --name $CLUSTER_NAME --resource-group $RESOURCE_GROUP \
  --enable-oidc-issuer --enable-workload-identity
```

### Permission denied to Key Vault

```bash
# Check access policy
az keyvault show-policy --name $VAULT_NAME

# Check managed identity assignment
az role assignment list --assignee $IDENTITY_PRINCIPAL_ID
```

### Pod missing Workload Identity label

The pod must have label: `azure.workload.identity/use: "true"`

Check: `kubectl get pod -l app=azure-secrets-example -o yaml | grep azure.workload.identity`

## Cleanup

```bash
kubectl delete -f deployment.yaml
kubectl delete -f serviceaccount.yaml

# Delete Azure resources
az identity federated-credential delete \
  --name keeper-k8s-federation \
  --identity-name keeper-secrets-access \
  --resource-group $RESOURCE_GROUP

az identity delete \
  --name keeper-secrets-access \
  --resource-group $RESOURCE_GROUP

az keyvault secret delete \
  --vault-name $VAULT_NAME \
  --name ksm-config
```

## See Also

- [AWS Secrets Manager Example](../08-aws-secrets-manager/)
- [GCP Secret Manager Example](../09-gcp-secret-manager/)
- [Cloud Secrets Guide](../../docs/cloud-secrets.md)
