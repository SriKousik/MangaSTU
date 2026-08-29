package core

import "time"

// Backup is the format-agnostic internal representation of a manga backup.
// All format readers produce this; all format writers consume this.
type Backup struct {
	Manga       []Manga
	Categories  []Category
	Sources     []Source
	Preferences []Preference
}

// Preference represents an app or tracker configuration key-value pair.
type Preference struct {
	Key       string
	Type      string
	Value     []byte
	StringVal string
}

// Manga represents a single manga entry with all associated data.
type Manga struct {
	Source       int64
	URL          string
	Title        string
	Artist       string
	Author       string
	Description  string
	Genre        []string
	Status       int    // 0=unknown, 1=ongoing, 2=completed, 3=licensed, 4=publishing_finished, 5=cancelled, 6=on_hiatus
	ThumbnailURL string

	DateAdded  int64 // epoch milliseconds
	Favorite   bool
	Initialized bool

	ViewerFlags  int
	ChapterFlags int

	UpdateStrategy string // "ALWAYS_UPDATE" or "ONLY_FETCH_ONCE"

	Chapters   []Chapter
	Categories []int64 // category order indices (0-based)
	Tracking   []Tracking
	History    []History
}

// Chapter represents a single chapter of a manga.
type Chapter struct {
	URL           string
	Name          string
	Scanlator     string
	Read          bool
	Bookmark      bool
	LastPageRead  int64
	DateFetch     int64   // epoch milliseconds
	DateUpload    int64   // epoch milliseconds
	ChapterNumber float32
	SourceOrder   int64
	LastReadAt    int64   // epoch milliseconds (TMB-specific, used for history)
}

// Category represents a library category.
type Category struct {
	ID    int64
	Name  string
	Order int64
	Flags int64
}

// Tracking represents a tracker entry (MAL, AniList, etc.).
type Tracking struct {
	SyncID             int    // tracker service ID (1=MAL, 2=AniList, 3=Kitsu, etc.)
	MediaID            int64
	LibraryID          int64
	Title              string
	TrackingURL        string
	LastChapterRead    float32
	TotalChapters      int
	Score              float32
	Status             int
	StartedReadingDate int64 // epoch milliseconds
	FinishedReadingDate int64
}

// History represents reading history for a chapter.
type History struct {
	ChapterURL   string // chapter URL (used by tachibk)
	LastRead     int64  // epoch milliseconds
	ReadDuration int64  // seconds
}

// Source maps a source ID to its human-readable name.
type Source struct {
	Name     string
	SourceID int64
}

// BackupStats provides a summary of a backup's contents.
type BackupStats struct {
	FormatName    string
	Extension     string
	CreatedAt     time.Time
	MangaCount    int
	ChapterCount  int
	CategoryCount int
	SourceCount   int
	TrackingCount int
	HistoryCount  int

	// Per-source breakdown
	SourceBreakdown map[string]int // source name → manga count

	// Capability flags (what data is present)
	HasTitle        bool
	HasAuthor       bool
	HasArtist       bool
	HasThumbnail    bool
	HasCategories   bool
	HasChapters     bool
	HasReadState    bool
	HasBookmarks    bool
	HasTracking     bool
	HasHistory      bool
	HasDescription  bool
}

// ComputeStats calculates summary statistics from a Backup.
func (b *Backup) ComputeStats(formatName, extension string, createdAt time.Time) BackupStats {
	stats := BackupStats{
		FormatName:      formatName,
		Extension:       extension,
		CreatedAt:       createdAt,
		MangaCount:      len(b.Manga),
		CategoryCount:   len(b.Categories),
		SourceCount:     len(b.Sources),
		SourceBreakdown: make(map[string]int),
	}

	// Build source ID→name map
	sourceMap := make(map[int64]string)
	for _, s := range b.Sources {
		sourceMap[s.SourceID] = s.Name
	}

	for _, m := range b.Manga {
		stats.ChapterCount += len(m.Chapters)
		stats.TrackingCount += len(m.Tracking)
		stats.HistoryCount += len(m.History)

		// Source breakdown
		name := sourceMap[m.Source]
		if name == "" {
			name = "Unknown"
		}
		stats.SourceBreakdown[name]++

		// Capability detection
		if m.Title != "" {
			stats.HasTitle = true
		}
		if m.Author != "" {
			stats.HasAuthor = true
		}
		if m.Artist != "" {
			stats.HasArtist = true
		}
		if m.ThumbnailURL != "" {
			stats.HasThumbnail = true
		}
		if m.Description != "" {
			stats.HasDescription = true
		}
		if len(m.Categories) > 0 {
			stats.HasCategories = true
		}
		if len(m.Tracking) > 0 {
			stats.HasTracking = true
		}
		if len(m.History) > 0 {
			stats.HasHistory = true
		}

		for _, ch := range m.Chapters {
			if ch.Read {
				stats.HasReadState = true
			}
			if ch.Bookmark {
				stats.HasBookmarks = true
			}
		}
		if len(m.Chapters) > 0 {
			stats.HasChapters = true
		}
	}

	return stats
}
