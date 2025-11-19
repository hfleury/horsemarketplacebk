# Horse Marketplace Backend

This is the backend service for the Horse Marketplace application, built with Go (Golang). It provides APIs for user authentication, management, and other marketplace features.

## 🚀 Tech Stack

- **Language:** [Go](https://go.dev/) (1.21+)
- **Framework:** [Gin Web Framework](https://github.com/gin-gonic/gin)
- **Database:** [PostgreSQL](https://www.postgresql.org/)
- **Authentication:** [PASETO](https://paseto.io/) (v2)
- **Configuration:** [Viper](https://github.com/spf13/viper)
- **Logging:** [Zerolog](https://github.com/rs/zerolog)
- **Containerization:** [Docker](https://www.docker.com/)
- **Orchestration:** [Kubernetes](https://kubernetes.io/) (Minikube)
- **Task Runner:** [Just](https://github.com/casey/just)

## 📋 Prerequisites

Ensure you have the following installed on your machine:

- [Go](https://go.dev/dl/)
- [Docker](https://docs.docker.com/get-docker/)
- [Minikube](https://minikube.sigs.k8s.io/docs/start/)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [Just](https://github.com/casey/just) (Command runner)

## ⚙️ Configuration

The application uses environment variables for configuration. You can set these in your environment or via Kubernetes secrets/configmaps.

| Variable | Description | Default |
|----------|-------------|---------|
| `ENVIRONMENT` | Application environment (development/production) | `development` |
| `PSQL_HOST` | PostgreSQL Host | `localhost` |
| `PSQL_PORT` | PostgreSQL Port | `5432` |
| `PSQL_USERNAME` | PostgreSQL Username | - |
| `PSQL_PASSWORD` | PostgreSQL Password | - |
| `PSQL_DB_NAME` | PostgreSQL Database Name | - |
| `PSQL_SSLMODE` | PostgreSQL SSL Mode | `disable` |
| `PASETO_KEY` | Symmetric Key for PASETO tokens (32 bytes) | - |

## 🛠️ Getting Started

### 1. Clone the repository

```bash
git clone https://github.com/hfleury/horsemarketplacebk.git
cd horsemarketplacebk
```

### 2. Local Development

To run the application locally, ensure you have a PostgreSQL instance running and the environment variables set.

```bash
# Install dependencies
go mod download

# Run the application
go run cmd/main.go
```

### 3. Docker & Kubernetes (Minikube)

We use `Just` to manage the Kubernetes environment.

**Start Minikube and Deploy:**
```bash
just up
```
This command will:
1. Start Minikube.
2. Build the Docker image inside Minikube's environment.
3. Apply all Kubernetes manifests (Database, ConfigMaps, Services, Deployments).

**Open Dashboard:**
```bash
just minikube-dashboard
```

**Tear Down:**
```bash
just k8s-delete
```

## 🧪 Testing

We use Go's built-in testing framework.

**Run all tests:**
```bash
just test
```

**Run tests with coverage:**
```bash
just test-coverage
```

## 🔌 API Endpoints

### Authentication

- **POST** `/auth/user` - Create a new user
- **GET** `/auth/users` - Get user by username (Query param)
- **GET** `/auth/login` - Login user (Returns PASETO token)
    - *Note: Accepts JSON body with `username` and `password_hash`*

## 📂 Project Structure

```
.
├── cmd
│   └── main.go             # Application entry point
├── config                  # Configuration and Logging setup
├── deploy                  # Kubernetes manifests and Dockerfile
│   └── local
├── internal
│   ├── auth                # Auth module (Handlers, Services, Repositories, Models)
│   ├── common              # Common utilities (API Response)
│   ├── db                  # Database connection
│   ├── middleware          # Gin Middleware (Logging)
│   └── router              # Route definitions
└── Justfile                # Task runner configuration
```
