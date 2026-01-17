# AWS Secrets Manager Authentication

Use AWS IRSA (IAM Roles for Service Accounts) to fetch Keeper configuration from AWS Secrets Manager instead of storing it in Kubernetes Secrets.

**Time to complete: ~20 minutes**

## What This Demonstrates

- No static credentials in Kubernetes cluster
- KSM config stored in AWS Secrets Manager
- IAM controls access to Keeper configuration
- CloudTrail audit logging of config access
- Industry-standard cloud-native authentication

## Prerequisites

1. EKS cluster with OIDC provider (enabled by default)
2. Keeper K8s Injector installed
3. AWS CLI configured
4. kubectl configured for your EKS cluster
5. Keeper Secrets Manager application with base64 config

## Quick Start

### Step 1: Store KSM Config in AWS Secrets Manager

```bash
# Get your KSM config from Keeper
# Vault → Secrets Manager → Select Application → Devices → Add Device → Base64
# Copy the base64 string

# Store in AWS Secrets Manager
aws secretsmanager create-secret \
  --name prod/keeper/ksm-config \
  --description "Keeper Secrets Manager configuration for production" \
  --secret-string '<paste-your-base64-ksm-config-here>' \
  --region us-west-2
```

### Step 2: Get EKS Cluster OIDC Provider

```bash
# Get cluster name and region
CLUSTER_NAME="my-cluster"
REGION="us-west-2"

# Get OIDC provider ARN
OIDC_PROVIDER=$(aws eks describe-cluster \
  --name $CLUSTER_NAME \
  --region $REGION \
  --query "cluster.identity.oidc.issuer" \
  --output text | sed -e "s/^https:\/\///")

echo "OIDC Provider: $OIDC_PROVIDER"

# Get AWS account ID
ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
```

### Step 3: Create IAM Role

Save this as `trust-policy.json`:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "arn:aws:iam::ACCOUNT_ID:oidc-provider/OIDC_PROVIDER"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "OIDC_PROVIDER:sub": "system:serviceaccount:default:myapp-sa",
          "OIDC_PROVIDER:aud": "sts.amazonaws.com"
        }
      }
    }
  ]
}
```

Replace `ACCOUNT_ID` and `OIDC_PROVIDER` with values from Step 2, then:

```bash
# Create IAM role
aws iam create-role \
  --role-name keeper-secrets-access \
  --assume-role-policy-document file://trust-policy.json

# Attach Secrets Manager policy
aws iam put-role-policy \
  --role-name keeper-secrets-access \
  --policy-name secrets-manager-access \
  --policy-document '{
    "Version": "2012-10-17",
    "Statement": [{
      "Effect": "Allow",
      "Action": "secretsmanager:GetSecretValue",
      "Resource": "arn:aws:secretsmanager:us-west-2:ACCOUNT_ID:secret:prod/keeper/ksm-config*"
    }]
  }'
```

### Step 4: Create Kubernetes Resources

Apply the example manifests:

```bash
kubectl apply -f serviceaccount.yaml
kubectl apply -f deployment.yaml
```

### Step 5: Verify

```bash
# Wait for pod to be ready
kubectl wait --for=condition=ready pod -l app=aws-secrets-example --timeout=120s

# Check sidecar logs
kubectl logs -l app=aws-secrets-example -c keeper-secrets-sidecar | grep "AWS"

# Expected output:
# {"level":"info","msg":"fetching KSM config from AWS Secrets Manager","secretId":"prod/keeper/ksm-config","region":"us-west-2"}
# {"level":"info","msg":"successfully fetched KSM config from AWS Secrets Manager"}

# Verify secrets were injected
kubectl exec deployment/aws-secrets-example -- ls -la /keeper/secrets/
```

## How It Works

```
1. Pod starts with ServiceAccount "myapp-sa"
   ↓
2. EKS injects AWS IRSA environment variables:
   - AWS_ROLE_ARN
   - AWS_WEB_IDENTITY_TOKEN_FILE (path to JWT token)
   ↓
3. Sidecar uses AWS SDK (automatic IRSA credential provider)
   ↓
4. AWS SDK exchanges K8s JWT for temporary AWS credentials
   ↓
5. Sidecar calls AWS Secrets Manager GetSecretValue
   ↓
6. Receives KSM config from AWS Secrets Manager
   ↓
7. Uses KSM config to authenticate with Keeper
   ↓
8. Fetches actual secrets from Keeper
```

## Security Advantages

- ✅ No KSM config in Kubernetes cluster
- ✅ IAM policies control who can access config
- ✅ CloudTrail logs every config access
- ✅ Automatic credential rotation via AWS
- ✅ Per-pod IAM role granularity
- ✅ Temporary credentials (15 min - 12 hr TTL)

## Troubleshooting

### Pod can't assume IAM role

**Symptoms:**
```
Error: An error occurred (AccessDenied) when calling the AssumeRoleWithWebIdentity operation
```

**Fixes:**
1. Verify ServiceAccount has role-arn annotation:
   ```bash
   kubectl get sa myapp-sa -o yaml | grep eks.amazonaws.com/role-arn
   ```

2. Check IAM trust policy matches exactly:
   - Account ID correct
   - OIDC provider URL matches cluster
   - Namespace and SA name match

3. Verify OIDC provider exists:
   ```bash
   aws iam list-open-id-connect-providers
   ```

### Can't access AWS Secrets Manager

**Symptoms:**
```
Error: AccessDeniedException: User is not authorized to perform: secretsmanager:GetSecretValue
```

**Fixes:**
1. Check IAM role has policy attached:
   ```bash
   aws iam list-role-policies --role-name keeper-secrets-access
   aws iam get-role-policy --role-name keeper-secrets-access --policy-name secrets-manager-access
   ```

2. Verify secret ARN in policy matches actual secret:
   ```bash
   aws secretsmanager describe-secret --secret-id prod/keeper/ksm-config
   ```

3. Check CloudTrail for denied API calls:
   ```bash
   aws cloudtrail lookup-events \
     --lookup-attributes AttributeKey=EventName,AttributeValue=GetSecretValue
   ```

### Sidecar can't read JWT token

**Symptoms:**
```
Error: AWS_WEB_IDENTITY_TOKEN_FILE not set
```

**Fixes:**
- Restart pod (EKS webhook might have failed)
- Verify EKS version supports IRSA (1.13+)
- Check if mutating webhooks are working

## Cleanup

```bash
kubectl delete -f deployment.yaml
kubectl delete -f serviceaccount.yaml

# Delete IAM role
aws iam delete-role-policy --role-name keeper-secrets-access --policy-name secrets-manager-access
aws iam delete-role --role-name keeper-secrets-access

# Delete secret (optional)
aws secretsmanager delete-secret --secret-id prod/keeper/ksm-config --force-delete-without-recovery
```

## Next Steps

- [GCP Secret Manager Example](../09-gcp-secret-manager/) - GCP equivalent
- [Azure Key Vault Example](../10-azure-key-vault/) - Azure equivalent
- [Cloud Secrets Guide](../../docs/cloud-secrets.md) - Complete documentation
