# set shell := ["bash", "-c"]

# Start minikube
minikube-start:
    minikube start

# Build docker image inside minikube
docker-build:
    eval $(minikube docker-env) && docker build -t horsemarketplacebk:latest -f deploy/local/Dockerfile .

# Apply kubernetes manifests
k8s-apply:
    kubectl apply -f deploy/local/go-namespace.yaml
    kubectl apply -f deploy/local/postgres-volume.yaml
    kubectl apply -f deploy/local/go-configmap.yaml
    kubectl apply -f deploy/local/postgres-deployment.yaml
    kubectl apply -f deploy/local/postgres-service.yaml
    kubectl apply -f deploy/local/go-deployment.yaml
    kubectl apply -f deploy/local/go-service.yaml

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

