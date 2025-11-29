# Local Development Workflow

This guide explains how to run the application locally without rebuilding Docker images on every code change.

## Prerequisites

- Minikube running with PostgreSQL deployed
- Go installed locally

## Quick Start

### Option 1: Using Justfile (Recommended)

```bash
# Make sure PostgreSQL is running in minikube
just k8s-apply-psql

# Run the application locally
just run-local
```

Now you can:
1. Make changes to your Go code
2. Stop the application (Ctrl+C)
3. Run `just run-local` again - much faster than rebuilding Docker images!

### Option 2: Using Environment File

```bash
# Source the environment variables
source .env.local

# Run the application
go run cmd/main.go
```

### Option 3: Port Forwarding (Alternative)

If you prefer to use `localhost:5432` instead of the minikube IP:

```bash
# In one terminal, start port forwarding
just postgres-port-forward

# In another terminal, update PSQL_HOST and run
export PSQL_HOST="localhost"
export PSQL_PORT="5432"
# ... (other env vars from .env.local)
go run cmd/main.go
```

## How It Works

- PostgreSQL is exposed from minikube via **NodePort on port 30007**
- The minikube IP is `192.168.49.2` (verify with `minikube ip`)
- The application connects to PostgreSQL using environment variables
- No Docker rebuild needed - just restart the Go application!

## Development Workflow Comparison

### Old Workflow (Slow)
```bash
# Make code changes
just docker-build        # Rebuild Docker image (slow!)
kubectl rollout restart  # Restart deployment
```

### New Workflow (Fast)
```bash
# Make code changes
just run-local          # Instant restart!
```

## Database Migrations

The project uses [golang-migrate](https://github.com/golang-migrate/migrate) for database migrations. All migration files are located in the `migrations/` directory.

### Running Migrations

Apply all pending migrations to the PostgreSQL database in minikube:

```bash
just migrate-up
```

### Checking Migration Status

Check the current migration version:

```bash
just migrate-version
```

### Rolling Back Migrations

Rollback the last migration:

```bash
just migrate-down
```

Rollback all migrations (use with caution):

```bash
just migrate-down-all
```

### Creating New Migrations

Create a new migration file:

```bash
just migrate-create add_users_table
```

This will create two files:
- `migrations/000XXX_add_users_table.up.sql` - Migration to apply
- `migrations/000XXX_add_users_table.down.sql` - Migration to rollback

### Force Migration Version

If you need to force the migration version (use with caution):

```bash
just migrate-force 5
```

### Migration Workflow

1. Create a new migration: `just migrate-create my_feature`
2. Edit the `.up.sql` and `.down.sql` files
3. Apply the migration: `just migrate-up`
4. Test your changes
5. If needed, rollback: `just migrate-down`

## Environment Variables

The following environment variables are configured in `.env.local`:

- `PSQL_HOST`: Minikube IP (192.168.49.2)
- `PSQL_PORT`: NodePort (30007)
- `PSQL_DB_NAME`: horsemktdb
- `PSQL_USERNAME`: horsemktuser
- `PSQL_PASSWORD`: @EUZ29tmw-yr2jnZY8M@
- `PSQL_SSLMODE`: disable
- `PASETO_KEY`: 0d2734c1bd19f2f273201165ca321914
- `ENVIRONMENT`: development

### Local SMTP (MailHog)

For local email testing we include a MailHog manifest in `deploy/local/mailhog.yaml` and SMTP settings in the `go-configmap.yaml` so the application can send mail via `mailhog:1025`.

To deploy MailHog to the `development` namespace:

```bash
# apply namespace and postgres manifests if not already applied
kubectl apply -f deploy/local/go-namespace.yaml
kubectl apply -f deploy/local/mailhog.yaml
kubectl apply -f deploy/local/go-configmap.yaml
kubectl rollout restart deployment/your-go-deployment -n development
```

Then open the MailHog UI at:

```
kubectl port-forward svc/mailhog 8025:8025 -n development
# then visit http://localhost:8025
```

The configmap sets:
- `SMTP_HOST=mailhog`
- `SMTP_PORT=1025`
- `MAIL_FROM=no-reply@example.local`

The application reads these via the configuration service and will prefer SMTP when configured. If you prefer not to use MailHog, unset these variables to fall back to the mock sender or configure Mailgun.

## Troubleshooting

### Cannot connect to PostgreSQL

1. Verify minikube is running:
   ```bash
   minikube status
   ```

2. Verify PostgreSQL is deployed:
   ```bash
   kubectl get pods -n development
   ```

3. Verify the minikube IP:
   ```bash
   minikube ip
   ```
   Update `.env.local` if the IP is different.

### Port already in use

If port 8080 is already in use, you can change it in `cmd/main.go` (line 71).

## Notes

- The `.env.local` file is gitignored to prevent committing credentials
- When deploying to production, use proper secrets management
- The NodePort (30007) is only accessible from your local machine
