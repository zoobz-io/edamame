# Integration Tests

End-to-end tests that verify edamame against real databases.

## Requirements

- Docker (for testcontainers)
- Go 1.24+

## Running

```bash
# Run all integration tests
make test-integration

# Run with verbose output
go test -v -tags=integration ./testing/integration/...

# Run specific test
go test -v -tags=integration -run TestPostgresQuery ./testing/integration/...
```

## Database Support

Currently tested databases:

- **PostgreSQL** via `postgres_test.go`

## Test Structure

Integration tests use the `integration` build tag:

```go
//go:build integration

package integration

import (
    "testing"
    "github.com/testcontainers/testcontainers-go"
)
```

## Writing Integration Tests

### Container Setup

Use testcontainers for database provisioning:

```go
func setupPostgres(t *testing.T) (*sqlx.DB, func()) {
    ctx := context.Background()

    container, err := postgres.Run(ctx,
        "postgres:16-alpine",
        postgres.WithDatabase("testdb"),
        postgres.WithUsername("test"),
        postgres.WithPassword("test"),
    )
    require.NoError(t, err)

    connStr, err := container.ConnectionString(ctx, "sslmode=disable")
    require.NoError(t, err)

    db, err := sqlx.Connect("postgres", connStr)
    require.NoError(t, err)

    return db, func() {
        db.Close()
        container.Terminate(ctx)
    }
}
```

### Test Pattern

```go
func TestFeature(t *testing.T) {
    db, cleanup := setupPostgres(t)
    defer cleanup()

    // Create schema
    _, err := db.Exec(`CREATE TABLE users (id SERIAL PRIMARY KEY, name TEXT)`)
    require.NoError(t, err)

    // Create executor and test
    exec, err := edamame.New[User](db, "users", postgres.New())
    require.NoError(t, err)

    // Execute and verify
    // ...
}
```

## CI Integration

Integration tests run in CI with PostgreSQL service:

```yaml
services:
  postgres:
    image: postgres:16-alpine
    env:
      POSTGRES_USER: test
      POSTGRES_PASSWORD: test
      POSTGRES_DB: testdb
```

## Troubleshooting

### Docker not available

Ensure Docker daemon is running:

```bash
docker info
```

### Port conflicts

Testcontainers uses random ports. If you see connection issues, check for orphaned containers:

```bash
docker ps -a | grep testcontainers
```
