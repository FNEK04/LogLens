package query

import (
	"testing"

	"LogLens/internal/domain"
)

func TestFilterEngine_EqualityFilter(t *testing.T) {
	engine := NewFilterEngine()
	conditions := []domain.FilterCondition{
		{Type: domain.FilterEquality, Field: "level", Value: "ERROR"},
	}
	filter, err := engine.BuildFilter(conditions)
	if err != nil {
		t.Fatalf("BuildFilter failed: %v", err)
	}

	record := domain.LogRecord{Level: "ERROR", Fields: make(map[string]interface{})}
	if !filter.Match(record) {
		t.Error("expected match for ERROR level")
	}

	record2 := domain.LogRecord{Level: "INFO", Fields: make(map[string]interface{})}
	if filter.Match(record2) {
		t.Error("expected no match for INFO level")
	}
}

func TestFilterEngine_ContainsFilter(t *testing.T) {
	engine := NewFilterEngine()
	conditions := []domain.FilterCondition{
		{Type: domain.FilterContains, Field: "message", Value: "connection"},
	}
	filter, err := engine.BuildFilter(conditions)
	if err != nil {
		t.Fatalf("BuildFilter failed: %v", err)
	}

	record := domain.LogRecord{Message: "Connection refused", Fields: make(map[string]interface{})}
	if !filter.Match(record) {
		t.Error("expected match for 'connection' in message")
	}

	record2 := domain.LogRecord{Message: "Request completed", Fields: make(map[string]interface{})}
	if filter.Match(record2) {
		t.Error("expected no match for 'Request completed'")
	}
}

func TestFilterEngine_RegexpFilter(t *testing.T) {
	engine := NewFilterEngine()
	conditions := []domain.FilterCondition{
		{Type: domain.FilterRegexp, Field: "message", Value: `timeout after \d+ms`},
	}
	filter, err := engine.BuildFilter(conditions)
	if err != nil {
		t.Fatalf("BuildFilter failed: %v", err)
	}

	record := domain.LogRecord{Message: "Request timeout after 3000ms", Fields: make(map[string]interface{})}
	if !filter.Match(record) {
		t.Error("expected match for regex pattern")
	}

	record2 := domain.LogRecord{Message: "Request completed", Fields: make(map[string]interface{})}
	if filter.Match(record2) {
		t.Error("expected no match for non-matching message")
	}
}

func TestFilterEngine_InvalidRegexp(t *testing.T) {
	engine := NewFilterEngine()
	conditions := []domain.FilterCondition{
		{Type: domain.FilterRegexp, Field: "message", Value: `[invalid`},
	}
	_, err := engine.BuildFilter(conditions)
	if err == nil {
		t.Error("expected error for invalid regex")
	}
}

func TestFilterEngine_RangeFilter(t *testing.T) {
	engine := NewFilterEngine()
	conditions := []domain.FilterCondition{
		{Type: domain.FilterRange, Field: "timestamp", Operator: "gt", Value: int64(1000)},
	}
	filter, err := engine.BuildFilter(conditions)
	if err != nil {
		t.Fatalf("BuildFilter failed: %v", err)
	}

	record := domain.LogRecord{Timestamp: 2000, Fields: make(map[string]interface{})}
	if !filter.Match(record) {
		t.Error("expected match for timestamp > 1000")
	}

	record2 := domain.LogRecord{Timestamp: 500, Fields: make(map[string]interface{})}
	if filter.Match(record2) {
		t.Error("expected no match for timestamp 500")
	}
}

func TestFilterEngine_MultipleFilters(t *testing.T) {
	engine := NewFilterEngine()
	conditions := []domain.FilterCondition{
		{Type: domain.FilterEquality, Field: "level", Value: "ERROR"},
		{Type: domain.FilterContains, Field: "message", Value: "timeout"},
	}
	filter, err := engine.BuildFilter(conditions)
	if err != nil {
		t.Fatalf("BuildFilter failed: %v", err)
	}

	record1 := domain.LogRecord{Level: "ERROR", Message: "Connection timeout", Fields: make(map[string]interface{})}
	if !filter.Match(record1) {
		t.Error("expected match for ERROR + timeout")
	}

	record2 := domain.LogRecord{Level: "ERROR", Message: "Connection refused", Fields: make(map[string]interface{})}
	if filter.Match(record2) {
		t.Error("expected no match for ERROR + refused")
	}

	record3 := domain.LogRecord{Level: "INFO", Message: "Connection timeout", Fields: make(map[string]interface{})}
	if filter.Match(record3) {
		t.Error("expected no match for INFO + timeout")
	}
}

func TestFilterEngine_NoFilters(t *testing.T) {
	engine := NewFilterEngine()
	filter, err := engine.BuildFilter([]domain.FilterCondition{})
	if err != nil {
		t.Fatalf("BuildFilter failed: %v", err)
	}

	record := domain.LogRecord{Level: "INFO", Fields: make(map[string]interface{})}
	if !filter.Match(record) {
		t.Error("NoOpFilter should match everything")
	}
}

func TestFilterEngine_ExclusionFilter(t *testing.T) {
	engine := NewFilterEngine()
	conditions := []domain.FilterCondition{
		{Type: domain.FilterExclusion, Field: "level", Value: "DEBUG"},
	}
	filter, err := engine.BuildFilter(conditions)
	if err != nil {
		t.Fatalf("BuildFilter failed: %v", err)
	}

	record := domain.LogRecord{Level: "ERROR", Fields: make(map[string]interface{})}
	if !filter.Match(record) {
		t.Error("expected match for non-DEBUG level")
	}

	record2 := domain.LogRecord{Level: "DEBUG", Fields: make(map[string]interface{})}
	if filter.Match(record2) {
		t.Error("expected no match for DEBUG level")
	}
}

func TestCompareValues(t *testing.T) {
	tests := []struct {
		name     string
		a, b     interface{}
		expected int
	}{
		{"equal ints", int64(42), int64(42), 0},
		{"a < b", int64(1), int64(2), -1},
		{"a > b", int64(2), int64(1), 1},
		{"equal floats", 3.14, 3.14, 0},
		{"nil vs value", nil, "x", -1},
		{"value vs nil", "x", nil, 1},
		{"both nil", nil, nil, 0},
		{"string comparison", "abc", "def", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compareValues(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("compareValues(%v, %v) = %d, want %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}
