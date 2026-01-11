# Testing

Test utilities and infrastructure for edamame.

## Structure

```
testing/
├── helpers.go           # Test utilities for edamame users
├── helpers_test.go      # Tests for helpers
├── benchmarks/          # Performance benchmarks
│   └── core_performance_test.go
└── integration/         # Database integration tests
    └── postgres_test.go
```

## Test Utilities

The `testing` package provides utilities for testing edamame-based applications:

### QueryCapture

Captures rendered SQL queries for verification:

```go
import edamametesting "github.com/zoobzio/edamame/testing"

capture := edamametesting.NewQueryCapture()

// Capture queries during test execution
capture.CaptureQuery("by-email", "select", "SELECT * FROM users WHERE email = $1", params)

// Verify captured queries
queries := capture.Queries()
last := capture.Last()
count := capture.Count()
byType := capture.ByType("select")
```

### ExecutorEventCapture

Captures executor creation events:

```go
capture := edamametesting.NewExecutorEventCapture()
handler := capture.Handler()

// Register with capitan
capitan.On(edamame.ExecutorCreated, handler)

// Verify events
tables := capture.Tables()
count := capture.Count()
```

### ParamBuilder

Fluent builder for test parameters:

```go
params := edamametesting.NewParamBuilder().
    Set("email", "test@example.com").
    Set("status", "active").
    Build()
```

## Running Tests

```bash
# Unit tests
make test

# Integration tests (requires Docker)
make test-integration

# Benchmarks
make test-bench

# All tests
make test-all
```

## Writing Tests

### Unit Tests

Each source file has a corresponding `_test.go` file. Tests should:

- Use table-driven tests where appropriate
- Test edge cases and error conditions
- Avoid external dependencies

### Integration Tests

Located in `testing/integration/`. These tests:

- Require a real database (PostgreSQL via testcontainers)
- Test actual SQL execution
- Verify end-to-end behavior

See [integration/README.md](integration/README.md) for details.

### Benchmarks

Located in `testing/benchmarks/`. These tests:

- Measure performance of core operations
- Track allocations
- Enable performance regression detection

See [benchmarks/README.md](benchmarks/README.md) for details.
