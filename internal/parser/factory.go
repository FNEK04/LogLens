package parser

import (
	"encoding/json"
	"fmt"
	"regexp"

	"LogLens/internal/domain"
)

var (
	factoryTimestampRe  = regexp.MustCompile(`\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}`)
	factoryLevelRe      = regexp.MustCompile(`\[(TRACE|DEBUG|INFO|WARN|ERROR|FATAL|PANIC)\]`)
	factoryServiceRe    = regexp.MustCompile(`\[([a-zA-Z0-9_-]+)\]`)
)

type ParserFactory struct{}

func NewParserFactory() *ParserFactory {
	return &ParserFactory{}
}

func (f *ParserFactory) CreateParser(config domain.ParserConfig) (domain.Parser, error) {
	switch config.Type {
	case domain.ParserPlain:
		return NewPlainParser(config), nil
	case domain.ParserJSON:
		return NewJSONParser(config), nil
	case domain.ParserRegex:
		return NewRegexParser(config)
	case domain.ParserGrok:
		return NewGrokParser(config)
	default:
		return nil, fmt.Errorf("unsupported parser type: %s", config.Type)
	}
}

func (f *ParserFactory) GetSupportedTypes() []domain.ParserType {
	return []domain.ParserType{
		domain.ParserPlain,
		domain.ParserJSON,
		domain.ParserRegex,
		domain.ParserGrok,
	}
}

func (f *ParserFactory) AutoDetectParser(sample string) (domain.ParserType, error) {
	if f.isJSON(sample) {
		return domain.ParserJSON, nil
	}
	
	if f.isStructuredLog(sample) {
		return domain.ParserPlain, nil
	}
	
	return domain.ParserPlain, nil
}

func (f *ParserFactory) isJSON(sample string) bool {
	return json.Valid([]byte(sample))
}

func (f *ParserFactory) isStructuredLog(sample string) bool {
	if factoryTimestampRe.MatchString(sample) {
		return true
	}
	if factoryLevelRe.MatchString(sample) {
		return true
	}
	if factoryServiceRe.MatchString(sample) {
		return true
	}
	return false
}
