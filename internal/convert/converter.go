package convert

import (
	"fmt"

	"github.com/misfit/mangastu/internal/core"
	"github.com/misfit/mangastu/internal/format"
)

// Converter handles format conversion using the format registry.
type Converter struct {
	Registry *format.Registry
}

// New creates a new Converter with the given format registry.
func New(registry *format.Registry) *Converter {
	return &Converter{Registry: registry}
}

// Convert reads a backup from inputPath and writes it to outputPath in the target format.
func (c *Converter) Convert(inputPath, outputPath string) (*core.ConversionResult, error) {
	// Detect source format
	srcFormat, err := c.Registry.Detect(inputPath)
	if err != nil {
		return nil, fmt.Errorf("input: %w", err)
	}

	// Determine target format by extension
	dstFormat, err := c.Registry.GetByExtension(outputPath)
	if err != nil {
		return nil, fmt.Errorf("output: %w", err)
	}

	// Check we're not converting to the same format
	if srcFormat.Name() == dstFormat.Name() {
		return nil, fmt.Errorf("source and target formats are the same: %s", srcFormat.Name())
	}

	// Read source
	result, err := srcFormat.Read(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", srcFormat.Name(), err)
	}

	// Write target
	if err := dstFormat.Write(outputPath, result.Backup); err != nil {
		return nil, fmt.Errorf("failed to write %s: %w", dstFormat.Name(), err)
	}

	// Compute stats for the result
	totalChapters := 0
	for _, m := range result.Backup.Manga {
		totalChapters += len(m.Chapters)
	}

	return &core.ConversionResult{
		InputFormat:  srcFormat.Name(),
		OutputFormat: dstFormat.Name(),
		MangaCount:   len(result.Backup.Manga),
		ChapterCount: totalChapters,
		Warnings:     result.Warnings,
	}, nil
}
