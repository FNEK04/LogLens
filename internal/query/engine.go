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
	
	var regexpFilters []domain.FilterCondition
	for _, f := range query.Filters {
		if f.Type == domain.FilterRegexp {
			regexpFilters = append(regexpFilters, f)
		}
	}
	
	if len(regexpFilters) > 0 {
		filterEngine := NewFilterEngine()
		filter, err := filterEngine.BuildFilter(regexpFilters)
		if err == nil {
			inCh := make(chan domain.LogRecord, len(result.Records))
			for _, r := range result.Records {
				inCh <- r
			}
			close(inCh)
			outCh := filterEngine.ApplyFilter(filter, inCh)
			filtered := make([]domain.LogRecord, 0)
			for r := range outCh {
				filtered = append(filtered, r)
			}
			result.Records = filtered
			result.Total = int64(len(filtered))
		}
	}
	
	if len(query.Aggregations) > 0 {
		aggregations := e.computeAggregations(query.Aggregations, result.Records)
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

func getRecordFieldValue(field string, record domain.LogRecord) interface{} {
	switch field {
	case "level":
		return record.Level
	case "service":
		return record.Service
	case "message":
		return record.Message
	default:
		if fieldValue, exists := record.Fields[field]; exists {
			return fieldValue
		}
		return nil
	}
}

func (e *QueryEngine) computeAggregations(aggregations []domain.Aggregation, records []domain.LogRecord) map[string]interface{} {
	results := make(map[string]interface{})
	
	for _, agg := range aggregations {
		alias := agg.Alias
		if alias == "" {
			alias = agg.Function
			if agg.Field != "" {
				alias += "_" + agg.Field
			}
		}
		
		switch agg.Function {
		case "count":
			if agg.Field == "" || agg.Field == "*" {
				results[alias] = len(records)
			} else {
				results[alias] = e.countDistinct(records, agg.Field)
			}
		case "avg":
			results[alias] = e.computeAverage(records, agg.Field)
		case "sum":
			results[alias] = e.computeSum(records, agg.Field)
		case "min":
			results[alias] = e.computeMin(records, agg.Field)
		case "max":
			results[alias] = e.computeMax(records, agg.Field)
		}
	}
	
	return results
}

func (e *QueryEngine) countDistinct(records []domain.LogRecord, field string) int {
	seen := make(map[interface{}]bool)
	
	for _, record := range records {
		if value := getRecordFieldValue(field, record); value != nil {
			seen[value] = true
		}
	}
	
	return len(seen)
}

func (e *QueryEngine) computeAverage(records []domain.LogRecord, field string) float64 {
	var sum float64
	var count int
	
	for _, record := range records {
		if numValue, ok := toFloat64(getRecordFieldValue(field, record)); ok {
			sum += numValue
			count++
		}
	}
	
	if count == 0 {
		return 0
	}
	
	return sum / float64(count)
}

func (e *QueryEngine) computeSum(records []domain.LogRecord, field string) float64 {
	var sum float64
	
	for _, record := range records {
		if numValue, ok := toFloat64(getRecordFieldValue(field, record)); ok {
			sum += numValue
		}
	}
	
	return sum
}

func (e *QueryEngine) computeMin(records []domain.LogRecord, field string) interface{} {
	var min interface{}
	
	for i, record := range records {
		value := getRecordFieldValue(field, record)
		if value != nil {
			if i == 0 || compareValues(value, min) < 0 {
				min = value
			}
		}
	}
	
	return min
}

func (e *QueryEngine) computeMax(records []domain.LogRecord, field string) interface{} {
	var max interface{}
	
	for i, record := range records {
		value := getRecordFieldValue(field, record)
		if value != nil {
			if i == 0 || compareValues(value, max) > 0 {
				max = value
			}
		}
	}
	
	return max
}
