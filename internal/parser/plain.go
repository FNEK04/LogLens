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

var (
	plainTimestampPatterns = []*regexp.Regexp{
		regexp.MustCompile(`\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}`),
		regexp.MustCompile(`\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}`),
		regexp.MustCompile(`\d{2}/\d{2}/\d{4} \d{2}:\d{2}:\d{2}`),
		regexp.MustCompile(`[A-Z][a-z]{2} \d{2} \d{2}:\d{2}:\d{2}`),
	}

	plainTimestampStripPatterns = []*regexp.Regexp{
		regexp.MustCompile(`\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}[\.\d]*[A-Z]*`),
		regexp.MustCompile(`\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}[\.\d]*[A-Z]*`),
		regexp.MustCompile(`\d{2}/\d{2}/\d{4} \d{2}:\d{2}:\d{2}[\.\d]*[A-Z]*`),
		regexp.MustCompile(`[A-Z][a-z]{2} \d{2} \d{2}:\d{2}:\d{2}[\.\d]*[A-Z]*`),
		regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}[\.\d]*[A-Z]*`),
	}

	plainLevelStripRe = regexp.MustCompile(`(TRACE|DEBUG|INFO|WARN|ERROR|FATAL|PANIC)`)

	plainServicePatterns = []*regexp.Regexp{
		regexp.MustCompile(`\[([a-zA-Z0-9_-]+)\]`),
		regexp.MustCompile(`([a-zA-Z0-9_-]+):`),
		regexp.MustCompile(`service=([a-zA-Z0-9_-]+)`),
	}

	plainServiceStripPatterns = []*regexp.Regexp{
		regexp.MustCompile(`\[[a-zA-Z0-9_-]+\]`),
		regexp.MustCompile(`[a-zA-Z0-9_-]+:`),
		regexp.MustCompile(`service=[a-zA-Z0-9_-]+`),
	}

	plainTimestampFormats = []string{
		"2006-01-02 15:04:05",
		"2006/01/02 15:04:05",
		"01/02/2006 15:04:05",
		"Jan 02 15:04:05",
		"2006-01-02T15:04:05",
	}
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

func (p *PlainParser) Parse(ctx context.Context, r io.Reader) (<-chan domain.LogRecord, error) {
	records := make(chan domain.LogRecord, 1000)
	
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

func (p *PlainParser) parseLine(line string, lineNum int) (*domain.LogRecord, error) {
	record := &domain.LogRecord{
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
	for i, re := range plainTimestampPatterns {
		match := re.FindString(line)
		if match != "" {
			for _, format := range plainTimestampFormats {
				if timestamp, err := time.Parse(format, match); err == nil {
					return &timestamp
				}
			}
			_ = i
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
	knownLevels := map[string]bool{
		"TRACE": true, "DEBUG": true, "INFO": true,
		"WARN": true, "ERROR": true, "FATAL": true, "PANIC": true,
	}

	searchFrom := 0
	for searchFrom < len(line) {
		matched := false
		for _, re := range plainServicePatterns {
			remaining := line[searchFrom:]
			loc := re.FindStringIndex(remaining)
			if loc == nil {
				continue
			}
			matches := re.FindStringSubmatch(remaining)
			if len(matches) > 1 {
				candidate := matches[1]
				upper := strings.ToUpper(candidate)
				allDigits := true
				for _, c := range candidate {
					if c < '0' || c > '9' {
						allDigits = false
						break
					}
				}
				if !knownLevels[upper] && !allDigits {
					return candidate
				}
			}
			searchFrom += loc[1]
			matched = true
			break
		}
		if !matched {
			break
		}
	}

	return ""
}

func (p *PlainParser) extractMessage(line string) string {
	message := line
	
	for _, re := range plainTimestampStripPatterns {
		message = re.ReplaceAllString(message, "")
	}
	
	message = plainLevelStripRe.ReplaceAllString(message, "")
	
	for _, re := range plainServiceStripPatterns {
		message = re.ReplaceAllString(message, "")
	}
	
	message = strings.TrimSpace(message)
	
	return message
}
