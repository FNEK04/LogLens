package parser

import (
	"context"
	"strings"
	"testing"

	"LogLens/internal/domain"
)

func TestPlainParser_Parse(t *testing.T) {
	input := `2024-01-15 10:30:45 [ERROR] [api-gateway] Connection refused to upstream
2024-01-15 10:30:46 [WARN] [api-gateway] Retrying request attempt 2
2024-01-15 10:30:47 [INFO] [worker] Request completed successfully
Some random line without timestamp
2024-01-15 10:30:48 [DEBUG] [db] Query executed in 50ms`

	parser := NewPlainParser(domain.ParserConfig{Type: domain.ParserPlain})
	records, err := parser.Parse(context.Background(), strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	var results []domain.LogRecord
	for r := range records {
		results = append(results, r)
	}

	if len(results) != 5 {
		t.Fatalf("expected 5 records, got %d", len(results))
	}

	if results[0].Level != "ERROR" {
		t.Errorf("expected level ERROR, got %s", results[0].Level)
	}
	if results[0].Service != "api-gateway" {
		t.Errorf("expected service api-gateway, got %s", results[0].Service)
	}
	if results[0].Timestamp == 0 {
		t.Error("expected non-zero timestamp")
	}

	if results[3].Level != "INFO" {
		t.Errorf("expected level INFO for random line, got %s", results[3].Level)
	}
}

func TestPlainParser_TimestampFormats(t *testing.T) {
	input := `2024-01-15 10:30:45 [INFO] test
2024/01/15 10:30:45 [INFO] test
12/25/2024 10:30:45 [INFO] test
Jan 15 10:30:45 [INFO] test`

	parser := NewPlainParser(domain.ParserConfig{Type: domain.ParserPlain})
	records, err := parser.Parse(context.Background(), strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	var results []domain.LogRecord
	for r := range records {
		results = append(results, r)
	}

	if len(results) != 4 {
		t.Fatalf("expected 4 records, got %d", len(results))
	}

	for i, r := range results {
		if r.Timestamp == 0 {
			t.Errorf("record %d: expected non-zero timestamp", i)
		}
	}
}

func TestPlainParser_Levels(t *testing.T) {
	input := `2024-01-15 10:30:45 [TRACE] trace message
2024-01-15 10:30:45 [DEBUG] debug message
2024-01-15 10:30:45 [INFO] info message
2024-01-15 10:30:45 [WARN] warn message
2024-01-15 10:30:45 [ERROR] error message
2024-01-15 10:30:45 [FATAL] fatal message
2024-01-15 10:30:45 [PANIC] panic message`

	parser := NewPlainParser(domain.ParserConfig{Type: domain.ParserPlain})
	records, err := parser.Parse(context.Background(), strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	expected := []string{"TRACE", "DEBUG", "INFO", "WARN", "ERROR", "FATAL", "PANIC"}
	var results []domain.LogRecord
	for r := range records {
		results = append(results, r)
	}

	if len(results) != len(expected) {
		t.Fatalf("expected %d records, got %d", len(expected), len(results))
	}

	for i, exp := range expected {
		if results[i].Level != exp {
			t.Errorf("record %d: expected level %s, got %s", i, exp, results[i].Level)
		}
	}
}

func TestPlainParser_EmptyLines(t *testing.T) {
	input := `2024-01-15 10:30:45 [INFO] first

2024-01-15 10:30:46 [INFO] second

`

	parser := NewPlainParser(domain.ParserConfig{Type: domain.ParserPlain})
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

func TestPlainParser_ContextCancel(t *testing.T) {
	input := "2024-01-15 10:30:45 [INFO] line1\n2024-01-15 10:30:46 [INFO] line2\n2024-01-15 10:30:47 [INFO] line3\n"

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	parser := NewPlainParser(domain.ParserConfig{Type: domain.ParserPlain})
	records, err := parser.Parse(ctx, strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	var count int
	for range records {
		count++
	}

	if count != 0 {
		t.Errorf("expected 0 records after cancel, got %d", count)
	}
}
