# API Keys Integration

Demonstrates injecting multiple API keys from different SaaS providers into custom file paths.

**Time to complete: ~10 minutes**

## What This Demonstrates

- Multiple secrets from different Keeper records
- Custom file paths for each secret
- Common SaaS integration pattern
- Masked secret display for security

## Use Case

Most applications integrate with multiple external services (Stripe for payments, Slack for notifications, AWS for cloud resources, etc.). Each service needs its own API key or credentials. This example shows how to manage all of them with Keeper.

## Prerequisites

- Keeper K8s Injector installed (see [Example 01 - Hello Secrets](../01-hello-secrets/) for complete installation)
- Keeper Secrets Manager application configured

## Quick Start

### 1. Create Your KSM Auth Secret

```bash
# If you haven't already
kubectl create secret generic keeper-credentials \
  --from-file=config=path/to/your/ksm-config.json
```

### 2. Create API Key Records in Keeper

Create three records in your Keeper vault:

#### Record 1: "stripe-api-key"
- Title: `stripe-api-key`
- Password field: `sk_test_abcd1234...` (your Stripe API key)

#### Record 2: "slack-webhook"
- Title: `slack-webhook`
- Password field: `https://hooks.slack.com/services/...` (your webhook URL)

#### Record 3: "aws-credentials"
- Title: `aws-credentials`
- Login field: `AKIAIOSFODNN7EXAMPLE`
- Password field: `wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY`

### 3. Deploy the Demo

```bash
kubectl apply -f deployment.yaml
```

### 4. Wait for Ready

```bash
kubectl wait --for=condition=ready pod -l app=api-keys-demo --timeout=120s
```

### 5. View the Demo

```bash
kubectl port-forward svc/api-keys-demo 8080:80
```

Open http://localhost:8080 in your browser.

You'll see a table showing:
- Each API key with a masked value
- Status (✓ Loaded or ✗ Missing)
- The file paths where they're stored

## Custom Path Mapping

The annotations in `deployment.yaml` control where each secret is written:

```yaml
annotations:
  keeper.security/secret-stripe: "stripe-api-key:/app/secrets/stripe.json"
  keeper.security/secret-slack: "slack-webhook:/app/secrets/slack.json"
  keeper.security/secret-aws: "aws-credentials:/app/secrets/aws.json"
```

Format: `keeper.security/secret-{name}: "{keeper-record-title}:{file-path}"`

## File Contents

Each JSON file contains the full Keeper record:

```json
{
  "type": "login",
  "login": "username_if_present",
  "password": "the_api_key_or_secret",
  "url": "url_if_present",
  "fields": [...]
}
```

Your application can parse these files to extract the needed credentials.

## Real-World Usage

### Node.js Example

```javascript
const fs = require('fs');

// Read Stripe API key
const stripeSecret = JSON.parse(fs.readFileSync('/app/secrets/stripe.json'));
const stripe = require('stripe')(stripeSecret.password);

// Read Slack webhook
const slackSecret = JSON.parse(fs.readFileSync('/app/secrets/slack.json'));
const webhookUrl = slackSecret.password;
```

### Python Example

```python
import json

# Read AWS credentials
with open('/app/secrets/aws.json') as f:
    aws_creds = json.load(f)

import boto3
session = boto3.Session(
    aws_access_key_id=aws_creds['login'],
    aws_secret_access_key=aws_creds['password']
)
```

## Try Secret Rotation

1. Go to Keeper and modify one of the API keys
2. Wait ~5 minutes (the configured refresh interval)
3. Refresh the demo page - you'll see the masked value update
4. Your app can re-read the file to get the new value without restart.

## Production Considerations

### Organize by environment

Use different Keeper records for dev/staging/prod:
- `stripe-api-key-prod`
- `stripe-api-key-staging`
- `stripe-api-key-dev`

### Set appropriate permissions

Ensure your Keeper application has access only to the records it needs.

### Monitor secret changes

Log when your application detects credential changes to track rotation events.

## Cleanup

```bash
kubectl delete -f deployment.yaml
```

## Troubleshooting

### Secret shows as "Missing"

1. Check the sidecar logs:
   ```bash
   kubectl logs deployment/api-keys-demo -c keeper-sidecar
   ```

2. Verify the record title in Keeper matches the annotation exactly

3. Ensure the Keeper application has access to the record

### Can't parse JSON

Check the file is valid JSON:
```bash
kubectl exec deployment/api-keys-demo -- cat /app/secrets/stripe.json | jq .
```

## Next Steps

- [Hello Secrets](../01-hello-secrets/) - Simpler getting started example
- [TLS Nginx](../05-tls-nginx/) - TLS certificate management
