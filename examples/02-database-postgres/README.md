# PostgreSQL Database Connection

A real-world example showing how to inject database credentials from Keeper into an application that connects to PostgreSQL.

**Time to complete: ~10 minutes**

## What This Demonstrates

- Database credentials injection (username/password)
- Real PostgreSQL connection using injected secrets
- Credential rotation without application restart
- Reading credentials from JSON files

## Why This Matters

Database credentials are one of the most common secrets in any application. This example shows:

- ✅ No hardcoded passwords in manifests
- ✅ Credentials never stored in etcd
- ✅ Rotation without redeployment
- ✅ File-based secrets (more secure than env vars)

## Prerequisites

- Keeper K8s Injector installed
- `keeper-credentials` secret created

**First time?** See [Example 01 - Hello Secrets](../01-hello-secrets/#complete-setup-from-zero) for complete installation instructions (Steps 1-2).

## Quick Start

### 1. Create Database Credentials in Keeper

In your Keeper vault:
1. Create a new record titled **"postgres-credentials"**
2. Set the **login** field to: `demouser`
3. Set the **password** field to: `initial-password-change-me`
4. Save the record

### 2. Deploy PostgreSQL and the Client App

```bash
# Deploy everything
kubectl apply -f database-postgres.yaml

# Wait for PostgreSQL to be ready
kubectl wait --for=condition=ready pod -l app=postgres --timeout=120s

# Wait for the client app
kubectl wait --for=condition=ready pod -l app=db-client --timeout=120s
```

### 3. View the Demo

```bash
kubectl port-forward svc/db-client 8080:80
```

Open http://localhost:8080 in your browser.

You should see:
- ✅ **Connected** status
- The username from Keeper
- Masked password (showing character count)

## Try Credential Rotation

This is where it gets interesting:

### Step 1: Update the password in Keeper

1. Go to your **"postgres-credentials"** record in Keeper
2. Change the password to something new (e.g., `super-secret-2024`)
3. Save the record

### Step 2: Update PostgreSQL password

```bash
kubectl exec -it deploy/postgres -- psql -U demouser -d demodb -c "ALTER USER demouser PASSWORD 'super-secret-2024';"
```

### Step 3: Verify the rotation

- The client app will pick up the new password within 60 seconds
- Connection will briefly fail, then succeed with new credentials
- No pod restart required

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        Cluster                              │
│                                                             │
│  ┌─────────────────────┐      ┌─────────────────────────┐  │
│  │   PostgreSQL Pod    │      │     Client App Pod      │  │
│  │                     │      │                         │  │
│  │  ┌───────────────┐  │      │  ┌─────────────────┐    │  │
│  │  │   postgres    │  │◄─────│  │    app          │    │  │
│  │  │   database    │  │      │  │                 │    │  │
│  │  └───────────────┘  │      │  │  reads creds    │    │  │
│  │                     │      │  │  from file      │    │  │
│  │                     │      │  └────────┬────────┘    │  │
│  │                     │      │           │             │  │
│  │                     │      │           ▼             │  │
│  │                     │      │  ┌─────────────────┐    │  │
│  │                     │      │  │ keeper-sidecar  │    │  │
│  │                     │      │  │                 │    │  │
│  │                     │      │  │ fetches from    │    │  │
│  │                     │      │  │ Keeper every    │    │  │
│  │                     │      │  │ 60 seconds      │    │  │
│  │                     │      │  └─────────────────┘    │  │
│  └─────────────────────┘      └─────────────────────────┘  │
│                                                             │
└─────────────────────────────────────────────────────────────┘
                                    │
                                    │ API calls
                                    ▼
                        ┌───────────────────────┐
                        │   Keeper Secrets      │
                        │   Manager             │
                        │                       │
                        │   postgres-creds      │
                        │   - login: demouser   │
                        │   - password: ****    │
                        └───────────────────────┘
```

## Keeper Record Format

The app expects a Keeper record with these fields:

| Field | Description | Example |
|-------|-------------|---------|
| `login` or `username` | Database username | `demouser` |
| `password` | Database password | `my-secret-password` |

The injected JSON looks like:
```json
{
  "login": "demouser",
  "password": "my-secret-password",
  "url": "...",
  "fields": [...]
}
```

## Production Considerations

### Use connection pooling

In production, use a connection pooler like PgBouncer that can handle credential changes gracefully.

### Set appropriate refresh intervals

```yaml
keeper.security/refresh-interval: "5m"  # Check every 5 minutes
```

### Use database-level monitoring

Monitor failed connection attempts to detect credential rotation issues early.

## Cleanup

```bash
kubectl delete -f database-postgres.yaml
```

## Troubleshooting

### Connection refused

1. Check PostgreSQL is running:
   ```bash
   kubectl get pods -l app=postgres
   kubectl logs deploy/postgres
   ```

2. Check the service exists:
   ```bash
   kubectl get svc postgres
   ```

### Authentication failed

1. Verify the password in Keeper matches PostgreSQL:
   ```bash
   # Check what's in Keeper (via sidecar logs)
   kubectl logs deploy/db-client -c keeper-sidecar

   # Test direct connection
   kubectl exec -it deploy/postgres -- psql -U demouser -d demodb
   ```

### Secret not loading

1. Check the sidecar logs:
   ```bash
   kubectl logs deploy/db-client -c keeper-sidecar
   ```

2. Verify the record title matches the annotation

## Next Steps

- [Hello Secrets](../01-hello-secrets/) - Simpler getting started example
- [Rotation Dashboard](../06-rotation-dashboard/) - Visualize rotation in action
