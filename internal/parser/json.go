package parser

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"LogLens/internal/domain"
)

type JSONParser struct {
	config domain.ParserConfig
}

func NewJSONParser(config domain.ParserConfig) *JSONParser {
	return &JSONParser{config: config}
}

func (p *JSONParser) Config() domain.ParserConfig {
	return p.config
}

func (p *JSONParser) Parse(ctx context.Context, r io.Reader) (<-chan domain.Record, error) {
	records := make(chan domain.Record, 1000)
	
	go func() {
		defer close(records)
		scanner := bufio.NewScanner(r)
		
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 10*1024*1024)
		
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			select {
			case <-ctx.Done():
				return
			default:
				line := strings.TrimSpace(scanner.Text())
				if line == "" {
					continue
				}
				
				record, err := p.parseJSONLine(line, lineNum)
				if err != nil {
					log.Printf("Error parsing line %d: %v", lineNum, err)
					continue
				}
				
				select {
				case records <- *record:
				case <-ctx.Done():
					return
				}
			}
		}
		
		if err := scanner.Err(); err != nil {
			log.Printf("Scanner error: %v", err)
		}
	}()
	
	return records, nil
}

func (p *JSONParser) parseJSONLine(line string, lineNum int) (*domain.Record, error) {
	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(line), &jsonData); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	
	record := &domain.Record{
		ID:     p.generateID(jsonData, lineNum),
		Raw:    line,
		Fields: make(map[string]interface{}),
	}
	
	if timestamp := p.extractTimestamp(jsonData); timestamp != nil {
		record.SetTimestamp(*timestamp)
	} else {
		record.SetTimestamp(time.Now())
	}
	
	record.Level = p.extractLevel(jsonData)
	record.Service = p.extractService(jsonData)
	record.Message = p.extractMessage(jsonData)
	
	for key, value := range jsonData {
		if !p.isStandardField(key) {
			record.Fields[key] = value
		}
	}
	
	return record, nil
}

func (p *JSONParser) generateID(jsonData map[string]interface{}, lineNum int) string {
	if id, exists := jsonData["id"]; exists {
		if idStr, ok := id.(string); ok && idStr != "" {
			return idStr
		}
	}
	
	return fmt.Sprintf("json_%d", lineNum)
}

func (p *JSONParser) extractTimestamp(jsonData map[string]interface{}) *time.Time {
	timestampFields := []string{"timestamp", "time", "@timestamp", "ts", "datetime"}
	
	for _, field := range timestampFields {
		if value, exists := jsonData[field]; exists {
			if timestamp, err := p.parseTimestamp(value); err == nil {
				return &timestamp
			}
		}
	}
	
	return nil
}

func (p *JSONParser) parseTimestamp(value interface{}) (time.Time, error) {
	switch v := value.(type) {
	case string:
		formats := []string{
			time.RFC3339,
			time.RFC3339Nano,
			"2006-01-02 15:04:05",
			"2006-01-02T15:04:05",
			"2006/01/02 15:04:05",
			"Jan 02 15:04:05",
		}
		
		for _, format := range formats {
			if timestamp, err := time.Parse(format, v); err == nil {
				return timestamp, nil
			}
		}
		
		if timestamp, err := time.Parse(time.RFC3339, v); err == nil {
			return timestamp, nil
		}
		
	case float64:
		return time.Unix(int64(v), 0), nil
		
	case int64:
		return time.Unix(v, 0), nil
	}
	
	return time.Time{}, fmt.Errorf("unsupported timestamp format: %v", value)
}

func (p *JSONParser) extractLevel(jsonData map[string]interface{}) string {
	levelFields := []string{"level", "severity", "priority", "loglevel"}
	
	for _, field := range levelFields {
		if value, exists := jsonData[field]; exists {
			if level, ok := value.(string); ok && level != "" {
				return strings.ToUpper(level)
			}
		}
	}
	
	return "INFO"
}

func (p *JSONParser) extractService(jsonData map[string]interface{}) string {
	serviceFields := []string{"service", "service_name", "application", "app", "component"}
	
	for _, field := range serviceFields {
		if value, exists := jsonData[field]; exists {
			if service, ok := value.(string); ok && service != "" {
				return service
			}
		}
	}
	
	return ""
}

func (p *JSONParser) extractMessage(jsonData map[string]interface{}) string {
	messageFields := []string{"message", "msg", "text", "content", "log"}
	
	for _, field := range messageFields {
		if value, exists := jsonData[field]; exists {
			if message, ok := value.(string); ok && message != "" {
				return message
			}
		}
	}
	
	if len(jsonData) > 0 {
		if bytes, err := json.Marshal(jsonData); err == nil {
			return string(bytes)
		}
	}
	
	return ""
}

func (p *JSONParser) isStandardField(field string) bool {
	standardFields := []string{
		"id", "timestamp", "time", "@timestamp", "ts", "datetime",
		"level", "severity", "priority", "loglevel",
		"service", "service_name", "application", "app", "component",
		"message", "msg", "text", "content", "log",
	}
	
	for _, standard := range standardFields {
		if field == standard {
			return true
		}
	}
	
	return false
}
