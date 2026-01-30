package parser

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"regexp"
	"strings"
	"time"

	"LogLens/internal/domain"
)

type RegexParser struct {
	config domain.ParserConfig
	regex  *regexp.Regexp
}

func NewRegexParser(config domain.ParserConfig) (*RegexParser, error) {
	if config.Pattern == "" {
		return nil, fmt.Errorf("regex pattern is required")
	}
	
	regex, err := regexp.Compile(config.Pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}
	
	return &RegexParser{
		config: config,
		regex:  regex,
	}, nil
}

func (p *RegexParser) Config() domain.ParserConfig {
	return p.config
}

func (p *RegexParser) Parse(ctx context.Context, r io.Reader) (<-chan domain.Record, error) {
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
				
				record, err := p.parseLine(line, lineNum)
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

func (p *RegexParser) parseLine(line string, lineNum int) (*domain.Record, error) {
	matches := p.regex.FindStringSubmatch(line)
	if matches == nil {
		return nil, fmt.Errorf("line doesn't match regex pattern")
	}
	
	record := &domain.Record{
		ID:     fmt.Sprintf("regex_%d", lineNum),
		Raw:    line,
		Fields: make(map[string]interface{}),
	}
	
	groupNames := p.regex.SubexpNames()
	for i, match := range matches {
		if i == 0 {
			continue
		}
		
		var fieldName string
		if i < len(groupNames) && groupNames[i] != "" {
			fieldName = groupNames[i]
		} else {
			fieldName = fmt.Sprintf("field_%d", i)
		}
		
		switch strings.ToLower(fieldName) {
		case "timestamp", "time", "ts":
			if timestamp, err := p.parseTimestamp(match); err == nil {
				record.SetTimestamp(timestamp)
			} else {
				record.SetTimestamp(time.Now())
			}
		case "level", "severity", "priority":
			record.Level = strings.ToUpper(match)
		case "service", "app", "application":
			record.Service = match
		case "message", "msg", "text":
			record.Message = match
		default:
			record.Fields[fieldName] = match
		}
	}
	
	if record.Timestamp == 0 {
		record.SetTimestamp(time.Now())
	}
	if record.Level == "" {
		record.Level = "INFO"
	}
	if record.Message == "" {
		record.Message = line
	}
	
	return record, nil
}

func (p *RegexParser) parseTimestamp(value string) (time.Time, error) {
	if p.config.TimeFormat != "" {
		if timestamp, err := time.Parse(p.config.TimeFormat, value); err == nil {
			return timestamp, nil
		}
	}
	
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006/01/02 15:04:05",
		"Jan 02 15:04:05",
		"2006-01-02 15:04:05.000",
		"2006-01-02 15:04:05.000000",
	}
	
	for _, format := range formats {
		if timestamp, err := time.Parse(format, value); err == nil {
			return timestamp, nil
		}
	}
	
	return time.Time{}, fmt.Errorf("unsupported timestamp format: %s", value)
}
