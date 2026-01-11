// Package edamame provides a statement-driven query executor for Go.
//
// This file serves as the public interface entry point, documenting
// and consolidating the package's exported API.
//
// # Core Types
//
// The primary types for working with edamame are:
//
//   - [Executor]: Generic query executor for a model type T
//   - [QueryStatement]: SELECT queries returning multiple records
//   - [SelectStatement]: SELECT queries returning a single record
//   - [UpdateStatement]: UPDATE mutations
//   - [DeleteStatement]: DELETE mutations
//   - [AggregateStatement]: Aggregate queries (COUNT, SUM, AVG, MIN, MAX)
//
// # Spec Types
//
// Statements are created from spec types that define query structure:
//
//   - [QuerySpec]: Specification for query statements
//   - [SelectSpec]: Specification for select statements
//   - [UpdateSpec]: Specification for update statements
//   - [DeleteSpec]: Specification for delete statements
//   - [AggregateSpec]: Specification for aggregate statements
//   - [ConditionSpec]: WHERE/HAVING clause conditions
//   - [OrderBySpec]: ORDER BY clause ordering
//   - [CompoundQuerySpec]: UNION/INTERSECT/EXCEPT queries
//
// # Statement Creation
//
// Create statements using the New*Statement constructors:
//
//   - [NewQueryStatement]: Create a query statement
//   - [NewSelectStatement]: Create a select statement
//   - [NewUpdateStatement]: Create an update statement
//   - [NewDeleteStatement]: Create a delete statement
//   - [NewAggregateStatement]: Create an aggregate statement
//
// # Aggregate Functions
//
// Available aggregate function constants:
//
//   - [AggCount]: COUNT aggregate
//   - [AggSum]: SUM aggregate
//   - [AggAvg]: AVG aggregate
//   - [AggMin]: MIN aggregate
//   - [AggMax]: MAX aggregate
//
// # Execution Methods
//
// The Executor provides these execution methods:
//
//   - ExecQuery: Execute a query statement, returns []T
//   - ExecSelect: Execute a select statement, returns T
//   - ExecUpdate: Execute an update statement, returns affected rows
//   - ExecDelete: Execute a delete statement, returns affected rows
//   - ExecAggregate: Execute an aggregate statement, returns float64
//   - ExecInsert: Insert a record, returns the inserted record
//   - ExecCompound: Execute a compound query, returns []T
//
// # Events
//
// The package emits structured events via capitan:
//
//   - [ExecutorCreated]: Emitted when a new Executor is created
//   - [KeyTable]: Event key for table name
//
// # Example
//
//	type User struct {
//	    ID    int    `db:"id"`
//	    Email string `db:"email"`
//	}
//
//	var ByEmail = edamame.NewSelectStatement("by-email", "Select by email",
//	    edamame.SelectSpec{
//	        Where: []edamame.ConditionSpec{
//	            {Field: "email", Operator: "=", Param: "email"},
//	        },
//	    },
//	)
//
//	exec, _ := edamame.New[User](db, "users", renderer)
//	user, _ := exec.ExecSelect(ctx, ByEmail, map[string]any{"email": "test@example.com"})
package edamame
