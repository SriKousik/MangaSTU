package merger

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/misfit/mangastu/internal/core"
	"github.com/misfit/mangastu/internal/format"
)

// Merger handles backup merging using the format registry.
type Merger struct {
	Registry *format.Registry
}

// New creates a new Merger instance.
func New(registry *format.Registry) *Merger {
	return &Merger{Registry: registry}
}

// MergeOptions contains configuration for the merge operation.
type MergeOptions struct {
	Verbose bool
}

// MergeReport summarizes the results of the merge operation.
type MergeReport struct {
	InputFiles      []string
	TotalInputManga int
	UniqueManga     int
	MergedManga     int
	TotalChapters   int
	ReadChapters    int
	CategoriesCount int
	SourcesCount    int
	TrackersCount   int
	HistoryCount    int
	Duration        time.Duration
}

// Merge reads multiple backup files, merges and deduplicates their content,
// and writes the result to outputPath in the format determined by its file extension.
func (m *Merger) Merge(inputPaths []string, outputPath string, opts MergeOptions) (*MergeReport, error) {
	if len(inputPaths) < 2 {
		return nil, fmt.Errorf("at least 2 input backup files are required to merge")
	}

	start := time.Now()
	var backups []*core.Backup
	totalInputManga := 0

	for _, p := range inputPaths {
		srcFormat, err := m.Registry.Detect(p)
		if err != nil {
			return nil, fmt.Errorf("failed to detect format for %s: %w", p, err)
		}

		res, err := srcFormat.Read(p)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", p, err)
		}

		backups = append(backups, res.Backup)
		totalInputManga += len(res.Backup.Manga)
	}

	mergedBackup, mergedCount := mergeBackups(backups)

	// Determine output format
	dstFormat, err := m.Registry.GetByExtension(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to determine output format for %s: %w", outputPath, err)
	}

	// Write output
	if err := dstFormat.Write(outputPath, mergedBackup); err != nil {
		return nil, fmt.Errorf("failed to write merged backup to %s: %w", outputPath, err)
	}

	totalChapters := 0
	readChapters := 0
	trackersCount := 0
	historyCount := 0
	for _, manga := range mergedBackup.Manga {
		totalChapters += len(manga.Chapters)
		for _, ch := range manga.Chapters {
			if ch.Read {
				readChapters++
			}
		}
		trackersCount += len(manga.Tracking)
		historyCount += len(manga.History)
	}

	return &MergeReport{
		InputFiles:      inputPaths,
		TotalInputManga: totalInputManga,
		UniqueManga:     len(mergedBackup.Manga),
		MergedManga:     mergedCount,
		TotalChapters:   totalChapters,
		ReadChapters:    readChapters,
		CategoriesCount: len(mergedBackup.Categories),
		SourcesCount:    len(mergedBackup.Sources),
		TrackersCount:   trackersCount,
		HistoryCount:    historyCount,
		Duration:        time.Since(start),
	}, nil
}

func mergeBackups(backups []*core.Backup) (*core.Backup, int) {
	result := &core.Backup{}

	// 1. Merge Categories (deduplicate by trimmed name)
	catNameToOrder := make(map[string]int64)
	var unifiedCategories []core.Category
	for _, b := range backups {
		for _, cat := range b.Categories {
			cleanName := strings.TrimSpace(cat.Name)
			if cleanName == "" {
				continue
			}
			if _, exists := catNameToOrder[cleanName]; !exists {
				order := int64(len(unifiedCategories))
				catNameToOrder[cleanName] = order
				unifiedCategories = append(unifiedCategories, core.Category{
					ID:    order + 1,
					Name:  cleanName,
					Order: order,
					Flags: cat.Flags,
				})
			}
		}
	}
	result.Categories = unifiedCategories

	// 2. Merge Sources
	sourceMap := make(map[int64]core.Source)
	for _, b := range backups {
		for _, src := range b.Sources {
			if existing, exists := sourceMap[src.SourceID]; !exists || (existing.Name == "" && src.Name != "") {
				sourceMap[src.SourceID] = src
			}
		}
	}

	// 3. Merge Manga, Chapters, Tracking, History
	type mangaKey struct {
		source int64
		url    string
	}
	mangaIndex := make(map[mangaKey]*core.Manga)
	var unifiedManga []*core.Manga
	mergedOverlapCount := 0

	for _, b := range backups {
		// Map this backup's category order -> category name
		backupCatOrderToName := make(map[int64]string)
		for _, cat := range b.Categories {
			backupCatOrderToName[cat.Order] = strings.TrimSpace(cat.Name)
		}

		for _, m := range b.Manga {
			// Resolve unified category order indices for this manga
			var resolvedCatOrders []int64
			for _, oldOrder := range m.Categories {
				catName := backupCatOrderToName[oldOrder]
				if newOrder, ok := catNameToOrder[catName]; ok {
					resolvedCatOrders = append(resolvedCatOrders, newOrder)
				}
			}

			key := mangaKey{source: m.Source, url: m.URL}
			existing, exists := mangaIndex[key]

			if !exists {
				// Clone manga
				mCopy := cloneManga(m)
				mCopy.Categories = deduplicateInt64(resolvedCatOrders)
				mangaIndex[key] = mCopy
				unifiedManga = append(unifiedManga, mCopy)
			} else {
				mergedOverlapCount++
				mergeMangaInto(existing, m, resolvedCatOrders)
			}

			// Ensure source is registered
			if _, exists := sourceMap[m.Source]; !exists {
				sourceMap[m.Source] = core.Source{
					SourceID: m.Source,
					Name:     fmt.Sprintf("Source %d", m.Source),
				}
			}
		}
	}

	// Flatten manga list
	result.Manga = make([]core.Manga, len(unifiedManga))
	for i, m := range unifiedManga {
		result.Manga[i] = *m
	}

	// Flatten sources list
	var unifiedSources []core.Source
	for _, src := range sourceMap {
		unifiedSources = append(unifiedSources, src)
	}
	sort.Slice(unifiedSources, func(i, j int) bool {
		return unifiedSources[i].SourceID < unifiedSources[j].SourceID
	})
	result.Sources = unifiedSources

	return result, mergedOverlapCount
}

func mergeMangaInto(target *core.Manga, source core.Manga, sourceResolvedCatOrders []int64) {
	// Library status: keep in library if true in either
	target.Favorite = target.Favorite || source.Favorite

	// Timestamps: earliest added
	if target.DateAdded == 0 || (source.DateAdded > 0 && source.DateAdded < target.DateAdded) {
		target.DateAdded = source.DateAdded
	}

	// Metadata completeness (prefer non-empty)
	if target.Title == "" && source.Title != "" {
		target.Title = source.Title
	}
	if target.Artist == "" && source.Artist != "" {
		target.Artist = source.Artist
	}
	if target.Author == "" && source.Author != "" {
		target.Author = source.Author
	}
	if target.Description == "" && source.Description != "" {
		target.Description = source.Description
	}
	if target.ThumbnailURL == "" && source.ThumbnailURL != "" {
		target.ThumbnailURL = source.ThumbnailURL
	}
	if len(target.Genre) == 0 && len(source.Genre) > 0 {
		target.Genre = source.Genre
	}
	if target.Status == 0 && source.Status > 0 {
		target.Status = source.Status
	}
	if target.ViewerFlags == 0 && source.ViewerFlags > 0 {
		target.ViewerFlags = source.ViewerFlags
	}
	if target.ChapterFlags == 0 && source.ChapterFlags > 0 {
		target.ChapterFlags = source.ChapterFlags
	}
	if strings.ToUpper(source.UpdateStrategy) == "ALWAYS_UPDATE" {
		target.UpdateStrategy = "ALWAYS_UPDATE"
	}

	// Categories: union
	target.Categories = deduplicateInt64(append(target.Categories, sourceResolvedCatOrders...))

	// Chapters: merge and reconcile
	target.Chapters = mergeChapters(target.Chapters, source.Chapters)

	// Tracking: merge by sync_id
	target.Tracking = mergeTracking(target.Tracking, source.Tracking)

	// History: merge by chapter_url
	target.History = mergeHistory(target.History, source.History)
}

func mergeChapters(ch1, ch2 []core.Chapter) []core.Chapter {
	chapMap := make(map[string]*core.Chapter)
	var unified []*core.Chapter

	for i := range ch1 {
		cCopy := ch1[i]
		chapMap[cCopy.URL] = &cCopy
		unified = append(unified, &cCopy)
	}

	for _, c := range ch2 {
		existing, exists := chapMap[c.URL]
		if !exists {
			cCopy := c
			chapMap[c.URL] = &cCopy
			unified = append(unified, &cCopy)
		} else {
			// Reconcile chapter progress and read state
			existing.Read = existing.Read || c.Read
			existing.Bookmark = existing.Bookmark || c.Bookmark
			if c.LastPageRead > existing.LastPageRead {
				existing.LastPageRead = c.LastPageRead
			}
			if c.LastReadAt > existing.LastReadAt {
				existing.LastReadAt = c.LastReadAt
			}
			if c.DateFetch > existing.DateFetch {
				existing.DateFetch = c.DateFetch
			}
			if existing.DateUpload == 0 && c.DateUpload > 0 {
				existing.DateUpload = c.DateUpload
			}
			if existing.Scanlator == "" && c.Scanlator != "" {
				existing.Scanlator = c.Scanlator
			}
		}
	}

	result := make([]core.Chapter, len(unified))
	for i, ch := range unified {
		result[i] = *ch
	}

	// Sort chapters by source order / chapter number
	sort.Slice(result, func(i, j int) bool {
		if result[i].SourceOrder != result[j].SourceOrder {
			return result[i].SourceOrder < result[j].SourceOrder
		}
		return result[i].ChapterNumber < result[j].ChapterNumber
	})

	return result
}

func mergeTracking(tr1, tr2 []core.Tracking) []core.Tracking {
	trMap := make(map[int]*core.Tracking)
	var unified []*core.Tracking

	for i := range tr1 {
		tCopy := tr1[i]
		trMap[tCopy.SyncID] = &tCopy
		unified = append(unified, &tCopy)
	}

	for _, t := range tr2 {
		existing, exists := trMap[t.SyncID]
		if !exists {
			tCopy := t
			trMap[t.SyncID] = &tCopy
			unified = append(unified, &tCopy)
		} else {
			if t.LastChapterRead > existing.LastChapterRead {
				existing.LastChapterRead = t.LastChapterRead
			}
			if t.Score > existing.Score {
				existing.Score = t.Score
			}
			if t.Status > 0 {
				existing.Status = t.Status
			}
			if existing.TrackingURL == "" && t.TrackingURL != "" {
				existing.TrackingURL = t.TrackingURL
			}
			if existing.StartedReadingDate == 0 && t.StartedReadingDate > 0 {
				existing.StartedReadingDate = t.StartedReadingDate
			}
			if existing.FinishedReadingDate == 0 && t.FinishedReadingDate > 0 {
				existing.FinishedReadingDate = t.FinishedReadingDate
			}
		}
	}

	result := make([]core.Tracking, len(unified))
	for i, tr := range unified {
		result[i] = *tr
	}
	return result
}

func mergeHistory(h1, h2 []core.History) []core.History {
	hMap := make(map[string]*core.History)
	var unified []*core.History

	for i := range h1 {
		hCopy := h1[i]
		hMap[hCopy.ChapterURL] = &hCopy
		unified = append(unified, &hCopy)
	}

	for _, h := range h2 {
		existing, exists := hMap[h.ChapterURL]
		if !exists {
			hCopy := h
			hMap[h.ChapterURL] = &hCopy
			unified = append(unified, &hCopy)
		} else {
			if h.LastRead > existing.LastRead {
				existing.LastRead = h.LastRead
			}
			if h.ReadDuration > existing.ReadDuration {
				existing.ReadDuration = h.ReadDuration
			}
		}
	}

	result := make([]core.History, len(unified))
	for i, h := range unified {
		result[i] = *h
	}
	return result
}

func cloneManga(m core.Manga) *core.Manga {
	c := m
	c.Genre = append([]string(nil), m.Genre...)
	c.Categories = append([]int64(nil), m.Categories...)
	c.Chapters = append([]core.Chapter(nil), m.Chapters...)
	c.Tracking = append([]core.Tracking(nil), m.Tracking...)
	c.History = append([]core.History(nil), m.History...)
	return &c
}

func deduplicateInt64(slice []int64) []int64 {
	seen := make(map[int64]bool)
	var result []int64
	for _, v := range slice {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
}
