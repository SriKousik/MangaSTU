package tachibk

import (
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/misfit/mangastu/internal/core"
	pb "github.com/misfit/mangastu/internal/format/tachibk/pb"
	"google.golang.org/protobuf/proto"
)

// Write serializes a Backup to a .tachibk file (gzip-compressed protobuf).
func (t *TachiBK) Write(path string, backup *core.Backup) error {
	pbBackup := toProto(backup)

	data, err := proto.Marshal(pbBackup)
	if err != nil {
		return fmt.Errorf("failed to marshal protobuf: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	if _, err := gw.Write(data); err != nil {
		return fmt.Errorf("failed to write gzip data: %w", err)
	}
	if err := gw.Close(); err != nil {
		return fmt.Errorf("failed to close gzip writer: %w", err)
	}

	return nil
}

// toProto converts the internal model to protobuf structs.
func toProto(backup *core.Backup) *pb.Backup {
	pbBackup := &pb.Backup{}

	// Convert categories
	for _, cat := range backup.Categories {
		pbBackup.BackupCategories = append(pbBackup.BackupCategories, &pb.BackupCategory{
			Name:  proto.String(cat.Name),
			Order: proto.Int64(cat.Order),
			Id:    proto.Int64(cat.ID),
			Flags: proto.Int64(cat.Flags),
		})
	}

	// Convert sources
	for _, src := range backup.Sources {
		pbBackup.BackupSources = append(pbBackup.BackupSources, &pb.BackupSource{
			Name:     proto.String(src.Name),
			SourceId: proto.Int64(src.SourceID),
		})
	}

	// Convert preferences & tracker tokens
	for _, pref := range backup.Preferences {
		prefType := pref.Type
		if prefType == "" {
			prefType = "eu.kanade.tachiyomi.data.backup.models.StringPreferenceValue"
		}
		pbBackup.BackupPreferences = append(pbBackup.BackupPreferences, &pb.BackupPreference{
			Key: proto.String(pref.Key),
			Value: &pb.PreferenceValue{
				Type:      proto.String(prefType),
				Truevalue: pref.Value,
			},
		})
	}

	// Convert manga
	for _, m := range backup.Manga {
		// Map update strategy
		updateStrategy := pb.UpdateStrategy_ALWAYS_UPDATE
		if strings.ToUpper(m.UpdateStrategy) == "ONLY_FETCH_ONCE" {
			updateStrategy = pb.UpdateStrategy_ONLY_FETCH_ONCE
		}

		pbManga := &pb.BackupManga{
			Source:         proto.Int64(m.Source),
			Url:            proto.String(m.URL),
			Title:          proto.String(m.Title),
			Artist:         proto.String(m.Artist),
			Author:         proto.String(m.Author),
			Description:    proto.String(m.Description),
			Genre:          m.Genre,
			Status:         proto.Int32(int32(m.Status)),
			ThumbnailUrl:   proto.String(m.ThumbnailURL),
			DateAdded:      proto.Int64(m.DateAdded),
			Favorite:       proto.Bool(m.Favorite),
			ChapterFlags:   proto.Int32(int32(m.ChapterFlags)),
			ViewerFlags:    proto.Int32(int32(m.ViewerFlags)),
			Categories:     m.Categories,
			UpdateStrategy: &updateStrategy,
		}

		// Convert chapters
		for _, ch := range m.Chapters {
			pbManga.Chapters = append(pbManga.Chapters, &pb.BackupChapter{
				Url:           proto.String(ch.URL),
				Name:          proto.String(ch.Name),
				Scanlator:     proto.String(ch.Scanlator),
				Read:          proto.Bool(ch.Read),
				Bookmark:      proto.Bool(ch.Bookmark),
				LastPageRead:  proto.Int64(ch.LastPageRead),
				DateFetch:     proto.Int64(ch.DateFetch),
				DateUpload:    proto.Int64(ch.DateUpload),
				ChapterNumber: proto.Float32(ch.ChapterNumber),
				SourceOrder:   proto.Int64(ch.SourceOrder),
			})
		}

		// Convert tracking
		for _, tr := range m.Tracking {
			pbManga.Tracking = append(pbManga.Tracking, &pb.BackupTracking{
				SyncId:              proto.Int32(int32(tr.SyncID)),
				MediaId:             proto.Int64(tr.MediaID),
				LibraryId:           proto.Int64(tr.LibraryID),
				Title:               proto.String(tr.Title),
				TrackingUrl:         proto.String(tr.TrackingURL),
				LastChapterRead:     proto.Float32(tr.LastChapterRead),
				TotalChapters:       proto.Int32(int32(tr.TotalChapters)),
				Score:               proto.Float32(tr.Score),
				Status:              proto.Int32(int32(tr.Status)),
				StartedReadingDate:  proto.Int64(tr.StartedReadingDate),
				FinishedReadingDate: proto.Int64(tr.FinishedReadingDate),
			})
		}

		// Convert history
		for _, h := range m.History {
			pbManga.History = append(pbManga.History, &pb.BackupHistory{
				Url:          proto.String(h.ChapterURL),
				LastRead:     proto.Int64(h.LastRead),
				ReadDuration: proto.Int64(h.ReadDuration),
			})
		}

		pbBackup.BackupManga = append(pbBackup.BackupManga, pbManga)
	}

	return pbBackup
}
