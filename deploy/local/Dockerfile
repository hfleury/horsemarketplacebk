# Start from the official Golang base image
FROM golang:1.24 AS builder

# Set the Current Working Directory
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source code to the container
COPY ./ ./

# Build the Go app
RUN go build -o server ./cmd

# Use a minimal base image for production
FROM debian:bookworm-slim

# Set the working directory
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/server .

# Expose the application port
EXPOSE 8080

# Command to run the executable
CMD ["./server"]
