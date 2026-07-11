package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"LogLens/internal/domain"
)

func newTestLogLens(t *testing.T) (*LogLens, func()) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	ll, err := NewLogLens(Config{DatabasePath: dbPath})
	if err != nil {
		t.Fatalf("failed to create LogLens: %v", err)
	}

	cleanup := func() {
		ll.Close()
		os.Remove(dbPath)
		os.Remove(dbPath + "-wal")
		os.Remove(dbPath + "-shm")
	}

	return ll, cleanup
}

func importPlainWithPrefix(t *testing.T, ll *LogLens, content string, prefix string) *domain.ImportResult {
	t.Helper()
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.log")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	config := domain.ParserConfig{Type: domain.ParserPlain, IDPrefix: prefix}
	result, err := ll.ImportFile(context.Background(), filePath, config, &noopReporter{})
	if err != nil {
		t.Fatalf("ImportFile failed: %v", err)
	}
	return result
}

func importPlain(t *testing.T, ll *LogLens, content string) *domain.ImportResult {
	return importPlainWithPrefix(t, ll, content, "test")
}

type noopReporter struct{}

func (r *noopReporter) ReportProgress(current, total int64, message string) {}
func (r *noopReporter) ReportError(err error)                              {}

func TestGetStats_SQLAggregation(t *testing.T) {
	ll, cleanup := newTestLogLens(t)
	defer cleanup()

	content := `2024-01-15 10:30:45 [ERROR] [api-gateway] Connection refused
2024-01-15 10:30:46 [INFO] [api-gateway] Request ok
2024-01-15 10:30:47 [WARN] [worker] Timeout
2024-01-15 10:30:48 [ERROR] [worker] Crash
2024-01-15 10:30:49 [INFO] [db] Query done`

	importPlain(t, ll, content)

	stats, err := ll.GetStats(context.Background())
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}

	if stats.TotalRecords != 5 {
		t.Errorf("expected 5 total records, got %d", stats.TotalRecords)
	}
	if stats.LevelCounts["ERROR"] != 2 {
		t.Errorf("expected 2 ERROR records, got %d", stats.LevelCounts["ERROR"])
	}
	if stats.LevelCounts["INFO"] != 2 {
		t.Errorf("expected 2 INFO records, got %d", stats.LevelCounts["INFO"])
	}
	if stats.LevelCounts["WARN"] != 1 {
		t.Errorf("expected 1 WARN record, got %d", stats.LevelCounts["WARN"])
	}
}

func TestMultiFileImport_NoIDCollision(t *testing.T) {
	ll, cleanup := newTestLogLens(t)
	defer cleanup()

	content1 := `2024-01-15 10:30:45 [ERROR] Service A error
2024-01-15 10:30:46 [INFO] Service A ok`

	content2 := `2024-01-15 10:30:45 [ERROR] Service B error
2024-01-15 10:30:46 [INFO] Service B ok`

	r1 := importPlainWithPrefix(t, ll, content1, "fileA")
	r2 := importPlainWithPrefix(t, ll, content2, "fileB")

	if r1.Processed != 2 {
		t.Errorf("first import: expected 2 processed, got %d", r1.Processed)
	}
	if r2.Processed != 2 {
		t.Errorf("second import: expected 2 processed, got %d", r2.Processed)
	}

	stats, err := ll.GetStats(context.Background())
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}

	if stats.TotalRecords != 4 {
		t.Errorf("expected 4 total records after two imports, got %d", stats.TotalRecords)
	}

	result, err := ll.Query(context.Background(), domain.Query{
		Filters: []domain.FilterCondition{
			{Type: domain.FilterContains, Field: "raw", Value: "Service A"},
		},
		Limit: 100,
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("expected 2 Service A records, got %d", result.Total)
	}

	result2, err := ll.Query(context.Background(), domain.Query{
		Filters: []domain.FilterCondition{
			{Type: domain.FilterContains, Field: "raw", Value: "Service B"},
		},
		Limit: 100,
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if result2.Total != 2 {
		t.Errorf("expected 2 Service B records, got %d", result2.Total)
	}
}

func TestQueryEngine_AggregationsOverFullDataset(t *testing.T) {
	ll, cleanup := newTestLogLens(t)
	defer cleanup()

	content := `2024-01-15 10:30:45 [ERROR] [api] err1
2024-01-15 10:30:46 [ERROR] [api] err2
2024-01-15 10:30:47 [INFO] [api] ok1
2024-01-15 10:30:48 [WARN] [api] warn1
2024-01-15 10:30:49 [INFO] [api] ok2
2024-01-15 10:30:50 [ERROR] [api] err3`

	importPlain(t, ll, content)

	q := domain.Query{
		Filters: []domain.FilterCondition{
			{Type: domain.FilterEquality, Field: "level", Value: "ERROR"},
		},
		Aggregations: []domain.Aggregation{
			{Function: "count", Field: "*", Alias: "error_count"},
		},
		Limit: 2,
	}

	result, err := ll.Query(context.Background(), q)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if result.Total != 3 {
		t.Errorf("expected total 3 ERROR records, got %d", result.Total)
	}
	if len(result.Records) != 2 {
		t.Errorf("expected 2 records on page (LIMIT 2), got %d", len(result.Records))
	}

	aggCount, ok := result.Aggregations["error_count"]
	if !ok {
		t.Fatal("expected error_count aggregation")
	}

	var count int64
	switch v := aggCount.(type) {
	case int64:
		count = v
	case int32:
		count = int64(v)
	case float64:
		count = int64(v)
	default:
		t.Fatalf("unexpected aggregation type: %T %v", aggCount, aggCount)
	}
	if count != 3 {
		t.Errorf("aggregation error_count=3 (full dataset), got %d", count)
	}
}

func TestQueryEngine_RegexpFilterInSQL(t *testing.T) {
	ll, cleanup := newTestLogLens(t)
	defer cleanup()

	content := `2024-01-15 10:30:45 [ERROR] Connection refused to 10.0.0.1
2024-01-15 10:30:46 [INFO] Connection established to 10.0.0.2
2024-01-15 10:30:47 [ERROR] Connection timed out to 192.168.1.1
2024-01-15 10:30:48 [INFO] Request completed`

	importPlain(t, ll, content)

	q := domain.Query{
		Filters: []domain.FilterCondition{
			{Type: domain.FilterRegexp, Field: "raw", Value: `Connection.*\d+\.\d+\.\d+\.\d+`},
		},
		Limit: 100,
	}

	result, err := ll.Query(context.Background(), q)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if result.Total != 3 {
		t.Errorf("expected 3 records matching Connection+IP pattern, got %d", result.Total)
	}
	for _, r := range result.Records {
		if !strings.Contains(r.Raw, "Connection") {
			t.Errorf("record does not match regexp: %s", r.Raw)
		}
	}
}
