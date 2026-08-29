package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/misfit/mangastu/internal/convert"
	"github.com/misfit/mangastu/internal/core"
	"github.com/misfit/mangastu/internal/format"
	"github.com/misfit/mangastu/internal/format/tachibk"
	"github.com/misfit/mangastu/internal/format/tmb"
	"github.com/misfit/mangastu/internal/inspect"
	"github.com/misfit/mangastu/internal/merger"
	"github.com/misfit/mangastu/internal/tracker"
)

// Server handles HTTP API requests and serves static web assets.
type Server struct {
	port      int
	staticDir string
	registry  *format.Registry
}

// New creates a new Server instance.
func New(port int, staticDir string) *Server {
	if port <= 0 {
		port = 8080
	}
	if staticDir == "" {
		staticDir = "./dist"
	}
	reg := format.NewRegistry()
	reg.Register(&tmb.TMB{})
	reg.Register(&tachibk.TachiBK{})

	return &Server{
		port:      port,
		staticDir: staticDir,
		registry:  reg,
	}
}

// Start runs the HTTP server.
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// API Routes
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/convert", s.handleConvert)
	mux.HandleFunc("/api/merge", s.handleMerge)
	mux.HandleFunc("/api/inspect", s.handleInspect)
	mux.HandleFunc("/api/trackers", s.handleTrackers)
	mux.HandleFunc("/api/tracking/export", s.handleTrackingExport)
	mux.HandleFunc("/api/search/anilist", s.handleSearchAniList)

	// Static Files & SPA Fallback
	mux.HandleFunc("/", s.handleStaticOrSPA)

	handler := s.corsMiddleware(mux)

	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("MangaSTU server listening on http://0.0.0.0:%d (Static Dir: %s)", s.port, s.staticDir)
	return http.ListenAndServe(addr, handler)
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Disposition, X-Manga-Count, X-Chapter-Count, X-Sources-Count, X-Output-Filename, X-Total-Manga, X-Unique-Manga, X-Overlap-Count, X-Total-Chapters, X-Read-Chapters, X-History-Count, X-Categories-Count, X-Extensions-Json, X-Categories-Json, X-Tracking-Added, X-Tracking-Updated")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":  "ok",
		"version": "0.1.0",
		"app":     "MangaSTU",
	})
}

func (s *Server) handleConvert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 500 MB max memory for upload
	if err := r.ParseMultipartForm(500 << 20); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse form: %v", err), http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Missing file in upload", http.StatusBadRequest)
		return
	}
	defer file.Close()

	targetFormat := strings.ToLower(strings.TrimSpace(r.FormValue("target_format")))
	if targetFormat == "" {
		if strings.HasSuffix(strings.ToLower(header.Filename), ".tmb") {
			targetFormat = "tachibk"
		} else {
			targetFormat = "tmb"
		}
	}

	tempDir, err := os.MkdirTemp("", "mangastu-convert-*")
	if err != nil {
		http.Error(w, "Failed to create temp directory", http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tempDir)

	inputPath := filepath.Join(tempDir, header.Filename)
	nowStr := time.Now().Format("2006-01-02_15-04")
	var outBaseName string
	if targetFormat == "tachibk" {
		outBaseName = fmt.Sprintf("app.komikku_%s", nowStr)
	} else {
		outBaseName = fmt.Sprintf("Tachimanga_backup_%s", nowStr)
	}
	outputPath := filepath.Join(tempDir, outBaseName+"."+targetFormat)

	dst, err := os.Create(inputPath)
	if err != nil {
		http.Error(w, "Failed to save uploaded file", http.StatusInternalServerError)
		return
	}
	if _, err := io.Copy(dst, file); err != nil {
		dst.Close()
		http.Error(w, "Failed to write uploaded file", http.StatusInternalServerError)
		return
	}
	dst.Close()

	converter := convert.New(s.registry)
	res, err := converter.Convert(inputPath, outputPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Conversion failed: %v", err), http.StatusBadRequest)
		return
	}

	type ExtensionInfo struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}

	var extensions []ExtensionInfo
	var categoriesList []tracker.CategoryCountItem
	var categoriesCount, historyCount, sourcesCount int

	if reader, err := s.registry.GetByExtension(outputPath); err == nil {
		if readRes, err := reader.Read(outputPath); err == nil {
			stats := readRes.Backup.ComputeStats(targetFormat, "."+targetFormat, time.Now())
			sourcesCount = stats.SourceCount
			categoriesCount = stats.CategoryCount
			historyCount = stats.HistoryCount
			categoriesList = tracker.ExtractCategoryCounts(readRes.Backup)

			for name, count := range stats.SourceBreakdown {
				extensions = append(extensions, ExtensionInfo{
					Name:  name,
					Count: count,
				})
			}

			sort.Slice(extensions, func(i, j int) bool {
				return extensions[i].Count > extensions[j].Count
			})
		}
	}

	extJSON, _ := json.Marshal(extensions)
	catJSON, _ := json.Marshal(categoriesList)

	outFilename := filepath.Base(outputPath)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", outFilename))
	w.Header().Set("X-Output-Filename", outFilename)
	w.Header().Set("X-Manga-Count", fmt.Sprintf("%d", res.MangaCount))
	w.Header().Set("X-Chapter-Count", fmt.Sprintf("%d", res.ChapterCount))
	w.Header().Set("X-Sources-Count", fmt.Sprintf("%d", sourcesCount))
	w.Header().Set("X-Categories-Count", fmt.Sprintf("%d", categoriesCount))
	w.Header().Set("X-History-Count", fmt.Sprintf("%d", historyCount))
	w.Header().Set("X-Extensions-Json", string(extJSON))
	w.Header().Set("X-Categories-Json", string(catJSON))

	http.ServeFile(w, r, outputPath)
}

func (s *Server) handleMerge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseMultipartForm(500 << 20); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse form: %v", err), http.StatusBadRequest)
		return
	}

	files := r.MultipartForm.File["files"]
	if len(files) < 2 {
		http.Error(w, "Merge requires at least 2 backup files", http.StatusBadRequest)
		return
	}

	targetFormat := strings.ToLower(strings.TrimSpace(r.FormValue("target_format")))
	if targetFormat == "" {
		targetFormat = "tachibk"
	}

	tempDir, err := os.MkdirTemp("", "mangastu-merge-*")
	if err != nil {
		http.Error(w, "Failed to create temp directory", http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tempDir)

	var inputPaths []string
	for idx, fh := range files {
		inputPath := filepath.Join(tempDir, fmt.Sprintf("input_%d_%s", idx, fh.Filename))
		src, err := fh.Open()
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to open file %s", fh.Filename), http.StatusBadRequest)
			return
		}
		dst, err := os.Create(inputPath)
		if err != nil {
			src.Close()
			http.Error(w, "Failed to create temp file", http.StatusInternalServerError)
			return
		}
		if _, err := io.Copy(dst, src); err != nil {
			src.Close()
			dst.Close()
			http.Error(w, "Failed to save file", http.StatusInternalServerError)
			return
		}
		src.Close()
		dst.Close()
		inputPaths = append(inputPaths, inputPath)
	}

	nowStr := time.Now().Format("2006-01-02_15-04")
	outputPath := filepath.Join(tempDir, fmt.Sprintf("merged_backup_%s.%s", nowStr, targetFormat))
	m := merger.New(s.registry)
	opts := merger.MergeOptions{
		Verbose: false,
	}

	res, err := m.Merge(inputPaths, outputPath, opts)
	if err != nil {
		http.Error(w, fmt.Sprintf("Merge failed: %v", err), http.StatusBadRequest)
		return
	}

	outFilename := filepath.Base(outputPath)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", outFilename))
	w.Header().Set("X-Output-Filename", outFilename)
	w.Header().Set("X-Total-Manga", fmt.Sprintf("%d", res.TotalInputManga))
	w.Header().Set("X-Unique-Manga", fmt.Sprintf("%d", res.UniqueManga))
	w.Header().Set("X-Overlap-Count", fmt.Sprintf("%d", res.MergedManga))
	w.Header().Set("X-Total-Chapters", fmt.Sprintf("%d", res.TotalChapters))
	w.Header().Set("X-Read-Chapters", fmt.Sprintf("%d", res.ReadChapters))
	w.Header().Set("X-History-Count", fmt.Sprintf("%d", res.HistoryCount))

	http.ServeFile(w, r, outputPath)
}

func (s *Server) handleInspect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseMultipartForm(500 << 20); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse form: %v", err), http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Missing file in upload", http.StatusBadRequest)
		return
	}
	defer file.Close()

	tempDir, err := os.MkdirTemp("", "mangastu-inspect-*")
	if err != nil {
		http.Error(w, "Failed to create temp directory", http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tempDir)

	inputPath := filepath.Join(tempDir, header.Filename)
	dst, err := os.Create(inputPath)
	if err != nil {
		http.Error(w, "Failed to save uploaded file", http.StatusInternalServerError)
		return
	}
	if _, err := io.Copy(dst, file); err != nil {
		dst.Close()
		http.Error(w, "Failed to write file", http.StatusInternalServerError)
		return
	}
	dst.Close()

	inspector := inspect.New(s.registry)
	report, err := inspector.Inspect(inputPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Inspect failed: %v", err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

func (s *Server) handleTrackers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseMultipartForm(500 << 20); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse form: %v", err), http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Missing file in upload", http.StatusBadRequest)
		return
	}
	defer file.Close()

	tempDir, err := os.MkdirTemp("", "mangastu-trackers-*")
	if err != nil {
		http.Error(w, "Failed to create temp directory", http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tempDir)

	inputPath := filepath.Join(tempDir, header.Filename)
	dst, err := os.Create(inputPath)
	if err != nil {
		http.Error(w, "Failed to save uploaded file", http.StatusInternalServerError)
		return
	}
	if _, err := io.Copy(dst, file); err != nil {
		dst.Close()
		http.Error(w, "Failed to write file", http.StatusInternalServerError)
		return
	}
	dst.Close()

	reader, err := s.registry.GetByExtension(inputPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Unsupported format: %v", err), http.StatusBadRequest)
		return
	}

	readRes, err := reader.Read(inputPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read backup: %v", err), http.StatusBadRequest)
		return
	}

	accounts := tracker.ExtractAccounts(readRes.Backup)
	trackedManga := tracker.ExtractTrackedManga(readRes.Backup)
	libraryManga := tracker.ExtractLibraryManga(readRes.Backup)
	categories := tracker.ExtractCategoryCounts(readRes.Backup)
	categoryNames := tracker.ExtractCategories(readRes.Backup)

	response := map[string]any{
		"accounts":       accounts,
		"tracked_manga":  trackedManga,
		"total_tracked":  len(trackedManga),
		"library_manga":  libraryManga,
		"total_library":  len(libraryManga),
		"categories":     categories,
		"category_names": categoryNames,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// trackingExportEntry is the small, client-owned part of a backup that can be
// safely changed by the tracking screen. The uploaded backup remains the source
// of truth for all other fields.
type trackingExportEntry struct {
	Title           string   `json:"title"`
	Source          int64    `json:"source"`
	DateAdded       int64    `json:"date_added"`
	MediaID         int64    `json:"media_id"`
	TrackerTitle    string   `json:"tracker_title"`
	TrackingURL     string   `json:"tracking_url"`
	Status          string   `json:"status"`
	Score           *float32 `json:"score"`
	LastChapterRead float32  `json:"last_chapter_read"`
	TotalChapters   int      `json:"total_chapters"`
}

// handleTrackingExport applies AniList matches selected in the web UI to the
// original uploaded backup and returns the same backup format for download.
func (s *Server) handleTrackingExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseMultipartForm(500 << 20); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse form: %v", err), http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Missing file in upload", http.StatusBadRequest)
		return
	}
	defer file.Close()

	var entries []trackingExportEntry
	if err := json.Unmarshal([]byte(r.FormValue("tracking_entries")), &entries); err != nil {
		http.Error(w, "Tracking changes could not be read", http.StatusBadRequest)
		return
	}

	tempDir, err := os.MkdirTemp("", "mangastu-tracking-export-*")
	if err != nil {
		http.Error(w, "Failed to create temp directory", http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tempDir)

	inputName := filepath.Base(header.Filename)
	inputPath := filepath.Join(tempDir, inputName)
	dst, err := os.Create(inputPath)
	if err != nil {
		http.Error(w, "Failed to save uploaded file", http.StatusInternalServerError)
		return
	}
	if _, err := io.Copy(dst, file); err != nil {
		dst.Close()
		http.Error(w, "Failed to write uploaded file", http.StatusInternalServerError)
		return
	}
	dst.Close()

	backupFormat, err := s.registry.GetByExtension(inputPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Unsupported format: %v", err), http.StatusBadRequest)
		return
	}
	readRes, err := backupFormat.Read(inputPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read backup: %v", err), http.StatusBadRequest)
		return
	}

	updates, additions := applyTrackingEntries(readRes.Backup, entries)
	extension := filepath.Ext(inputName)
	if extension == "" {
		extension = backupFormat.Extensions()[0]
	}
	baseName := strings.TrimSuffix(inputName, extension)
	outputName := fmt.Sprintf("%s_tracked%s", baseName, extension)
	outputPath := filepath.Join(tempDir, outputName)

	if err := backupFormat.Write(outputPath, readRes.Backup); err != nil {
		http.Error(w, fmt.Sprintf("Failed to write updated backup: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", outputName))
	w.Header().Set("X-Output-Filename", outputName)
	w.Header().Set("X-Tracking-Added", fmt.Sprintf("%d", additions))
	w.Header().Set("X-Tracking-Updated", fmt.Sprintf("%d", updates))
	http.ServeFile(w, r, outputPath)
}

func applyTrackingEntries(backup *core.Backup, entries []trackingExportEntry) (updated, added int) {
	for _, entry := range entries {
		if strings.TrimSpace(entry.Title) == "" || entry.MediaID <= 0 {
			continue
		}

		for mangaIndex := range backup.Manga {
			manga := &backup.Manga[mangaIndex]
			if manga.Title != entry.Title {
				continue
			}
			if entry.Source != 0 && manga.Source != entry.Source {
				continue
			}
			if entry.DateAdded != 0 && manga.DateAdded != entry.DateAdded {
				continue
			}

			trackIndex := -1
			for i, existing := range manga.Tracking {
				if existing.SyncID == tracker.SyncIDAniList {
					trackIndex = i
					break
				}
			}

			track := core.Tracking{
				SyncID:          tracker.SyncIDAniList,
				MediaID:         entry.MediaID,
				Title:           entry.TrackerTitle,
				TrackingURL:     entry.TrackingURL,
				LastChapterRead: entry.LastChapterRead,
				TotalChapters:   entry.TotalChapters,
				Status:          trackingStatusCode(entry.Status),
			}
			if track.Title == "" {
				track.Title = manga.Title
			}
			if track.TrackingURL == "" {
				track.TrackingURL = fmt.Sprintf("https://anilist.co/manga/%d", entry.MediaID)
			}

			if trackIndex >= 0 {
				existing := manga.Tracking[trackIndex]
				track.LibraryID = existing.LibraryID
				track.StartedReadingDate = existing.StartedReadingDate
				track.FinishedReadingDate = existing.FinishedReadingDate
				if entry.Score != nil {
					track.Score = *entry.Score
				} else {
					track.Score = existing.Score
				}
				manga.Tracking[trackIndex] = track
				updated++
			} else {
				if entry.Score != nil {
					track.Score = *entry.Score
				}
				manga.Tracking = append(manga.Tracking, track)
				added++
			}
			break
		}
	}

	return updated, added
}

func trackingStatusCode(status string) int {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "completed":
		return 2
	case "on hold", "on-hold", "on_hold", "paused":
		return 3
	case "dropped":
		return 4
	case "planning", "plan to read", "plan-to-read":
		return 5
	case "repeating":
		return 6
	default:
		return 1
	}
}

func (s *Server) handleStaticOrSPA(w http.ResponseWriter, r *http.Request) {
	// If staticDir doesn't exist, return simple status
	if _, err := os.Stat(s.staticDir); os.IsNotExist(err) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "MangaSTU API Server v0.1.0\nStatic directory %s not found. Build frontend with `npm run build`.", s.staticDir)
		return
	}

	path := filepath.Join(s.staticDir, filepath.Clean(r.URL.Path))
	info, err := os.Stat(path)

	// If file exists and is not a directory, serve it
	if err == nil && !info.IsDir() {
		http.ServeFile(w, r, path)
		return
	}

	// SPA fallback: serve index.html
	indexPath := filepath.Join(s.staticDir, "index.html")
	if _, err := os.Stat(indexPath); err == nil {
		http.ServeFile(w, r, indexPath)
		return
	}

	http.NotFound(w, r)
}

func (s *Server) handleSearchAniList(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		http.Error(w, "Query parameter q is required", http.StatusBadRequest)
		return
	}

	gqlBody, _ := json.Marshal(map[string]any{
		"query": `query ($search: String) {
			Page(page: 1, perPage: 8) {
				media(search: $search, type: MANGA) {
					id
					title {
						romaji
						english
						native
					}
					coverImage {
						medium
						large
					}
					format
					status
					chapters
					averageScore
					siteUrl
				}
			}
		}`,
		"variables": map[string]string{
			"search": query,
		},
	})

	client := &http.Client{Timeout: 8 * time.Second}
	req, err := http.NewRequest("POST", "https://graphql.anilist.co", bytes.NewBuffer(gqlBody))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("AniList API error: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
