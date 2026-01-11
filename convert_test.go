package edamame

import (
	"strings"
	"testing"

	"github.com/zoobzio/astql/postgres"
)

func TestToCondition(t *testing.T) {
	tests := []struct {
		name string
		spec ConditionSpec
	}{
		{
			name: "simple condition",
			spec: ConditionSpec{Field: "age", Operator: ">=", Param: "min_age"},
		},
		{
			name: "null condition",
			spec: ConditionSpec{Field: "email", IsNull: true},
		},
		{
			name: "not null condition",
			spec: ConditionSpec{Field: "email", IsNull: true, Operator: "IS NOT NULL"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			// Exercise the code path - soy.Condition is opaque
			// so we just verify it doesn't panic
			_ = tt.spec.toCondition()
		})
	}
}

func TestToConditions(t *testing.T) {
	specs := []ConditionSpec{
		{Field: "age", Operator: ">=", Param: "min_age"},
		{Field: "status", Operator: "=", Param: "status"},
		// Group should be filtered out
		{
			Logic: "OR",
			Group: []ConditionSpec{
				{Field: "x", Operator: "=", Param: "x"},
			},
		},
	}

	conditions := toConditions(specs)

	// Should have 2 conditions (group filtered out)
	if len(conditions) != 2 {
		t.Errorf("toConditions() returned %d conditions, want 2", len(conditions))
	}
}

func TestQueryFromSpec(t *testing.T) {
	factory, err := New[User](nil, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	limit := 10
	offset := 5
	spec := QuerySpec{
		Fields:     []string{"id", "name"},
		Where:      []ConditionSpec{{Field: "age", Operator: ">=", Param: "min_age"}},
		OrderBy:    []OrderBySpec{{Field: "name", Direction: "asc"}},
		GroupBy:    []string{"name"},
		Limit:      &limit,
		Offset:     &offset,
		Distinct:   true,
		ForLocking: "update",
	}

	builder, err := factory.queryFromSpec(spec)
	if err != nil {
		t.Fatalf("queryFromSpec() failed: %v", err)
	}
	if builder == nil {
		t.Fatal("queryFromSpec() returned nil")
	}

	result, err := builder.Render()
	if err != nil {
		t.Fatalf("Render() failed: %v", err)
	}

	sql := result.SQL
	if sql == "" {
		t.Error("Render() produced empty SQL")
	}

	// Verify key clauses are present
	checks := []string{"SELECT", "FROM", "WHERE", "ORDER BY", "GROUP BY", "LIMIT", "OFFSET", "DISTINCT", "FOR UPDATE"}
	for _, check := range checks {
		if !strings.Contains(strings.ToUpper(sql), check) {
			t.Errorf("SQL missing %s clause: %s", check, sql)
		}
	}
}

func TestQueryFromSpecWithOrderByVariants(t *testing.T) {
	factory, err := New[User](nil, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	tests := []struct {
		name    string
		orderBy OrderBySpec
	}{
		{
			name:    "simple",
			orderBy: OrderBySpec{Field: "name", Direction: "asc"},
		},
		{
			name:    "with nulls",
			orderBy: OrderBySpec{Field: "name", Direction: "asc", Nulls: "last"},
		},
		{
			name:    "expression",
			orderBy: OrderBySpec{Field: "age", Operator: "<->", Param: "vec", Direction: "asc"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := QuerySpec{
				OrderBy: []OrderBySpec{tt.orderBy},
			}
			builder, err := factory.queryFromSpec(spec)
			if err != nil {
				t.Fatalf("queryFromSpec() failed: %v", err)
			}
			if builder == nil {
				t.Fatal("queryFromSpec() returned nil")
			}

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

func TestQueryFromSpecWithConditionGroups(t *testing.T) {
	factory, err := New[User](nil, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	spec := QuerySpec{
		Where: []ConditionSpec{
			{Field: "age", Operator: ">=", Param: "min_age"},
			{
				Logic: "OR",
				Group: []ConditionSpec{
					{Field: "name", Operator: "=", Param: "name1"},
					{Field: "name", Operator: "=", Param: "name2"},
				},
			},
		},
	}

	builder, err := factory.queryFromSpec(spec)
	if err != nil {
		t.Fatalf("queryFromSpec() failed: %v", err)
	}
	result, err := builder.Render()
	if err != nil {
		t.Fatalf("Render() failed: %v", err)
	}

	if result.SQL == "" {
		t.Error("Render() produced empty SQL")
	}
}

func TestSelectFromSpec(t *testing.T) {
	factory, err := New[User](nil, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	spec := SelectSpec{
		Fields:     []string{"id", "name"},
		Where:      []ConditionSpec{{Field: "id", Operator: "=", Param: "id"}},
		ForLocking: "share",
	}

	builder, err := factory.selectFromSpec(spec)
	if err != nil {
		t.Fatalf("selectFromSpec() failed: %v", err)
	}
	if builder == nil {
		t.Fatal("selectFromSpec() returned nil")
	}

	result, err := builder.Render()
	if err != nil {
		t.Fatalf("Render() failed: %v", err)
	}

	if result.SQL == "" {
		t.Error("Render() produced empty SQL")
	}
}

func TestModifyFromSpec(t *testing.T) {
	factory, err := New[User](nil, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	spec := UpdateSpec{
		Set:   map[string]string{"name": "new_name"},
		Where: []ConditionSpec{{Field: "id", Operator: "=", Param: "id"}},
	}

	builder := factory.modifyFromSpec(spec)
	if builder == nil {
		t.Fatal("modifyFromSpec() returned nil")
	}

	result, err := builder.Render()
	if err != nil {
		t.Fatalf("Render() failed: %v", err)
	}

	if result.SQL == "" {
		t.Error("Render() produced empty SQL")
	}

	if !strings.Contains(strings.ToUpper(result.SQL), "UPDATE") {
		t.Error("SQL missing UPDATE clause")
	}
}

func TestRemoveFromSpec(t *testing.T) {
	factory, err := New[User](nil, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	spec := DeleteSpec{
		Where: []ConditionSpec{{Field: "id", Operator: "=", Param: "id"}},
	}

	builder := factory.removeFromSpec(spec)
	if builder == nil {
		t.Fatal("removeFromSpec() returned nil")
	}

	result, err := builder.Render()
	if err != nil {
		t.Fatalf("Render() failed: %v", err)
	}

	if result.SQL == "" {
		t.Error("Render() produced empty SQL")
	}

	if !strings.Contains(strings.ToUpper(result.SQL), "DELETE") {
		t.Error("SQL missing DELETE clause")
	}
}

func TestAggregateFromSpec(t *testing.T) {
	factory, err := New[User](nil, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	spec := AggregateSpec{
		Field: "age",
		Where: []ConditionSpec{{Field: "name", Operator: "=", Param: "name"}},
	}

	t.Run("count", func(t *testing.T) {
		builder := factory.countFromSpec(spec)
		result, err := builder.Render()
		if err != nil {
			t.Fatalf("Render() failed: %v", err)
		}
		if result.SQL == "" {
			t.Error("Render() produced empty SQL")
		}
	})

	t.Run("sum", func(t *testing.T) {
		builder := factory.sumFromSpec(spec)
		result, err := builder.Render()
		if err != nil {
			t.Fatalf("Render() failed: %v", err)
		}
		if result.SQL == "" {
			t.Error("Render() produced empty SQL")
		}
	})

	t.Run("avg", func(t *testing.T) {
		builder := factory.avgFromSpec(spec)
		result, err := builder.Render()
		if err != nil {
			t.Fatalf("Render() failed: %v", err)
		}
		if result.SQL == "" {
			t.Error("Render() produced empty SQL")
		}
	})

	t.Run("min", func(t *testing.T) {
		builder := factory.minFromSpec(spec)
		result, err := builder.Render()
		if err != nil {
			t.Fatalf("Render() failed: %v", err)
		}
		if result.SQL == "" {
			t.Error("Render() produced empty SQL")
		}
	})

	t.Run("max", func(t *testing.T) {
		builder := factory.maxFromSpec(spec)
		result, err := builder.Render()
		if err != nil {
			t.Fatalf("Render() failed: %v", err)
		}
		if result.SQL == "" {
			t.Error("Render() produced empty SQL")
		}
	})
}

func TestInsertFromSpec(t *testing.T) {
	factory, err := New[User](nil, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	tests := []struct {
		name string
		spec CreateSpec
	}{
		{
			name: "simple insert",
			spec: CreateSpec{},
		},
		{
			name: "on conflict do nothing",
			spec: CreateSpec{
				OnConflict:     []string{"email"},
				ConflictAction: "nothing",
			},
		},
		{
			name: "on conflict do update",
			spec: CreateSpec{
				OnConflict:     []string{"email"},
				ConflictAction: "update",
				ConflictSet:    map[string]string{"name": "new_name"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder, err := factory.insertFromSpec(tt.spec)
			if err != nil {
				t.Fatalf("insertFromSpec() failed: %v", err)
			}
			if builder == nil {
				t.Fatal("insertFromSpec() returned nil")
			}

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

func TestApplyForLocking(t *testing.T) {
	factory, err := New[User](nil, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	tests := []struct {
		name       string
		forLocking string
		contains   string
		wantErr    bool
	}{
		{"update", "update", "FOR UPDATE", false},
		{"no_key_update", "no_key_update", "FOR NO KEY UPDATE", false},
		{"share", "share", "FOR SHARE", false},
		{"key_share", "key_share", "FOR KEY SHARE", false},
		{"empty", "", "", false},
		{"invalid lock mode", "invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := QuerySpec{ForLocking: tt.forLocking}
			builder, err := factory.queryFromSpec(spec)
			if tt.wantErr {
				if err == nil {
					t.Error("queryFromSpec() should have returned an error for invalid lock mode")
				}
				return
			}
			if err != nil {
				t.Fatalf("queryFromSpec() failed: %v", err)
			}

			result, err := builder.Render()
			if err != nil {
				t.Fatalf("Render() failed: %v", err)
			}

			if tt.contains != "" {
				if !strings.Contains(strings.ToUpper(result.SQL), tt.contains) {
					t.Errorf("SQL should contain %q: %s", tt.contains, result.SQL)
				}
			}
		})
	}
}

func TestNullConditions(t *testing.T) {
	factory, err := New[User](nil, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	tests := []struct {
		name     string
		spec     ConditionSpec
		contains string
	}{
		{
			name:     "is null",
			spec:     ConditionSpec{Field: "email", IsNull: true, Operator: "IS NULL"},
			contains: "IS NULL",
		},
		{
			name:     "is not null",
			spec:     ConditionSpec{Field: "email", IsNull: true, Operator: "IS NOT NULL"},
			contains: "IS NOT NULL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			querySpec := QuerySpec{Where: []ConditionSpec{tt.spec}}
			builder, err := factory.queryFromSpec(querySpec)
			if err != nil {
				t.Fatalf("queryFromSpec() failed: %v", err)
			}
			result, err := builder.Render()
			if err != nil {
				t.Fatalf("Render() failed: %v", err)
			}

			if !strings.Contains(strings.ToUpper(result.SQL), tt.contains) {
				t.Errorf("SQL should contain %q: %s", tt.contains, result.SQL)
			}
		})
	}
}

func TestBetweenConditions(t *testing.T) {
	factory, err := New[User](nil, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	tests := []struct {
		name     string
		spec     QuerySpec
		contains string
	}{
		{
			name: "between",
			spec: QuerySpec{
				Where: []ConditionSpec{
					{Field: "age", Between: true, LowParam: "min_age", HighParam: "max_age"},
				},
			},
			contains: "BETWEEN",
		},
		{
			name: "not between",
			spec: QuerySpec{
				Where: []ConditionSpec{
					{Field: "age", NotBetween: true, LowParam: "min_age", HighParam: "max_age"},
				},
			},
			contains: "NOT BETWEEN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder, err := factory.queryFromSpec(tt.spec)
			if err != nil {
				t.Fatalf("queryFromSpec() failed: %v", err)
			}
			result, err := builder.Render()
			if err != nil {
				t.Fatalf("Render() failed: %v", err)
			}
			if !strings.Contains(strings.ToUpper(result.SQL), tt.contains) {
				t.Errorf("SQL should contain %q: %s", tt.contains, result.SQL)
			}
		})
	}
}

func TestFieldToFieldComparison(t *testing.T) {
	factory, err := New[User](nil, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	spec := QuerySpec{
		Where: []ConditionSpec{
			{Field: "id", Operator: "<", RightField: "age"},
		},
	}

	builder, err := factory.queryFromSpec(spec)
	if err != nil {
		t.Fatalf("queryFromSpec() failed: %v", err)
	}
	result, err := builder.Render()
	if err != nil {
		t.Fatalf("Render() failed: %v", err)
	}

	// Should compare two fields, not a field and param
	sql := result.SQL
	if !strings.Contains(sql, `"id"`) || !strings.Contains(sql, `"age"`) {
		t.Errorf("SQL should compare two fields: %s", sql)
	}
}

func TestParameterizedPagination(t *testing.T) {
	factory, err := New[User](nil, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	tests := []struct {
		name     string
		spec     QuerySpec
		contains string
	}{
		{
			name: "limit param",
			spec: QuerySpec{
				LimitParam: "page_size",
			},
			contains: ":page_size",
		},
		{
			name: "offset param",
			spec: QuerySpec{
				OffsetParam: "page_offset",
			},
			contains: ":page_offset",
		},
		{
			name: "both params",
			spec: QuerySpec{
				LimitParam:  "page_size",
				OffsetParam: "page_offset",
			},
			contains: ":page_size",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder, err := factory.queryFromSpec(tt.spec)
			if err != nil {
				t.Fatalf("queryFromSpec() failed: %v", err)
			}
			result, err := builder.Render()
			if err != nil {
				t.Fatalf("Render() failed: %v", err)
			}
			if !strings.Contains(result.SQL, tt.contains) {
				t.Errorf("SQL should contain %q: %s", tt.contains, result.SQL)
			}
		})
	}
}

func TestSelectExpressions(t *testing.T) {
	factory, err := New[User](nil, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	tests := []struct {
		name     string
		expr     SelectExprSpec
		contains string
	}{
		// String functions
		{
			name:     "upper",
			expr:     SelectExprSpec{Func: "upper", Field: "name", Alias: "upper_name"},
			contains: "UPPER",
		},
		{
			name:     "lower",
			expr:     SelectExprSpec{Func: "lower", Field: "email", Alias: "lower_email"},
			contains: "LOWER",
		},
		{
			name:     "length",
			expr:     SelectExprSpec{Func: "length", Field: "name", Alias: "name_len"},
			contains: "LENGTH",
		},
		{
			name:     "trim",
			expr:     SelectExprSpec{Func: "trim", Field: "name", Alias: "trimmed"},
			contains: "TRIM",
		},
		{
			name:     "ltrim",
			expr:     SelectExprSpec{Func: "ltrim", Field: "name", Alias: "ltrimmed"},
			contains: "LTRIM",
		},
		{
			name:     "rtrim",
			expr:     SelectExprSpec{Func: "rtrim", Field: "name", Alias: "rtrimmed"},
			contains: "RTRIM",
		},
		{
			name:     "substring",
			expr:     SelectExprSpec{Func: "substring", Field: "name", Params: []string{"start_pos", "length_val"}, Alias: "sub"},
			contains: "SUBSTRING",
		},
		{
			name:     "replace",
			expr:     SelectExprSpec{Func: "replace", Field: "name", Params: []string{"old", "new"}, Alias: "replaced"},
			contains: "REPLACE",
		},
		{
			name:     "concat",
			expr:     SelectExprSpec{Func: "concat", Fields: []string{"name", "email"}, Alias: "combined"},
			contains: "CONCAT",
		},
		// Math functions
		{
			name:     "abs",
			expr:     SelectExprSpec{Func: "abs", Field: "age", Alias: "abs_age"},
			contains: "ABS",
		},
		{
			name:     "ceil",
			expr:     SelectExprSpec{Func: "ceil", Field: "age", Alias: "ceil_age"},
			contains: "CEIL",
		},
		{
			name:     "floor",
			expr:     SelectExprSpec{Func: "floor", Field: "age", Alias: "floor_age"},
			contains: "FLOOR",
		},
		{
			name:     "round",
			expr:     SelectExprSpec{Func: "round", Field: "age", Alias: "round_age"},
			contains: "ROUND",
		},
		{
			name:     "sqrt",
			expr:     SelectExprSpec{Func: "sqrt", Field: "age", Alias: "sqrt_age"},
			contains: "SQRT",
		},
		{
			name:     "power",
			expr:     SelectExprSpec{Func: "power", Field: "age", Params: []string{"exponent"}, Alias: "squared"},
			contains: "POWER",
		},
		// Date/Time functions
		{
			name:     "now",
			expr:     SelectExprSpec{Func: "now", Alias: "current_ts"},
			contains: "NOW",
		},
		{
			name:     "current_date",
			expr:     SelectExprSpec{Func: "current_date", Alias: "today"},
			contains: "CURRENT_DATE",
		},
		{
			name:     "current_time",
			expr:     SelectExprSpec{Func: "current_time", Alias: "now_time"},
			contains: "CURRENT_TIME",
		},
		{
			name:     "current_timestamp",
			expr:     SelectExprSpec{Func: "current_timestamp", Alias: "now_ts"},
			contains: "CURRENT_TIMESTAMP",
		},
		// Type casting
		{
			name:     "cast",
			expr:     SelectExprSpec{Func: "cast", Field: "age", CastType: "text", Alias: "age_text"},
			contains: "CAST",
		},
		// Aggregate functions
		{
			name:     "count_star",
			expr:     SelectExprSpec{Func: "count_star", Alias: "total"},
			contains: "COUNT(*)",
		},
		{
			name:     "count",
			expr:     SelectExprSpec{Func: "count", Field: "id", Alias: "id_count"},
			contains: "COUNT",
		},
		{
			name:     "count_distinct",
			expr:     SelectExprSpec{Func: "count_distinct", Field: "email", Alias: "unique_emails"},
			contains: "DISTINCT",
		},
		{
			name:     "sum",
			expr:     SelectExprSpec{Func: "sum", Field: "age", Alias: "total_age"},
			contains: "SUM",
		},
		{
			name:     "avg",
			expr:     SelectExprSpec{Func: "avg", Field: "age", Alias: "avg_age"},
			contains: "AVG",
		},
		{
			name:     "min",
			expr:     SelectExprSpec{Func: "min", Field: "age", Alias: "min_age"},
			contains: "MIN",
		},
		{
			name:     "max",
			expr:     SelectExprSpec{Func: "max", Field: "age", Alias: "max_age"},
			contains: "MAX",
		},
		// Conditional functions
		{
			name:     "coalesce",
			expr:     SelectExprSpec{Func: "coalesce", Params: []string{"name", "default_name"}, Alias: "result"},
			contains: "COALESCE",
		},
		{
			name:     "nullif",
			expr:     SelectExprSpec{Func: "nullif", Params: []string{"age", "compare_val"}, Alias: "nullif_age"},
			contains: "NULLIF",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := QuerySpec{
				SelectExprs: []SelectExprSpec{tt.expr},
			}
			builder, err := factory.queryFromSpec(spec)
			if err != nil {
				t.Fatalf("queryFromSpec() failed: %v", err)
			}
			result, err := builder.Render()
			if err != nil {
				t.Fatalf("Render() failed: %v", err)
			}
			if !strings.Contains(strings.ToUpper(result.SQL), tt.contains) {
				t.Errorf("SQL should contain %q: %s", tt.contains, result.SQL)
			}
		})
	}
}

func TestCompoundQueryFromSpec(t *testing.T) {
	factory, err := New[User](nil, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	tests := []struct {
		name     string
		spec     CompoundQuerySpec
		contains string
		wantErr  bool
	}{
		{
			name: "union",
			spec: CompoundQuerySpec{
				Base: QuerySpec{Fields: []string{"id", "name"}},
				Operands: []SetOperandSpec{
					{Operation: "union", Query: QuerySpec{Fields: []string{"id", "name"}, Where: []ConditionSpec{{Field: "age", Operator: ">", Param: "min_age"}}}},
				},
			},
			contains: "UNION",
		},
		{
			name: "union_all",
			spec: CompoundQuerySpec{
				Base: QuerySpec{Fields: []string{"id", "name"}},
				Operands: []SetOperandSpec{
					{Operation: "union_all", Query: QuerySpec{Fields: []string{"id", "name"}}},
				},
			},
			contains: "UNION ALL",
		},
		{
			name: "intersect",
			spec: CompoundQuerySpec{
				Base: QuerySpec{Fields: []string{"id"}},
				Operands: []SetOperandSpec{
					{Operation: "intersect", Query: QuerySpec{Fields: []string{"id"}}},
				},
			},
			contains: "INTERSECT",
		},
		{
			name: "intersect_all",
			spec: CompoundQuerySpec{
				Base: QuerySpec{Fields: []string{"id"}},
				Operands: []SetOperandSpec{
					{Operation: "intersect_all", Query: QuerySpec{Fields: []string{"id"}}},
				},
			},
			contains: "INTERSECT ALL",
		},
		{
			name: "except",
			spec: CompoundQuerySpec{
				Base: QuerySpec{Fields: []string{"id"}},
				Operands: []SetOperandSpec{
					{Operation: "except", Query: QuerySpec{Fields: []string{"id"}}},
				},
			},
			contains: "EXCEPT",
		},
		{
			name: "except_all",
			spec: CompoundQuerySpec{
				Base: QuerySpec{Fields: []string{"id"}},
				Operands: []SetOperandSpec{
					{Operation: "except_all", Query: QuerySpec{Fields: []string{"id"}}},
				},
			},
			contains: "EXCEPT ALL",
		},
		{
			name: "with order by",
			spec: CompoundQuerySpec{
				Base: QuerySpec{Fields: []string{"id", "name"}},
				Operands: []SetOperandSpec{
					{Operation: "union", Query: QuerySpec{Fields: []string{"id", "name"}}},
				},
				OrderBy: []OrderBySpec{{Field: "name", Direction: "asc"}},
			},
			contains: "ORDER BY",
		},
		{
			name: "with limit and offset",
			spec: CompoundQuerySpec{
				Base: QuerySpec{Fields: []string{"id"}},
				Operands: []SetOperandSpec{
					{Operation: "union", Query: QuerySpec{Fields: []string{"id"}}},
				},
				Limit:  intPtr(10),
				Offset: intPtr(5),
			},
			contains: "LIMIT",
		},
		{
			name: "multiple operands",
			spec: CompoundQuerySpec{
				Base: QuerySpec{Fields: []string{"id"}},
				Operands: []SetOperandSpec{
					{Operation: "union", Query: QuerySpec{Fields: []string{"id"}}},
					{Operation: "except", Query: QuerySpec{Fields: []string{"id"}}},
				},
			},
			contains: "EXCEPT",
		},
		{
			name: "no operands",
			spec: CompoundQuerySpec{
				Base: QuerySpec{Fields: []string{"id"}},
			},
			wantErr: true,
		},
		{
			name: "invalid operation",
			spec: CompoundQuerySpec{
				Base: QuerySpec{Fields: []string{"id"}},
				Operands: []SetOperandSpec{
					{Operation: "invalid", Query: QuerySpec{Fields: []string{"id"}}},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder, err := factory.Compound(tt.spec)
			if tt.wantErr {
				if err == nil {
					t.Error("Compound() should have returned an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Compound() failed: %v", err)
			}

			result, err := builder.Render()
			if err != nil {
				t.Fatalf("Render() failed: %v", err)
			}

			if tt.contains != "" && !strings.Contains(strings.ToUpper(result.SQL), strings.ToUpper(tt.contains)) {
				t.Errorf("SQL should contain %q: %s", tt.contains, result.SQL)
			}
		})
	}
}

func TestRenderCompound(t *testing.T) {
	factory, err := New[User](nil, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	spec := CompoundQuerySpec{
		Base: QuerySpec{Fields: []string{"id", "name"}},
		Operands: []SetOperandSpec{
			{Operation: "union", Query: QuerySpec{Fields: []string{"id", "name"}}},
		},
	}

	sql, err := factory.RenderCompound(spec)
	if err != nil {
		t.Fatalf("RenderCompound() failed: %v", err)
	}

	if !strings.Contains(strings.ToUpper(sql), "UNION") {
		t.Errorf("SQL should contain UNION: %s", sql)
	}
}

func TestSelectConditionVariants(t *testing.T) {
	factory, err := New[User](nil, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	tests := []struct {
		name     string
		spec     SelectSpec
		contains string
	}{
		{
			name: "between",
			spec: SelectSpec{
				Where: []ConditionSpec{
					{Field: "age", Between: true, LowParam: "min_age", HighParam: "max_age"},
				},
			},
			contains: "BETWEEN",
		},
		{
			name: "not between",
			spec: SelectSpec{
				Where: []ConditionSpec{
					{Field: "age", NotBetween: true, LowParam: "min_age", HighParam: "max_age"},
				},
			},
			contains: "NOT BETWEEN",
		},
		{
			name: "is null",
			spec: SelectSpec{
				Where: []ConditionSpec{
					{Field: "email", IsNull: true, Operator: "IS NULL"},
				},
			},
			contains: "IS NULL",
		},
		{
			name: "is not null",
			spec: SelectSpec{
				Where: []ConditionSpec{
					{Field: "email", IsNull: true, Operator: "IS NOT NULL"},
				},
			},
			contains: "IS NOT NULL",
		},
		{
			name: "field comparison",
			spec: SelectSpec{
				Where: []ConditionSpec{
					{Field: "id", Operator: "<", RightField: "age"},
				},
			},
			contains: `"age"`,
		},
		{
			name: "condition group or",
			spec: SelectSpec{
				Where: []ConditionSpec{
					{
						Logic: "OR",
						Group: []ConditionSpec{
							{Field: "name", Operator: "=", Param: "name1"},
							{Field: "name", Operator: "=", Param: "name2"},
						},
					},
				},
			},
			contains: "OR",
		},
		{
			name: "condition group and",
			spec: SelectSpec{
				Where: []ConditionSpec{
					{
						Logic: "AND",
						Group: []ConditionSpec{
							{Field: "name", Operator: "=", Param: "name1"},
							{Field: "age", Operator: ">", Param: "min_age"},
						},
					},
				},
			},
			contains: "AND",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder, err := factory.selectFromSpec(tt.spec)
			if err != nil {
				t.Fatalf("selectFromSpec() failed: %v", err)
			}
			result, err := builder.Render()
			if err != nil {
				t.Fatalf("Render() failed: %v", err)
			}
			if !strings.Contains(result.SQL, tt.contains) {
				t.Errorf("SQL should contain %q: %s", tt.contains, result.SQL)
			}
		})
	}
}

func TestUpdateConditionVariants(t *testing.T) {
	factory, err := New[User](nil, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	tests := []struct {
		name     string
		spec     UpdateSpec
		contains string
	}{
		{
			name: "between",
			spec: UpdateSpec{
				Set: map[string]string{"name": "new_name"},
				Where: []ConditionSpec{
					{Field: "age", Between: true, LowParam: "min_age", HighParam: "max_age"},
				},
			},
			contains: "BETWEEN",
		},
		{
			name: "not between",
			spec: UpdateSpec{
				Set: map[string]string{"name": "new_name"},
				Where: []ConditionSpec{
					{Field: "age", NotBetween: true, LowParam: "min_age", HighParam: "max_age"},
				},
			},
			contains: "NOT BETWEEN",
		},
		{
			name: "is null",
			spec: UpdateSpec{
				Set: map[string]string{"name": "new_name"},
				Where: []ConditionSpec{
					{Field: "email", IsNull: true, Operator: "IS NULL"},
				},
			},
			contains: "IS NULL",
		},
		{
			name: "is not null",
			spec: UpdateSpec{
				Set: map[string]string{"name": "new_name"},
				Where: []ConditionSpec{
					{Field: "email", IsNull: true, Operator: "IS NOT NULL"},
				},
			},
			contains: "IS NOT NULL",
		},
		{
			name: "condition group or",
			spec: UpdateSpec{
				Set: map[string]string{"name": "new_name"},
				Where: []ConditionSpec{
					{
						Logic: "OR",
						Group: []ConditionSpec{
							{Field: "name", Operator: "=", Param: "name1"},
							{Field: "name", Operator: "=", Param: "name2"},
						},
					},
				},
			},
			contains: "OR",
		},
		{
			name: "condition group and",
			spec: UpdateSpec{
				Set: map[string]string{"name": "new_name"},
				Where: []ConditionSpec{
					{
						Logic: "AND",
						Group: []ConditionSpec{
							{Field: "name", Operator: "=", Param: "name1"},
							{Field: "age", Operator: ">", Param: "min_age"},
						},
					},
				},
			},
			contains: "AND",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := factory.modifyFromSpec(tt.spec)
			result, err := builder.Render()
			if err != nil {
				t.Fatalf("Render() failed: %v", err)
			}
			if !strings.Contains(result.SQL, tt.contains) {
				t.Errorf("SQL should contain %q: %s", tt.contains, result.SQL)
			}
		})
	}
}

func TestDeleteConditionVariants(t *testing.T) {
	factory, err := New[User](nil, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	tests := []struct {
		name     string
		spec     DeleteSpec
		contains string
	}{
		{
			name: "between",
			spec: DeleteSpec{
				Where: []ConditionSpec{
					{Field: "age", Between: true, LowParam: "min_age", HighParam: "max_age"},
				},
			},
			contains: "BETWEEN",
		},
		{
			name: "not between",
			spec: DeleteSpec{
				Where: []ConditionSpec{
					{Field: "age", NotBetween: true, LowParam: "min_age", HighParam: "max_age"},
				},
			},
			contains: "NOT BETWEEN",
		},
		{
			name: "is null",
			spec: DeleteSpec{
				Where: []ConditionSpec{
					{Field: "email", IsNull: true, Operator: "IS NULL"},
				},
			},
			contains: "IS NULL",
		},
		{
			name: "is not null",
			spec: DeleteSpec{
				Where: []ConditionSpec{
					{Field: "email", IsNull: true, Operator: "IS NOT NULL"},
				},
			},
			contains: "IS NOT NULL",
		},
		{
			name: "field comparison",
			spec: DeleteSpec{
				Where: []ConditionSpec{
					{Field: "id", Operator: "<", RightField: "age"},
				},
			},
			contains: `"age"`,
		},
		{
			name: "condition group or",
			spec: DeleteSpec{
				Where: []ConditionSpec{
					{
						Logic: "OR",
						Group: []ConditionSpec{
							{Field: "name", Operator: "=", Param: "name1"},
							{Field: "name", Operator: "=", Param: "name2"},
						},
					},
				},
			},
			contains: "OR",
		},
		{
			name: "condition group and",
			spec: DeleteSpec{
				Where: []ConditionSpec{
					{
						Logic: "AND",
						Group: []ConditionSpec{
							{Field: "name", Operator: "=", Param: "name1"},
							{Field: "age", Operator: ">", Param: "min_age"},
						},
					},
				},
			},
			contains: "AND",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := factory.removeFromSpec(tt.spec)
			result, err := builder.Render()
			if err != nil {
				t.Fatalf("Render() failed: %v", err)
			}
			if !strings.Contains(result.SQL, tt.contains) {
				t.Errorf("SQL should contain %q: %s", tt.contains, result.SQL)
			}
		})
	}
}

func TestAggregateConditionVariants(t *testing.T) {
	factory, err := New[User](nil, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	tests := []struct {
		name     string
		spec     AggregateSpec
		contains string
	}{
		{
			name: "between",
			spec: AggregateSpec{
				Field: "id",
				Where: []ConditionSpec{
					{Field: "age", Between: true, LowParam: "min_age", HighParam: "max_age"},
				},
			},
			contains: "BETWEEN",
		},
		{
			name: "not between",
			spec: AggregateSpec{
				Field: "id",
				Where: []ConditionSpec{
					{Field: "age", NotBetween: true, LowParam: "min_age", HighParam: "max_age"},
				},
			},
			contains: "NOT BETWEEN",
		},
		{
			name: "is null",
			spec: AggregateSpec{
				Field: "id",
				Where: []ConditionSpec{
					{Field: "email", IsNull: true, Operator: "IS NULL"},
				},
			},
			contains: "IS NULL",
		},
		{
			name: "is not null",
			spec: AggregateSpec{
				Field: "id",
				Where: []ConditionSpec{
					{Field: "email", IsNull: true, Operator: "IS NOT NULL"},
				},
			},
			contains: "IS NOT NULL",
		},
		{
			name: "field comparison",
			spec: AggregateSpec{
				Field: "id",
				Where: []ConditionSpec{
					{Field: "id", Operator: "<", RightField: "age"},
				},
			},
			contains: `"age"`,
		},
		{
			name: "condition group or",
			spec: AggregateSpec{
				Field: "id",
				Where: []ConditionSpec{
					{
						Logic: "OR",
						Group: []ConditionSpec{
							{Field: "name", Operator: "=", Param: "name1"},
							{Field: "name", Operator: "=", Param: "name2"},
						},
					},
				},
			},
			contains: "OR",
		},
		{
			name: "condition group and",
			spec: AggregateSpec{
				Field: "id",
				Where: []ConditionSpec{
					{
						Logic: "AND",
						Group: []ConditionSpec{
							{Field: "name", Operator: "=", Param: "name1"},
							{Field: "age", Operator: ">", Param: "min_age"},
						},
					},
				},
			},
			contains: "AND",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := factory.countFromSpec(tt.spec)
			result, err := builder.Render()
			if err != nil {
				t.Fatalf("Render() failed: %v", err)
			}
			if !strings.Contains(result.SQL, tt.contains) {
				t.Errorf("SQL should contain %q: %s", tt.contains, result.SQL)
			}
		})
	}
}

func TestSelectExpressionsForSelect(t *testing.T) {
	factory, err := New[User](nil, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	tests := []struct {
		name     string
		expr     SelectExprSpec
		contains string
	}{
		// String functions
		{
			name:     "upper",
			expr:     SelectExprSpec{Func: "upper", Field: "name", Alias: "upper_name"},
			contains: "UPPER",
		},
		{
			name:     "lower",
			expr:     SelectExprSpec{Func: "lower", Field: "name", Alias: "lower_name"},
			contains: "LOWER",
		},
		{
			name:     "length",
			expr:     SelectExprSpec{Func: "length", Field: "name", Alias: "name_len"},
			contains: "LENGTH",
		},
		{
			name:     "trim",
			expr:     SelectExprSpec{Func: "trim", Field: "name", Alias: "trimmed"},
			contains: "TRIM",
		},
		{
			name:     "ltrim",
			expr:     SelectExprSpec{Func: "ltrim", Field: "name", Alias: "ltrimmed"},
			contains: "LTRIM",
		},
		{
			name:     "rtrim",
			expr:     SelectExprSpec{Func: "rtrim", Field: "name", Alias: "rtrimmed"},
			contains: "RTRIM",
		},
		{
			name:     "substring",
			expr:     SelectExprSpec{Func: "substring", Field: "name", Params: []string{"start", "length"}, Alias: "sub"},
			contains: "SUBSTRING",
		},
		{
			name:     "replace",
			expr:     SelectExprSpec{Func: "replace", Field: "name", Params: []string{"old", "new"}, Alias: "replaced"},
			contains: "REPLACE",
		},
		{
			name:     "concat",
			expr:     SelectExprSpec{Func: "concat", Fields: []string{"name", "email"}, Alias: "combined"},
			contains: "CONCAT",
		},
		// Math functions
		{
			name:     "abs",
			expr:     SelectExprSpec{Func: "abs", Field: "age", Alias: "abs_age"},
			contains: "ABS",
		},
		{
			name:     "ceil",
			expr:     SelectExprSpec{Func: "ceil", Field: "age", Alias: "ceil_age"},
			contains: "CEIL",
		},
		{
			name:     "floor",
			expr:     SelectExprSpec{Func: "floor", Field: "age", Alias: "floor_age"},
			contains: "FLOOR",
		},
		{
			name:     "round",
			expr:     SelectExprSpec{Func: "round", Field: "age", Alias: "round_age"},
			contains: "ROUND",
		},
		{
			name:     "sqrt",
			expr:     SelectExprSpec{Func: "sqrt", Field: "age", Alias: "sqrt_age"},
			contains: "SQRT",
		},
		{
			name:     "power",
			expr:     SelectExprSpec{Func: "power", Field: "age", Params: []string{"exp"}, Alias: "powered"},
			contains: "POWER",
		},
		// Date/Time functions
		{
			name:     "now",
			expr:     SelectExprSpec{Func: "now", Alias: "current_ts"},
			contains: "NOW",
		},
		{
			name:     "current_date",
			expr:     SelectExprSpec{Func: "current_date", Alias: "today"},
			contains: "CURRENT_DATE",
		},
		{
			name:     "current_time",
			expr:     SelectExprSpec{Func: "current_time", Alias: "now_time"},
			contains: "CURRENT_TIME",
		},
		{
			name:     "current_timestamp",
			expr:     SelectExprSpec{Func: "current_timestamp", Alias: "now_ts"},
			contains: "CURRENT_TIMESTAMP",
		},
		// Type casting
		{
			name:     "cast",
			expr:     SelectExprSpec{Func: "cast", Field: "age", CastType: "text", Alias: "age_text"},
			contains: "CAST",
		},
		// Aggregate functions
		{
			name:     "count_star",
			expr:     SelectExprSpec{Func: "count_star", Alias: "total"},
			contains: "COUNT(*)",
		},
		{
			name:     "count",
			expr:     SelectExprSpec{Func: "count", Field: "id", Alias: "id_count"},
			contains: "COUNT",
		},
		{
			name:     "count_distinct",
			expr:     SelectExprSpec{Func: "count_distinct", Field: "email", Alias: "unique_emails"},
			contains: "COUNT(DISTINCT",
		},
		{
			name:     "sum",
			expr:     SelectExprSpec{Func: "sum", Field: "age", Alias: "total_age"},
			contains: "SUM",
		},
		{
			name:     "avg",
			expr:     SelectExprSpec{Func: "avg", Field: "age", Alias: "avg_age"},
			contains: "AVG",
		},
		{
			name:     "min",
			expr:     SelectExprSpec{Func: "min", Field: "age", Alias: "min_age"},
			contains: "MIN",
		},
		{
			name:     "max",
			expr:     SelectExprSpec{Func: "max", Field: "age", Alias: "max_age"},
			contains: "MAX",
		},
		// Conditional functions
		{
			name:     "coalesce",
			expr:     SelectExprSpec{Func: "coalesce", Params: []string{"name", "default_name"}, Alias: "result"},
			contains: "COALESCE",
		},
		{
			name:     "nullif",
			expr:     SelectExprSpec{Func: "nullif", Params: []string{"age", "compare_val"}, Alias: "nullif_age"},
			contains: "NULLIF",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := SelectSpec{
				SelectExprs: []SelectExprSpec{tt.expr},
				Where:       []ConditionSpec{{Field: "id", Operator: "=", Param: "id"}},
			}
			builder, err := factory.selectFromSpec(spec)
			if err != nil {
				t.Fatalf("selectFromSpec() failed: %v", err)
			}
			result, err := builder.Render()
			if err != nil {
				t.Fatalf("Render() failed: %v", err)
			}
			if !strings.Contains(strings.ToUpper(result.SQL), tt.contains) {
				t.Errorf("SQL should contain %q: %s", tt.contains, result.SQL)
			}
		})
	}
}

func TestSelectExpressionsWithFilters(t *testing.T) {
	factory, err := New[User](nil, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	filter := &ConditionSpec{
		Field:    "name",
		Operator: "=",
		Param:    "filter_name",
	}

	tests := []struct {
		name     string
		expr     SelectExprSpec
		contains string
	}{
		{
			name:     "count_with_filter",
			expr:     SelectExprSpec{Func: "count", Field: "id", Filter: filter, Alias: "active_count"},
			contains: "FILTER",
		},
		{
			name:     "count_distinct_with_filter",
			expr:     SelectExprSpec{Func: "count_distinct", Field: "email", Filter: filter, Alias: "active_unique"},
			contains: "FILTER",
		},
		{
			name:     "sum_with_filter",
			expr:     SelectExprSpec{Func: "sum", Field: "age", Filter: filter, Alias: "active_sum"},
			contains: "FILTER",
		},
		{
			name:     "avg_with_filter",
			expr:     SelectExprSpec{Func: "avg", Field: "age", Filter: filter, Alias: "active_avg"},
			contains: "FILTER",
		},
		{
			name:     "min_with_filter",
			expr:     SelectExprSpec{Func: "min", Field: "age", Filter: filter, Alias: "active_min"},
			contains: "FILTER",
		},
		{
			name:     "max_with_filter",
			expr:     SelectExprSpec{Func: "max", Field: "age", Filter: filter, Alias: "active_max"},
			contains: "FILTER",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := SelectSpec{
				SelectExprs: []SelectExprSpec{tt.expr},
				Where:       []ConditionSpec{{Field: "id", Operator: ">", Param: "min_id"}},
			}
			builder, err := factory.selectFromSpec(spec)
			if err != nil {
				t.Fatalf("selectFromSpec() failed: %v", err)
			}
			result, err := builder.Render()
			if err != nil {
				t.Fatalf("Render() failed: %v", err)
			}
			if !strings.Contains(strings.ToUpper(result.SQL), tt.contains) {
				t.Errorf("SQL should contain %q: %s", tt.contains, result.SQL)
			}
		})
	}
}

func TestSelectForLocking(t *testing.T) {
	factory, err := New[User](nil, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	tests := []struct {
		name       string
		forLocking string
		contains   string
		shouldErr  bool
	}{
		{
			name:       "for_update",
			forLocking: "update",
			contains:   "FOR UPDATE",
		},
		{
			name:       "for_share",
			forLocking: "share",
			contains:   "FOR SHARE",
		},
		{
			name:       "for_no_key_update",
			forLocking: "no_key_update",
			contains:   "FOR NO KEY UPDATE",
		},
		{
			name:       "for_key_share",
			forLocking: "key_share",
			contains:   "FOR KEY SHARE",
		},
		{
			name:       "invalid_lock",
			forLocking: "invalid_mode",
			shouldErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := SelectSpec{
				Where:      []ConditionSpec{{Field: "id", Operator: "=", Param: "id"}},
				ForLocking: tt.forLocking,
			}
			builder, err := factory.selectFromSpec(spec)
			if tt.shouldErr {
				if err == nil {
					t.Errorf("expected error for invalid lock mode, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("selectFromSpec() failed: %v", err)
			}
			result, err := builder.Render()
			if err != nil {
				t.Fatalf("Render() failed: %v", err)
			}
			if !strings.Contains(strings.ToUpper(result.SQL), tt.contains) {
				t.Errorf("SQL should contain %q: %s", tt.contains, result.SQL)
			}
		})
	}
}

func TestSelectSpecFeatures(t *testing.T) {
	factory, err := New[User](nil, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	tests := []struct {
		name     string
		spec     SelectSpec
		contains []string
	}{
		{
			name: "group_by",
			spec: SelectSpec{
				Fields:  []string{"name"},
				GroupBy: []string{"name"},
				Where:   []ConditionSpec{{Field: "id", Operator: ">", Param: "min_id"}},
			},
			contains: []string{"GROUP BY"},
		},
		{
			name: "having",
			spec: SelectSpec{
				Fields:  []string{"name"},
				GroupBy: []string{"name"},
				Having:  []ConditionSpec{{Field: "name", Operator: "!=", Param: "excluded"}},
				Where:   []ConditionSpec{{Field: "id", Operator: ">", Param: "min_id"}},
			},
			contains: []string{"HAVING"},
		},
		{
			name: "having_agg",
			spec: SelectSpec{
				Fields:    []string{"name"},
				GroupBy:   []string{"name"},
				HavingAgg: []HavingAggSpec{{Func: "count", Field: "id", Operator: ">", Param: "min_count"}},
				Where:     []ConditionSpec{{Field: "id", Operator: ">", Param: "min_id"}},
			},
			contains: []string{"HAVING", "COUNT"},
		},
		{
			name: "distinct",
			spec: SelectSpec{
				Fields:   []string{"name"},
				Distinct: true,
				Where:    []ConditionSpec{{Field: "id", Operator: ">", Param: "min_id"}},
			},
			contains: []string{"DISTINCT"},
		},
		{
			name: "distinct_on",
			spec: SelectSpec{
				Fields:     []string{"name", "email"},
				DistinctOn: []string{"name"},
				Where:      []ConditionSpec{{Field: "id", Operator: ">", Param: "min_id"}},
			},
			contains: []string{"DISTINCT ON"},
		},
		{
			name: "limit_param",
			spec: SelectSpec{
				Fields:     []string{"name"},
				LimitParam: "page_size",
				Where:      []ConditionSpec{{Field: "id", Operator: ">", Param: "min_id"}},
			},
			contains: []string{"LIMIT"},
		},
		{
			name: "offset_param",
			spec: SelectSpec{
				Fields:      []string{"name"},
				OffsetParam: "page_offset",
				Where:       []ConditionSpec{{Field: "id", Operator: ">", Param: "min_id"}},
			},
			contains: []string{"OFFSET"},
		},
		{
			name: "order_by_nulls",
			spec: SelectSpec{
				Fields:  []string{"name"},
				OrderBy: []OrderBySpec{{Field: "age", Direction: "asc", Nulls: "first"}},
				Where:   []ConditionSpec{{Field: "id", Operator: ">", Param: "min_id"}},
			},
			contains: []string{"ORDER BY", "NULLS FIRST"},
		},
		{
			name: "order_by_expression",
			spec: SelectSpec{
				Fields:  []string{"name"},
				OrderBy: []OrderBySpec{{Field: "name", Operator: "<->", Param: "query_vec", Direction: "asc"}},
				Where:   []ConditionSpec{{Field: "id", Operator: ">", Param: "min_id"}},
			},
			contains: []string{"ORDER BY", "<->"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder, err := factory.selectFromSpec(tt.spec)
			if err != nil {
				t.Fatalf("selectFromSpec() failed: %v", err)
			}
			result, err := builder.Render()
			if err != nil {
				t.Fatalf("Render() failed: %v", err)
			}
			sql := strings.ToUpper(result.SQL)
			for _, contain := range tt.contains {
				if !strings.Contains(sql, contain) {
					t.Errorf("SQL should contain %q: %s", contain, result.SQL)
				}
			}
		})
	}
}

func TestConditionSpecHelpers(t *testing.T) {
	tests := []struct {
		name           string
		spec           ConditionSpec
		isBetween      bool
		isNotBetween   bool
		isFieldCompare bool
		isGroup        bool
	}{
		{
			name:      "between",
			spec:      ConditionSpec{Field: "age", Between: true, LowParam: "min", HighParam: "max"},
			isBetween: true,
		},
		{
			name:         "not between",
			spec:         ConditionSpec{Field: "age", NotBetween: true, LowParam: "min", HighParam: "max"},
			isNotBetween: true,
		},
		{
			name:           "field comparison",
			spec:           ConditionSpec{Field: "created_at", Operator: "<", RightField: "updated_at"},
			isFieldCompare: true,
		},
		{
			name:    "group",
			spec:    ConditionSpec{Logic: "OR", Group: []ConditionSpec{{Field: "x", Operator: "=", Param: "y"}}},
			isGroup: true,
		},
		{
			name: "simple condition",
			spec: ConditionSpec{Field: "age", Operator: "=", Param: "val"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.spec.IsBetween(); got != tt.isBetween {
				t.Errorf("IsBetween() = %v, want %v", got, tt.isBetween)
			}
			if got := tt.spec.IsNotBetween(); got != tt.isNotBetween {
				t.Errorf("IsNotBetween() = %v, want %v", got, tt.isNotBetween)
			}
			if got := tt.spec.IsFieldComparison(); got != tt.isFieldCompare {
				t.Errorf("IsFieldComparison() = %v, want %v", got, tt.isFieldCompare)
			}
			if got := tt.spec.IsGroup(); got != tt.isGroup {
				t.Errorf("IsGroup() = %v, want %v", got, tt.isGroup)
			}
		})
	}
}

func TestSelectExprSpecErrors(t *testing.T) {
	factory, err := New[User](nil, "users", postgres.New())
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	tests := []struct {
		name        string
		expr        SelectExprSpec
		errContains string
	}{
		{
			name:        "unknown_function",
			expr:        SelectExprSpec{Func: "unknown_func", Field: "name", Alias: "result"},
			errContains: "unknown select expression function",
		},
		{
			name:        "substring_missing_params",
			expr:        SelectExprSpec{Func: "substring", Field: "name", Params: []string{"1"}, Alias: "sub"},
			errContains: "substring requires 2 params",
		},
		{
			name:        "replace_missing_params",
			expr:        SelectExprSpec{Func: "replace", Field: "name", Params: []string{"old"}, Alias: "replaced"},
			errContains: "replace requires 2 params",
		},
		{
			name:        "power_missing_params",
			expr:        SelectExprSpec{Func: "power", Field: "val", Alias: "powered"},
			errContains: "power requires 1 param",
		},
		{
			name:        "nullif_missing_params",
			expr:        SelectExprSpec{Func: "nullif", Params: []string{"one"}, Alias: "result"},
			errContains: "nullif requires 2 params",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+"_query", func(t *testing.T) {
			spec := QuerySpec{
				SelectExprs: []SelectExprSpec{tt.expr},
			}
			_, err := factory.queryFromSpec(spec)
			if err == nil {
				t.Errorf("expected error containing %q, got nil", tt.errContains)
				return
			}
			if !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("error should contain %q, got: %v", tt.errContains, err)
			}
		})

		t.Run(tt.name+"_select", func(t *testing.T) {
			spec := SelectSpec{
				SelectExprs: []SelectExprSpec{tt.expr},
			}
			_, err := factory.selectFromSpec(spec)
			if err == nil {
				t.Errorf("expected error containing %q, got nil", tt.errContains)
				return
			}
			if !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("error should contain %q, got: %v", tt.errContains, err)
			}
		})
	}
}
