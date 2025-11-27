# set shell := ["bash", "-c"]

# Start minikube
minikube-start:
    minikube start

# Build docker image inside minikube
docker-build:
    eval $(minikube docker-env) && docker build -t hfcardoso/golang-server:latest -f deploy/local/Dockerfile .

# Apply kubernetes manifests
k8s-apply:
    kubectl apply -f deploy/local/go-namespace.yaml
    kubectl apply -f deploy/local/postgres-volume.yaml
    kubectl apply -f deploy/local/go-configmap.yaml
    kubectl apply -f deploy/local/postgres-deployment.yaml
    kubectl apply -f deploy/local/postgres-service.yaml
    kubectl apply -f deploy/local/go-deployment.yaml
    kubectl apply -f deploy/local/go-service.yaml


# Apply PSQL kubernetes manifests 
k8s-apply-psql:
    kubectl apply -f deploy/local/go-namespace.yaml
    kubectl apply -f deploy/local/postgres-volume.yaml
    kubectl apply -f deploy/local/postgres-deployment.yaml
    kubectl apply -f deploy/local/postgres-service.yaml

# Delete kubernetes manifests
k8s-delete:
    kubectl delete -f deploy/local/go-service.yaml
    kubectl delete -f deploy/local/go-deployment.yaml
    kubectl delete -f deploy/local/postgres-service.yaml
    kubectl delete -f deploy/local/postgres-deployment.yaml
    kubectl delete -f deploy/local/go-configmap.yaml
    kubectl delete -f deploy/local/postgres-volume.yaml
    kubectl delete -f deploy/local/go-namespace.yaml

# Open minikube dashboard
minikube-dashboard:
    minikube dashboard

# Run all (start, build, apply)
up: minikube-start docker-build k8s-apply

# Run tests with coverage
test-coverage:
    go test -coverprofile=coverage.out ./...
    go tool cover -func=coverage.out

# Run all tests
test:
    go test ./...


# Update minikube deployment
update-minikube: docker-build
    kubectl rollout restart deployment golang-server -n development

# Run application locally (connects to PostgreSQL in minikube)
run-local:
    @echo "Starting application locally..."
    @echo "Make sure PostgreSQL is running in minikube (just k8s-apply)"
    @echo "Connecting to PostgreSQL at $(minikube ip):30007"
    export PSQL_HOST=$(minikube ip) && \
    export PSQL_DB_NAME="horsemktdb" && \
    export PSQL_USERNAME="horsemktuser" && \
    export PSQL_PORT="30007" && \
    export PSQL_PASSWORD="@EUZ29tmw-yr2jnZY8M@" && \
    export PSQL_SSLMODE="disable" && \
    export PASETO_KEY="0d2734c1bd19f2f273201165ca321914" && \
    export ENVIRONMENT="development" && \
    go run cmd/main.go

# Port forward PostgreSQL for local development (alternative to NodePort)
postgres-port-forward:
    @echo "Port forwarding PostgreSQL from minikube to localhost:5432"
    kubectl port-forward -n development svc/postgres-service 5432:5432

# Run database migrations up (apply all pending migrations)
migrate-up:
    @echo "Running migrations against PostgreSQL in minikube..."
    migrate -path migrations -database "postgresql://horsemktuser:@EUZ29tmw-yr2jnZY8M@@$(minikube ip):30007/horsemktdb?sslmode=disable" up

# Rollback last migration
migrate-down:
    @echo "Rolling back last migration..."
    migrate -path migrations -database "postgresql://horsemktuser:@EUZ29tmw-yr2jnZY8M@@$(minikube ip):30007/horsemktdb?sslmode=disable" down 1

# Rollback all migrations
migrate-down-all:
    @echo "Rolling back all migrations..."
    migrate -path migrations -database "postgresql://horsemktuser:@EUZ29tmw-yr2jnZY8M@@$(minikube ip):30007/horsemktdb?sslmode=disable" down -all

# Force migration version (use with caution)
migrate-force VERSION:
    @echo "Forcing migration version to {{VERSION}}..."
    migrate -path migrations -database "postgresql://horsemktuser:@EUZ29tmw-yr2jnZY8M@@$(minikube ip):30007/horsemktdb?sslmode=disable" force {{VERSION}}

# Check migration status
migrate-version:
    @echo "Checking current migration version..."
    migrate -path migrations -database "postgresql://horsemktuser:@EUZ29tmw-yr2jnZY8M@@$(minikube ip):30007/horsemktdb?sslmode=disable" version

# Create a new migration file
migrate-create NAME:
    @echo "Creating new migration: {{NAME}}"
    migrate create -ext sql -dir migrations -seq {{NAME}}
