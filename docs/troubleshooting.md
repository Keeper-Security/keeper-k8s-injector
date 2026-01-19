# Troubleshooting

## Common Issues

### Pod stuck in CrashLoopBackOff or Pending

**Symptom**: Pod never becomes Ready, `kubectl wait` times out

**Check init container logs**:
```bash
kubectl logs YOUR-POD -c keeper-secrets-init --previous
```

**Common errors and solutions**:

#### Error: "403 Forbidden - access_denied"
```
Error: 403 Forbidden
Message: Unable to validate Keeper application access
```

**Cause**: Your KSM config in the `keeper-credentials` secret is invalid, expired, or revoked.

**Solution**:
1. Generate a new KSM device config in Keeper:
   - Go to Keeper Vault → Secrets Manager → Your Application
   - Devices tab → Add Device → Configuration File → Base64
   - Copy the base64 string

2. Update the Kubernetes secret:
   ```bash
   kubectl delete secret keeper-credentials
   kubectl create secret generic keeper-credentials \
     --from-literal=config='<paste-new-base64-config-here>'
   ```

3. Delete and recreate your pod:
   ```bash
   kubectl delete pod YOUR-POD
   kubectl apply -f your-pod.yaml
   ```

#### Error: "no record found with title: my-secret"

**Cause**: The secret name in your annotation doesn't exist in KSM, or your KSM application doesn't have access to it.

**Solution**:
1. Verify the secret title exists in Keeper (case-sensitive)
2. Check your KSM application has access to the secret (Keeper UI → Secrets Manager → Application → Secrets tab)
3. Use the exact title as shown in Keeper
4. Or use the record UID instead: `keeper.security/secret: "ABC123def456ghi789"`

### My secrets aren't appearing

**Check 1: Is injection enabled?**
```bash
kubectl get pod my-pod -o jsonpath='{.metadata.annotations}'
```
Verify `keeper.security/inject: "true"` is set.

**Check 2: Is the namespace excluded?**
```bash
kubectl get ns my-namespace --show-labels
```
Namespaces with `keeper.security/inject=disabled` are excluded.

**Check 3: Is the webhook running?**
```bash
kubectl get pods -n keeper-security
```

**Check 4: Check init container logs**
```bash
kubectl logs my-pod -c keeper-secrets-init
```

**Check 5: Verify auth secret exists**
```bash
kubectl get secret keeper-auth -o yaml
```

### Authentication failed

**Error**: `failed to create KSM client: authentication failed`

**Solution**:
1. Verify your KSM config.json is valid
2. Check the secret contains the correct key:
```bash
kubectl get secret keeper-auth -o jsonpath='{.data.config}' | base64 -d
```
3. Ensure the KSM application has access to the requested records

### Pod won't start

**Error**: `Init container keeper-secrets-init failed`

**Solution**:
1. Check init container logs:
```bash
kubectl logs my-pod -c keeper-secrets-init
```
2. Common causes:
   - Invalid auth credentials
   - Secret not found in KSM
   - Network connectivity to KSM

**To debug without blocking pod startup**:
```yaml
annotations:
  keeper.security/fail-on-error: "false"
```

### Secret not found

**Error**: `no record found with title: my-secret`

**Solution**:
1. Verify the secret title matches exactly (case-sensitive)
2. Check that your KSM application has access to the secret
3. Try using the record UID instead of title

### Multiple records match

**Error**: `multiple records (3) found with title: database (strict mode enabled)`

**Solution**:
1. Use a unique title or UID
2. Disable strict mode: `keeper.security/strict-lookup: "false"` (uses first match)
3. Use folder-scoped titles: `keeper.security/secret: "production/database"`

### Sidecar not refreshing

**Check 1: Is sidecar running?**
```bash
kubectl get pod my-pod -o jsonpath='{.spec.containers[*].name}'
```

**Check 2: Check sidecar logs**
```bash
kubectl logs my-pod -c keeper-secrets-sidecar
```

**Check 3: Verify refresh interval**
```bash
kubectl get pod my-pod -o jsonpath='{.metadata.annotations.keeper\.security/refresh-interval}'
```

### Webhook timeout

**Error**: `context deadline exceeded` or `i/o timeout`

**Solution**:
1. Increase timeout in MutatingWebhookConfiguration
2. Check webhook pod resources (may need more CPU/memory)
3. Verify network connectivity between API server and webhook

### Certificate errors

**Error**: `x509: certificate signed by unknown authority`

**Solution**:
1. Ensure cert-manager is installed and healthy:
```bash
kubectl get pods -n cert-manager
```
2. Check certificate status:
```bash
kubectl get certificate -n keeper-security
```
3. Restart webhook pods after certificate renewal:
```bash
kubectl rollout restart deployment -n keeper-security keeper-injector
```

## Debugging Commands

### View all injector resources
```bash
kubectl get all -n keeper-security
```

### Check webhook configuration
```bash
kubectl get mutatingwebhookconfiguration keeper-injector -o yaml
```

### Test webhook connectivity
```bash
kubectl run test --image=curlimages/curl --rm -it --restart=Never -- \
  curl -k https://keeper-injector-webhook.keeper-security:443/healthz
```

### View injection for a specific pod
```bash
kubectl get pod my-pod -o yaml | grep -A 50 "initContainers:"
```

### Check events
```bash
kubectl get events --field-selector reason=FailedCreate
```

## Getting Help

1. Check the [FAQ](https://docs.keeper.io/k8s/faq)
2. Search [GitHub Issues](https://github.com/keeper-security/keeper-k8s-injector/issues)
3. Contact [Keeper Support](https://www.keepersecurity.com/support.html)

When reporting issues, include:
- Kubernetes version: `kubectl version`
- Injector version: `kubectl get deployment -n keeper-security keeper-injector -o jsonpath='{.spec.template.spec.containers[0].image}'`
- Pod annotations
- Init container and sidecar logs
- Events: `kubectl get events -n <namespace>`

---

**[← Back to Documentation Index](INDEX.md)**
