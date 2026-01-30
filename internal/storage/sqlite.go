package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"LogLens/internal/domain"
	_ "modernc.org/sqlite"
)

type SQLiteStorage struct {
	db *sql.DB
}

func NewSQLiteStorage(dbPath string) (*SQLiteStorage, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	
	storage := &SQLiteStorage{db: db}
	
	if err := storage.init(); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}
	
	return storage, nil
}

func (s *SQLiteStorage) init() error {
	if _, err := s.db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return fmt.Errorf("failed to enable WAL mode: %w", err)
	}
	
	createRecordsTable := `
	CREATE TABLE IF NOT EXISTS records (
		id TEXT PRIMARY KEY,
		timestamp INTEGER NOT NULL,
		level TEXT NOT NULL,
		message TEXT NOT NULL,
		service TEXT,
		fields TEXT,
		raw TEXT NOT NULL,
		created_at INTEGER DEFAULT (strftime('%s', 'now'))
	);
	CREATE INDEX IF NOT EXISTS idx_records_timestamp ON records(timestamp);
	CREATE INDEX IF NOT EXISTS idx_records_timestamp_level ON records(timestamp, level);
	CREATE INDEX IF NOT EXISTS idx_records_timestamp_service ON records(timestamp, service);
	CREATE INDEX IF NOT EXISTS idx_records_level ON records(level);
	CREATE INDEX IF NOT EXISTS idx_records_service ON records(service);
	CREATE INDEX IF NOT EXISTS idx_records_created_at ON records(created_at);
	`
	
	if _, err := s.db.Exec(createRecordsTable); err != nil {
		return fmt.Errorf("failed to create records table: %w", err)
	}
	
	return nil
}

func (s *SQLiteStorage) allowedColumn(field string) (string, bool) {
	switch strings.ToLower(field) {
	case "id":
		return "id", true
	case "timestamp":
		return "timestamp", true
	case "level":
		return "level", true
	case "message":
		return "message", true
	case "service":
		return "service", true
	case "raw":
		return "raw", true
	default:
		return "", false
	}
}

func (s *SQLiteStorage) Store(ctx context.Context, records <-chan domain.Record) (*domain.ImportResult, error) {
	startTime := time.Now()
	result := &domain.ImportResult{}
	
	stmt, err := s.db.PrepareContext(ctx, `
		INSERT OR REPLACE INTO records (id, timestamp, level, message, service, fields, raw)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare insert statement: %w", err)
	}
	defer stmt.Close()
	
	batchSize := 1000
	batch := make([]domain.Record, 0, batchSize)
	
	for record := range records {
		batch = append(batch, record)
		result.TotalRecords++
		
		if len(batch) >= batchSize {
			if err := s.insertBatch(ctx, stmt, batch); err != nil {
				result.Errors = append(result.Errors, err.Error())
				continue
			}
			result.Processed += int64(len(batch))
			batch = batch[:0] // Reset batch
		}
	}
	
	if len(batch) > 0 {
		if err := s.insertBatch(ctx, stmt, batch); err != nil {
			result.Errors = append(result.Errors, err.Error())
		} else {
			result.Processed += int64(len(batch))
		}
	}
	
	result.Duration = time.Since(startTime).Milliseconds()
	return result, nil
}

func (s *SQLiteStorage) insertBatch(ctx context.Context, stmt *sql.Stmt, batch []domain.Record) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	for _, record := range batch {
		fieldsJSON, _ := json.Marshal(record.Fields)
		ts := record.Timestamp
		
		_, err := tx.StmtContext(ctx, stmt).Exec(
			record.ID,
			ts,
			record.Level,
			record.Message,
			record.Service,
			string(fieldsJSON),
			record.Raw,
		)
		if err != nil {
			return fmt.Errorf("failed to insert record %s: %w", record.ID, err)
		}
	}
	
	return tx.Commit()
}

func (s *SQLiteStorage) Query(ctx context.Context, query domain.Query) (*domain.QueryResult, error) {
	startTime := time.Now()
	
	sqlQuery, args, err := s.buildSQLQuery(query)
	if err != nil {
		return nil, fmt.Errorf("failed to build SQL query: %w", err)
	}
	
	rows, err := s.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()
	
	records := make([]domain.Record, 0)
	for rows.Next() {
		var record domain.Record
		var ts int64
		var fieldsJSON string
		
		err := rows.Scan(
			&record.ID,
			&ts,
			&record.Level,
			&record.Message,
			&record.Service,
			&fieldsJSON,
			&record.Raw,
		)
		if err != nil {
			log.Printf("Failed to scan row: %v", err)
			continue
		}
		
		record.Timestamp = ts
		if fieldsJSON != "" {
			if err := json.Unmarshal([]byte(fieldsJSON), &record.Fields); err != nil {
				log.Printf("Failed to unmarshal fields: %v", err)
				record.Fields = make(map[string]interface{})
			}
		} else {
			record.Fields = make(map[string]interface{})
		}
		
		records = append(records, record)
	}
	
	total, err := s.getTotalCount(ctx, query)
	if err != nil {
		log.Printf("Failed to get total count: %v", err)
		total = int64(len(records))
	}
	
	return &domain.QueryResult{
		Records: records,
		Total:   total,
		Took:    time.Since(startTime).Milliseconds(),
	}, nil
}

func (s *SQLiteStorage) Timeline(ctx context.Context, filters []domain.FilterCondition, bucketMs int64) ([]domain.TimelinePoint, error) {
	if bucketMs <= 0 {
		return nil, fmt.Errorf("bucketMs must be > 0")
	}

	var whereClauses []string
	var whereArgs []interface{}

	for _, filter := range filters {
		clause, clauseArgs, err := s.buildFilterClause(filter)
		if err != nil {
			return nil, err
		}
		if clause != "" {
			whereClauses = append(whereClauses, clause)
			whereArgs = append(whereArgs, clauseArgs...)
		}
	}

	q := "SELECT (CAST(timestamp / ? AS INTEGER) * ?) AS bucket_start, COUNT(*) AS cnt FROM records"
	args := []interface{}{bucketMs, bucketMs}
	if len(whereClauses) > 0 {
		q += " WHERE " + strings.Join(whereClauses, " AND ")
		args = append(args, whereArgs...)
	}
	q += " GROUP BY bucket_start ORDER BY bucket_start ASC"

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute timeline query: %w", err)
	}
	defer rows.Close()

	points := make([]domain.TimelinePoint, 0)
	for rows.Next() {
		var bucketStart int64
		var cnt int64
		if err := rows.Scan(&bucketStart, &cnt); err != nil {
			return nil, fmt.Errorf("failed to scan timeline row: %w", err)
		}
		points = append(points, domain.TimelinePoint{BucketStart: bucketStart, Count: cnt})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("timeline rows error: %w", err)
	}

	return points, nil
}

func (s *SQLiteStorage) buildSQLQuery(query domain.Query) (string, []interface{}, error) {
	var whereClauses []string
	var args []interface{}
	
	for _, filter := range query.Filters {
		clause, clauseArgs, err := s.buildFilterClause(filter)
		if err != nil {
			return "", nil, err
		}
		if clause != "" {
			whereClauses = append(whereClauses, clause)
			args = append(args, clauseArgs...)
		}
	}
	
	baseQuery := "SELECT id, timestamp, level, message, service, fields, raw FROM records"
	
	if len(whereClauses) > 0 {
		baseQuery += " WHERE " + strings.Join(whereClauses, " AND ")
	}
	
	if query.SortBy != "" {
		col, ok := s.allowedColumn(query.SortBy)
		if !ok {
			return "", nil, fmt.Errorf("invalid sort field: %s", query.SortBy)
		}
		direction := "ASC"
		if query.SortDesc {
			direction = "DESC"
		}
		baseQuery += fmt.Sprintf(" ORDER BY %s %s", col, direction)
	} else {
		baseQuery += " ORDER BY timestamp DESC"
	}
	
	if query.Limit > 0 {
		baseQuery += fmt.Sprintf(" LIMIT %d", query.Limit)
		if query.Offset > 0 {
			baseQuery += fmt.Sprintf(" OFFSET %d", query.Offset)
		}
	}
	
	return baseQuery, args, nil
}

func (s *SQLiteStorage) buildFilterClause(filter domain.FilterCondition) (string, []interface{}, error) {
	col, ok := s.allowedColumn(filter.Field)
	if !ok {
		return "", nil, fmt.Errorf("invalid filter field: %s", filter.Field)
	}

	switch filter.Type {
	case domain.FilterEquality:
		return fmt.Sprintf("%s = ?", col), []interface{}{filter.Value}, nil
	case domain.FilterExclusion:
		return fmt.Sprintf("%s != ?", col), []interface{}{filter.Value}, nil
	case domain.FilterContains:
		return fmt.Sprintf("%s LIKE ?", col), []interface{}{"%" + fmt.Sprintf("%v", filter.Value) + "%"}, nil
	case domain.FilterRegexp:
		return fmt.Sprintf("%s REGEXP ?", col), []interface{}{filter.Value}, nil
	case domain.FilterRange:
		switch filter.Operator {
		case "gt":
			return fmt.Sprintf("%s > ?", col), []interface{}{filter.Value}, nil
		case "lt":
			return fmt.Sprintf("%s < ?", col), []interface{}{filter.Value}, nil
		case "gte":
			return fmt.Sprintf("%s >= ?", col), []interface{}{filter.Value}, nil
		case "lte":
			return fmt.Sprintf("%s <= ?", col), []interface{}{filter.Value}, nil
		default:
			return "", nil, nil
		}
	default:
		return "", nil, nil
	}
}

func (s *SQLiteStorage) getTotalCount(ctx context.Context, query domain.Query) (int64, error) {
	var whereClauses []string
	var args []interface{}
	
	for _, filter := range query.Filters {
		clause, clauseArgs, err := s.buildFilterClause(filter)
		if err != nil {
			return 0, err
		}
		if clause != "" {
			whereClauses = append(whereClauses, clause)
			args = append(args, clauseArgs...)
		}
	}
	
	countQuery := "SELECT COUNT(*) FROM records"
	if len(whereClauses) > 0 {
		countQuery += " WHERE " + strings.Join(whereClauses, " AND ")
	}
	
	var count int64
	err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&count)
	return count, err
}

func (s *SQLiteStorage) GetRecord(ctx context.Context, id string) (*domain.Record, error) {
	query := `
		SELECT id, timestamp, level, message, service, fields, raw
		FROM records WHERE id = ?
	`
	
	var record domain.Record
	var ts int64
	var fieldsJSON string
	
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&record.ID,
		&ts,
		&record.Level,
		&record.Message,
		&record.Service,
		&fieldsJSON,
		&record.Raw,
	)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get record: %w", err)
	}
	
	record.Timestamp = ts
	if fieldsJSON != "" {
		if err := json.Unmarshal([]byte(fieldsJSON), &record.Fields); err != nil {
			log.Printf("Failed to unmarshal fields: %v", err)
			record.Fields = make(map[string]interface{})
		}
	} else {
		record.Fields = make(map[string]interface{})
	}
	
	return &record, nil
}

func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}
