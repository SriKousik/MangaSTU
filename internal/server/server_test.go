package server

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/misfit/mangastu/internal/core"
	"github.com/misfit/mangastu/internal/format/tachibk"
	"github.com/misfit/mangastu/internal/tracker"
)

func TestApplyTrackingEntriesAddsAndUpdatesAniListTracking(t *testing.T) {
	backup := &core.Backup{Manga: []core.Manga{
		{
			Title:     "New match",
			Source:    101,
			DateAdded: 1000,
			Chapters:  []core.Chapter{{ChapterNumber: 7, Read: true}},
		},
		{
			Title:     "Existing match",
			Source:    202,
			DateAdded: 2000,
			Tracking: []core.Tracking{{
				SyncID:             tracker.SyncIDAniList,
				MediaID:            10,
				Title:              "Old title",
				Score:              5,
				LibraryID:          44,
				StartedReadingDate: 900,
			}},
		},
	}}

	score := float32(9.5)
	updated, added := applyTrackingEntries(backup, []trackingExportEntry{
		{
			Title:           "New match",
			Source:          101,
			DateAdded:       1000,
			MediaID:         111,
			TrackerTitle:    "Matched title",
			LastChapterRead: 7,
			TotalChapters:   12,
			Status:          "Planning",
		},
		{
			Title:           "Existing match",
			Source:          202,
			DateAdded:       2000,
			MediaID:         222,
			TrackerTitle:    "Updated title",
			TrackingURL:     "https://anilist.co/manga/222",
			Status:          "Completed",
			Score:           &score,
			LastChapterRead: 12,
			TotalChapters:   12,
		},
	})

	if added != 1 || updated != 1 {
		t.Fatalf("expected 1 added and 1 updated tracking entry, got %d added and %d updated", added, updated)
	}

	newTrack := backup.Manga[0].Tracking[0]
	if newTrack.SyncID != tracker.SyncIDAniList || newTrack.MediaID != 111 || newTrack.Status != 5 {
		t.Fatalf("new AniList track was not written correctly: %+v", newTrack)
	}
	if newTrack.TrackingURL != "https://anilist.co/manga/111" {
		t.Fatalf("expected generated AniList URL, got %q", newTrack.TrackingURL)
	}

	updatedTrack := backup.Manga[1].Tracking[0]
	if updatedTrack.MediaID != 222 || updatedTrack.Score != score || updatedTrack.Status != 2 {
		t.Fatalf("existing AniList track was not updated correctly: %+v", updatedTrack)
	}
	if updatedTrack.LibraryID != 44 || updatedTrack.StartedReadingDate != 900 {
		t.Fatalf("existing track fields should be preserved: %+v", updatedTrack)
	}
}

func TestHandleTrackingExportReturnsBackupWithNewAniListEntry(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "library.tachibk")
	input := &core.Backup{Manga: []core.Manga{{
		Title:     "A linked title",
		Source:    123,
		DateAdded: 456,
		Favorite:  true,
		Chapters:  []core.Chapter{{ChapterNumber: 4, Read: true}},
	}}}
	if err := (&tachibk.TachiBK{}).Write(inputPath, input); err != nil {
		t.Fatalf("write input backup: %v", err)
	}

	inputData, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatalf("read input backup: %v", err)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	fileWriter, err := writer.CreateFormFile("file", "library.tachibk")
	if err != nil {
		t.Fatalf("create upload file: %v", err)
	}
	if _, err := fileWriter.Write(inputData); err != nil {
		t.Fatalf("write upload file: %v", err)
	}
	entries, err := json.Marshal([]trackingExportEntry{{
		Title:           "A linked title",
		Source:          123,
		DateAdded:       456,
		MediaID:         789,
		TrackerTitle:    "AniList title",
		Status:          "Reading",
		LastChapterRead: 4,
		TotalChapters:   12,
	}})
	if err != nil {
		t.Fatalf("marshal tracking entries: %v", err)
	}
	if err := writer.WriteField("tracking_entries", string(entries)); err != nil {
		t.Fatalf("write tracking entries: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	server := New(0, "")
	req := httptest.NewRequest(http.MethodPost, "/api/tracking/export", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()
	server.handleTrackingExport(response, req)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	if got := response.Header().Get("X-Output-Filename"); got != "library_tracked.tachibk" {
		t.Fatalf("unexpected output filename: %q", got)
	}
	if got := response.Header().Get("X-Tracking-Added"); got != "1" {
		t.Fatalf("expected one added entry, got %q", got)
	}

	outputPath := filepath.Join(tempDir, "library_tracked.tachibk")
	if err := os.WriteFile(outputPath, response.Body.Bytes(), 0o600); err != nil {
		t.Fatalf("write returned backup: %v", err)
	}
	reader := &tachibk.TachiBK{}
	result, err := reader.Read(outputPath)
	if err != nil {
		t.Fatalf("read returned backup: %v", err)
	}
	if len(result.Backup.Manga) != 1 || len(result.Backup.Manga[0].Tracking) != 1 {
		t.Fatalf("expected one manga with one tracking entry: %+v", result.Backup)
	}
	track := result.Backup.Manga[0].Tracking[0]
	if track.MediaID != 789 || track.Title != "AniList title" || track.LastChapterRead != 4 {
		t.Fatalf("unexpected returned tracking entry: %+v", track)
	}

}
