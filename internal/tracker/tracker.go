package tracker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/misfit/mangastu/internal/core"
)

// Service definition and constants.
const (
	SyncIDMyAnimeList  = 1
	SyncIDAniList      = 2
	SyncIDKitsu        = 3
	SyncIDShikimori    = 4
	SyncIDBangumi      = 5
	SyncIDMangaUpdates = 7 // Komikku / Tachiyomi uses 6 or 7
	SyncIDMangaDex     = 60
)

// ServiceName returns human-readable service name.
func ServiceName(syncID int) string {
	switch syncID {
	case SyncIDMyAnimeList:
		return "MyAnimeList"
	case SyncIDAniList:
		return "AniList"
	case SyncIDKitsu:
		return "Kitsu"
	case SyncIDShikimori:
		return "Shikimori"
	case SyncIDBangumi:
		return "Bangumi"
	case 6, SyncIDMangaUpdates:
		return "MangaUpdates"
	case SyncIDMangaDex:
		return "MangaDex"
	default:
		return fmt.Sprintf("Tracker Service %d", syncID)
	}
}

// ServiceColor returns brand color for UI badges.
func ServiceColor(syncID int) string {
	switch syncID {
	case SyncIDMyAnimeList:
		return "#2e51a2"
	case SyncIDAniList:
		return "#02a9ff"
	case SyncIDKitsu:
		return "#e4405f"
	case SyncIDShikimori:
		return "#4b5563"
	case SyncIDBangumi:
		return "#f09199"
	case 6, SyncIDMangaUpdates:
		return "#ff6600"
	case SyncIDMangaDex:
		return "#ff6740"
	default:
		return "#0ea5e9"
	}
}

// TrackerAccount represents an authenticated tracker session found in the backup.
type TrackerAccount struct {
	SyncID       int       `json:"sync_id"`
	ServiceName  string    `json:"service_name"`
	ServiceColor string    `json:"service_color"`
	Username     string    `json:"username,omitempty"`
	AccessToken  string    `json:"access_token,omitempty"`
	TokenType    string    `json:"token_type,omitempty"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
	IsExpired    bool      `json:"is_expired"`
	ScoreType    string    `json:"score_type,omitempty"`
	TrackedCount int       `json:"tracked_count"`
	TokenPreview string    `json:"token_preview,omitempty"`
}

// TrackedMangaItem represents a manga entry linked to an external tracker.
type TrackedMangaItem struct {
	MangaTitle      string  `json:"manga_title"`
	TrackerTitle    string  `json:"tracker_title"`
	SyncID          int     `json:"sync_id"`
	ServiceName     string  `json:"service_name"`
	ServiceColor    string  `json:"service_color"`
	MediaID         int64   `json:"media_id"`
	LastChapterRead float32 `json:"last_chapter_read"`
	TotalChapters   int     `json:"total_chapters"`
	Score           float32 `json:"score"`
	Status          string  `json:"status"`
	TrackingURL     string  `json:"tracking_url"`
}

type TokenPayload struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Expires     int64  `json:"expires"`
	ExpiresIn   int64  `json:"expires_in"`
}

// ExtractAccounts discovers all tracker tokens, user credentials, and active links.
func ExtractAccounts(backup *core.Backup) []TrackerAccount {
	accountsMap := make(map[int]*TrackerAccount)

	// Count tracked titles per service
	trackedCounts := make(map[int]int)
	for _, m := range backup.Manga {
		for _, t := range m.Tracking {
			trackedCounts[t.SyncID]++
		}
	}

	// Initialize known services with tracked manga
	for syncID, count := range trackedCounts {
		accountsMap[syncID] = &TrackerAccount{
			SyncID:       syncID,
			ServiceName:  ServiceName(syncID),
			ServiceColor: ServiceColor(syncID),
			TrackedCount: count,
		}
	}

	// Parse backup preferences for credentials & tokens
	for _, pref := range backup.Preferences {
		key := pref.Key

		// Check __PRIVATE_track_token_<id>
		if strings.HasPrefix(key, "__PRIVATE_track_token_") {
			idStr := strings.TrimPrefix(key, "__PRIVATE_track_token_")
			if id, err := strconv.Atoi(idStr); err == nil {
				acc, exists := accountsMap[id]
				if !exists {
					acc = &TrackerAccount{
						SyncID:       id,
						ServiceName:  ServiceName(id),
						ServiceColor: ServiceColor(id),
					}
					accountsMap[id] = acc
				}

				// Extract JSON substring if wrapped with protobuf tags
				jsonStr := pref.StringVal
				startIdx := strings.Index(jsonStr, "{")
				endIdx := strings.LastIndex(jsonStr, "}")
				if startIdx != -1 && endIdx != -1 && endIdx > startIdx {
					jsonStr = jsonStr[startIdx : endIdx+1]
				}

				var tok TokenPayload
				if err := json.Unmarshal([]byte(jsonStr), &tok); err == nil {
					acc.AccessToken = tok.AccessToken
					acc.TokenType = tok.TokenType
					if tok.Expires > 0 {
						acc.ExpiresAt = time.UnixMilli(tok.Expires)
						acc.IsExpired = time.Now().After(acc.ExpiresAt)
					}
				}

				if len(acc.AccessToken) > 16 {
					acc.TokenPreview = acc.AccessToken[:12] + "..." + acc.AccessToken[len(acc.AccessToken)-4:]
				} else if len(acc.AccessToken) > 0 {
					acc.TokenPreview = "Valid Session"
				}
			}
		}

		// Check __PRIVATE_pref_mangasync_username_<id>
		if strings.HasPrefix(key, "__PRIVATE_pref_mangasync_username_") {
			idStr := strings.TrimPrefix(key, "__PRIVATE_pref_mangasync_username_")
			if id, err := strconv.Atoi(idStr); err == nil {
				acc, exists := accountsMap[id]
				if !exists {
					acc = &TrackerAccount{
						SyncID:       id,
						ServiceName:  ServiceName(id),
						ServiceColor: ServiceColor(id),
					}
					accountsMap[id] = acc
				}
				// Clean non-printable protobuf tag bytes
				cleanUser := strings.TrimFunc(pref.StringVal, func(r rune) bool {
					return r < 32 || r > 126
				})
				acc.Username = strings.TrimSpace(cleanUser)
			}
		}

		// Check score types (e.g. anilist_score_type)
		if strings.HasSuffix(key, "_score_type") {
			svcName := strings.TrimSuffix(key, "_score_type")
			cleanScore := strings.TrimFunc(pref.StringVal, func(r rune) bool {
				return r < 32 || r > 126
			})
			for _, acc := range accountsMap {
				if strings.EqualFold(acc.ServiceName, svcName) {
					acc.ScoreType = strings.TrimSpace(cleanScore)
				}
			}
		}
	}

	var results []TrackerAccount
	for _, acc := range accountsMap {
		if acc.SyncID == SyncIDAniList {
			resolveAniListUsername(acc)
		}
		results = append(results, *acc)
	}

	return results
}

// resolveAniListUsername fetches the human-readable username from AniList GraphQL
func resolveAniListUsername(acc *TrackerAccount) {
	client := &http.Client{Timeout: 3 * time.Second}
	var query string
	var variables map[string]any

	if acc.AccessToken != "" {
		query = `query { Viewer { id name } }`
	} else if acc.Username != "" {
		if id, err := strconv.Atoi(acc.Username); err == nil {
			query = `query ($id: Int) { User(id: $id) { id name } }`
			variables = map[string]any{"id": id}
		}
	}

	if query == "" {
		return
	}

	body, _ := json.Marshal(map[string]any{
		"query":     query,
		"variables": variables,
	})

	req, err := http.NewRequest("POST", "https://graphql.anilist.co", bytes.NewBuffer(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if acc.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+acc.AccessToken)
	}

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		var gqlRes struct {
			Data struct {
				Viewer *struct {
					Name string `json:"name"`
				} `json:"Viewer"`
				User *struct {
					Name string `json:"name"`
				} `json:"User"`
			} `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&gqlRes); err == nil {
			if gqlRes.Data.Viewer != nil && gqlRes.Data.Viewer.Name != "" {
				acc.Username = gqlRes.Data.Viewer.Name
			} else if gqlRes.Data.User != nil && gqlRes.Data.User.Name != "" {
				acc.Username = gqlRes.Data.User.Name
			}
		}
	}
}

// ExtractTrackedManga extracts all individual series tracked across services.
func ExtractTrackedManga(backup *core.Backup) []TrackedMangaItem {
	var items []TrackedMangaItem

	statusLabels := map[int]string{
		1: "Reading",
		2: "Completed",
		3: "On Hold",
		4: "Dropped",
		5: "Plan to Read",
		6: "Repeating",
	}

	for _, m := range backup.Manga {
		for _, t := range m.Tracking {
			statusStr := statusLabels[t.Status]
			if statusStr == "" {
				statusStr = "Active"
			}

			items = append(items, TrackedMangaItem{
				MangaTitle:      m.Title,
				TrackerTitle:    t.Title,
				SyncID:          t.SyncID,
				ServiceName:     ServiceName(t.SyncID),
				ServiceColor:    ServiceColor(t.SyncID),
				MediaID:         t.MediaID,
				LastChapterRead: t.LastChapterRead,
				TotalChapters:   t.TotalChapters,
				Score:           t.Score,
				Status:          statusStr,
				TrackingURL:     t.TrackingURL,
			})
		}
	}

	return items
}

// CategoryCountItem represents a category name and the number of entries in it.
type CategoryCountItem struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// LibraryMangaItem represents an entry from the library with full chapter & tracking details.
type LibraryMangaItem struct {
	Title           string             `json:"title"`
	Artist          string             `json:"artist,omitempty"`
	Author          string             `json:"author,omitempty"`
	Source          int64              `json:"source"`
	TotalChapters   int                `json:"total_chapters"`
	ReadChapters    int                `json:"read_chapters"`
	UnreadChapters  int                `json:"unread_chapters"`
	LastReadChapter float32            `json:"last_read_chapter"`
	DateAdded       int64              `json:"date_added"`
	Favorite        bool               `json:"favorite"`
	Status          string             `json:"status"`
	Categories      []string           `json:"categories,omitempty"`
	Tracking        []TrackedMangaItem `json:"tracking,omitempty"`
	IsTracked       bool               `json:"is_tracked"`
}

// ExtractCategoryCounts returns all categories with their manga counts for library entries.
func ExtractCategoryCounts(backup *core.Backup) []CategoryCountItem {
	catNameMap := make(map[int64]string)
	var orderedNames []string
	seen := make(map[string]bool)

	for idx, c := range backup.Categories {
		name := strings.TrimSpace(c.Name)
		if name != "" {
			catNameMap[int64(idx)] = name
			catNameMap[c.ID] = name
			catNameMap[c.Order] = name
			if !seen[name] {
				seen[name] = true
				orderedNames = append(orderedNames, name)
			}
		}
	}

	hasFavorites := false
	for _, m := range backup.Manga {
		if m.Favorite {
			hasFavorites = true
			break
		}
	}

	counts := make(map[string]int)
	for _, m := range backup.Manga {
		if hasFavorites && !m.Favorite {
			continue
		}
		mangaSeen := make(map[string]bool)
		for _, cid := range m.Categories {
			if name, ok := catNameMap[cid]; ok && name != "" && !mangaSeen[name] {
				mangaSeen[name] = true
				counts[name]++
			}
		}
	}

	var results []CategoryCountItem
	for _, name := range orderedNames {
		results = append(results, CategoryCountItem{
			Name:  name,
			Count: counts[name],
		})
	}
	return results
}

// ExtractCategories returns a list of all distinct category names in the backup.
func ExtractCategories(backup *core.Backup) []string {
	var cats []string
	seen := make(map[string]bool)
	for _, c := range backup.Categories {
		name := strings.TrimSpace(c.Name)
		if name != "" && !seen[name] {
			seen[name] = true
			cats = append(cats, name)
		}
	}
	return cats
}

// ExtractLibraryManga returns all library manga in the backup with their read progress, categories, and tracking status, sorted by DateAdded DESC.
func ExtractLibraryManga(backup *core.Backup) []LibraryMangaItem {
	var results []LibraryMangaItem

	statusMap := map[int]string{
		0: "Unknown",
		1: "Ongoing",
		2: "Completed",
		3: "Licensed",
		4: "Publishing Finished",
		5: "Cancelled",
		6: "On Hiatus",
	}
	trackingStatusMap := map[int]string{
		1: "Reading",
		2: "Completed",
		3: "On Hold",
		4: "Dropped",
		5: "Planning",
		6: "Repeating",
	}

	catNameMap := make(map[int64]string)
	for idx, c := range backup.Categories {
		name := strings.TrimSpace(c.Name)
		if name != "" {
			catNameMap[int64(idx)] = name
			catNameMap[c.ID] = name
			catNameMap[c.Order] = name
		}
	}

	hasFavorites := false
	for _, m := range backup.Manga {
		if m.Favorite {
			hasFavorites = true
			break
		}
	}

	for _, m := range backup.Manga {
		if hasFavorites && !m.Favorite {
			continue
		}

		readCount := 0
		var maxRead float32 = 0

		for _, ch := range m.Chapters {
			if ch.Read {
				readCount++
				if ch.ChapterNumber > maxRead {
					maxRead = ch.ChapterNumber
				}
			}
		}

		var trackingList []TrackedMangaItem
		for _, t := range m.Tracking {
			trackStatus := trackingStatusMap[t.Status]
			if trackStatus == "" {
				trackStatus = "Reading"
			}
			trackingList = append(trackingList, TrackedMangaItem{
				MangaTitle:      m.Title,
				TrackerTitle:    t.Title,
				SyncID:          t.SyncID,
				ServiceName:     ServiceName(t.SyncID),
				ServiceColor:    ServiceColor(t.SyncID),
				MediaID:         t.MediaID,
				LastChapterRead: t.LastChapterRead,
				TotalChapters:   t.TotalChapters,
				Score:           t.Score,
				Status:          trackStatus,
				TrackingURL:     t.TrackingURL,
			})
		}

		statusStr := statusMap[m.Status]
		if statusStr == "" {
			statusStr = "Ongoing"
		}

		var mangaCats []string
		for _, catIdx := range m.Categories {
			if name, ok := catNameMap[catIdx]; ok && name != "" {
				mangaCats = append(mangaCats, name)
			}
		}

		totalCh := len(m.Chapters)
		unreadCh := totalCh - readCount
		if unreadCh < 0 {
			unreadCh = 0
		}

		results = append(results, LibraryMangaItem{
			Title:           m.Title,
			Artist:          m.Artist,
			Author:          m.Author,
			Source:          m.Source,
			TotalChapters:   totalCh,
			ReadChapters:    readCount,
			UnreadChapters:  unreadCh,
			LastReadChapter: maxRead,
			DateAdded:       m.DateAdded,
			Favorite:        m.Favorite,
			Status:          statusStr,
			Categories:      mangaCats,
			Tracking:        trackingList,
			IsTracked:       len(trackingList) > 0,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].DateAdded > results[j].DateAdded
	})

	return results
}
