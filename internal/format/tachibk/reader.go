package tachibk

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/misfit/mangastu/internal/core"
	"github.com/misfit/mangastu/internal/format"
	pb "github.com/misfit/mangastu/internal/format/tachibk/pb"
	"google.golang.org/protobuf/proto"
)

// Detect checks if a file is a tachibk backup by looking for gzip magic bytes.
func (t *TachiBK) Detect(path string) (bool, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".tachibk" {
		return true, nil
	}

	// Check for gzip magic bytes (0x1f 0x8b)
	f, err := os.Open(path)
	if err != nil {
		return false, nil
	}
	defer f.Close()

	magic := make([]byte, 2)
	if _, err := io.ReadFull(f, magic); err != nil {
		return false, nil
	}

	return magic[0] == 0x1f && magic[1] == 0x8b, nil
}

// Read parses a .tachibk backup file and returns the internal model.
func (t *TachiBK) Read(path string) (*format.ReadResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open tachibk file: %w", err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("failed to open gzip reader: %w", err)
	}
	defer gr.Close()

	data, err := io.ReadAll(gr)
	if err != nil {
		return nil, fmt.Errorf("failed to read decompressed data: %w", err)
	}

	pbBackup := &pb.Backup{}
	if err := proto.Unmarshal(data, pbBackup); err != nil {
		return nil, fmt.Errorf("failed to decode protobuf: %w", err)
	}

	backup := fromProto(pbBackup)

	// Try to get creation time from file modification time
	stat, _ := os.Stat(path)
	createdAt := time.Now()
	if stat != nil {
		createdAt = stat.ModTime()
	}

	return &format.ReadResult{
		Backup:    backup,
		CreatedAt: createdAt,
	}, nil
}

// fromProto converts protobuf structs to the internal model.
func fromProto(pbBackup *pb.Backup) *core.Backup {
	backup := &core.Backup{}

	// Convert categories
	for _, pbCat := range pbBackup.BackupCategories {
		backup.Categories = append(backup.Categories, core.Category{
			ID:    pbCat.GetId(),
			Name:  pbCat.GetName(),
			Order: pbCat.GetOrder(),
			Flags: pbCat.GetFlags(),
		})
	}

	// Convert preferences & tracker tokens
	for _, pbPref := range pbBackup.BackupPreferences {
		backup.Preferences = append(backup.Preferences, core.Preference{
			Key:       pbPref.GetKey(),
			Type:      pbPref.GetValue().GetType(),
			Value:     pbPref.GetValue().GetTruevalue(),
			StringVal: string(pbPref.GetValue().GetTruevalue()),
		})
	}

	// Convert sources
	for _, pbSrc := range pbBackup.BackupSources {
		backup.Sources = append(backup.Sources, core.Source{
			Name:     pbSrc.GetName(),
			SourceID: pbSrc.GetSourceId(),
		})
	}

	// Convert manga
	for _, pbManga := range pbBackup.BackupManga {
		m := core.Manga{
			Source:       pbManga.GetSource(),
			URL:          pbManga.GetUrl(),
			Title:        pbManga.GetTitle(),
			Artist:       pbManga.GetArtist(),
			Author:       pbManga.GetAuthor(),
			Description:  pbManga.GetDescription(),
			Genre:        pbManga.GetGenre(),
			Status:       int(pbManga.GetStatus()),
			ThumbnailURL: pbManga.GetThumbnailUrl(),
			DateAdded:    pbManga.GetDateAdded(),
			Favorite:     pbManga.GetFavorite(),
			ChapterFlags: int(pbManga.GetChapterFlags()),
			ViewerFlags:  int(pbManga.GetViewerFlags()),
			Categories:   pbManga.GetCategories(),
		}

		// In Komikku's kotlinx.serialization, favorite defaults to true.
		// proto2 optional bool defaults to false when not set.
		// If the field was not explicitly set, default to true (Komikku behavior).
		if pbManga.Favorite == nil {
			m.Favorite = true
		}

		// Map update strategy
		switch pbManga.GetUpdateStrategy() {
		case pb.UpdateStrategy_ONLY_FETCH_ONCE:
			m.UpdateStrategy = "ONLY_FETCH_ONCE"
		default:
			m.UpdateStrategy = "ALWAYS_UPDATE"
		}

		// Convert chapters
		for _, pbCh := range pbManga.GetChapters() {
			m.Chapters = append(m.Chapters, core.Chapter{
				URL:           pbCh.GetUrl(),
				Name:          pbCh.GetName(),
				Scanlator:     pbCh.GetScanlator(),
				Read:          pbCh.GetRead(),
				Bookmark:      pbCh.GetBookmark(),
				LastPageRead:  pbCh.GetLastPageRead(),
				DateFetch:     pbCh.GetDateFetch(),
				DateUpload:    pbCh.GetDateUpload(),
				ChapterNumber: pbCh.GetChapterNumber(),
				SourceOrder:   pbCh.GetSourceOrder(),
			})
		}

		// Convert tracking
		for _, pbTr := range pbManga.GetTracking() {
			mediaID := pbTr.GetMediaId()
			if mediaID == 0 {
				mediaID = int64(pbTr.GetMediaIdInt())
			}
			m.Tracking = append(m.Tracking, core.Tracking{
				SyncID:              int(pbTr.GetSyncId()),
				MediaID:             mediaID,
				LibraryID:           pbTr.GetLibraryId(),
				Title:               pbTr.GetTitle(),
				TrackingURL:         pbTr.GetTrackingUrl(),
				LastChapterRead:     pbTr.GetLastChapterRead(),
				TotalChapters:       int(pbTr.GetTotalChapters()),
				Score:               pbTr.GetScore(),
				Status:              int(pbTr.GetStatus()),
				StartedReadingDate:  pbTr.GetStartedReadingDate(),
				FinishedReadingDate: pbTr.GetFinishedReadingDate(),
			})
		}

		// Convert history
		for _, pbH := range pbManga.GetHistory() {
			m.History = append(m.History, core.History{
				ChapterURL:   pbH.GetUrl(),
				LastRead:     pbH.GetLastRead(),
				ReadDuration: pbH.GetReadDuration(),
			})
		}

		backup.Manga = append(backup.Manga, m)
	}

	return backup
}
