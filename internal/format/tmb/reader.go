package tmb

import (
	"archive/zip"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/misfit/mangastu/internal/core"
	"github.com/misfit/mangastu/internal/format"
)

// Detect checks if a file is a TMB backup by looking for ZIP with meta.json + contents.zip.
func (t *TMB) Detect(path string) (bool, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".tmb" {
		return true, nil
	}

	// Try to open as ZIP and check for expected entries
	r, err := zip.OpenReader(path)
	if err != nil {
		return false, nil
	}
	defer r.Close()

	hasMeta, hasContents := false, false
	for _, f := range r.File {
		switch f.Name {
		case "meta.json":
			hasMeta = true
		case "contents.zip":
			hasContents = true
		}
	}
	return hasMeta && hasContents, nil
}

// Read parses a .tmb backup file and returns the internal model.
func (t *TMB) Read(path string) (*format.ReadResult, error) {
	// Open the outer ZIP
	outerZip, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open TMB archive: %w", err)
	}
	defer outerZip.Close()

	// Parse meta.json
	meta, err := readMeta(outerZip)
	if err != nil {
		return nil, fmt.Errorf("failed to read meta.json: %w", err)
	}

	// Extract contents.zip to a temp directory
	tmpDir, err := os.MkdirTemp("", "mangastu-tmb-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	contentsZipPath := filepath.Join(tmpDir, "contents.zip")
	if err := extractFile(outerZip, "contents.zip", contentsZipPath); err != nil {
		return nil, fmt.Errorf("failed to extract contents.zip: %w", err)
	}

	// Open contents.zip and extract the SQLite database
	innerZip, err := zip.OpenReader(contentsZipPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open contents.zip: %w", err)
	}
	defer innerZip.Close()

	dbPath := filepath.Join(tmpDir, "tachimanga.db")
	if err := extractFile2(innerZip, "tachimanga.db", dbPath); err != nil {
		return nil, fmt.Errorf("failed to extract tachimanga.db: %w", err)
	}

	// Open SQLite database
	db, err := sql.Open("sqlite", dbPath+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Read all data from the database
	var warnings []core.Warning

	categories, err := readCategories(db)
	if err != nil {
		return nil, fmt.Errorf("failed to read categories: %w", err)
	}

	manga, w, err := readManga(db, categories)
	if err != nil {
		return nil, fmt.Errorf("failed to read manga: %w", err)
	}
	warnings = append(warnings, w...)

	sources := buildSources(db, manga)

	// Report unmappable data
	warnings = append(warnings, reportUnmappable(innerZip)...)

	backup := &core.Backup{
		Manga:      manga,
		Categories: categories,
		Sources:    sources,
	}

	createdAt := time.Unix(meta.CreateAt, 0)

	return &format.ReadResult{
		Backup:    backup,
		CreatedAt: createdAt,
		Warnings:  warnings,
	}, nil
}

// readMeta parses meta.json from the outer ZIP.
func readMeta(zr *zip.ReadCloser) (*tmbMeta, error) {
	for _, f := range zr.File {
		if f.Name == "meta.json" {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()

			var meta tmbMeta
			if err := json.NewDecoder(rc).Decode(&meta); err != nil {
				return nil, err
			}
			return &meta, nil
		}
	}
	return nil, fmt.Errorf("meta.json not found in archive")
}

// readCategories reads categories from the database.
func readCategories(db *sql.DB) ([]core.Category, error) {
	rows, err := db.Query(`SELECT id, name, "order" FROM Category WHERE is_delete = 0 ORDER BY "order"`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []core.Category
	for i := int64(0); rows.Next(); i++ {
		var c core.Category
		var origOrder int64
		if err := rows.Scan(&c.ID, &c.Name, &origOrder); err != nil {
			return nil, err
		}
		c.Order = i // 0-based order for Komikku/tachibk compatibility
		categories = append(categories, c)
	}
	return categories, rows.Err()
}

// readManga reads manga and all associated data from the database.
func readManga(db *sql.DB, categories []core.Category) ([]core.Manga, []core.Warning, error) {
	var warnings []core.Warning

	// Build category ID → 0-based order index map
	catIDToOrder := make(map[int64]int64)
	for _, c := range categories {
		catIDToOrder[c.ID] = c.Order
	}

	// Read library manga
	rows, err := db.Query(`
		SELECT id, source, url, title,
			COALESCE(artist, ''), COALESCE(author, ''), COALESCE(description, ''),
			COALESCE(genre, ''), status, COALESCE(thumbnail_url, ''),
			in_library_at, in_library, initialized, update_strategy
		FROM Manga
		WHERE in_library = 1
		ORDER BY id
	`)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var mangaList []core.Manga
	for rows.Next() {
		var m core.Manga
		var mangaID int64
		var genreStr, updateStrategy string

		err := rows.Scan(
			&mangaID, &m.Source, &m.URL, &m.Title,
			&m.Artist, &m.Author, &m.Description,
			&genreStr, &m.Status, &m.ThumbnailURL,
			&m.DateAdded, &m.Favorite, &m.Initialized, &updateStrategy,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to scan manga: %w", err)
		}

		m.Favorite = true // in_library = 1 means favorite
		m.UpdateStrategy = updateStrategy

		// TMB stores in_library_at as epoch seconds; tachibk expects millis
		m.DateAdded = toMillis(m.DateAdded)

		// Parse genres (comma-separated)
		if genreStr != "" {
			for _, g := range strings.Split(genreStr, ", ") {
				g = strings.TrimSpace(g)
				if g != "" {
					m.Genre = append(m.Genre, g)
				}
			}
		}

		// Read chapters
		chapters, err := readChapters(db, mangaID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read chapters for manga %d: %w", mangaID, err)
		}
		m.Chapters = chapters

		// Read category assignments
		catIDs, err := readMangaCategories(db, mangaID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read categories for manga %d: %w", mangaID, err)
		}
		for _, catID := range catIDs {
			if order, ok := catIDToOrder[catID]; ok {
				m.Categories = append(m.Categories, order)
			}
		}

		// Read tracking
		tracking, err := readTracking(db, mangaID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read tracking for manga %d: %w", mangaID, err)
		}
		m.Tracking = tracking

		// Read history
		history, err := readHistory(db, mangaID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read history for manga %d: %w", mangaID, err)
		}
		m.History = history

		mangaList = append(mangaList, m)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	return mangaList, warnings, nil
}

// readChapters reads all chapters for a given manga ID.
func readChapters(db *sql.DB, mangaID int64) ([]core.Chapter, error) {
	rows, err := db.Query(`
		SELECT url, name, COALESCE(scanlator, ''), read, bookmark,
			last_page_read, date_upload, chapter_number, source_order,
			last_read_at, fetched_at
		FROM Chapter
		WHERE manga = ?
		ORDER BY source_order
	`, mangaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chapters []core.Chapter
	for rows.Next() {
		var ch core.Chapter
		err := rows.Scan(
			&ch.URL, &ch.Name, &ch.Scanlator, &ch.Read, &ch.Bookmark,
			&ch.LastPageRead, &ch.DateUpload, &ch.ChapterNumber, &ch.SourceOrder,
			&ch.LastReadAt, &ch.DateFetch,
		)
		if err != nil {
			return nil, err
		}
		// TMB stores fetched_at and last_read_at as epoch seconds; tachibk expects millis
		// date_upload is already in millis
		ch.DateFetch = toMillis(ch.DateFetch)
		ch.LastReadAt = toMillis(ch.LastReadAt)
		chapters = append(chapters, ch)
	}
	return chapters, rows.Err()
}

// readMangaCategories reads category IDs assigned to a manga.
func readMangaCategories(db *sql.DB, mangaID int64) ([]int64, error) {
	rows, err := db.Query(`SELECT category FROM CategoryManga WHERE manga = ?`, mangaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// readTracking reads tracker records for a manga.
func readTracking(db *sql.DB, mangaID int64) ([]core.Tracking, error) {
	rows, err := db.Query(`
		SELECT sync_id, remote_id, COALESCE(library_id, 0), title,
			last_chapter_read, total_chapters, status, score,
			remote_url, start_date, finish_date
		FROM TrackRecord
		WHERE manga_id = ? AND is_delete = 0
	`, mangaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tracking []core.Tracking
	for rows.Next() {
		var t core.Tracking
		err := rows.Scan(
			&t.SyncID, &t.MediaID, &t.LibraryID, &t.Title,
			&t.LastChapterRead, &t.TotalChapters, &t.Status, &t.Score,
			&t.TrackingURL, &t.StartedReadingDate, &t.FinishedReadingDate,
		)
		if err != nil {
			return nil, err
		}
		tracking = append(tracking, t)
	}
	return tracking, rows.Err()
}

// readHistory reads reading history for a manga.
// TMB stores history with chapter_id references, so we need to join to get URLs.
func readHistory(db *sql.DB, mangaID int64) ([]core.History, error) {
	rows, err := db.Query(`
		SELECT c.url, h.last_read_at, h.read_duration
		FROM History h
		JOIN Chapter c ON c.id = h.last_chapter_id
		WHERE h.manga_id = ? AND h.is_delete = 0
	`, mangaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []core.History
	for rows.Next() {
		var hist core.History
		var lastReadEpoch, readDurationSec int64
		err := rows.Scan(&hist.ChapterURL, &lastReadEpoch, &readDurationSec)
		if err != nil {
			return nil, err
		}
		// TMB stores last_read_at as epoch seconds; tachibk expects millis
		hist.LastRead = toMillis(lastReadEpoch)
		// TMB stores read_duration in seconds; tachibk expects millis
		hist.ReadDuration = readDurationSec * 1000
		history = append(history, hist)
	}
	return history, rows.Err()
}

// buildSources creates a list of unique sources from the manga data,
// using the Source table for name resolution.
func buildSources(db *sql.DB, manga []core.Manga) []core.Source {
	// Query the Source table to get source ID → name mapping
	sourceNames := make(map[int64]string)
	rows, err := db.Query(`SELECT id, name FROM Source`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id int64
			var name string
			if err := rows.Scan(&id, &name); err == nil {
				sourceNames[id] = name
			}
		}
	}

	seen := make(map[int64]bool)
	var sources []core.Source

	for _, m := range manga {
		if seen[m.Source] {
			continue
		}
		seen[m.Source] = true

		name := sourceNames[m.Source]
		if name == "" {
			name = fmt.Sprintf("Source %d", m.Source)
		}
		sources = append(sources, core.Source{
			Name:     name,
			SourceID: m.Source,
		})
	}

	return sources
}

// reportUnmappable generates warnings for data in the TMB that can't be converted.
func reportUnmappable(innerZip *zip.ReadCloser) []core.Warning {
	var warnings []core.Warning

	hasExtensions := false
	hasPlists := false

	for _, f := range innerZip.File {
		if strings.HasPrefix(f.Name, "extensions/") && strings.HasSuffix(f.Name, ".jar") {
			hasExtensions = true
		}
		if strings.HasPrefix(f.Name, "prefs/") && strings.HasSuffix(f.Name, ".plist") {
			hasPlists = true
		}
	}

	if hasExtensions {
		warnings = append(warnings, core.Warning{
			Level:   core.WarnMinor,
			Field:   "extensions",
			Message: "Extension JAR files cannot be included in tachibk format",
		})
	}

	if hasPlists {
		warnings = append(warnings, core.Warning{
			Level:   core.WarnInfo,
			Field:   "source_preferences",
			Message: "Source preference plist files are not converted (format-specific)",
		})
	}

	warnings = append(warnings, core.Warning{
		Level:   core.WarnInfo,
		Field:   "app_preferences",
		Message: "Tachimanga app preferences (pref.json) are not converted (app-specific)",
	})

	return warnings
}

// extractFile extracts a named file from a zip.ReadCloser to the destination path.
func extractFile(zr *zip.ReadCloser, name string, dst string) error {
	for _, f := range zr.File {
		if f.Name == name {
			return extractZipEntry(f, dst)
		}
	}
	return fmt.Errorf("file %s not found in archive", name)
}

// extractFile2 extracts a named file from a zip.ReadCloser (inner zip).
func extractFile2(zr *zip.ReadCloser, name string, dst string) error {
	for _, f := range zr.File {
		if f.Name == name {
			return extractZipEntry(f, dst)
		}
	}
	return fmt.Errorf("file %s not found in archive", name)
}

// extractZipEntry extracts a single zip entry to the destination path.
func extractZipEntry(f *zip.File, dst string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, rc)
	return err
}

// toMillis converts an epoch timestamp to milliseconds.
// If the value looks like seconds (< 1e12), it multiplies by 1000.
// If it already looks like milliseconds (>= 1e12), it returns as-is.
func toMillis(ts int64) int64 {
	if ts == 0 {
		return 0
	}
	if ts < 1e12 {
		return ts * 1000
	}
	return ts
}
