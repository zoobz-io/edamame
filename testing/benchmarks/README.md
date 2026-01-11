# Benchmarks

Performance benchmarks for edamame core operations.

## Running Benchmarks

```bash
# Run all benchmarks
make test-bench

# Run with more iterations
go test -v -bench=. -benchmem -benchtime=5s ./testing/benchmarks/...

# Run specific benchmark
go test -v -bench=BenchmarkQueryRender -benchmem ./testing/benchmarks/...

# Generate CPU profile
go test -bench=. -cpuprofile=cpu.prof ./testing/benchmarks/...

# Generate memory profile
go test -bench=. -memprofile=mem.prof ./testing/benchmarks/...
```

## Benchmark Categories

### Statement Creation

Measures cost of creating statements:

- `BenchmarkNewQueryStatement`
- `BenchmarkNewSelectStatement`
- `BenchmarkNewUpdateStatement`

### Query Rendering

Measures SQL generation performance:

- `BenchmarkQueryRender`
- `BenchmarkSelectRender`
- `BenchmarkComplexQueryRender`

### Spec Conversion

Measures spec-to-builder conversion:

- `BenchmarkConditionConversion`
- `BenchmarkOrderByConversion`

## Interpreting Results

```
BenchmarkQueryRender-8    500000    2345 ns/op    1024 B/op    12 allocs/op
```

- `500000`: Number of iterations
- `2345 ns/op`: Nanoseconds per operation
- `1024 B/op`: Bytes allocated per operation
- `12 allocs/op`: Allocations per operation

## Performance Goals

| Operation | Target |
|-----------|--------|
| Statement creation | < 1μs |
| Simple query render | < 5μs |
| Complex query render | < 20μs |
| Allocations per query | < 20 |

## Tracking Regressions

Benchmark results are uploaded as CI artifacts. Compare across commits:

```bash
# Save baseline
go test -bench=. -benchmem ./testing/benchmarks/... > baseline.txt

# Run after changes
go test -bench=. -benchmem ./testing/benchmarks/... > current.txt

# Compare (requires benchstat)
benchstat baseline.txt current.txt
```

## Writing Benchmarks

```go
func BenchmarkOperation(b *testing.B) {
    // Setup outside the loop
    stmt := edamame.NewQueryStatement("test", "Test", edamame.QuerySpec{})
    exec, _ := edamame.New[Model](db, "table", renderer)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        // Operation to benchmark
        _, _ = exec.RenderQuery(stmt)
    }
}
```

### Sub-benchmarks

```go
func BenchmarkQueries(b *testing.B) {
    cases := []struct {
        name string
        spec edamame.QuerySpec
    }{
        {"simple", edamame.QuerySpec{}},
        {"filtered", edamame.QuerySpec{Where: conditions}},
    }

    for _, tc := range cases {
        b.Run(tc.name, func(b *testing.B) {
            // ...
        })
    }
}
```
