package inspect

import (
	"fmt"
	"sort"
	"strings"

	"github.com/misfit/mangastu/internal/core"
	"github.com/misfit/mangastu/internal/format"
)

// Inspector generates human-readable summaries of backup files.
type Inspector struct {
	Registry *format.Registry
}

// New creates a new Inspector with the given format registry.
func New(registry *format.Registry) *Inspector {
	return &Inspector{Registry: registry}
}

// InspectResult holds the formatted output of an inspection.
type InspectResult struct {
	Stats   core.BackupStats
	Summary string // human-readable summary
}

// Inspect reads a backup file and generates a summary.
func (i *Inspector) Inspect(path string) (*InspectResult, error) {
	srcFormat, err := i.Registry.Detect(path)
	if err != nil {
		return nil, err
	}

	result, err := srcFormat.Read(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", srcFormat.Name(), err)
	}

	stats := result.Backup.ComputeStats(srcFormat.Name(), srcFormat.Extensions()[0], result.CreatedAt)
	summary := formatSummary(stats)

	return &InspectResult{
		Stats:   stats,
		Summary: summary,
	}, nil
}

// Validate reads a backup file and checks for issues.
func (i *Inspector) Validate(path string) (string, error) {
	srcFormat, err := i.Registry.Detect(path)
	if err != nil {
		return "", err
	}

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Validating: %s\n\n", path))

	result, err := srcFormat.Read(path)
	if err != nil {
		sb.WriteString(fmt.Sprintf("✗ Failed to read: %v\n", err))
		return sb.String(), nil
	}

	sb.WriteString("✓ Archive readable\n")
	sb.WriteString(fmt.Sprintf("✓ Backup structure recognized (%s)\n", srcFormat.Name()))

	backup := result.Backup

	// Validate metadata
	if len(backup.Manga) == 0 {
		sb.WriteString("⚠ No manga entries found\n")
	} else {
		sb.WriteString(fmt.Sprintf("✓ %d manga records\n", len(backup.Manga)))
	}

	totalChapters := 0
	missingURLs := 0
	missingTitles := 0
	for _, m := range backup.Manga {
		totalChapters += len(m.Chapters)
		if m.URL == "" {
			missingURLs++
		}
		if m.Title == "" {
			missingTitles++
		}
		for _, ch := range m.Chapters {
			if ch.URL == "" {
				missingURLs++
			}
		}
	}

	if totalChapters > 0 {
		sb.WriteString(fmt.Sprintf("✓ %d chapters\n", totalChapters))
	} else {
		sb.WriteString("⚠ No chapters found\n")
	}

	if missingURLs > 0 {
		sb.WriteString(fmt.Sprintf("⚠ %d entries have missing URLs\n", missingURLs))
	}

	if missingTitles > 0 {
		sb.WriteString(fmt.Sprintf("⚠ %d manga have missing titles\n", missingTitles))
	}

	if len(backup.Categories) > 0 {
		sb.WriteString(fmt.Sprintf("✓ %d categories\n", len(backup.Categories)))
	}

	if len(backup.Sources) > 0 {
		sb.WriteString(fmt.Sprintf("✓ %d sources\n", len(backup.Sources)))
	}

	// Check conversion capability
	sb.WriteString("\nConversion compatibility:\n")
	for _, f := range i.Registry.List() {
		if f.Name() == srcFormat.Name() {
			continue
		}
		sb.WriteString(fmt.Sprintf("  → %s: ✓ Can convert\n", f.Name()))
	}

	// Report warnings
	if len(result.Warnings) > 0 {
		sb.WriteString("\nWarnings:\n")
		for _, w := range result.Warnings {
			sb.WriteString(fmt.Sprintf("  %s\n", w.String()))
		}
	}

	return sb.String(), nil
}

// formatSummary creates a human-readable summary string.
func formatSummary(stats core.BackupStats) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Format:     %s\n", stats.FormatName))
	sb.WriteString(fmt.Sprintf("Extension:  %s\n", stats.Extension))
	if !stats.CreatedAt.IsZero() {
		sb.WriteString(fmt.Sprintf("Created:    %s\n", stats.CreatedAt.Format("2006-01-02 15:04:05")))
	}
	sb.WriteString("\n")

	sb.WriteString(fmt.Sprintf("Manga:      %d\n", stats.MangaCount))
	sb.WriteString(fmt.Sprintf("Chapters:   %d\n", stats.ChapterCount))
	sb.WriteString(fmt.Sprintf("Categories: %d\n", stats.CategoryCount))
	sb.WriteString(fmt.Sprintf("Sources:    %d\n", stats.SourceCount))

	if stats.TrackingCount > 0 {
		sb.WriteString(fmt.Sprintf("Tracking:   %d\n", stats.TrackingCount))
	}
	if stats.HistoryCount > 0 {
		sb.WriteString(fmt.Sprintf("History:    %d\n", stats.HistoryCount))
	}

	// Source breakdown
	if len(stats.SourceBreakdown) > 0 {
		sb.WriteString("\nSources:\n")

		// Sort sources by count (descending)
		type sourceCount struct {
			name  string
			count int
		}
		sorted := make([]sourceCount, 0, len(stats.SourceBreakdown))
		for name, count := range stats.SourceBreakdown {
			sorted = append(sorted, sourceCount{name, count})
		}
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].count > sorted[j].count
		})

		for _, sc := range sorted {
			sb.WriteString(fmt.Sprintf("  - %s (%d)\n", sc.name, sc.count))
		}
	}

	// Metadata capabilities
	sb.WriteString("\nMetadata:\n")
	writeCapability(&sb, "title", stats.HasTitle)
	writeCapability(&sb, "author", stats.HasAuthor)
	writeCapability(&sb, "artist", stats.HasArtist)
	writeCapability(&sb, "thumbnail", stats.HasThumbnail)
	writeCapability(&sb, "description", stats.HasDescription)
	writeCapability(&sb, "categories", stats.HasCategories)
	writeCapability(&sb, "chapters", stats.HasChapters)
	writeCapability(&sb, "read state", stats.HasReadState)
	writeCapability(&sb, "bookmarks", stats.HasBookmarks)
	writeCapability(&sb, "tracking", stats.HasTracking)
	writeCapability(&sb, "history", stats.HasHistory)

	return sb.String()
}

func writeCapability(sb *strings.Builder, name string, present bool) {
	if present {
		sb.WriteString(fmt.Sprintf("  ✓ %s\n", name))
	} else {
		sb.WriteString(fmt.Sprintf("  ✗ %s\n", name))
	}
}
