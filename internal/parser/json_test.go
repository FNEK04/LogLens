package parser

import (
	"context"
	"strings"
	"testing"

	"LogLens/internal/domain"
)

func TestJSONParser_Parse(t *testing.T) {
	input := `{"timestamp":"2024-01-15T10:30:00Z","level":"ERROR","message":"Connection refused","service":"api-gateway","host":"server-1"}
{"time":"2024-01-15T10:31:00Z","severity":"WARN","msg":"Retrying request","service":"api-gateway"}
{"@timestamp":"2024-01-15T10:32:00Z","level":"INFO","message":"Request completed","service":"worker","duration_ms":150}`

	parser := NewJSONParser(domain.ParserConfig{Type: domain.ParserJSON})
	records, err := parser.Parse(context.Background(), strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	var results []domain.LogRecord
	for r := range records {
		results = append(results, r)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 records, got %d", len(results))
	}

	if results[0].Level != "ERROR" {
		t.Errorf("expected level ERROR, got %s", results[0].Level)
	}
	if results[0].Service != "api-gateway" {
		t.Errorf("expected service api-gateway, got %s", results[0].Service)
	}
	if results[0].Message != "Connection refused" {
		t.Errorf("expected message 'Connection refused', got %s", results[0].Message)
	}
	if results[0].Timestamp == 0 {
		t.Error("expected non-zero timestamp")
	}

	if results[1].Level != "WARN" {
		t.Errorf("expected level WARN, got %s", results[1].Level)
	}
	if results[1].Message != "Retrying request" {
		t.Errorf("expected message 'Retrying request', got %s", results[1].Message)
	}

	if results[2].Fields["duration_ms"] != float64(150) {
		t.Errorf("expected duration_ms=150, got %v", results[2].Fields["duration_ms"])
	}
}

func TestJSONParser_EmptyLines(t *testing.T) {
	input := `{"level":"INFO","message":"line1"}

{"level":"ERROR","message":"line2"}`

	parser := NewJSONParser(domain.ParserConfig{Type: domain.ParserJSON})
	records, err := parser.Parse(context.Background(), strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	var results []domain.LogRecord
	for r := range records {
		results = append(results, r)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 records, got %d", len(results))
	}
}

func TestJSONParser_InvalidJSON(t *testing.T) {
	input := `not json at all
{"level":"INFO","message":"valid"}`

	parser := NewJSONParser(domain.ParserConfig{Type: domain.ParserJSON})
	records, err := parser.Parse(context.Background(), strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	var results []domain.LogRecord
	for r := range records {
		results = append(results, r)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 valid record, got %d", len(results))
	}
}

func TestJSONParser_GeneratedID(t *testing.T) {
	input := `{"level":"INFO","message":"no id field"}`

	parser := NewJSONParser(domain.ParserConfig{Type: domain.ParserJSON})
	records, err := parser.Parse(context.Background(), strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	var results []domain.LogRecord
	for r := range records {
		results = append(results, r)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 record, got %d", len(results))
	}

	if results[0].ID != "json_1" {
		t.Errorf("expected ID 'json_1', got %s", results[0].ID)
	}
}

func TestJSONParser_ExistingID(t *testing.T) {
	input := `{"id":"custom-123","level":"INFO","message":"custom id"}`

	parser := NewJSONParser(domain.ParserConfig{Type: domain.ParserJSON})
	records, err := parser.Parse(context.Background(), strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	var results []domain.LogRecord
	for r := range records {
		results = append(results, r)
	}

	if results[0].ID != "custom-123" {
		t.Errorf("expected ID 'custom-123', got %s", results[0].ID)
	}
}

func TestJSONParser_TimestampFormats(t *testing.T) {
	tests := []struct {
		name  string
		field string
		value string
	}{
		{"RFC3339", "timestamp", "2024-01-15T10:30:00Z"},
		{"RFC3339Nano", "timestamp", "2024-01-15T10:30:00.123456789Z"},
		{"space separated", "time", "2024-01-15 10:30:00"},
		{"at timestamp", "@timestamp", "2024-01-15T10:30:00Z"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := `{"` + tt.field + `":"` + tt.value + `","level":"INFO","message":"test"}`
			parser := NewJSONParser(domain.ParserConfig{Type: domain.ParserJSON})
			records, err := parser.Parse(context.Background(), strings.NewReader(input))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			var results []domain.LogRecord
			for r := range records {
				results = append(results, r)
			}

			if results[0].Timestamp == 0 {
				t.Errorf("expected non-zero timestamp for format %s", tt.name)
			}
		})
	}
}
