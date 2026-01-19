# Folder Path Lookup

Demonstrates how to reference Keeper secrets using folder paths instead of UIDs, making your configurations more readable and maintainable.

**Time to complete: ~10 minutes**

## What This Demonstrates

- Referencing secrets by folder path + title
- Folder-based organization of secrets
- Extracting specific fields from folder-located secrets
- Multiple folder path notations in a single pod

## Why Use Folder Paths?

**Before (UID-based):**
```yaml
keeper.security/secret: "QabbP Id M8Unw4hwVM-F8VQ"  # What is this?
```

**After (folder path):**
```yaml
keeper.security/secret: "Production/Databases/mysql-prod"  # Clear!
```

**Benefits:**
- **Readable**: Know what secret you're referencing at a glance
- **Maintainable**: Easy to update when reorganizing folders
- **Contextual**: Folder structure provides organizational meaning
- **Case-sensitive**: Precise matching prevents accidental lookups

## Prerequisites

- Kubernetes cluster (minikube, kind, EKS, GKE, AKS, or any K8s 1.21+)
- kubectl configured
- Keeper Secrets Manager account with folders set up

## Folder Structure Setup

### Step 1: Create Folder Structure in Keeper

Create the following folder hierarchy in your Keeper vault:

```
Production/
├── Databases/
│   ├── mysql-prod
│   └── postgres-prod
└── APIs/
    └── stripe-api

Development/
├── Databases/
│   └── mysql-dev
└── APIs/
    └── test-api
```

### Step 2: Add Records to Folders

Create sample records in each folder with fields like:
- `password`
- `username`
- `host`
- `api_key` (for API records)

## Complete Setup

### Step 1: Install Keeper K8s Injector

**Using Helm:**
```bash
helm upgrade --install keeper-injector oci://registry-1.docker.io/keeper/keeper-injector \
  --namespace keeper-security \
  --create-namespace
```

**Using kubectl:**
```bash
kubectl apply -f https://github.com/Keeper-Security/keeper-k8s-injector/releases/latest/download/install.yaml
```

### Step 2: Create KSM Auth Secret

Get your KSM config from Keeper:
1. Log into Keeper Vault
2. Navigate to: Vault → Secrets Manager → Select your Application
3. Ensure your shared folders are accessible to the KSM application
4. Go to Devices tab → Add Device
5. Select "Configuration File" method and "Base64" type
6. Copy the base64 string

```bash
kubectl create secret generic keeper-credentials \
  --from-literal=config='<paste-base64-config-here>'
```

### Step 3: Deploy the Demo

```bash
kubectl apply -f folder-paths.yaml
```

### Step 4: Verify Injection

```bash
# Check pod logs
kubectl logs -l app=folder-demo

# Check injected secrets
kubectl exec deployment/folder-demo -- cat /keeper/secrets/prod-db-password.txt
kubectl exec deployment/folder-demo -- cat /keeper/secrets/dev-db-full.json
kubectl exec deployment/folder-demo -- cat /keeper/secrets/api-key.txt
```

## What Gets Injected

The demo deploys a pod with multiple folder path references:

| Annotation | Folder Path | Field | Output File |
|------------|-------------|-------|-------------|
| `secret-prod-db-pass` | Production/Databases/mysql-prod | password | `/keeper/secrets/prod-db-password.txt` |
| `secret-dev-db-full` | Development/Databases/mysql-dev | (entire record) | `/keeper/secrets/dev-db-full.json` |
| `secret-api-key` | Production/APIs/stripe-api | api_key | `/keeper/secrets/api-key.txt` |
| `secret-prod-username` | Production/Databases/postgres-prod | username | `/keeper/secrets/prod-username.txt` |

## Key Annotations Explained

```yaml
# Extract single field using folder path
keeper.security/secret-prod-db-pass: "Production/Databases/mysql-prod/field/password:/keeper/secrets/prod-db-password.txt"
#                                      └─────────────┬─────────────┘ └───┬───┘ └──┬───┘ └────────────┬───────────────┘
#                                              Folder Path            Record  Selector    Output Path

# Get entire record
keeper.security/secret-dev-db-full: "Development/Databases/mysql-dev:/keeper/secrets/dev-db-full.json"
#                                    └──────────┬──────────┘ └───┬──┘ └──────────┬────────────┘
#                                          Folder Path        Record   Output Path
```

## Testing Folder Paths

### Test 1: View Extracted Password

```bash
kubectl exec deployment/folder-demo -- cat /keeper/secrets/prod-db-password.txt
```

Expected: Raw password value (no JSON wrapping)

### Test 2: View Full Record

```bash
kubectl exec deployment/folder-demo -- cat /keeper/secrets/dev-db-full.json | jq .
```

Expected: Complete record with all fields as JSON

### Test 3: Change Secret in Keeper and Wait

1. Edit the `mysql-prod` password in Keeper (in Production/Databases folder)
2. Wait 1 minute (default refresh interval)
3. Check the file again:

```bash
kubectl exec deployment/folder-demo -- cat /keeper/secrets/prod-db-password.txt
```

The password should update automatically!

## Folder Path Notation Formats

### With keeper:// prefix

```yaml
keeper.security/secret: "keeper://Production/Databases/mysql-prod/field/password:/app/secrets/db-pass"
```

### Without prefix (also works)

```yaml
keeper.security/secret: "Production/Databases/mysql-prod/field/password:/app/secrets/db-pass"
```

### Supported Selectors

```yaml
# Extract field
"Production/Databases/mysql-prod/field/password:/app/secrets/pass.txt"

# Extract custom field
"Production/APIs/stripe/custom_field/api_token:/app/secrets/token.txt"

# Download file attachment
"Production/Certs/ssl-cert/file/cert.pem:/app/certs/cert.pem"

# Get record type
"Production/Databases/mysql-prod/type:/app/metadata/type.txt"

# Get record title
"Production/Databases/mysql-prod/title:/app/metadata/title.txt"

# Get entire record (no selector)
"Production/Databases/mysql-prod:/app/secrets/full-record.json"
```

## Folder vs. UID: When to Use Each

| Scenario | Use Folder Path | Use UID |
|----------|----------------|---------|
| Readable configs | ✅ Folder Path | ❌ UID is cryptic |
| Team collaboration | ✅ Folder Path | ❌ Hard to remember UIDs |
| Programmatic access | Either | ✅ UID is guaranteed unique |
| Unique record names | ✅ Folder Path | Either |
| Duplicate record names | Either (folder disambiguates) | ✅ UID is always unique |

## Cleanup

```bash
kubectl delete -f folder-paths.yaml
kubectl delete secret keeper-credentials
```

## Troubleshooting

### Error: "folder not found at path 'Production'"

**Problem**: Folder path is case-sensitive and must match exactly.

**Solution**: Check folder names in Keeper vault - spaces, capitalization, and special characters must match exactly.

### Error: "no record found with name 'mysql-prod' in folder path"

**Problem**: Record doesn't exist in the specified folder or name doesn't match.

**Solution**:
- Verify the record exists in that exact folder
- Check record title matches exactly (case-sensitive)
- Ensure the KSM application has access to the shared folder

### Error: "failed to resolve folder path"

**Problem**: Folder hierarchy doesn't exist or KSM application lacks access.

**Solution**:
- Verify folder structure exists in Keeper
- Ensure shared folders are accessible to your KSM application
- Check that parent folders exist (e.g., `Production` must exist for `Production/Databases`)

## Next Steps

- Try [Example 12: Environment Variables](../12-env-vars/) to inject folder path secrets as env vars
- See [docs/annotations.md](../../docs/annotations.md#folder-path-notation) for complete notation reference
- Learn about [folder-based organization](../../docs/features.md#folder-support) in the features guide

## Related Documentation

- [Annotations Reference](../../docs/annotations.md)
- [Features Guide](../../docs/features.md)
- [Troubleshooting](../../docs/troubleshooting.md)
