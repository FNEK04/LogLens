package parser

import (
	"context"
	"fmt"
	"io"

	"LogLens/internal/domain"
)

type GrokParser struct {
	config domain.ParserConfig
}

func NewGrokParser(config domain.ParserConfig) (*GrokParser, error) {
	return &GrokParser{
		config: config,
	}, nil
}

func (p *GrokParser) Config() domain.ParserConfig {
	return p.config
}

func (p *GrokParser) Parse(ctx context.Context, r io.Reader) (<-chan domain.Record, error) {
	return nil, fmt.Errorf("grok parser not yet implemented")
}
