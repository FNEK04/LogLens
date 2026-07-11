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
		valueStr, _ := condition.Value.(string)
		return &ContainsFilter{
			field:      condition.Field,
			value:      strings.ToLower(valueStr),
			ignoreCase: true,
		}, nil
		
	case domain.FilterRegexp:
		regexStr, _ := condition.Value.(string)
		regex, err := regexp.Compile(regexStr)
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

func (e *FilterEngine) ApplyFilter(filter domain.Filter, records <-chan domain.LogRecord) <-chan domain.LogRecord {
	filtered := make(chan domain.LogRecord, 1000)
	
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

func getFieldValue(field string, record domain.LogRecord) interface{} {
	switch field {
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
		if value, exists := record.Fields[field]; exists {
			return value
		}
		return nil
	}
}

func compareValues(a, b interface{}) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}
	
	if aNum, aOk := toFloat64(a); aOk {
		if bNum, bOk := toFloat64(b); bOk {
			if aNum < bNum {
				return -1
			} else if aNum > bNum {
				return 1
			}
			return 0
		}
	}
	
	aStr := toString(a)
	bStr := toString(b)
	
	if aStr < bStr {
		return -1
	} else if aStr > bStr {
		return 1
	}
	return 0
}

func toFloat64(value interface{}) (float64, bool) {
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

func toString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	default:
		return ""
	}
}

type NoOpFilter struct{}

func (f *NoOpFilter) Match(record domain.LogRecord) bool {
	return true
}

type EqualityFilter struct {
	field string
	value interface{}
}

func (f *EqualityFilter) Match(record domain.LogRecord) bool {
	recordValue := getFieldValue(f.field, record)
	return compareValues(recordValue, f.value) == 0
}

type ExclusionFilter struct {
	field string
	value interface{}
}

func (f *ExclusionFilter) Match(record domain.LogRecord) bool {
	recordValue := getFieldValue(f.field, record)
	return compareValues(recordValue, f.value) != 0
}

type ContainsFilter struct {
	field      string
	value      string
	ignoreCase bool
}

func (f *ContainsFilter) Match(record domain.LogRecord) bool {
	recordValue := getFieldValue(f.field, record)
	if recordValue == nil {
		return false
	}
	
	recordStr := toString(recordValue)
	if f.ignoreCase {
		recordStr = strings.ToLower(recordStr)
	}
	
	return strings.Contains(recordStr, f.value)
}

type RegexpFilter struct {
	field string
	regex *regexp.Regexp
}

func (f *RegexpFilter) Match(record domain.LogRecord) bool {
	recordValue := getFieldValue(f.field, record)
	if recordValue == nil {
		return false
	}
	
	recordStr := toString(recordValue)
	return f.regex.MatchString(recordStr)
}

type RangeFilter struct {
	field    string
	operator string
	value    interface{}
}

func (f *RangeFilter) Match(record domain.LogRecord) bool {
	recordValue := getFieldValue(f.field, record)
	if recordValue == nil {
		return false
	}
	
	comparison := compareValues(recordValue, f.value)
	
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

func (f *AndFilter) Match(record domain.LogRecord) bool {
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

func (f *OrFilter) Match(record domain.LogRecord) bool {
	for _, filter := range f.filters {
		if filter.Match(record) {
			return true
		}
	}
	return false
}
