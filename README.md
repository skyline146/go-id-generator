# Go ID Generator

This project is a Go-based ID generator service that supports both HTTP and gRPC interfaces.

## Server Configuration

The server can be configured using command-line variables to specify the ports for HTTP and gRPC servers.

### `./cmd/server/server.go`

Use the following command-line variables to specify the ports:

- `--http-port`: Specify the port for the HTTP server (default: `3000`)
- `--grpc-port`: Specify the port for the gRPC server (default: `3001`)

Example commands to run the server:

```bash
go run ./cmd/server/server.go
go run ./cmd/server/server.go --http-port 3002 --grpc-port 3003
```

## In-Memory Database

The project uses [Dragonfly](https://dragonfly.io/) as an in-memory database, which is fully compatible with the Go Redis client. A locking mechanism is implemented to prevent race conditions.

### Starting the Database

To start the Dragonfly database using Docker, run the following command:

```bash
docker compose up -d
```

## Test Client

A test client is provided to simulate and test the service with two instances running simultaneously.

### `./cmd/test-client/client.go`

To test the service, you need to run two instances of the server in separate terminals:

1. **First Terminal:**
```bash
go run ./cmd/server/server.go
```

2. **Second Terminal:**
```bash
go run ./cmd/server/server.go --http-port 3002 --grpc-port 3003
```

3. **Third Terminal (Client):**
```bash
go run ./cmd/test-client/client.go
```
