package query

import (
	"context"
	"fmt"
	"strings"

	"LogLens/internal/domain"
)

type QueryEngine struct {
	storage domain.Storage
}

func NewQueryEngine(storage domain.Storage) *QueryEngine {
	return &QueryEngine{
		storage: storage,
	}
}

func (e *QueryEngine) Execute(ctx context.Context, query domain.Query) (*domain.QueryResult, error) {
	if err := e.validateQuery(query); err != nil {
		return nil, fmt.Errorf("invalid query: %w", err)
	}

	result, err := e.storage.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	if len(query.Aggregations) > 0 {
		aggregations, err := e.storage.Aggregate(ctx, query.Filters, query.Aggregations)
		if err != nil {
			return nil, fmt.Errorf("failed to compute aggregations: %w", err)
		}
		result.Aggregations = aggregations
	}

	return result, nil
}

func (e *QueryEngine) Explain(query domain.Query) (string, error) {
	var explanation strings.Builder
	
	explanation.WriteString("Query Execution Plan:\n")
	explanation.WriteString("====================\n\n")
	
	if len(query.Filters) > 0 {
		explanation.WriteString("Filters:\n")
		for i, filter := range query.Filters {
			explanation.WriteString(fmt.Sprintf("  %d. %s %s %v\n", i+1, filter.Field, filter.Type, filter.Value))
		}
		explanation.WriteString("\n")
	}
	
	if query.SortBy != "" {
		direction := "ASC"
		if query.SortDesc {
			direction = "DESC"
		}
		explanation.WriteString(fmt.Sprintf("Sort: %s %s\n\n", query.SortBy, direction))
	}
	
	if query.Limit > 0 {
		explanation.WriteString(fmt.Sprintf("Limit: %d\n", query.Limit))
		if query.Offset > 0 {
			explanation.WriteString(fmt.Sprintf("Offset: %d\n", query.Offset))
		}
		explanation.WriteString("\n")
	}
	
	if len(query.Aggregations) > 0 {
		explanation.WriteString("Aggregations:\n")
		for i, agg := range query.Aggregations {
			field := "*"
			if agg.Field != "" {
				field = agg.Field
			}
			alias := agg.Function
			if agg.Alias != "" {
				alias = agg.Alias
			}
			explanation.WriteString(fmt.Sprintf("  %d. %s(%s) AS %s\n", i+1, agg.Function, field, alias))
		}
		explanation.WriteString("\n")
	}
	
	return explanation.String(), nil
}

func (e *QueryEngine) validateQuery(query domain.Query) error {
	for _, filter := range query.Filters {
		if filter.Field == "" {
			return fmt.Errorf("filter field cannot be empty")
		}
		
		switch filter.Type {
		case domain.FilterRange:
			if filter.Operator == "" {
				return fmt.Errorf("range filter requires operator")
			}
			validOperators := []string{"gt", "lt", "gte", "lte"}
			valid := false
			for _, op := range validOperators {
				if filter.Operator == op {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("invalid range operator: %s", filter.Operator)
			}
		}
	}
	
	for _, agg := range query.Aggregations {
		validFunctions := []string{"count", "avg", "sum", "min", "max"}
		valid := false
		for _, fn := range validFunctions {
			if agg.Function == fn {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid aggregation function: %s", agg.Function)
		}
	}
	
	if query.Limit < 0 {
		return fmt.Errorf("limit cannot be negative")
	}
	if query.Offset < 0 {
		return fmt.Errorf("offset cannot be negative")
	}
	
	return nil
}
