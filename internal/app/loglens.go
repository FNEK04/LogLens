package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"LogLens/internal/domain"
	"LogLens/internal/parser"
	"LogLens/internal/query"
	"LogLens/internal/storage"
)

type LogLens struct {
	storage      domain.Storage
	queryEngine  domain.QueryEngine
	filterEngine domain.FilterEngine
	parserFactory *parser.ParserFactory
}



type Config struct {
	DatabasePath string `json:"databasePath"`
}

func NewLogLens(config Config) (*LogLens, error) {
	if err := os.MkdirAll(filepath.Dir(config.DatabasePath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}
	
	storage, err := storage.NewSQLiteStorage(config.DatabasePath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}
	
	queryEngine := query.NewQueryEngine(storage)
	
	filterEngine := query.NewFilterEngine()
	
	parserFactory := parser.NewParserFactory()
	
	return &LogLens{
		storage:       storage,
		queryEngine:   queryEngine,
		filterEngine:  filterEngine,
		parserFactory: parserFactory,
	}, nil
}

func (ll *LogLens) ImportFile(ctx context.Context, filePath string, parserConfig domain.ParserConfig, reporter domain.ProgressReporter) (*domain.ImportResult, error) {
	parser, err := ll.parserFactory.CreateParser(parserConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create parser: %w", err)
	}
	
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()
	
	records, err := parser.Parse(ctx, file)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file: %w", err)
	}
	
	result, err := ll.storage.Store(ctx, records)
	if err != nil {
		return nil, fmt.Errorf("failed to store records: %w", err)
	}
	
	return result, nil
}

func (ll *LogLens) AutoImportFile(ctx context.Context, filePath string, reporter domain.ProgressReporter) (*domain.ImportResult, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()
	
	buffer := make([]byte, 1024)
	n, err := file.Read(buffer)
	if err != nil {
		return nil, fmt.Errorf("failed to read file sample: %w", err)
	}
	
	sample := string(buffer[:n])
	
	parserType, err := ll.parserFactory.AutoDetectParser(sample)
	if err != nil {
		return nil, fmt.Errorf("failed to auto-detect parser: %w", err)
	}
	
	parserConfig := domain.ParserConfig{
		Type: parserType,
	}
	
	return ll.ImportFile(ctx, filePath, parserConfig, reporter)
}

func (ll *LogLens) Query(ctx context.Context, query domain.Query) (*domain.QueryResult, error) {
	return ll.queryEngine.Execute(ctx, query)
}

func (ll *LogLens) ExplainQuery(query domain.Query) (string, error) {
	return ll.queryEngine.Explain(query)
}

func (ll *LogLens) GetRecord(ctx context.Context, id string) (*domain.Record, error) {
	return ll.storage.GetRecord(ctx, id)
}

func (ll *LogLens) GetTimeline(ctx context.Context, req domain.TimelineRequest) ([]domain.TimelinePoint, error) {
	provider, ok := ll.storage.(interface{ Timeline(context.Context, []domain.FilterCondition, int64) ([]domain.TimelinePoint, error) })
	if !ok {
		return nil, fmt.Errorf("timeline not supported by storage")
	}
	return provider.Timeline(ctx, req.Filters, req.BucketMs)
}

func (ll *LogLens) GetSupportedParserTypes() []domain.ParserType {
	return ll.parserFactory.GetSupportedTypes()
}

func (ll *LogLens) CreateParser(config domain.ParserConfig) (domain.Parser, error) {
	return ll.parserFactory.CreateParser(config)
}

func (ll *LogLens) Close() error {
	return ll.storage.Close()
}

func (ll *LogLens) GetStats(ctx context.Context) (*Stats, error) {
	totalQuery := domain.Query{
		Limit: 0,
	}
	
	result, err := ll.storage.Query(ctx, totalQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}
	
	stats := &Stats{
		TotalRecords: result.Total,
		LastUpdated:  time.Now().UnixMilli(),
		LevelCounts:  make(map[string]int64),
	}
	
	levelQueries := []string{"ERROR", "WARN", "INFO", "DEBUG"}
	for _, level := range levelQueries {
		levelQuery := domain.Query{
			Filters: []domain.FilterCondition{
				{
					Type:  domain.FilterEquality,
					Field: "level",
					Value: level,
				},
			},
			Limit: 0,
		}
		
		result, err := ll.storage.Query(ctx, levelQuery)
		if err != nil {
			continue
		}
		
		stats.LevelCounts[level] = result.Total
	}
	
	return stats, nil
}

type Stats struct {
	TotalRecords int64            `json:"totalRecords"`
	LevelCounts  map[string]int64 `json:"levelCounts"`
	LastUpdated  int64            `json:"lastUpdated"`
}
