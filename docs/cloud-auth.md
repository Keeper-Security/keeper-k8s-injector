# Cloud Secrets Provider Authentication

Store Keeper Secrets Manager configuration in cloud secrets stores (AWS Secrets Manager, GCP Secret Manager, Azure Key Vault) instead of Kubernetes Secrets.

## Why Use Cloud Secrets?

### Security Benefits

| Aspect | K8s Secret | Cloud Secrets Store |
|--------|------------|---------------------|
| **Access control** | K8s RBAC | Cloud IAM |
| **Audit logging** | K8s audit logs | CloudTrail/Cloud Logging |
| **Rotation** | Manual | Automatic via cloud |
| **Blast radius** | Namespace-level | IAM role-specific |
| **Compliance** | K8s policies | Cloud compliance (SOC2, HIPAA, etc.) |

### When to Use

**Use cloud secrets when:**
- Running in EKS, GKE, or AKS
- Want centralized secret management
- Need cloud-native audit trails
- Have existing cloud IAM infrastructure

**Use K8s Secrets when:**
- Running on-prem or multi-cloud
- Want simplicity
- Don't have cloud provider dependencies

---

## AWS Secrets Manager (EKS)

### Prerequisites

1. **EKS cluster** with OIDC provider (enabled by default)
2. **AWS Secrets Manager** secret containing KSM config
3. **IAM role** with Secrets Manager access
4. **ServiceAccount** with IAM role annotation

### Step 1: Store KSM Config in AWS Secrets Manager

Get your KSM config from Keeper:
- Go to Keeper: Vault → Secrets Manager → Application → Devices → Add Device → Base64
- Copy the base64 string

```bash
aws secretsmanager create-secret \
  --name prod/keeper/ksm-config \
  --description "Keeper Secrets Manager configuration" \
  --secret-string '<your-base64-ksm-config>' \
  --region us-west-2
```

**Example output:**
```json
{
  "ARN": "arn:aws:secretsmanager:us-west-2:123456789012:secret:prod/keeper/ksm-config-AbCdEf",
  "Name": "prod/keeper/ksm-config"
}
```

### Step 2: Get EKS Cluster OIDC Provider

**What is this?** Your EKS cluster name is what you named it when you created it (e.g., `my-prod-cluster`). You need the OIDC provider ID to configure IAM trust policies.

```bash
# Get your cluster name (if you don't know it)
aws eks list-clusters --region us-west-2

# Get the OIDC provider URL for your cluster
aws eks describe-cluster --name YOUR-CLUSTER-NAME --region us-west-2 \
  --query "cluster.identity.oidc.issuer" --output text
```

**Example output:**
```
https://oidc.eks.us-west-2.amazonaws.com/id/A1B2C3D4E5F6G7H8I9J0K1L2M3N4O5P6
                                           ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
                                           This is your CLUSTER_ID
```

**Save these values** - you'll need them in the next step:
- `CLUSTER_NAME`: Your EKS cluster name (e.g., `my-prod-cluster`)
- `CLUSTER_ID`: The hex string after `/id/` (e.g., `A1B2C3D4E5F6G7H8I9J0K1L2M3N4O5P6`)
- `REGION`: Your AWS region (e.g., `us-west-2`)
- `ACCOUNT_ID`: Your AWS account ID (12 digits, get with `aws sts get-caller-identity --query Account --output text`)

### Step 3: Create IAM Role

Create IAM role with trust policy for IRSA (replace placeholders with values from Step 2):

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "arn:aws:iam::ACCOUNT_ID:oidc-provider/oidc.eks.REGION.amazonaws.com/id/CLUSTER_ID"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "oidc.eks.REGION.amazonaws.com/id/CLUSTER_ID:sub": "system:serviceaccount:NAMESPACE:SERVICE_ACCOUNT_NAME",
          "oidc.eks.REGION.amazonaws.com/id/CLUSTER_ID:aud": "sts.amazonaws.com"
        }
      }
    }
  ]
}
```

**Real example with actual values:**
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "arn:aws:iam::123456789012:oidc-provider/oidc.eks.us-west-2.amazonaws.com/id/A1B2C3D4E5F6G7H8I9J0K1L2M3N4O5P6"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "oidc.eks.us-west-2.amazonaws.com/id/A1B2C3D4E5F6G7H8I9J0K1L2M3N4O5P6:sub": "system:serviceaccount:production:myapp-sa",
          "oidc.eks.us-west-2.amazonaws.com/id/A1B2C3D4E5F6G7H8I9J0K1L2M3N4O5P6:aud": "sts.amazonaws.com"
        }
      }
    }
  ]
}
```

Attach policy for Secrets Manager access:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "secretsmanager:GetSecretValue"
      ],
      "Resource": "arn:aws:secretsmanager:REGION:ACCOUNT:secret:prod/keeper/ksm-config*"
    }
  ]
}
```

### Step 4: Create Kubernetes ServiceAccount

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: myapp-sa
  namespace: production
  annotations:
    eks.amazonaws.com/role-arn: arn:aws:iam::123456789:role/keeper-secrets-access
```

### Step 5: Use in Pod

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: my-app
  annotations:
    keeper.security/inject: "true"
    keeper.security/auth-method: "aws-secrets-manager"
    keeper.security/aws-secret-id: "prod/keeper/ksm-config"
    keeper.security/aws-region: "us-west-2"
    keeper.security/secret: "my-database-creds"
spec:
  serviceAccountName: myapp-sa  # ← Important!
  containers:
    - name: app
      image: myapp:latest
```

### Step 6: Verify

```bash
# Check sidecar logs
kubectl logs pod/my-app -c keeper-secrets-sidecar | grep "AWS Secrets Manager"

# Expected output:
# {"level":"info","msg":"fetching KSM config from AWS Secrets Manager","secretId":"prod/keeper/ksm-config"}
# {"level":"info","msg":"successfully fetched KSM config from AWS Secrets Manager"}

# Verify secrets were injected
kubectl exec pod/my-app -- cat /keeper/secrets/my-database-creds.json
```

### Troubleshooting AWS

**Error: "AWS_WEB_IDENTITY_TOKEN_FILE not set"**
- ServiceAccount missing `eks.amazonaws.com/role-arn` annotation
- Check: `kubectl get sa myapp-sa -o yaml`

**Error: "Access Denied" from Secrets Manager**
- IAM role lacks `secretsmanager:GetSecretValue` permission
- IAM trust policy namespace/SA mismatch
- Check CloudTrail for denied API calls

**Error: "secret not found"**
- Wrong secret ID or region
- Secret doesn't exist in Secrets Manager
- Verify: `aws secretsmanager describe-secret --secret-id prod/keeper/ksm-config`

---

## GCP Secret Manager (GKE)

### Prerequisites

1. **GKE cluster** with Workload Identity enabled
2. **GCP Secret Manager** secret containing KSM config
3. **GCP service account** with Secret Manager access
4. **K8s ServiceAccount** with GCP SA binding

### Step 1: Get GCP Project and Cluster Info

**What is project name?** Your GCP project ID (e.g., `my-company-prod`). Find it with:

```bash
# List your GCP projects
gcloud projects list

# Get current project
gcloud config get-value project
```

**Example output:**
```
PROJECT_ID         NAME              PROJECT_NUMBER
my-company-prod    Production        123456789012
```

### Step 2: Store KSM Config in GCP Secret Manager

Get your KSM config from Keeper, then:

```bash
echo -n '<your-base64-ksm-config>' | gcloud secrets create ksm-config \
  --data-file=- \
  --project=my-company-prod \
  --replication-policy=automatic
```

**Example output:**
```
Created secret [ksm-config].
```

### Step 3: Create GCP Service Account

```bash
# Create service account
gcloud iam service-accounts create keeper-secrets-access \
  --project=my-project \
  --description="Keeper K8s Injector access" \
  --display-name="Keeper Secrets Access"

# Grant Secret Manager access
gcloud secrets add-iam-policy-binding ksm-config \
  --project=my-project \
  --member="serviceAccount:keeper-secrets-access@my-project.iam.gserviceaccount.com" \
  --role="roles/secretmanager.secretAccessor"
```

### Step 4: Bind to Kubernetes ServiceAccount

```bash
# Allow K8s SA to impersonate GCP SA
gcloud iam service-accounts add-iam-policy-binding \
  keeper-secrets-access@my-project.iam.gserviceaccount.com \
  --role roles/iam.workloadIdentityUser \
  --member "serviceAccount:my-project.svc.id.goog[production/myapp-sa]" \
  --project=my-project
```

### Step 4: Create Kubernetes ServiceAccount

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: myapp-sa
  namespace: production
  annotations:
    iam.gke.io/gcp-service-account: keeper-secrets-access@my-project.iam.gserviceaccount.com
```

### Step 5: Use in Pod

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: my-app
  annotations:
    keeper.security/inject: "true"
    keeper.security/auth-method: "gcp-secret-manager"
    keeper.security/gcp-secret-id: "projects/my-project/secrets/ksm-config/versions/latest"
    keeper.security/secret: "my-database-creds"
spec:
  serviceAccountName: myapp-sa  # ← Important!
  containers:
    - name: app
      image: myapp:latest
```

### Troubleshooting GCP

**Error: "failed to create GCP Secret Manager client"**
- Workload Identity not enabled on cluster
- ServiceAccount missing GCP annotation
- Check: `kubectl get sa myapp-sa -o yaml`

**Error: "Permission denied"**
- GCP SA lacks Secret Manager Secret Accessor role
- Workload Identity binding missing
- Check: `gcloud secrets get-iam-policy ksm-config`

**Error: "secret not found"**
- Wrong secret ID format
- Secret doesn't exist
- Verify: `gcloud secrets describe ksm-config --project=my-project`

---

## Azure Key Vault (AKS)

### Prerequisites

1. **AKS cluster** with Workload Identity enabled
2. **Azure Key Vault** with KSM config
3. **Managed identity** with Key Vault access
4. **Federated credential** linking identity to K8s SA

### Step 1: Get AKS Cluster and Resource Info

**What are these?** Your resource group and cluster name. Find them with:

```bash
# List resource groups
az group list --output table

# List AKS clusters in a resource group
az aks list --resource-group mygroup --output table
```

**Example output:**
```
Name             Location    ResourceGroup
my-prod-cluster  eastus      mygroup
```

### Step 2: Store KSM Config in Azure Key Vault

Get your KSM config from Keeper, then:

```bash
az keyvault secret set \
  --vault-name mykeyvault \
  --name ksm-config \
  --value '<your-base64-ksm-config>'
```

**Example output:**
```json
{
  "id": "https://mykeyvault.vault.azure.net/secrets/ksm-config/abc123",
  "name": "ksm-config"
}
```

### Step 3: Create Managed Identity

```bash
# Create user-assigned managed identity
az identity create \
  --name keeper-secrets-access \
  --resource-group mygroup \
  --location eastus

# Get client ID and principal ID
IDENTITY_CLIENT_ID=$(az identity show --name keeper-secrets-access --resource-group mygroup --query clientId -o tsv)
IDENTITY_PRINCIPAL_ID=$(az identity show --name keeper-secrets-access --resource-group mygroup --query principalId -o tsv)

# Grant Key Vault access
az keyvault set-policy \
  --name mykeyvault \
  --object-id $IDENTITY_PRINCIPAL_ID \
  --secret-permissions get
```

### Step 4: Get AKS OIDC Issuer

**Get the OIDC issuer URL** (needed for federated credential):

```bash
# Replace 'my-prod-cluster' with your actual cluster name from Step 1
OIDC_ISSUER=$(az aks show --name my-prod-cluster --resource-group mygroup --query oidcIssuerProfile.issuerUrl -o tsv)
echo $OIDC_ISSUER
```

**Example output:**
```
https://eastus.oic.prod-aks.azure.com/12345678-1234-1234-1234-123456789012/abcdef01-2345-6789-abcd-ef0123456789/
```

### Step 5: Create Federated Credential

```bash
# Create federated credential
az identity federated-credential create \
  --name keeper-k8s-fed \
  --identity-name keeper-secrets-access \
  --resource-group mygroup \
  --issuer "$OIDC_ISSUER" \
  --subject system:serviceaccount:production:myapp-sa \
  --audience api://AzureADTokenExchange
```

### Step 6: Create Kubernetes ServiceAccount

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: myapp-sa
  namespace: production
  annotations:
    azure.workload.identity/client-id: "12345678-1234-1234-1234-123456789012"
```

### Step 7: Use in Pod

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: my-app
  labels:
    azure.workload.identity/use: "true"  # ← Required for Azure
  annotations:
    keeper.security/inject: "true"
    keeper.security/auth-method: "azure-key-vault"
    keeper.security/azure-vault-name: "mykeyvault"
    keeper.security/azure-secret-name: "ksm-config"
    keeper.security/secret: "my-database-creds"
spec:
  serviceAccountName: myapp-sa  # ← Important!
  containers:
    - name: app
      image: myapp:latest
```

### Troubleshooting Azure

**Error: "failed to create Azure credential"**
- Workload Identity not enabled
- ServiceAccount missing client-id annotation
- Pod missing `azure.workload.identity/use: "true"` label

**Error: "Access denied to Key Vault"**
- Managed identity lacks Key Vault Get Secrets permission
- Federated credential not created/misconfigured
- Check: `az keyvault show-policy --name mykeyvault`

**Error: "secret not found"**
- Wrong vault name or secret name
- Secret doesn't exist in Key Vault
- Verify: `az keyvault secret show --vault-name mykeyvault --name ksm-config`

---

## Comparison: Auth Methods

| Auth Method | Setup Complexity | Security | Cloud Dependency |
|-------------|------------------|----------|------------------|
| **K8s Secret** (default) | Low | Medium | None |
| **AWS Secrets Manager** | Medium | High | AWS EKS |
| **GCP Secret Manager** | Medium | High | GCP GKE |
| **Azure Key Vault** | Medium | High | Azure AKS |

---

## Migration Guide

### From K8s Secret to AWS Secrets Manager

**Current setup:**
```yaml
# Step 1: Get KSM config from K8s Secret
kubectl get secret keeper-credentials -o jsonpath='{.data.config}' | base64 -d > ksm-config.b64

# Step 2: Store in AWS Secrets Manager
aws secretsmanager create-secret \
  --name prod/keeper/ksm-config \
  --secret-string "$(cat ksm-config.b64)" \
  --region us-west-2

# Step 3: Create IAM role (see above)

# Step 4: Update pod annotations
# Remove: keeper.security/auth-secret
# Add: keeper.security/auth-method: "aws-secrets-manager"
# Add: keeper.security/aws-secret-id: "prod/keeper/ksm-config"

# Step 5: Delete K8s Secret (optional, after verifying)
kubectl delete secret keeper-credentials
```

---

## Best Practices

### Separate Configs Per Environment

```
AWS Secrets:
  prod/keeper/ksm-config     → KSM app with only PROD secrets
  staging/keeper/ksm-config  → KSM app with only STAGING secrets
  dev/keeper/ksm-config      → KSM app with only DEV secrets
```

### Use Least Privilege IAM

**AWS:**
```json
{
  "Effect": "Allow",
  "Action": "secretsmanager:GetSecretValue",
  "Resource": "arn:aws:secretsmanager:*:*:secret:prod/keeper/*"
}
```

**GCP:**
```bash
# Only grant access to specific secret, not all secrets
gcloud secrets add-iam-policy-binding ksm-config \
  --member="serviceAccount:app@project.iam.gserviceaccount.com" \
  --role="roles/secretmanager.secretAccessor"
```

**Azure:**
```bash
# Specific Key Vault, not all vaults
az keyvault set-policy \
  --name mykeyvault \
  --object-id $PRINCIPAL_ID \
  --secret-permissions get
```

### Enable Audit Logging

**AWS CloudTrail:**
```bash
# Log all Secrets Manager API calls
aws cloudtrail create-trail --name keeper-audit ...
```

**GCP Cloud Logging:**
- Automatically logs Secret Manager access
- View in Cloud Console → Logging

**Azure Monitor:**
- Key Vault diagnostic settings
- Log all secret access events

### Rotation

Cloud providers support automatic rotation:

**AWS:**
```bash
aws secretsmanager rotate-secret \
  --secret-id prod/keeper/ksm-config \
  --rotation-lambda-arn arn:aws:lambda:...
```

**GCP:**
- Use Secret Manager versioning
- Update to new version, old versions remain accessible

**Azure:**
- Update secret value
- Previous versions retained

---

## Verification Scripts

### AWS Verification

```bash
#!/bin/bash
# verify-aws-irsa.sh

POD_NAME=$1

echo "Checking AWS IRSA configuration..."

# 1. Check ServiceAccount annotation
SA=$(kubectl get pod $POD_NAME -o jsonpath='{.spec.serviceAccountName}')
ROLE_ARN=$(kubectl get sa $SA -o jsonpath='{.metadata.annotations.eks\.amazonaws\.com/role-arn}')
echo "ServiceAccount: $SA"
echo "IAM Role: $ROLE_ARN"

# 2. Check environment variables in pod
kubectl exec $POD_NAME -c keeper-secrets-sidecar -- env | grep AWS

# 3. Check sidecar logs
kubectl logs $POD_NAME -c keeper-secrets-sidecar | grep "AWS Secrets Manager"

echo "✓ Verification complete"
```

### GCP Verification

```bash
#!/bin/bash
# verify-gcp-wi.sh

POD_NAME=$1

echo "Checking GCP Workload Identity..."

# Check ServiceAccount annotation
SA=$(kubectl get pod $POD_NAME -o jsonpath='{.spec.serviceAccountName}')
GCP_SA=$(kubectl get sa $SA -o jsonpath='{.metadata.annotations.iam\.gke\.io/gcp-service-account}')
echo "K8s ServiceAccount: $SA"
echo "GCP ServiceAccount: $GCP_SA"

# Check sidecar logs
kubectl logs $POD_NAME -c keeper-secrets-sidecar | grep "GCP Secret Manager"

echo "✓ Verification complete"
```

### Azure Verification

```bash
#!/bin/bash
# verify-azure-wi.sh

POD_NAME=$1

echo "Checking Azure Workload Identity..."

# Check ServiceAccount and pod label
SA=$(kubectl get pod $POD_NAME -o jsonpath='{.spec.serviceAccountName}')
CLIENT_ID=$(kubectl get sa $SA -o jsonpath='{.metadata.annotations.azure\.workload\.identity/client-id}')
USE_WI=$(kubectl get pod $POD_NAME -o jsonpath='{.metadata.labels.azure\.workload\.identity/use}')

echo "ServiceAccount: $SA"
echo "Azure Client ID: $CLIENT_ID"
echo "Workload Identity enabled: $USE_WI"

# Check environment variables
kubectl exec $POD_NAME -c keeper-secrets-sidecar -- env | grep AZURE

# Check sidecar logs
kubectl logs $POD_NAME -c keeper-secrets-sidecar | grep "Azure Key Vault"

echo "✓ Verification complete"
```

---

## FAQ

### Can I use both K8s Secret and cloud secrets?

Yes, on different pods. Each pod specifies its auth-method:

```yaml
# Pod A: K8s Secret
annotations:
  keeper.security/auth-method: "secret"
  keeper.security/auth-secret: "keeper-credentials"

# Pod B: AWS Secrets Manager
annotations:
  keeper.security/auth-method: "aws-secrets-manager"
  keeper.security/aws-secret-id: "prod/keeper/ksm-config"
```

### What if my cluster isn't in AWS/GCP/Azure?

Use K8s Secret auth (the default). Cloud auth only works in native cloud K8s.

### Can I use AWS Secrets Manager in GKE?

No. Each cloud provider's Workload Identity only authenticates to that cloud:
- EKS + IRSA → AWS services
- GKE + Workload Identity → GCP services
- AKS + Workload Identity → Azure services

### Does this work on-prem?

No. Requires cloud provider OIDC infrastructure. Use K8s Secret auth for on-prem.

---

## See Also

- [Configuration Guide](configuration.md) - All annotations
- [Examples](../examples/) - Working examples
- [Corporate Proxy Guide](corporate-proxy.md) - SSL inspection setup

---

**[← Back to Documentation Index](INDEX.md)**
