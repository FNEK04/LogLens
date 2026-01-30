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

type PlainParser struct {
	config domain.ParserConfig
}

func NewPlainParser(config domain.ParserConfig) *PlainParser {
	return &PlainParser{config: config}
}

func (p *PlainParser) Config() domain.ParserConfig {
	return p.config
}

func (p *PlainParser) Parse(ctx context.Context, r io.Reader) (<-chan domain.Record, error) {
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
				line := scanner.Text()
				if strings.TrimSpace(line) == "" {
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

func (p *PlainParser) parseLine(line string, lineNum int) (*domain.Record, error) {
	record := &domain.Record{
		ID:      fmt.Sprintf("line_%d", lineNum),
		Raw:     line,
		Fields:  make(map[string]interface{}),
	}
	
	if timestamp := p.extractTimestamp(line); timestamp != nil {
		record.SetTimestamp(*timestamp)
	} else {
		record.SetTimestamp(time.Now())
	}
	
	if level := p.extractLevel(line); level != "" {
		record.Level = level
	} else {
		record.Level = "INFO"
	}
	
	if service := p.extractService(line); service != "" {
		record.Service = service
	}
	
	record.Message = p.extractMessage(line)
	
	return record, nil
}

func (p *PlainParser) extractTimestamp(line string) *time.Time {
	patterns := []string{
		`\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}`,           // 2023-12-25 10:30:45
		`\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}`,            // 2023/12/25 10:30:45
		`\d{2}/\d{2}/\d{4} \d{2}:\d{2}:\d{2}`,            // 12/25/2023 10:30:45
		`[A-Z][a-z]{2} \d{2} \d{2}:\d{2}:\d{2}`,          // Dec 25 10:30:45
	}
	
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		match := re.FindString(line)
		if match != "" {
			formats := []string{
				"2006-01-02 15:04:05",
				"2006/01/02 15:04:05",
				"01/02/2006 15:04:05",
				"Jan 02 15:04:05",
				"2006-01-02T15:04:05",
			}
			
			for _, format := range formats {
				if timestamp, err := time.Parse(format, match); err == nil {
					return &timestamp
				}
			}
		}
	}
	
	return nil
}

func (p *PlainParser) extractLevel(line string) string {
	levels := []string{"TRACE", "DEBUG", "INFO", "WARN", "ERROR", "FATAL", "PANIC"}
	lineUpper := strings.ToUpper(line)
	
	for _, level := range levels {
		if strings.Contains(lineUpper, level) {
			return level
		}
	}
	
	return ""
}

func (p *PlainParser) extractService(line string) string {
	patterns := []string{
		`\[([a-zA-Z0-9_-]+)\]`,           // [service-name]
		`([a-zA-Z0-9_-]+):`,              // service-name:
		`service=([a-zA-Z0-9_-]+)`,       // service=service-name
	}
	
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(line)
		if len(matches) > 1 {
			return matches[1]
		}
	}
	
	return ""
}

func (p *PlainParser) extractMessage(line string) string {
	message := line
	
	timestampPatterns := []string{
		`\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}[\.\d]*[A-Z]*`,
		`\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}[\.\d]*[A-Z]*`,
		`\d{2}/\d{2}/\d{4} \d{2}:\d{2}:\d{2}[\.\d]*[A-Z]*`,
		`[A-Z][a-z]{2} \d{2} \d{2}:\d{2}:\d{2}[\.\d]*[A-Z]*`,
		`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}[\.\d]*[A-Z]*`,
	}
	
	for _, pattern := range timestampPatterns {
		re := regexp.MustCompile(pattern)
		message = re.ReplaceAllString(message, "")
	}
	
	levelPattern := `(TRACE|DEBUG|INFO|WARN|ERROR|FATAL|PANIC)`
	re := regexp.MustCompile(levelPattern)
	message = re.ReplaceAllString(message, "")
	
	servicePatterns := []string{
		`\[[a-zA-Z0-9_-]+\]`,
		`[a-zA-Z0-9_-]+:`,
		`service=[a-zA-Z0-9_-]+`,
	}
	
	for _, pattern := range servicePatterns {
		re := regexp.MustCompile(pattern)
		message = re.ReplaceAllString(message, "")
	}
	
	message = strings.TrimSpace(message)
	
	return message
}
