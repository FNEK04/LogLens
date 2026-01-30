package query

import (
	"regexp"
	"strings"

	"LogLens/internal/domain"
)

type FilterEngine struct{}

func NewFilterEngine() *FilterEngine {
	return &FilterEngine{}
}

func (e *FilterEngine) BuildFilter(conditions []domain.FilterCondition) (domain.Filter, error) {
	if len(conditions) == 0 {
		return &NoOpFilter{}, nil
	}
	
	if len(conditions) == 1 {
		return e.buildSingleFilter(conditions[0])
	}
	
	filters := make([]domain.Filter, len(conditions))
	for i, condition := range conditions {
		filter, err := e.buildSingleFilter(condition)
		if err != nil {
			return nil, err
		}
		filters[i] = filter
	}
	
	return &AndFilter{filters: filters}, nil
}

func (e *FilterEngine) buildSingleFilter(condition domain.FilterCondition) (domain.Filter, error) {
	switch condition.Type {
	case domain.FilterEquality:
		return &EqualityFilter{
			field: condition.Field,
			value: condition.Value,
		}, nil
		
	case domain.FilterExclusion:
		return &ExclusionFilter{
			field: condition.Field,
			value: condition.Value,
		}, nil
		
	case domain.FilterContains:
		return &ContainsFilter{
			field:   condition.Field,
			value:   strings.ToLower(condition.Value.(string)),
			ignoreCase: true,
		}, nil
		
	case domain.FilterRegexp:
		regex, err := regexp.Compile(condition.Value.(string))
		if err != nil {
			return nil, err
		}
		return &RegexpFilter{
			field: condition.Field,
			regex: regex,
		}, nil
		
	case domain.FilterRange:
		return &RangeFilter{
			field:    condition.Field,
			operator: condition.Operator,
			value:    condition.Value,
		}, nil
		
	default:
		return &NoOpFilter{}, nil
	}
}

func (e *FilterEngine) ApplyFilter(filter domain.Filter, records <-chan domain.Record) <-chan domain.Record {
	filtered := make(chan domain.Record, 1000)
	
	go func() {
		defer close(filtered)
		for record := range records {
			if filter.Match(record) {
				filtered <- record
			}
		}
	}()
	
	return filtered
}

type NoOpFilter struct{}

func (f *NoOpFilter) Match(record domain.Record) bool {
	return true
}

type EqualityFilter struct {
	field string
	value interface{}
}

func (f *EqualityFilter) Match(record domain.Record) bool {
	recordValue := f.getFieldValue(record)
	return f.compareValues(recordValue, f.value) == 0
}

type ExclusionFilter struct {
	field string
	value interface{}
}

func (f *ExclusionFilter) Match(record domain.Record) bool {
	recordValue := f.getFieldValue(record)
	return f.compareValues(recordValue, f.value) != 0
}

type ContainsFilter struct {
	field       string
	value       string
	ignoreCase  bool
}

func (f *ContainsFilter) Match(record domain.Record) bool {
	recordValue := f.getFieldValue(record)
	if recordValue == nil {
		return false
	}
	
	recordStr := f.toString(recordValue)
	if f.ignoreCase {
		recordStr = strings.ToLower(recordStr)
	}
	
	return strings.Contains(recordStr, f.value)
}

type RegexpFilter struct {
	field string
	regex *regexp.Regexp
}

func (f *RegexpFilter) Match(record domain.Record) bool {
	recordValue := f.getFieldValue(record)
	if recordValue == nil {
		return false
	}
	
	recordStr := f.toString(recordValue)
	return f.regex.MatchString(recordStr)
}

type RangeFilter struct {
	field    string
	operator string
	value    interface{}
}

func (f *RangeFilter) Match(record domain.Record) bool {
	recordValue := f.getFieldValue(record)
	if recordValue == nil {
		return false
	}
	
	comparison := f.compareValues(recordValue, f.value)
	
	switch f.operator {
	case "gt":
		return comparison > 0
	case "gte":
		return comparison >= 0
	case "lt":
		return comparison < 0
	case "lte":
		return comparison <= 0
	default:
		return false
	}
}

type AndFilter struct {
	filters []domain.Filter
}

func (f *AndFilter) Match(record domain.Record) bool {
	for _, filter := range f.filters {
		if !filter.Match(record) {
			return false
		}
	}
	return true
}

type OrFilter struct {
	filters []domain.Filter
}

func (f *OrFilter) Match(record domain.Record) bool {
	for _, filter := range f.filters {
		if filter.Match(record) {
			return true
		}
	}
	return false
}

func (f *EqualityFilter) getFieldValue(record domain.Record) interface{} {
	switch f.field {
	case "id":
		return record.ID
	case "timestamp":
		return record.Timestamp
	case "level":
		return record.Level
	case "message":
		return record.Message
	case "service":
		return record.Service
	case "raw":
		return record.Raw
	default:
		if value, exists := record.Fields[f.field]; exists {
			return value
		}
		return nil
	}
}

func (f *ExclusionFilter) getFieldValue(record domain.Record) interface{} {
	e := &EqualityFilter{field: f.field}
	return e.getFieldValue(record)
}

func (f *ContainsFilter) getFieldValue(record domain.Record) interface{} {
	e := &EqualityFilter{field: f.field}
	return e.getFieldValue(record)
}

func (f *RegexpFilter) getFieldValue(record domain.Record) interface{} {
	e := &EqualityFilter{field: f.field}
	return e.getFieldValue(record)
}

func (f *RangeFilter) getFieldValue(record domain.Record) interface{} {
	e := &EqualityFilter{field: f.field}
	return e.getFieldValue(record)
}

func (f *EqualityFilter) compareValues(a, b interface{}) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}
	
	if aNum, aOk := f.toFloat64(a); aOk {
		if bNum, bOk := f.toFloat64(b); bOk {
			if aNum < bNum {
				return -1
			} else if aNum > bNum {
				return 1
			}
			return 0
		}
	}
	
	aStr := f.toString(a)
	bStr := f.toString(b)
	
	if aStr < bStr {
		return -1
	} else if aStr > bStr {
		return 1
	}
	return 0
}

func (f *ExclusionFilter) compareValues(a, b interface{}) int {
	e := &EqualityFilter{}
	return e.compareValues(a, b)
}

func (f *RangeFilter) compareValues(a, b interface{}) int {
	e := &EqualityFilter{}
	return e.compareValues(a, b)
}

func (f *EqualityFilter) toFloat64(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case int32:
		return float64(v), true
	default:
		return 0, false
	}
}

func (f *EqualityFilter) toString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	default:
		return ""
	}
}

func (f *ExclusionFilter) toFloat64(value interface{}) (float64, bool) {
	e := &EqualityFilter{}
	return e.toFloat64(value)
}

func (f *ContainsFilter) toString(value interface{}) string {
	e := &EqualityFilter{}
	return e.toString(value)
}

func (f *RegexpFilter) toString(value interface{}) string {
	e := &EqualityFilter{}
	return e.toString(value)
}

func (f *RangeFilter) toFloat64(value interface{}) (float64, bool) {
	e := &EqualityFilter{}
	return e.toFloat64(value)
}
