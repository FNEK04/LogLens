package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"LogLens/internal/app"
	"LogLens/internal/domain"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx     context.Context
	loglens *app.LogLens
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	
	config := app.Config{
		DatabasePath: a.getDatabasePath(),
	}
	
	loglens, err := app.NewLogLens(config)
	if err != nil {
		fmt.Printf("Failed to initialize LogLens: %v\n", err)
		return
	}
	
	a.loglens = loglens
}

func (a *App) shutdown(ctx context.Context) {
	if a.loglens != nil {
		a.loglens.Close()
	}
}

func (a *App) getDatabasePath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "./loglens.db"
	}
	
	loglensDir := filepath.Join(homeDir, ".loglens")
	os.MkdirAll(loglensDir, 0755)
	
	return filepath.Join(loglensDir, "loglens.db")
}

func (a *App) SelectLogFile() (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("app not initialized")
	}

	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select log file",
		Filters: []runtime.FileFilter{
			{DisplayName: "Log files", Pattern: "*.log;*.txt;*.json;*.ndjson;*.csv"},
			{DisplayName: "All files", Pattern: "*"},
		},
	})
	if err != nil {
		return "", err
	}

	return path, nil
}

func (a *App) GetTimeline(req domain.TimelineRequest) ([]domain.TimelinePoint, error) {
	if a.loglens == nil {
		return nil, fmt.Errorf("LogLens not initialized")
	}
	return a.loglens.GetTimeline(a.ctx, req)
}

func (a *App) ExportReport(query domain.Query, bucketMs int64) (string, error) {
	if a.loglens == nil {
		return "", fmt.Errorf("LogLens not initialized")
	}

	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Save LogLens Report",
		DefaultFilename: "loglens-report.json",
		Filters: []runtime.FileFilter{
			{DisplayName: "JSON", Pattern: "*.json"},
			{DisplayName: "All files", Pattern: "*"},
		},
	})
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", nil
	}

	res, err := a.loglens.Query(a.ctx, query)
	if err != nil {
		return "", err
	}

	points, err := a.loglens.GetTimeline(a.ctx, domain.TimelineRequest{Filters: query.Filters, BucketMs: bucketMs})
	if err != nil {
		return "", err
	}

	report := domain.Report{
		GeneratedAt: time.Now().UnixMilli(),
		Query:       query,
		BucketMs:    bucketMs,
		Timeline:    points,
		Total:       res.Total,
		Took:        res.Took,
		Records:     res.Records,
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", err
	}

	return path, nil
}

func (a *App) ImportFile(filePath string, parserConfig domain.ParserConfig) (*domain.ImportResult, error) {
	if a.loglens == nil {
		return nil, fmt.Errorf("LogLens not initialized")
	}
	
	return a.loglens.ImportFile(a.ctx, filePath, parserConfig, &defaultProgressReporter{})
}

func (a *App) AutoImportFile(filePath string) (*domain.ImportResult, error) {
	if a.loglens == nil {
		return nil, fmt.Errorf("LogLens not initialized")
	}
	
	return a.loglens.AutoImportFile(a.ctx, filePath, &defaultProgressReporter{})
}

func (a *App) Query(query domain.Query) (*domain.QueryResult, error) {
	if a.loglens == nil {
		return nil, fmt.Errorf("LogLens not initialized")
	}
	
	return a.loglens.Query(a.ctx, query)
}

func (a *App) ExplainQuery(query domain.Query) (string, error) {
	if a.loglens == nil {
		return "", fmt.Errorf("LogLens not initialized")
	}
	
	return a.loglens.ExplainQuery(query)
}

func (a *App) GetRecord(id string) (*domain.Record, error) {
	if a.loglens == nil {
		return nil, fmt.Errorf("LogLens not initialized")
	}
	
	return a.loglens.GetRecord(a.ctx, id)
}

func (a *App) GetSupportedParserTypes() []domain.ParserType {
	if a.loglens == nil {
		return []domain.ParserType{}
	}
	
	return a.loglens.GetSupportedParserTypes()
}

func (a *App) GetStats() (*app.Stats, error) {
	if a.loglens == nil {
		return nil, fmt.Errorf("LogLens not initialized")
	}
	
	return a.loglens.GetStats(a.ctx)
}

func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, Welcome to LogLens!", name)
}

type defaultProgressReporter struct{}

func (r *defaultProgressReporter) ReportProgress(current, total int64, message string) {
	fmt.Printf("Progress: %d/%d - %s\n", current, total, message)
}

func (r *defaultProgressReporter) ReportError(err error) {
	fmt.Printf("Error: %v\n", err)
}
