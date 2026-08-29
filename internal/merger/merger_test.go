package merger

import (
	"testing"

	"github.com/misfit/mangastu/internal/core"
)

func TestMergeBackups(t *testing.T) {
	b1 := &core.Backup{
		Categories: []core.Category{
			{ID: 1, Name: "Action", Order: 0},
			{ID: 2, Name: "Fantasy", Order: 1},
		},
		Sources: []core.Source{
			{SourceID: 100, Name: "SourceA"},
		},
		Manga: []core.Manga{
			{
				Source:     100,
				URL:        "/manga/solo-leveling",
				Title:      "Solo Leveling",
				Favorite:   true,
				Categories: []int64{0}, // Action
				Chapters: []core.Chapter{
					{URL: "/ch/1", Name: "Chapter 1", Read: true, LastPageRead: 10, LastReadAt: 1000},
					{URL: "/ch/2", Name: "Chapter 2", Read: false, LastPageRead: 0, LastReadAt: 0},
				},
				Tracking: []core.Tracking{
					{SyncID: 1, Title: "Solo Leveling", LastChapterRead: 1, Score: 9.0},
				},
				History: []core.History{
					{ChapterURL: "/ch/1", LastRead: 1000, ReadDuration: 300},
				},
			},
			{
				Source:   100,
				URL:      "/manga/one-piece",
				Title:    "One Piece",
				Favorite: true,
				Chapters: []core.Chapter{
					{URL: "/op/1", Name: "Chapter 1", Read: true},
				},
			},
		},
	}

	b2 := &core.Backup{
		Categories: []core.Category{
			{ID: 1, Name: "Action", Order: 0},
			{ID: 2, Name: "Shounen", Order: 1},
		},
		Sources: []core.Source{
			{SourceID: 100, Name: "SourceA"},
			{SourceID: 200, Name: "SourceB"},
		},
		Manga: []core.Manga{
			{
				Source:     100,
				URL:        "/manga/solo-leveling",
				Title:      "Solo Leveling",
				Favorite:   false,
				Categories: []int64{1}, // Shounen
				Chapters: []core.Chapter{
					{URL: "/ch/1", Name: "Chapter 1", Read: true, LastPageRead: 25, LastReadAt: 2000},
					{URL: "/ch/2", Name: "Chapter 2", Read: true, LastPageRead: 15, LastReadAt: 2500},
					{URL: "/ch/3", Name: "Chapter 3", Read: false, LastPageRead: 0, LastReadAt: 0},
				},
				Tracking: []core.Tracking{
					{SyncID: 1, Title: "Solo Leveling", LastChapterRead: 2, Score: 9.5},
				},
				History: []core.History{
					{ChapterURL: "/ch/1", LastRead: 2000, ReadDuration: 600},
					{ChapterURL: "/ch/2", LastRead: 2500, ReadDuration: 400},
				},
			},
			{
				Source:   200,
				URL:      "/manga/naruto",
				Title:    "Naruto",
				Favorite: true,
				Chapters: []core.Chapter{
					{URL: "/naruto/1", Name: "Chapter 1", Read: true},
				},
			},
		},
	}

	merged, overlapCount := mergeBackups([]*core.Backup{b1, b2})

	if overlapCount != 1 {
		t.Fatalf("expected 1 overlapping manga, got %d", overlapCount)
	}

	if len(merged.Manga) != 3 {
		t.Fatalf("expected 3 total unique manga, got %d", len(merged.Manga))
	}

	// Verify categories deduplication
	if len(merged.Categories) != 3 { // Action, Fantasy, Shounen
		t.Fatalf("expected 3 unique categories, got %d", len(merged.Categories))
	}

	// Verify merged Solo Leveling
	var solo *core.Manga
	for i := range merged.Manga {
		if merged.Manga[i].URL == "/manga/solo-leveling" {
			solo = &merged.Manga[i]
			break
		}
	}
	if solo == nil {
		t.Fatalf("Solo Leveling not found in merged manga")
	}

	if !solo.Favorite {
		t.Errorf("expected Solo Leveling Favorite=true")
	}

	if len(solo.Chapters) != 3 {
		t.Fatalf("expected 3 merged chapters for Solo Leveling, got %d", len(solo.Chapters))
	}

	// Chapter 1 reconciliation
	ch1 := solo.Chapters[0]
	if !ch1.Read || ch1.LastPageRead != 25 || ch1.LastReadAt != 2000 {
		t.Errorf("chapter 1 reconciled incorrectly: Read=%v, LastPageRead=%d, LastReadAt=%d", ch1.Read, ch1.LastPageRead, ch1.LastReadAt)
	}

	// Chapter 2 reconciliation (was unread in b1, read in b2)
	ch2 := solo.Chapters[1]
	if !ch2.Read || ch2.LastPageRead != 15 || ch2.LastReadAt != 2500 {
		t.Errorf("chapter 2 reconciled incorrectly: Read=%v, LastPageRead=%d, LastReadAt=%d", ch2.Read, ch2.LastPageRead, ch2.LastReadAt)
	}

	// Tracking reconciliation
	if len(solo.Tracking) != 1 || solo.Tracking[0].LastChapterRead != 2 || solo.Tracking[0].Score != 9.5 {
		t.Errorf("tracking reconciled incorrectly: %+v", solo.Tracking)
	}

	// History reconciliation
	if len(solo.History) != 2 {
		t.Fatalf("expected 2 history entries, got %d", len(solo.History))
	}
}
