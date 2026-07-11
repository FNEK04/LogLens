package domain

import (
	"context"
	"io"
)

type Parser interface {
	Parse(ctx context.Context, r io.Reader) (<-chan LogRecord, error)
	Config() ParserConfig
}

type Storage interface {
	Store(ctx context.Context, records <-chan LogRecord) (*ImportResult, error)
	Query(ctx context.Context, query Query) (*QueryResult, error)
	GetRecord(ctx context.Context, id string) (*LogRecord, error)
	GetTotalCount(ctx context.Context) (int64, error)
	GetLevelCounts(ctx context.Context) (map[string]int64, error)
	Aggregate(ctx context.Context, filters []FilterCondition, aggs []Aggregation) (map[string]interface{}, error)
	Close() error
}

type QueryEngine interface {
	Execute(ctx context.Context, query Query) (*QueryResult, error)
	Explain(query Query) (string, error)
}

type FilterEngine interface {
	BuildFilter(conditions []FilterCondition) (Filter, error)
	ApplyFilter(filter Filter, records <-chan LogRecord) <-chan LogRecord
}

type ProgressReporter interface {
	ReportProgress(current, total int64, message string)
	ReportError(err error)
}
