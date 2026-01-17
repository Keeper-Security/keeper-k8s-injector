# MySQL Database Connection

A real-world example showing how to inject database credentials from Keeper into an application that connects to MySQL.

**Time to complete: ~10 minutes**

## What This Demonstrates

- MySQL database credentials injection (username/password)
- Real MySQL connection using injected secrets
- Credential rotation without application restart
- Reading credentials from JSON files

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

### 2. Create Database Credentials in Keeper

In your Keeper vault:
1. Create a new record titled **"mysql-credentials"**
2. Set the **login** field to: `demouser`
3. Set the **password** field to: `initial-password-change-me`
4. Save the record

### 3. Deploy MySQL and the Client App

```bash
# Deploy everything
kubectl apply -f .

# Wait for MySQL to be ready
kubectl wait --for=condition=ready pod -l app=mysql --timeout=120s

# Wait for the client app
kubectl wait --for=condition=ready pod -l app=mysql-client --timeout=120s
```

### 4. View the Demo

```bash
kubectl port-forward svc/mysql-client 8080:80
```

Open http://localhost:8080 in your browser.

You should see:
- ✅ **Connected** status
- The username from Keeper
- Masked password (showing character count)

## Try Credential Rotation

### Step 1: Update the password in Keeper

1. Go to your **"mysql-credentials"** record in Keeper
2. Change the password to something new (e.g., `super-secret-2024`)
3. Save the record

### Step 2: Update MySQL password

```bash
kubectl exec -it deploy/mysql -- mysql -u root -pinitial-password-change-me -e "ALTER USER 'demouser'@'%' IDENTIFIED BY 'super-secret-2024';"
```

### Step 3: Verify the rotation

- The client app will pick up the new password within 60 seconds
- Connection will briefly fail, then succeed with new credentials
- No pod restart required

## Keeper Record Format

The app expects a Keeper record with these fields:

| Field | Description | Example |
|-------|-------------|---------|
| `login` or `username` | Database username | `demouser` |
| `password` | Database password | `my-secret-password` |

## Production Considerations

### Use connection pooling

In production, use a connection pooler that can handle credential changes gracefully.

### Set appropriate refresh intervals

```yaml
keeper.security/refresh-interval: "5m"  # Check every 5 minutes
```

### Use database-level monitoring

Monitor failed connection attempts to detect credential rotation issues early.

## Cleanup

```bash
kubectl delete -f .
```

## Troubleshooting

### Connection refused

1. Check MySQL is running:
   ```bash
   kubectl get pods -l app=mysql
   kubectl logs deploy/mysql
   ```

2. Check the service exists:
   ```bash
   kubectl get svc mysql
   ```

### Authentication failed

1. Verify the password in Keeper matches MySQL:
   ```bash
   # Check what's in Keeper (via sidecar logs)
   kubectl logs deploy/mysql-client -c keeper-sidecar

   # Test direct connection
   kubectl exec -it deploy/mysql -- mysql -u demouser -p
   ```

### Secret not loading

1. Check the sidecar logs:
   ```bash
   kubectl logs deploy/mysql-client -c keeper-sidecar
   ```

2. Verify the record title matches the annotation

## Next Steps

- [PostgreSQL Example](../02-database-postgres/) - Alternative database
- [Rotation Dashboard](../06-rotation-dashboard/) - Visualize rotation in action
