package domain

import (
	"testing"
	"time"
)

func TestLogRecord_SetTimestamp(t *testing.T) {
	r := &LogRecord{}
	now := time.Now()
	r.SetTimestamp(now)

	if r.Timestamp != now.UnixMilli() {
		t.Errorf("expected timestamp %d, got %d", now.UnixMilli(), r.Timestamp)
	}
}

func TestLogRecord_SetTimestamp_Zero(t *testing.T) {
	r := &LogRecord{}
	before := time.Now().UnixMilli()
	r.SetTimestamp(time.Time{})
	after := time.Now().UnixMilli()

	if r.Timestamp < before || r.Timestamp > after {
		t.Errorf("expected timestamp between %d and %d, got %d", before, after, r.Timestamp)
	}
}

func TestLogRecord_GetTimestamp(t *testing.T) {
	now := time.Now().Truncate(time.Millisecond)
	r := &LogRecord{Timestamp: now.UnixMilli()}
	got := r.GetTimestamp()

	if !got.Equal(now) {
		t.Errorf("expected %v, got %v", now, got)
	}
}

func TestLogRecord_GetTimestamp_Zero(t *testing.T) {
	r := &LogRecord{Timestamp: 0}
	got := r.GetTimestamp()

	if got.IsZero() {
		t.Error("expected non-zero time for zero timestamp")
	}
}

func TestFilterCondition_Fields(t *testing.T) {
	fc := FilterCondition{
		Type:     FilterContains,
		Field:    "message",
		Value:    "error",
		Operator: "",
	}

	if fc.Type != FilterContains {
		t.Errorf("expected FilterContains, got %s", fc.Type)
	}
	if fc.Field != "message" {
		t.Errorf("expected 'message', got %s", fc.Field)
	}
}
