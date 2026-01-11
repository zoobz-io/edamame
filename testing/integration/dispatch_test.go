// Package integration provides integration tests for edamame using testcontainers.
//
// Run with: go test -tags=integration ./testing/integration/...
//
//go:build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/zoobzio/astql/postgres"
	"github.com/zoobzio/edamame"
)

// testDB holds the shared database connection for dispatch tests.
var testDB *sqlx.DB

// testContainer holds the shared container reference.
var testContainer testcontainers.Container

// Test statements
var (
	queryAll = edamame.NewQueryStatement("query-all", "Query all users", edamame.QuerySpec{})

	selectByID = edamame.NewSelectStatement("select-by-id", "Select user by ID", edamame.SelectSpec{
		Where: []edamame.ConditionSpec{{Field: "id", Operator: "=", Param: "id"}},
	})

	updateName = edamame.NewUpdateStatement("update-name", "Update user name", edamame.UpdateSpec{
		Set:   map[string]string{"name": "new_name"},
		Where: []edamame.ConditionSpec{{Field: "id", Operator: "=", Param: "id"}},
	})

	deleteByID = edamame.NewDeleteStatement("delete-by-id", "Delete user by ID", edamame.DeleteSpec{
		Where: []edamame.ConditionSpec{{Field: "id", Operator: "=", Param: "id"}},
	})

	countAll = edamame.NewAggregateStatement("count-all", "Count all users", edamame.AggCount, edamame.AggregateSpec{})
)

// TestMain sets up a shared postgres container for all tests.
func TestMain(m *testing.M) {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		WaitingFor:   wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(60 * time.Second),
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "testdb",
		},
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start container: %v\n", err)
		os.Exit(1)
	}
	testContainer = container

	host, err := container.Host(ctx)
	if err != nil {
		container.Terminate(ctx)
		fmt.Fprintf(os.Stderr, "failed to get container host: %v\n", err)
		os.Exit(1)
	}

	mappedPort, err := container.MappedPort(ctx, "5432")
	if err != nil {
		container.Terminate(ctx)
		fmt.Fprintf(os.Stderr, "failed to get container port: %v\n", err)
		os.Exit(1)
	}

	dsn := fmt.Sprintf("host=%s port=%s user=test password=test dbname=testdb sslmode=disable", host, mappedPort.Port())
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		container.Terminate(ctx)
		fmt.Fprintf(os.Stderr, "failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	testDB = db

	_, err = testDB.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			email TEXT NOT NULL UNIQUE,
			name TEXT,
			age INTEGER
		)
	`)
	if err != nil {
		db.Close()
		container.Terminate(ctx)
		fmt.Fprintf(os.Stderr, "failed to create table: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()

	db.Close()
	container.Terminate(ctx)

	os.Exit(code)
}

// truncateUsers clears the users table between tests.
func truncateUsers(t *testing.T) {
	t.Helper()
	_, err := testDB.ExecContext(context.Background(), `TRUNCATE TABLE users RESTART IDENTITY CASCADE`)
	if err != nil {
		t.Fatalf("failed to truncate users: %v", err)
	}
}

// insertTestUser inserts a test user and returns the ID.
func insertTestUser(t *testing.T, email, name string, age *int) int {
	t.Helper()
	var id int
	err := testDB.QueryRowContext(context.Background(),
		`INSERT INTO users (email, name, age) VALUES ($1, $2, $3) RETURNING id`,
		email, name, age,
	).Scan(&id)
	if err != nil {
		t.Fatalf("failed to insert user: %v", err)
	}
	return id
}

func TestDispatch_ExecQuery(t *testing.T) {
	truncateUsers(t)
	ctx := context.Background()

	age1, age2 := 25, 30
	insertTestUser(t, "alice@test.com", "Alice", &age1)
	insertTestUser(t, "bob@test.com", "Bob", &age2)

	factory, err := edamame.New[User](testDB, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	users, err := factory.ExecQuery(ctx, queryAll, nil)
	if err != nil {
		t.Fatalf("ExecQuery() failed: %v", err)
	}

	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
}

func TestDispatch_ExecQueryTx(t *testing.T) {
	truncateUsers(t)
	ctx := context.Background()

	age := 25
	insertTestUser(t, "alice@test.com", "Alice", &age)

	factory, err := edamame.New[User](testDB, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	tx, err := testDB.BeginTxx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTxx() failed: %v", err)
	}
	defer tx.Rollback()

	users, err := factory.ExecQueryTx(ctx, tx, queryAll, nil)
	if err != nil {
		t.Fatalf("ExecQueryTx() failed: %v", err)
	}

	if len(users) != 1 {
		t.Errorf("expected 1 user, got %d", len(users))
	}
}

func TestDispatch_ExecSelect(t *testing.T) {
	truncateUsers(t)
	ctx := context.Background()

	age := 25
	id := insertTestUser(t, "alice@test.com", "Alice", &age)

	factory, err := edamame.New[User](testDB, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	user, err := factory.ExecSelect(ctx, selectByID, map[string]any{"id": id})
	if err != nil {
		t.Fatalf("ExecSelect() failed: %v", err)
	}

	if user.Name != "Alice" {
		t.Errorf("expected name 'Alice', got %q", user.Name)
	}
}

func TestDispatch_ExecSelectTx(t *testing.T) {
	truncateUsers(t)
	ctx := context.Background()

	age := 25
	id := insertTestUser(t, "alice@test.com", "Alice", &age)

	factory, err := edamame.New[User](testDB, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	tx, err := testDB.BeginTxx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTxx() failed: %v", err)
	}
	defer tx.Rollback()

	user, err := factory.ExecSelectTx(ctx, tx, selectByID, map[string]any{"id": id})
	if err != nil {
		t.Fatalf("ExecSelectTx() failed: %v", err)
	}

	if user.Name != "Alice" {
		t.Errorf("expected name 'Alice', got %q", user.Name)
	}
}

func TestDispatch_ExecUpdate(t *testing.T) {
	truncateUsers(t)
	ctx := context.Background()

	age := 25
	id := insertTestUser(t, "alice@test.com", "Alice", &age)

	factory, err := edamame.New[User](testDB, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	updated, err := factory.ExecUpdate(ctx, updateName, map[string]any{"id": id, "new_name": "Updated"})
	if err != nil {
		t.Fatalf("ExecUpdate() failed: %v", err)
	}

	if updated.Name != "Updated" {
		t.Errorf("expected name 'Updated', got %q", updated.Name)
	}
}

func TestDispatch_ExecUpdateTx(t *testing.T) {
	truncateUsers(t)
	ctx := context.Background()

	age := 25
	id := insertTestUser(t, "alice@test.com", "Alice", &age)

	factory, err := edamame.New[User](testDB, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	tx, err := testDB.BeginTxx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTxx() failed: %v", err)
	}

	updated, err := factory.ExecUpdateTx(ctx, tx, updateName, map[string]any{"id": id, "new_name": "TxUpdated"})
	if err != nil {
		tx.Rollback()
		t.Fatalf("ExecUpdateTx() failed: %v", err)
	}

	if updated.Name != "TxUpdated" {
		tx.Rollback()
		t.Errorf("expected name 'TxUpdated', got %q", updated.Name)
	}

	tx.Rollback()

	user, _ := factory.ExecSelect(ctx, selectByID, map[string]any{"id": id})
	if user.Name != "Alice" {
		t.Errorf("expected name 'Alice' after rollback, got %q", user.Name)
	}
}

func TestDispatch_ExecDelete(t *testing.T) {
	truncateUsers(t)
	ctx := context.Background()

	age := 25
	id := insertTestUser(t, "alice@test.com", "Alice", &age)

	factory, err := edamame.New[User](testDB, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	count, err := factory.ExecDelete(ctx, deleteByID, map[string]any{"id": id})
	if err != nil {
		t.Fatalf("ExecDelete() failed: %v", err)
	}

	if count != 1 {
		t.Errorf("expected 1 deleted row, got %d", count)
	}
}

func TestDispatch_ExecDeleteTx(t *testing.T) {
	truncateUsers(t)
	ctx := context.Background()

	age := 25
	id := insertTestUser(t, "alice@test.com", "Alice", &age)

	factory, err := edamame.New[User](testDB, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	tx, err := testDB.BeginTxx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTxx() failed: %v", err)
	}

	count, err := factory.ExecDeleteTx(ctx, tx, deleteByID, map[string]any{"id": id})
	if err != nil {
		tx.Rollback()
		t.Fatalf("ExecDeleteTx() failed: %v", err)
	}

	if count != 1 {
		tx.Rollback()
		t.Errorf("expected 1 deleted row, got %d", count)
	}

	tx.Rollback()

	user, err := factory.ExecSelect(ctx, selectByID, map[string]any{"id": id})
	if err != nil {
		t.Fatalf("user should exist after rollback: %v", err)
	}
	if user.Name != "Alice" {
		t.Errorf("expected name 'Alice', got %q", user.Name)
	}
}

func TestDispatch_ExecAggregate(t *testing.T) {
	truncateUsers(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		age := 20 + i
		insertTestUser(t, fmt.Sprintf("user%d@test.com", i), fmt.Sprintf("User%d", i), &age)
	}

	factory, err := edamame.New[User](testDB, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	count, err := factory.ExecAggregate(ctx, countAll, nil)
	if err != nil {
		t.Fatalf("ExecAggregate() failed: %v", err)
	}

	if count != 5 {
		t.Errorf("expected count 5, got %f", count)
	}
}

func TestDispatch_ExecAggregateTx(t *testing.T) {
	truncateUsers(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		age := 20 + i
		insertTestUser(t, fmt.Sprintf("user%d@test.com", i), fmt.Sprintf("User%d", i), &age)
	}

	factory, err := edamame.New[User](testDB, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	tx, err := testDB.BeginTxx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTxx() failed: %v", err)
	}
	defer tx.Rollback()

	count, err := factory.ExecAggregateTx(ctx, tx, countAll, nil)
	if err != nil {
		t.Fatalf("ExecAggregateTx() failed: %v", err)
	}

	if count != 3 {
		t.Errorf("expected count 3, got %f", count)
	}
}

func TestDispatch_ExecInsert(t *testing.T) {
	truncateUsers(t)
	ctx := context.Background()

	factory, err := edamame.New[User](testDB, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	age := 28
	user := &User{
		Email: "charlie@test.com",
		Name:  "Charlie",
		Age:   &age,
	}

	inserted, err := factory.ExecInsert(ctx, user)
	if err != nil {
		t.Fatalf("ExecInsert() failed: %v", err)
	}

	if inserted.ID == 0 {
		t.Error("expected non-zero ID after insert")
	}
	if inserted.Name != "Charlie" {
		t.Errorf("expected name 'Charlie', got %q", inserted.Name)
	}
}

func TestDispatch_ExecInsertTx(t *testing.T) {
	truncateUsers(t)
	ctx := context.Background()

	factory, err := edamame.New[User](testDB, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	tx, err := testDB.BeginTxx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTxx() failed: %v", err)
	}

	age := 28
	user := &User{
		Email: "txinsert@test.com",
		Name:  "TxInsert",
		Age:   &age,
	}

	inserted, err := factory.ExecInsertTx(ctx, tx, user)
	if err != nil {
		tx.Rollback()
		t.Fatalf("ExecInsertTx() failed: %v", err)
	}

	tx.Rollback()

	_, err = factory.ExecSelect(ctx, selectByID, map[string]any{"id": inserted.ID})
	if err == nil {
		t.Error("user should not exist after rollback")
	}
}

func TestDispatch_ExecInsertBatch(t *testing.T) {
	truncateUsers(t)
	ctx := context.Background()

	factory, err := edamame.New[User](testDB, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	users := make([]*User, 5)
	for i := 0; i < 5; i++ {
		age := 20 + i
		users[i] = &User{
			Email: fmt.Sprintf("batch%d@test.com", i),
			Name:  fmt.Sprintf("Batch%d", i),
			Age:   &age,
		}
	}

	count, err := factory.ExecInsertBatch(ctx, users)
	if err != nil {
		t.Fatalf("ExecInsertBatch() failed: %v", err)
	}

	if count != 5 {
		t.Errorf("expected 5 inserted, got %d", count)
	}

	totalCount, err := factory.ExecAggregate(ctx, countAll, nil)
	if err != nil {
		t.Fatalf("ExecAggregate() failed: %v", err)
	}

	if totalCount != 5 {
		t.Errorf("expected total count 5, got %f", totalCount)
	}
}

func TestDispatch_ExecInsertBatchTx(t *testing.T) {
	truncateUsers(t)
	ctx := context.Background()

	factory, err := edamame.New[User](testDB, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	tx, err := testDB.BeginTxx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTxx() failed: %v", err)
	}

	users := make([]*User, 3)
	for i := 0; i < 3; i++ {
		age := 20 + i
		users[i] = &User{
			Email: fmt.Sprintf("txbatch%d@test.com", i),
			Name:  fmt.Sprintf("TxBatch%d", i),
			Age:   &age,
		}
	}

	count, err := factory.ExecInsertBatchTx(ctx, tx, users)
	if err != nil {
		tx.Rollback()
		t.Fatalf("ExecInsertBatchTx() failed: %v", err)
	}

	if count != 3 {
		tx.Rollback()
		t.Errorf("expected 3 inserted, got %d", count)
	}

	tx.Rollback()

	totalCount, err := factory.ExecAggregate(ctx, countAll, nil)
	if err != nil {
		t.Fatalf("ExecAggregate() failed: %v", err)
	}

	if totalCount != 0 {
		t.Errorf("expected 0 after rollback, got %f", totalCount)
	}
}

func TestDispatch_ExecCompound(t *testing.T) {
	truncateUsers(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		age := 20 + i*5
		insertTestUser(t, fmt.Sprintf("user%d@test.com", i), fmt.Sprintf("User%d", i), &age)
	}

	factory, err := edamame.New[User](testDB, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	spec := edamame.CompoundQuerySpec{
		Base: edamame.QuerySpec{
			Fields: []string{"id", "name", "email", "age"},
			Where:  []edamame.ConditionSpec{{Field: "age", Operator: "<", Param: "young_max"}},
		},
		Operands: []edamame.SetOperandSpec{
			{
				Operation: "union",
				Query: edamame.QuerySpec{
					Fields: []string{"id", "name", "email", "age"},
					Where:  []edamame.ConditionSpec{{Field: "age", Operator: ">", Param: "old_min"}},
				},
			},
		},
		OrderBy: []edamame.OrderBySpec{{Field: "age", Direction: "asc"}},
	}

	users, err := factory.ExecCompound(ctx, spec, map[string]any{"q0_young_max": 22, "q1_old_min": 38})
	if err != nil {
		t.Fatalf("ExecCompound() failed: %v", err)
	}

	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
}

func TestDispatch_ExecQueryAtom(t *testing.T) {
	truncateUsers(t)
	ctx := context.Background()

	age1, age2 := 25, 30
	insertTestUser(t, "alice@test.com", "Alice", &age1)
	insertTestUser(t, "bob@test.com", "Bob", &age2)

	factory, err := edamame.New[User](testDB, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	atoms, err := factory.ExecQueryAtom(ctx, queryAll, nil)
	if err != nil {
		t.Fatalf("ExecQueryAtom() failed: %v", err)
	}

	if len(atoms) != 2 {
		t.Errorf("expected 2 atoms, got %d", len(atoms))
	}
}

func TestDispatch_ExecSelectAtom(t *testing.T) {
	truncateUsers(t)
	ctx := context.Background()

	age := 25
	id := insertTestUser(t, "alice@test.com", "Alice", &age)

	factory, err := edamame.New[User](testDB, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	atom, err := factory.ExecSelectAtom(ctx, selectByID, map[string]any{"id": id})
	if err != nil {
		t.Fatalf("ExecSelectAtom() failed: %v", err)
	}

	if atom == nil {
		t.Fatal("expected non-nil atom")
	}
}

func TestDispatch_ExecInsertAtom(t *testing.T) {
	truncateUsers(t)
	ctx := context.Background()

	factory, err := edamame.New[User](testDB, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	age := 28
	atom, err := factory.ExecInsertAtom(ctx, map[string]any{
		"email": "charlie@test.com",
		"name":  "Charlie",
		"age":   &age,
	})
	if err != nil {
		t.Fatalf("ExecInsertAtom() failed: %v", err)
	}

	if atom == nil {
		t.Fatal("expected non-nil atom")
	}
}

func TestDispatch_ExecUpdateBatch(t *testing.T) {
	truncateUsers(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		age := 20 + i
		insertTestUser(t, fmt.Sprintf("user%d@test.com", i), fmt.Sprintf("User%d", i), &age)
	}

	factory, err := edamame.New[User](testDB, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	users, err := factory.ExecQuery(ctx, queryAll, nil)
	if err != nil {
		t.Fatalf("ExecQuery() failed: %v", err)
	}

	batchParams := make([]map[string]any, len(users))
	for i, u := range users {
		batchParams[i] = map[string]any{
			"id":       u.ID,
			"new_name": fmt.Sprintf("Updated%d", i),
		}
	}

	count, err := factory.ExecUpdateBatch(ctx, updateName, batchParams)
	if err != nil {
		t.Fatalf("ExecUpdateBatch() failed: %v", err)
	}

	if count != 3 {
		t.Errorf("expected 3 updated, got %d", count)
	}

	updated, err := factory.ExecQuery(ctx, queryAll, nil)
	if err != nil {
		t.Fatalf("ExecQuery() failed: %v", err)
	}

	for _, u := range updated {
		if u.Name[:7] != "Updated" {
			t.Errorf("expected name to start with 'Updated', got %q", u.Name)
		}
	}
}

func TestDispatch_ExecUpdateBatchTx(t *testing.T) {
	truncateUsers(t)
	ctx := context.Background()

	for i := 0; i < 2; i++ {
		age := 20 + i
		insertTestUser(t, fmt.Sprintf("user%d@test.com", i), fmt.Sprintf("User%d", i), &age)
	}

	factory, err := edamame.New[User](testDB, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	users, err := factory.ExecQuery(ctx, queryAll, nil)
	if err != nil {
		t.Fatalf("ExecQuery() failed: %v", err)
	}

	tx, err := testDB.BeginTxx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTxx() failed: %v", err)
	}

	batchParams := make([]map[string]any, len(users))
	for i, u := range users {
		batchParams[i] = map[string]any{
			"id":       u.ID,
			"new_name": fmt.Sprintf("TxUpdated%d", i),
		}
	}

	count, err := factory.ExecUpdateBatchTx(ctx, tx, updateName, batchParams)
	if err != nil {
		tx.Rollback()
		t.Fatalf("ExecUpdateBatchTx() failed: %v", err)
	}

	if count != 2 {
		tx.Rollback()
		t.Errorf("expected 2 updated, got %d", count)
	}

	tx.Rollback()

	afterRollback, err := factory.ExecQuery(ctx, queryAll, nil)
	if err != nil {
		t.Fatalf("ExecQuery() failed: %v", err)
	}

	for _, u := range afterRollback {
		if u.Name[:4] != "User" {
			t.Errorf("expected name to start with 'User' after rollback, got %q", u.Name)
		}
	}
}

func TestDispatch_ExecDeleteBatch(t *testing.T) {
	truncateUsers(t)
	ctx := context.Background()

	ids := make([]int, 3)
	for i := 0; i < 3; i++ {
		age := 20 + i
		ids[i] = insertTestUser(t, fmt.Sprintf("user%d@test.com", i), fmt.Sprintf("User%d", i), &age)
	}

	factory, err := edamame.New[User](testDB, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	batchParams := []map[string]any{
		{"id": ids[0]},
		{"id": ids[1]},
	}

	count, err := factory.ExecDeleteBatch(ctx, deleteByID, batchParams)
	if err != nil {
		t.Fatalf("ExecDeleteBatch() failed: %v", err)
	}

	if count != 2 {
		t.Errorf("expected 2 deleted, got %d", count)
	}

	remaining, err := factory.ExecQuery(ctx, queryAll, nil)
	if err != nil {
		t.Fatalf("ExecQuery() failed: %v", err)
	}

	if len(remaining) != 1 {
		t.Errorf("expected 1 remaining user, got %d", len(remaining))
	}
}

func TestDispatch_ExecDeleteBatchTx(t *testing.T) {
	truncateUsers(t)
	ctx := context.Background()

	ids := make([]int, 2)
	for i := 0; i < 2; i++ {
		age := 20 + i
		ids[i] = insertTestUser(t, fmt.Sprintf("user%d@test.com", i), fmt.Sprintf("User%d", i), &age)
	}

	factory, err := edamame.New[User](testDB, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	tx, err := testDB.BeginTxx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTxx() failed: %v", err)
	}

	batchParams := []map[string]any{
		{"id": ids[0]},
		{"id": ids[1]},
	}

	count, err := factory.ExecDeleteBatchTx(ctx, tx, deleteByID, batchParams)
	if err != nil {
		tx.Rollback()
		t.Fatalf("ExecDeleteBatchTx() failed: %v", err)
	}

	if count != 2 {
		tx.Rollback()
		t.Errorf("expected 2 deleted, got %d", count)
	}

	tx.Rollback()

	remaining, err := factory.ExecQuery(ctx, queryAll, nil)
	if err != nil {
		t.Fatalf("ExecQuery() failed: %v", err)
	}

	if len(remaining) != 2 {
		t.Errorf("expected 2 users after rollback, got %d", len(remaining))
	}
}

func TestDispatch_ExecCompoundTx(t *testing.T) {
	truncateUsers(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		age := 20 + i*5
		insertTestUser(t, fmt.Sprintf("user%d@test.com", i), fmt.Sprintf("User%d", i), &age)
	}

	factory, err := edamame.New[User](testDB, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	tx, err := testDB.BeginTxx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTxx() failed: %v", err)
	}
	defer tx.Rollback()

	spec := edamame.CompoundQuerySpec{
		Base: edamame.QuerySpec{
			Fields: []string{"id", "name", "email", "age"},
			Where:  []edamame.ConditionSpec{{Field: "age", Operator: "<", Param: "young_max"}},
		},
		Operands: []edamame.SetOperandSpec{
			{
				Operation: "union",
				Query: edamame.QuerySpec{
					Fields: []string{"id", "name", "email", "age"},
					Where:  []edamame.ConditionSpec{{Field: "age", Operator: ">", Param: "old_min"}},
				},
			},
		},
		OrderBy: []edamame.OrderBySpec{{Field: "age", Direction: "asc"}},
	}

	users, err := factory.ExecCompoundTx(ctx, tx, spec, map[string]any{"q0_young_max": 22, "q1_old_min": 38})
	if err != nil {
		t.Fatalf("ExecCompoundTx() failed: %v", err)
	}

	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
}
