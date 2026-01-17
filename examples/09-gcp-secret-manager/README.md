# GCP Secret Manager Authentication

Use GCP Workload Identity to fetch Keeper configuration from GCP Secret Manager instead of Kubernetes Secrets.

**Time to complete: ~20 minutes**

## Prerequisites

1. GKE cluster with Workload Identity enabled
2. Keeper K8s Injector installed
3. gcloud CLI configured
4. Keeper Secrets Manager application with base64 config

## Quick Start

### Step 1: Store KSM Config in GCP Secret Manager

```bash
# Get your KSM config (base64 string from Keeper)

# Set project
PROJECT_ID="my-project"

# Create secret
echo -n '<your-base64-ksm-config>' | gcloud secrets create ksm-config \
  --data-file=- \
  --project=$PROJECT_ID \
  --replication-policy=automatic

# Verify
gcloud secrets describe ksm-config --project=$PROJECT_ID
```

### Step 2: Create GCP Service Account

```bash
# Create service account
gcloud iam service-accounts create keeper-secrets-access \
  --project=$PROJECT_ID \
  --description="Keeper K8s Injector" \
  --display-name="Keeper Secrets Access"

# Grant Secret Manager access
gcloud secrets add-iam-policy-binding ksm-config \
  --project=$PROJECT_ID \
  --member="serviceAccount:keeper-secrets-access@${PROJECT_ID}.iam.gserviceaccount.com" \
  --role="roles/secretmanager.secretAccessor"
```

### Step 3: Bind to Kubernetes ServiceAccount

```bash
# Get GKE cluster info
CLUSTER_NAME="my-cluster"
CLUSTER_REGION="us-central1"

# Create IAM binding
gcloud iam service-accounts add-iam-policy-binding \
  keeper-secrets-access@${PROJECT_ID}.iam.gserviceaccount.com \
  --project=$PROJECT_ID \
  --role roles/iam.workloadIdentityUser \
  --member "serviceAccount:${PROJECT_ID}.svc.id.goog[default/myapp-sa]"
```

### Step 4: Deploy Kubernetes Resources

```bash
kubectl apply -f serviceaccount.yaml
kubectl apply -f deployment.yaml
```

### Step 5: Verify

```bash
# Check logs
kubectl logs -l app=gcp-secrets-example -c keeper-secrets-sidecar | grep "GCP"

# Expected:
# {"level":"info","msg":"fetching KSM config from GCP Secret Manager"}
# {"level":"info","msg":"successfully fetched KSM config from GCP Secret Manager"}

# Verify secrets injected
kubectl exec deployment/gcp-secrets-example -- ls -la /keeper/secrets/
```

## Troubleshooting

### Workload Identity not enabled

```bash
# Check if WI is enabled on cluster
gcloud container clusters describe $CLUSTER_NAME \
  --region=$CLUSTER_REGION \
  --format="value(workloadIdentityConfig.workloadPool)"

# Should output: PROJECT_ID.svc.id.goog
# If empty, enable WI on cluster
```

### Permission denied

```bash
# Verify IAM binding
gcloud iam service-accounts get-iam-policy \
  keeper-secrets-access@${PROJECT_ID}.iam.gserviceaccount.com

# Verify secret access
gcloud secrets get-iam-policy ksm-config --project=$PROJECT_ID
```

### ServiceAccount annotation missing

```bash
kubectl get sa myapp-sa -o yaml | grep iam.gke.io
```

## Cleanup

```bash
kubectl delete -f deployment.yaml
kubectl delete -f serviceaccount.yaml

# Delete GCP resources
gcloud iam service-accounts delete keeper-secrets-access@${PROJECT_ID}.iam.gserviceaccount.com --project=$PROJECT_ID
gcloud secrets delete ksm-config --project=$PROJECT_ID
```

## See Also

- [AWS Secrets Manager Example](../08-aws-secrets-manager/)
- [Azure Key Vault Example](../10-azure-key-vault/)
- [Cloud Secrets Guide](../../docs/cloud-secrets.md)
