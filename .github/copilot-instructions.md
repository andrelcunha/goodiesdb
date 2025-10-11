# GoodiesDB AI Coding Guide

## Project Overview
GoodiesDB is a Redis-compatible in-memory data store implemented in Go. It features a TCP server with Redis protocol compatibility, dual persistence mechanisms (RDB snapshots + AOF logging), and multi-database support.

## Architecture Components

### Core Structure
- **`cmd/goodiesdb-server/main.go`**: Entry point with graceful shutdown handling and signal management
- **`internal/core/server/`**: TCP server with Redis-compatible command parsing and connection management
- **`internal/core/store/`**: Thread-safe in-memory storage with multi-database support (16 databases, indices 0-15)
- **`internal/persistence/`**: Dual persistence via RDB snapshots (`rdb/`) and AOF logging (`aof/`)

### Key Patterns

#### Database Architecture
```go
// Store manages 16 separate databases (Redis-compatible)
// All fields are private for proper encapsulation
type Store struct {
    data    []map[string]interface{}  // One map per database (private)
    expires []map[string]time.Time    // TTL tracking per database (private)
    mu      sync.RWMutex             // Thread safety (private)
    aofChan chan string              // AOF command logging (private)
}
```

#### Store Encapsulation
- **All fields are private** - access only through methods
- **No exposed Lock/Unlock methods** - synchronization handled internally
- **Persistence interface**: Use `GetSnapshot()` and `RestoreFromSnapshot()` for RDB operations
- **Test helpers**: Use `GetListLength()`, `GetList()`, and `SetRawValue()` for testing only

#### Command Flow
1. TCP connection → `handleConnection()` → `handleCommand()`
2. Authentication check (except AUTH command)
3. Database index resolution per connection
4. Command execution with AOF logging
5. Response sent to client

#### Persistence Strategy
- **AOF**: Every write command logged to channel → async file writer
- **RDB**: Periodic snapshots using Go's `gob` encoding
- **Recovery**: AOF replay on startup if persistence enabled

## Development Workflows

### Build & Run
```bash
make build    # Builds to bin/goodiesdb-server with version from git
make run      # Build + run server on localhost:6379
```

### Testing
```bash
go test ./...                    # Run all tests
go test ./pkg/store -v          # Verbose store tests
go test ./pkg/persistence/aof   # AOF-specific tests
```

### Docker Development
```bash
docker-compose up               # Run with persistent volume
# Server available on localhost:6379 with password from compose file
```

## Critical Conventions

### Configuration
- Environment-based config via `.env` files (see `pkg/server/config.go`)
- Default password: "guest", port: 6379
- `USE_RDB=true` and `USE_AOF=true` enable dual persistence

### Connection Management
```go
// Per-connection state tracking
authenticatedConnections map[net.Conn]bool  // Auth status
connectionDbs            map[net.Conn]int   // Current DB index (0-15)
```

### Thread Safety
- All store operations use `RLock()`/`Lock()` patterns
- AOF channel prevents blocking on writes
- Graceful shutdown closes AOF channel to flush writes

### Error Patterns
- Redis-compatible error responses: `"ERR <message>"`
- Authentication errors: `"NOAUTH Authentication required"`
- Type validation for numeric operations (INCR/DECR)

## Data Types & Commands

### Supported Types
- **string**: Basic key-value pairs
- **list**: `[]string` with head/tail operations
- **TTL**: Per-key expiration with automatic cleanup

### Command Implementation
Commands are implemented in `pkg/server/server.go` with direct store method calls. Each write command triggers AOF logging with format: `"COMMAND dbIndex key [args...]"`

## Testing Approach
- Unit tests for each store operation (`pkg/store/store_test.go`)
- Test patterns use mock AOF channels: `make(chan string, 100)`
- List operations tested for Redis-compatible ordering and edge cases
- TTL tests use real time delays for expiration validation

## Integration Points
- **TCP Protocol**: Raw socket communication (not HTTP)
- **Client Testing**: Use telnet/netcat on port 6379 or Redis CLI
- **Persistence Files**: `data/dump.rdb` and `data/appendonly.aof`
- **Docker**: Multi-stage build with Alpine base for minimal image

## Common Gotchas
- Database indices are connection-specific, not global
- AOF replay during startup requires exact command format matching
- List operations follow Redis semantics (LPUSH prepends in reverse order)
- Type checking required for numeric operations (string values only)