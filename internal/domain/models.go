package domain

import (
	"time"
)

type Record struct {
	ID        string                 `json:"id"`
	Timestamp int64                  `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Service   string                 `json:"service,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Raw       string                 `json:"raw"`
}

func (r *Record) SetTimestamp(t time.Time) {
	if t.IsZero() {
		r.Timestamp = time.Now().Unix() * 1000
	} else {
		r.Timestamp = t.Unix() * 1000
	}
}

func (r *Record) GetTimestamp() time.Time {
	if r.Timestamp == 0 {
		return time.Now()
	}
	return time.UnixMilli(r.Timestamp)
}

type Filter interface {
	Match(r Record) bool
}

type FilterType string

const (
	FilterEquality   FilterType = "equality"
	FilterExclusion  FilterType = "exclusion"
	FilterContains   FilterType = "contains"
	FilterRegexp     FilterType = "regexp"
	FilterRange      FilterType = "range"
)

type FilterCondition struct {
	Type     FilterType `json:"type"`
	Field    string     `json:"field"`
	Value    interface{} `json:"value"`
	Operator string     `json:"operator,omitempty"`
}

type Query struct {
	Filters    []FilterCondition `json:"filters"`
	GroupBy    []string          `json:"groupBy,omitempty"`
	Aggregations []Aggregation   `json:"aggregations,omitempty"`
	SortBy     string            `json:"sortBy,omitempty"`
	SortDesc   bool              `json:"sortDesc,omitempty"`
	Limit      int               `json:"limit,omitempty"`
	Offset     int               `json:"offset,omitempty"`
}

type Aggregation struct {
	Function string `json:"function"`
	Field    string `json:"field,omitempty"`
	Alias    string `json:"alias,omitempty"`
}

type QueryResult struct {
	Records      []Record               `json:"records"`
	Aggregations map[string]interface{} `json:"aggregations,omitempty"`
	Total        int64                  `json:"total"`
	Took         int64                  `json:"took"`
}

type LogGroup struct {
	ID        string `json:"id"`
	Pattern   string `json:"pattern"`
	Count     int    `json:"count"`
	FirstSeen int64  `json:"firstSeen"`
	LastSeen  int64  `json:"lastSeen"`
	Sample    Record `json:"sample"`
	Level     string `json:"level"`
	Service   string `json:"service,omitempty"`
}

type ImportResult struct {
	TotalRecords int64  `json:"totalRecords"`
	Processed    int64  `json:"processed"`
	Errors       []string `json:"errors,omitempty"`
	Duration     int64  `json:"duration"`
}

type ParserType string

const (
	ParserPlain  ParserType = "plain"
	ParserJSON   ParserType = "json"
	ParserRegex  ParserType = "regex"
	ParserGrok   ParserType = "grok"
)

type ParserConfig struct {
	Type     ParserType          `json:"type"`
	Pattern  string             `json:"pattern,omitempty"`
	Fields   map[string]string  `json:"fields,omitempty"`
	TimeFormat string            `json:"timeFormat,omitempty"`
}

type IndexType string

const (
	IndexTime      IndexType = "time"
	IndexInverted  IndexType = "inverted"
	IndexBloom     IndexType = "bloom"
)

type IndexConfig struct {
	Type   IndexType `json:"type"`
	Field  string    `json:"field"`
	Config map[string]interface{} `json:"config,omitempty"`
}

type TimelineRequest struct {
	Filters  []FilterCondition `json:"filters"`
	BucketMs int64             `json:"bucketMs"`
}

type TimelinePoint struct {
	BucketStart int64 `json:"bucketStart"`
	Count       int64 `json:"count"`
}

type Report struct {
	GeneratedAt int64          `json:"generatedAt"`
	Query       Query          `json:"query"`
	BucketMs    int64          `json:"bucketMs"`
	Timeline    []TimelinePoint `json:"timeline"`
	Total       int64          `json:"total"`
	Took        int64          `json:"took"`
	Records     []Record       `json:"records"`
}
