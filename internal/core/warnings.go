package core

import "fmt"

// WarningLevel indicates the severity of a conversion warning.
type WarningLevel int

const (
	WarnInfo    WarningLevel = iota // informational, no data loss
	WarnMinor                      // minor data loss, non-critical field
	WarnMajor                      // significant data cannot be represented
)

// Warning represents a piece of data that couldn't be fully converted.
type Warning struct {
	Level   WarningLevel
	Field   string // which field or data type
	Message string
}

func (w Warning) String() string {
	var prefix string
	switch w.Level {
	case WarnInfo:
		prefix = "ℹ"
	case WarnMinor:
		prefix = "⚠"
	case WarnMajor:
		prefix = "✗"
	}
	return fmt.Sprintf("%s %s: %s", prefix, w.Field, w.Message)
}

// ConversionResult holds the outcome of a format conversion.
type ConversionResult struct {
	InputFormat  string
	OutputFormat string
	MangaCount   int
	ChapterCount int
	Warnings     []Warning
}
