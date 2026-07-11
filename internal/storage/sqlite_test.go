package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"LogLens/internal/domain"
)

func newTestStorage(t *testing.T) (*SQLiteStorage, func()) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	storage, err := NewSQLiteStorage(dbPath)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	cleanup := func() {
		storage.Close()
		os.Remove(dbPath)
		os.Remove(dbPath + "-wal")
		os.Remove(dbPath + "-shm")
	}

	return storage, cleanup
}

func storeRecords(t *testing.T, storage *SQLiteStorage, records []domain.LogRecord) *domain.ImportResult {
	t.Helper()
	ch := make(chan domain.LogRecord, len(records))
	for _, r := range records {
		ch <- r
	}
	close(ch)

	result, err := storage.Store(context.Background(), ch)
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}
	return result
}

func TestSQLiteStorage_StoreAndQuery(t *testing.T) {
	storage, cleanup := newTestStorage(t)
	defer cleanup()

	records := []domain.LogRecord{
		{ID: "1", Timestamp: 1000, Level: "ERROR", Message: "Connection refused", Service: "api", Fields: make(map[string]interface{}), Raw: "raw1"},
		{ID: "2", Timestamp: 2000, Level: "INFO", Message: "Request completed", Service: "api", Fields: make(map[string]interface{}), Raw: "raw2"},
		{ID: "3", Timestamp: 3000, Level: "ERROR", Message: "Timeout", Service: "worker", Fields: make(map[string]interface{}), Raw: "raw3"},
	}

	result := storeRecords(t, storage, records)

	if result.Processed != 3 {
		t.Errorf("expected 3 processed, got %d", result.Processed)
	}

	queryResult, err := storage.Query(context.Background(), domain.Query{Limit: 100})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if queryResult.Total != 3 {
		t.Errorf("expected 3 total, got %d", queryResult.Total)
	}
	if len(queryResult.Records) != 3 {
		t.Errorf("expected 3 records, got %d", len(queryResult.Records))
	}
}

func TestSQLiteStorage_QueryWithFilter(t *testing.T) {
	storage, cleanup := newTestStorage(t)
	defer cleanup()

	records := []domain.LogRecord{
		{ID: "1", Timestamp: 1000, Level: "ERROR", Message: "Connection refused", Fields: make(map[string]interface{}), Raw: "raw1"},
		{ID: "2", Timestamp: 2000, Level: "INFO", Message: "Request completed", Fields: make(map[string]interface{}), Raw: "raw2"},
		{ID: "3", Timestamp: 3000, Level: "ERROR", Message: "Timeout", Fields: make(map[string]interface{}), Raw: "raw3"},
	}
	storeRecords(t, storage, records)

	queryResult, err := storage.Query(context.Background(), domain.Query{
		Filters: []domain.FilterCondition{
			{Type: domain.FilterEquality, Field: "level", Value: "ERROR"},
		},
		Limit: 100,
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if queryResult.Total != 2 {
		t.Errorf("expected 2 ERROR records, got %d", queryResult.Total)
	}
}

func TestSQLiteStorage_GetRecord(t *testing.T) {
	storage, cleanup := newTestStorage(t)
	defer cleanup()

	records := []domain.LogRecord{
		{ID: "test-id", Timestamp: 1000, Level: "INFO", Message: "test message", Fields: map[string]interface{}{"key": "value"}, Raw: "raw"},
	}
	storeRecords(t, storage, records)

	record, err := storage.GetRecord(context.Background(), "test-id")
	if err != nil {
		t.Fatalf("GetRecord failed: %v", err)
	}

	if record.ID != "test-id" {
		t.Errorf("expected ID 'test-id', got %s", record.ID)
	}
	if record.Message != "test message" {
		t.Errorf("expected message 'test message', got %s", record.Message)
	}
	if record.Fields["key"] != "value" {
		t.Errorf("expected field key=value, got %v", record.Fields["key"])
	}
}

func TestSQLiteStorage_GetRecord_NotFound(t *testing.T) {
	storage, cleanup := newTestStorage(t)
	defer cleanup()

	_, err := storage.GetRecord(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent record")
	}
}

func TestSQLiteStorage_Timeline(t *testing.T) {
	storage, cleanup := newTestStorage(t)
	defer cleanup()

	records := []domain.LogRecord{
		{ID: "1", Timestamp: 1000, Level: "INFO", Message: "msg1", Fields: make(map[string]interface{}), Raw: "raw1"},
		{ID: "2", Timestamp: 1500, Level: "INFO", Message: "msg2", Fields: make(map[string]interface{}), Raw: "raw2"},
		{ID: "3", Timestamp: 3000, Level: "INFO", Message: "msg3", Fields: make(map[string]interface{}), Raw: "raw3"},
		{ID: "4", Timestamp: 3200, Level: "INFO", Message: "msg4", Fields: make(map[string]interface{}), Raw: "raw4"},
	}
	storeRecords(t, storage, records)

	points, err := storage.Timeline(context.Background(), nil, 1000)
	if err != nil {
		t.Fatalf("Timeline failed: %v", err)
	}

	if len(points) != 2 {
		t.Fatalf("expected 2 buckets, got %d", len(points))
	}

	if points[0].Count != 2 {
		t.Errorf("expected bucket 0 count=2, got %d", points[0].Count)
	}
	if points[1].Count != 2 {
		t.Errorf("expected bucket 1 count=2, got %d", points[1].Count)
	}
}

func TestSQLiteStorage_BatchInsert(t *testing.T) {
	storage, cleanup := newTestStorage(t)
	defer cleanup()

	records := make([]domain.LogRecord, 2500)
	for i := range records {
		records[i] = domain.LogRecord{
			ID:        "batch_" + string(rune('0'+i%10)) + "_" + string(rune('a'+i/10%26)),
			Timestamp: int64(i * 100),
			Level:     "INFO",
			Message:   "batch record",
			Fields:    make(map[string]interface{}),
			Raw:       "raw",
		}
	}

	result := storeRecords(t, storage, records)

	if result.Processed != 2500 {
		t.Errorf("expected 2500 processed, got %d", result.Processed)
	}
}

func TestSQLiteStorage_Close(t *testing.T) {
	storage, cleanup := newTestStorage(t)
	defer cleanup()

	err := storage.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}
