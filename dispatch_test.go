package edamame

import (
	"testing"

	"github.com/zoobz-io/astql/postgres"
)

// User is a test model.
type User struct {
	ID    int    `db:"id" type:"integer" constraints:"primarykey"`
	Email string `db:"email" type:"text" constraints:"notnull,unique"`
	Name  string `db:"name" type:"text"`
	Age   *int   `db:"age" type:"integer"`
}

// Test statements
var (
	queryAll = NewQueryStatement("query-all", "Query all users", QuerySpec{})

	selectByID = NewSelectStatement("select-by-id", "Select user by ID", SelectSpec{
		Where: []ConditionSpec{{Field: "id", Operator: "=", Param: "id"}},
	})

	updateName = NewUpdateStatement("update-name", "Update user name", UpdateSpec{
		Set:   map[string]string{"name": "new_name"},
		Where: []ConditionSpec{{Field: "id", Operator: "=", Param: "id"}},
	})

	deleteByID = NewDeleteStatement("delete-by-id", "Delete user by ID", DeleteSpec{
		Where: []ConditionSpec{{Field: "id", Operator: "=", Param: "id"}},
	})

	countAll = NewAggregateStatement("count-all", "Count all users", AggCount, AggregateSpec{})

	queryByAge = NewQueryStatement("query-by-age", "Query users by age", QuerySpec{
		Where:   []ConditionSpec{{Field: "age", Operator: ">=", Param: "min_age"}},
		OrderBy: []OrderBySpec{{Field: "age", Direction: "desc"}},
		Limit:   intPtr(10),
	})

	sumAge = NewAggregateStatement("sum-age", "Sum of ages", AggSum, AggregateSpec{Field: "age"})
	avgAge = NewAggregateStatement("avg-age", "Average age", AggAvg, AggregateSpec{Field: "age"})
	minAge = NewAggregateStatement("min-age", "Minimum age", AggMin, AggregateSpec{Field: "age"})
	maxAge = NewAggregateStatement("max-age", "Maximum age", AggMax, AggregateSpec{Field: "age"})
)

// Helper for creating int pointers
func intPtr(i int) *int {
	return &i
}

func TestQueryDispatch(t *testing.T) {
	factory, err := New[User](nil, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	builder, err := factory.Query(queryAll)
	if err != nil {
		t.Fatalf("Query() failed: %v", err)
	}
	if builder == nil {
		t.Fatal("Query() returned nil")
	}

	result, err := builder.Render()
	if err != nil {
		t.Fatalf("Render() failed: %v", err)
	}

	if result.SQL == "" {
		t.Error("Render() produced empty SQL")
	}
}

func TestSelectDispatch(t *testing.T) {
	factory, err := New[User](nil, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	builder, err := factory.Select(selectByID)
	if err != nil {
		t.Fatalf("Select() failed: %v", err)
	}
	if builder == nil {
		t.Fatal("Select() returned nil")
	}

	result, err := builder.Render()
	if err != nil {
		t.Fatalf("Render() failed: %v", err)
	}

	if result.SQL == "" {
		t.Error("Render() produced empty SQL")
	}
}

func TestUpdateDispatch(t *testing.T) {
	factory, err := New[User](nil, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	builder := factory.Update(updateName)
	if builder == nil {
		t.Fatal("Update() returned nil")
	}

	result, err := builder.Render()
	if err != nil {
		t.Fatalf("Render() failed: %v", err)
	}

	if result.SQL == "" {
		t.Error("Render() produced empty SQL")
	}
}

func TestDeleteDispatch(t *testing.T) {
	factory, err := New[User](nil, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	builder := factory.Delete(deleteByID)
	if builder == nil {
		t.Fatal("Delete() returned nil")
	}

	result, err := builder.Render()
	if err != nil {
		t.Fatalf("Render() failed: %v", err)
	}

	if result.SQL == "" {
		t.Error("Render() produced empty SQL")
	}
}

func TestAggregateDispatch(t *testing.T) {
	factory, err := New[User](nil, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	builder := factory.Aggregate(countAll)
	if builder == nil {
		t.Fatal("Aggregate() returned nil")
	}

	result, err := builder.Render()
	if err != nil {
		t.Fatalf("Render() failed: %v", err)
	}

	if result.SQL == "" {
		t.Error("Render() produced empty SQL")
	}
}

func TestInsertDispatch(t *testing.T) {
	factory, err := New[User](nil, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	builder := factory.Insert()
	if builder == nil {
		t.Fatal("Insert() returned nil")
	}

	result, err := builder.Render()
	if err != nil {
		t.Fatalf("Render() failed: %v", err)
	}

	if result.SQL == "" {
		t.Error("Render() produced empty SQL")
	}
}

func TestCustomQueryDispatch(t *testing.T) {
	factory, err := New[User](nil, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	builder, err := factory.Query(queryByAge)
	if err != nil {
		t.Fatalf("Query() failed: %v", err)
	}

	result, err := builder.Render()
	if err != nil {
		t.Fatalf("Render() failed: %v", err)
	}

	if result.SQL == "" {
		t.Error("Render() produced empty SQL")
	}
}

func TestAggregateDispatchVariants(t *testing.T) {
	factory, err := New[User](nil, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	tests := []struct {
		name string
		stmt AggregateStatement
	}{
		{"sum-age", sumAge},
		{"avg-age", avgAge},
		{"min-age", minAge},
		{"max-age", maxAge},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := factory.Aggregate(tt.stmt)

			result, err := builder.Render()
			if err != nil {
				t.Fatalf("Render() failed: %v", err)
			}

			if result.SQL == "" {
				t.Error("Render() produced empty SQL")
			}
		})
	}
}

func TestBuilderChaining(t *testing.T) {
	factory, err := New[User](nil, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	builder, err := factory.Query(queryAll)
	if err != nil {
		t.Fatalf("Query() failed: %v", err)
	}

	result, err := builder.
		Where("age", ">=", "min_age").
		OrderBy("name", "asc").
		Limit(10).
		Render()
	if err != nil {
		t.Fatalf("Render() failed: %v", err)
	}

	if result.SQL == "" {
		t.Error("Render() produced empty SQL")
	}
}

func TestRenderMethods(t *testing.T) {
	factory, err := New[User](nil, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	tests := []struct {
		name   string
		render func() (string, error)
	}{
		{"RenderQuery", func() (string, error) { return factory.RenderQuery(queryAll) }},
		{"RenderSelect", func() (string, error) { return factory.RenderSelect(selectByID) }},
		{"RenderUpdate", func() (string, error) { return factory.RenderUpdate(updateName) }},
		{"RenderDelete", func() (string, error) { return factory.RenderDelete(deleteByID) }},
		{"RenderAggregate_Count", func() (string, error) { return factory.RenderAggregate(countAll) }},
		{"RenderAggregate_Sum", func() (string, error) { return factory.RenderAggregate(sumAge) }},
		{"RenderAggregate_Avg", func() (string, error) { return factory.RenderAggregate(avgAge) }},
		{"RenderAggregate_Min", func() (string, error) { return factory.RenderAggregate(minAge) }},
		{"RenderAggregate_Max", func() (string, error) { return factory.RenderAggregate(maxAge) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, err := tt.render()
			if err != nil {
				t.Fatalf("%s failed: %v", tt.name, err)
			}
			if sql == "" {
				t.Errorf("%s returned empty SQL", tt.name)
			}
		})
	}
}
