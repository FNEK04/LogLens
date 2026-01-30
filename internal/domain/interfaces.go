package domain

import (
	"context"
	"io"
)

type Parser interface {
	Parse(ctx context.Context, r io.Reader) (<-chan Record, error)
	Config() ParserConfig
}

type Storage interface {
	Store(ctx context.Context, records <-chan Record) (*ImportResult, error)
	Query(ctx context.Context, query Query) (*QueryResult, error)
	GetRecord(ctx context.Context, id string) (*Record, error)
	Close() error
}

type Index interface {
	Index(records <-chan Record) error
	Search(query Query) ([]string, error)
	Close() error
}

type QueryEngine interface {
	Execute(ctx context.Context, query Query) (*QueryResult, error)
	Explain(query Query) (string, error)
}

type FilterEngine interface {
	BuildFilter(conditions []FilterCondition) (Filter, error)
	ApplyFilter(filter Filter, records <-chan Record) <-chan Record
}

type GroupingEngine interface {
	GroupRecords(records <-chan Record) (<-chan LogGroup, error)
	SetGroupingConfig(config GroupingConfig)
}

type GroupingConfig struct {
	Enabled         bool    `json:"enabled"`
	NormalizeIDs    bool    `json:"normalizeIds"`
	NormalizeNumbers bool   `json:"normalizeNumbers"`
	NormalizeDates   bool   `json:"normalizeDates"`
	SimilarityThreshold float64 `json:"similarityThreshold,omitempty"`
}

type ProgressReporter interface {
	ReportProgress(current, total int64, message string)
	ReportError(err error)
}

type FileImporter interface {
	ImportFile(ctx context.Context, filePath string, parser Parser, reporter ProgressReporter) (*ImportResult, error)
	SupportedExtensions() []string
}

type Plugin interface {
	Name() string
	Version() string
	CreateParser(config ParserConfig) (Parser, error)
	SupportedTypes() []ParserType
}
