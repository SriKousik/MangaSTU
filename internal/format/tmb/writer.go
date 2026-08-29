package tmb

import (
	"archive/zip"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/misfit/mangastu/internal/core"
)

const (
	tmbVersion = 491
)

// ddlSchema defines the complete SQLite schema matching Tachimanga database v491.
const ddlSchema = `
CREATE TABLE IF NOT EXISTS Migrations (version INT NOT NULL PRIMARY KEY, "name" VARCHAR(400) NOT NULL, executed_at TEXT NOT NULL);
CREATE UNIQUE INDEX IF NOT EXISTS Migrations_name ON Migrations ("name");

CREATE TABLE IF NOT EXISTS Extension (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    apk_name VARCHAR(1024) NOT NULL,
    icon_url VARCHAR(2048) DEFAULT 'https://raw.githubusercontent.com/tachiyomiorg/tachiyomi/64ba127e7d43b1d7e6d58a6f5c9b2bd5fe0543f7/app/src/main/res/mipmap-xxxhdpi/ic_local_source.webp' NOT NULL,
    "name" VARCHAR(128) NOT NULL,
    pkg_name VARCHAR(128) NOT NULL,
    version_name VARCHAR(16) NOT NULL,
    version_code INT NOT NULL,
    lang VARCHAR(32) NOT NULL,
    is_nsfw BOOLEAN NOT NULL,
    is_installed BOOLEAN DEFAULT 0 NOT NULL,
    has_update BOOLEAN DEFAULT 0 NOT NULL,
    is_obsolete BOOLEAN DEFAULT 0 NOT NULL,
    class_name VARCHAR(1024) DEFAULT '' NOT NULL,
    pkg_factory VARCHAR(128) NULL,
    has_changelog BOOLEAN DEFAULT 0 NOT NULL,
    has_readme BOOLEAN DEFAULT 0 NOT NULL,
    repo_id INT DEFAULT 0 NOT NULL,
    "create_at" BIGINT NOT NULL DEFAULT 0,
    "update_at" BIGINT NOT NULL DEFAULT 0,
    "dirty" BOOLEAN NOT NULL DEFAULT 0,
    "commit_id" BIGINT NOT NULL DEFAULT 0,
    "base_url" VARCHAR(2048) NULL,
    "alternate_name" VARCHAR(256) NULL,
    "install_path_type" VARCHAR(32) NULL,
    "apk_url" VARCHAR(2048) NULL,
    "jar_url" VARCHAR(2048) NULL,
    "content_warning" INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS Extension_idx_dirty ON Extension (id, dirty);

CREATE TABLE IF NOT EXISTS "Source" (
    id BIGINT NOT NULL,
    "name" VARCHAR(128) NOT NULL,
    lang VARCHAR(32) NOT NULL,
    extension INT NOT NULL,
    is_nsfw BOOLEAN DEFAULT 0 NOT NULL,
    is_direct BOOLEAN DEFAULT NULL,
    random_ua BOOLEAN DEFAULT NULL,
    CONSTRAINT fk_Source_extension__id FOREIGN KEY (extension) REFERENCES Extension(id) ON DELETE RESTRICT ON UPDATE RESTRICT
);

CREATE TABLE IF NOT EXISTS Category (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    "name" VARCHAR(64) NOT NULL,
    "order" INT DEFAULT 0 NOT NULL,
    is_default BOOLEAN DEFAULT 0 NOT NULL,
    "uuid" VARCHAR(128) NULL,
    "create_at" BIGINT NOT NULL DEFAULT 0,
    "update_at" BIGINT NOT NULL DEFAULT 0,
    "is_delete" BOOLEAN NOT NULL DEFAULT 0,
    "dirty" BOOLEAN NOT NULL DEFAULT 0,
    "commit_id" BIGINT NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS Category_idx_dirty ON Category (id, dirty);

CREATE TABLE IF NOT EXISTS Manga (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    url VARCHAR(2048) NOT NULL,
    title VARCHAR(512) NOT NULL,
    initialized BOOLEAN DEFAULT 0 NOT NULL,
    artist VARCHAR(512) NULL,
    author VARCHAR(512) NULL,
    description VARCHAR(2147483647) NULL,
    genre VARCHAR(2147483647) NULL,
    status INT DEFAULT 0 NOT NULL,
    thumbnail_url VARCHAR(2048) NULL,
    thumbnail_url_last_fetched BIGINT DEFAULT 0 NOT NULL,
    in_library BOOLEAN DEFAULT 0 NOT NULL,
    default_category BOOLEAN DEFAULT 1 NOT NULL,
    in_library_at BIGINT DEFAULT 0 NOT NULL,
    "source" BIGINT NOT NULL,
    real_url VARCHAR(2048) NULL,
    last_fetched_at BIGINT DEFAULT 0 NOT NULL,
    chapters_last_fetched_at BIGINT DEFAULT 0 NOT NULL,
    update_strategy VARCHAR(256) DEFAULT 'ALWAYS_UPDATE' NOT NULL,
    last_download_at BIGINT DEFAULT 0 NOT NULL,
    "create_at" BIGINT NOT NULL DEFAULT 0,
    "update_at" BIGINT NOT NULL DEFAULT 0,
    "dirty" BOOLEAN NOT NULL DEFAULT 0,
    "commit_id" BIGINT NOT NULL DEFAULT 0,
    "memo" TEXT NOT NULL DEFAULT '{}'
);
CREATE INDEX IF NOT EXISTS Manga_idx_dirty ON Manga (id, dirty);
CREATE INDEX IF NOT EXISTS Manga_idx_in_library ON Manga(in_library);
CREATE INDEX IF NOT EXISTS Manga_idx_source ON Manga(source);

CREATE TABLE IF NOT EXISTS CategoryManga (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    category INT NOT NULL,
    manga INT NOT NULL,
    CONSTRAINT fk_CategoryManga_category__id FOREIGN KEY (category) REFERENCES Category(id) ON DELETE RESTRICT ON UPDATE RESTRICT,
    CONSTRAINT fk_CategoryManga_manga__id FOREIGN KEY (manga) REFERENCES Manga(id) ON DELETE RESTRICT ON UPDATE RESTRICT
);
CREATE INDEX IF NOT EXISTS CategoryManga_idx_manga ON CategoryManga(manga);
CREATE INDEX IF NOT EXISTS CategoryManga_idx_category ON CategoryManga(category);

CREATE TABLE IF NOT EXISTS Chapter (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    url VARCHAR(2048) NOT NULL,
    "name" VARCHAR(512) NOT NULL,
    date_upload BIGINT DEFAULT 0 NOT NULL,
    chapter_number SINGLE DEFAULT -1.0 NOT NULL,
    scanlator VARCHAR(128) NULL,
    "read" BOOLEAN DEFAULT 0 NOT NULL,
    bookmark BOOLEAN DEFAULT 0 NOT NULL,
    last_page_read INT DEFAULT 0 NOT NULL,
    last_read_at BIGINT DEFAULT 0 NOT NULL,
    fetched_at BIGINT DEFAULT 0 NOT NULL,
    source_order INT NOT NULL,
    real_url VARCHAR(2048) NULL,
    is_downloaded BOOLEAN DEFAULT 0 NOT NULL,
    page_count INT DEFAULT -1 NOT NULL,
    manga INT NOT NULL,
    original_chapter_id INTEGER,
    "create_at" BIGINT NOT NULL DEFAULT 0,
    "update_at" BIGINT NOT NULL DEFAULT 0,
    "dirty" BOOLEAN NOT NULL DEFAULT 0,
    "commit_id" BIGINT NOT NULL DEFAULT 0,
    "memo" TEXT NOT NULL DEFAULT '{}',
    CONSTRAINT fk_Chapter_manga__id FOREIGN KEY (manga) REFERENCES Manga(id) ON DELETE RESTRICT ON UPDATE RESTRICT
);
CREATE INDEX IF NOT EXISTS Chapter_idx_dirty ON Chapter (id, dirty);
CREATE INDEX IF NOT EXISTS Chapter_idx_manga ON Chapter(manga);

CREATE TABLE IF NOT EXISTS TrackRecord (
    id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    manga_id INTEGER NOT NULL,
    sync_id INTEGER NOT NULL,
    remote_id INTEGER NOT NULL,
    library_id INTEGER,
    title TEXT NOT NULL,
    last_chapter_read REAL NOT NULL,
    total_chapters INTEGER NOT NULL,
    status INTEGER NOT NULL,
    score REAL NOT NULL,
    remote_url TEXT NOT NULL,
    start_date INTEGER NOT NULL,
    finish_date INTEGER NOT NULL,
    "create_at" BIGINT NOT NULL DEFAULT 0,
    "update_at" BIGINT NOT NULL DEFAULT 0,
    "is_delete" BOOLEAN NOT NULL DEFAULT 0,
    "dirty" BOOLEAN NOT NULL DEFAULT 0,
    "commit_id" BIGINT NOT NULL DEFAULT 0,
    UNIQUE (manga_id, sync_id) ON CONFLICT REPLACE
);
CREATE INDEX IF NOT EXISTS TrackRecord_idx_dirty ON TrackRecord (id, dirty, is_delete);

CREATE TABLE IF NOT EXISTS History (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    create_at BIGINT DEFAULT 0 NOT NULL,
    update_at BIGINT DEFAULT 0 NOT NULL,
    is_delete BOOLEAN DEFAULT 0 NOT NULL,
    manga_id INTEGER NOT NULL,
    last_chapter_id INTEGER NOT NULL,
    last_read_at BIGINT DEFAULT 0 NOT NULL,
    "dirty" BOOLEAN NOT NULL DEFAULT 0,
    "commit_id" BIGINT NOT NULL DEFAULT 0,
    "last_chapter_name" VARCHAR(512) NULL,
    "read_duration" INT DEFAULT 0 NOT NULL,
    "sync_buff" INT DEFAULT 0 NOT NULL,
    UNIQUE (manga_id) ON CONFLICT REPLACE
);
CREATE INDEX IF NOT EXISTS History_idx_dirty ON History (id, dirty);
CREATE INDEX IF NOT EXISTS idx_last_read_at ON History (last_read_at);

CREATE TABLE IF NOT EXISTS "Stats" (
    "id" INTEGER PRIMARY KEY AUTOINCREMENT,
    "create_at" BIGINT NOT NULL DEFAULT 0,
    "update_at" BIGINT NOT NULL DEFAULT 0,
    "is_delete" BOOLEAN NOT NULL DEFAULT 0,
    "dirty" BOOLEAN NOT NULL DEFAULT 0,
    "commit_id" BIGINT NOT NULL DEFAULT 0,
    "day" INTEGER NOT NULL DEFAULT 0,
    "manga_id" INTEGER NOT NULL,
    "read_duration" INT NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS Stats_idx_dirty ON Stats (id, dirty);
CREATE INDEX IF NOT EXISTS Stats_idx_manga_id ON Stats (manga_id);
CREATE INDEX IF NOT EXISTS Stats_idx_day ON Stats (day);

CREATE TABLE IF NOT EXISTS Setting (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    create_at BIGINT DEFAULT 0 NOT NULL,
    update_at BIGINT DEFAULT 0 NOT NULL,
    is_delete BOOLEAN DEFAULT 0 NOT NULL,
    key VARCHAR(256) NOT NULL,
    value VARCHAR(2048) NOT NULL,
    UNIQUE (key) ON CONFLICT REPLACE
);

CREATE TABLE IF NOT EXISTS Repo (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    "type" INT DEFAULT 0 NOT NULL,
    "name" VARCHAR(1024) NOT NULL,
    meta_url VARCHAR(2048) NOT NULL,
    base_url VARCHAR(2048) NOT NULL,
    homepage VARCHAR(2048) NULL,
    deleted BOOLEAN DEFAULT 0 NOT NULL,
    create_at BIGINT DEFAULT 0 NOT NULL,
    update_at BIGINT DEFAULT 0 NOT NULL,
    "dirty" BOOLEAN NOT NULL DEFAULT 0,
    "commit_id" BIGINT NOT NULL DEFAULT 0,
    "index_url" VARCHAR(2048) NULL,
    "extension_list_url" VARCHAR(2048) NULL,
    "badge_label" VARCHAR(1024) NULL,
    "signing_key_fingerprint" VARCHAR(2048) NULL,
    "contact_discord" VARCHAR(2048) NULL,
    "is_legacy" BOOLEAN NOT NULL DEFAULT 1,
    "last_meta_refresh_at" BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS CategoryMeta (id INTEGER PRIMARY KEY AUTOINCREMENT, "key" VARCHAR(256) NOT NULL, "value" VARCHAR(4096) NOT NULL, category_ref INT NOT NULL);
CREATE TABLE IF NOT EXISTS ChapterMeta (id INTEGER PRIMARY KEY AUTOINCREMENT, "key" VARCHAR(256) NOT NULL, "value" VARCHAR(4096) NOT NULL, chapter_ref INT NOT NULL);
CREATE TABLE IF NOT EXISTS MangaMeta (id INTEGER PRIMARY KEY AUTOINCREMENT, "key" VARCHAR(256) NOT NULL, "value" VARCHAR(4096) NOT NULL, manga_ref INT NOT NULL);
CREATE TABLE IF NOT EXISTS SourceMeta (id INTEGER PRIMARY KEY AUTOINCREMENT, "create_at" BIGINT NOT NULL DEFAULT 0, "update_at" BIGINT NOT NULL DEFAULT 0, "is_delete" BOOLEAN NOT NULL DEFAULT 0, "dirty" BOOLEAN NOT NULL DEFAULT 0, "commit_id" BIGINT NOT NULL DEFAULT 0, "source_id" BIGINT NOT NULL, "key" VARCHAR(256) NOT NULL, "value" VARCHAR(4096) NOT NULL, UNIQUE (source_id, key) ON CONFLICT REPLACE);
CREATE TABLE IF NOT EXISTS Page (id INTEGER PRIMARY KEY AUTOINCREMENT, "index" INT NOT NULL, url VARCHAR(2048) NOT NULL, imageUrl VARCHAR(2048) NULL, chapter INT NOT NULL);
CREATE TABLE IF NOT EXISTS SyncState (id INTEGER PRIMARY KEY AUTOINCREMENT, create_at BIGINT DEFAULT 0 NOT NULL, update_at BIGINT DEFAULT 0 NOT NULL, is_delete BOOLEAN DEFAULT 0 NOT NULL, email VARCHAR(256) NOT NULL, enable BOOLEAN DEFAULT 0 NOT NULL, max_commit_id BIGINT DEFAULT 0 NOT NULL, last_sync_time BIGINT DEFAULT 0 NOT NULL, interval INTEGER NULL, UNIQUE (email) ON CONFLICT REPLACE);
CREATE TABLE IF NOT EXISTS SyncCommit (id INTEGER PRIMARY KEY AUTOINCREMENT, server_id BIGINT DEFAULT 0 NOT NULL, create_at BIGINT DEFAULT 0 NOT NULL, data_type VARCHAR(16) NOT NULL, data_id BIGINT DEFAULT 0 NOT NULL, content TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS ChapterSync (id INTEGER PRIMARY KEY AUTOINCREMENT, url VARCHAR(2048) NOT NULL, manga_id INT NOT NULL, read BOOLEAN NOT NULL DEFAULT 0, bookmark BOOLEAN NOT NULL DEFAULT 0, last_page_read INT NOT NULL DEFAULT 0, last_read_at BIGINT NOT NULL DEFAULT 0, create_at BIGINT NOT NULL DEFAULT 0, update_at BIGINT NOT NULL DEFAULT 0, dirty BOOLEAN NOT NULL DEFAULT 0, commit_id BIGINT NOT NULL DEFAULT 0, "name" VARCHAR(512) NULL, UNIQUE (manga_id, url) ON CONFLICT REPLACE);
CREATE TABLE IF NOT EXISTS UpdateRecord (id INTEGER PRIMARY KEY AUTOINCREMENT, "create_at" BIGINT NOT NULL DEFAULT 0, "update_at" BIGINT NOT NULL DEFAULT 0, "finish_at" BIGINT NOT NULL DEFAULT 0, "type" INT NOT NULL DEFAULT 0, "status" INT NOT NULL DEFAULT 0, "err_code" VARCHAR(32), "err_msg" VARCHAR(256), "total_count" INT NOT NULL DEFAULT 0, "succ_count" INT NOT NULL DEFAULT 0, "failed_count" INT NOT NULL DEFAULT 0, "skip_count" INT NOT NULL DEFAULT 0, "new_chapter_count" INT NOT NULL DEFAULT 0);
`

var migrations = []struct {
	version int
	name    string
}{
	{1, "Initial"},
	{2, "SourceV2_1"},
	{3, "SourceV2_2"},
	{4, "TrackRecord"},
	{5, "Manga"},
	{6, "PkgFactory"},
	{7, "Extension_Changelog"},
	{8, "Extension_Readme"},
	{9, "Extension_RepoId"},
	{10, "Repo"},
	{11, "Setting"},
	{12, "History"},
	{13, "History_index"},
	{14, "Chapter_ChapterId"},
	{15, "Sync"},
	{16, "HistorySyncFix"},
	{17, "History_ReadDuration"},
	{18, "Stats"},
	{19, "SourceMeta"},
	{20, "History_SyncBuff"},
	{21, "Chapter_manga_index"},
	{22, "Manga_source_index"},
	{23, "Manga_in_library_index"},
	{24, "Page_chapter_index"},
	{25, "CategoryManga_Index"},
	{26, "MetaTable_Index"},
	{27, "UpdateRecord"},
	{28, "Extension_BaseUrl"},
	{29, "Extension_AlternateName"},
	{30, "Memo"},
	{31, "Extension_InstallPathType"},
	{32, "ExtensionStore"},
}

// tmbMeta represents the exact meta.json structure required by Tachimanga on iOS.
type tmbMeta struct {
	State            int         `json:"state"`
	ExtInfo          interface{} `json:"extInfo"`
	DownloadProgress int         `json:"downloadProgress"`
	Version          int         `json:"version"`
	CloudBackup      bool        `json:"cloudBackup"`
	CreateAt         int64       `json:"createAt"`
	Checksum         string      `json:"checksum"`
	Size             int64       `json:"size"`
	Type             int         `json:"type"`
	UpdateAt         int64       `json:"updateAt"`
	BackupID         int64       `json:"backupId"`
	Downloaded       bool        `json:"downloaded"`
	RemoteBackup     bool        `json:"remoteBackup"`
	Name             string      `json:"name"`
}

// Write creates a .tmb archive from the internal Backup model.
func (t *TMB) Write(outputPath string, backup *core.Backup) error {
	tmpDir, err := os.MkdirTemp("", "mangastu_tmb_*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "tachimanga.db")
	if err := createDatabase(dbPath, backup); err != nil {
		return fmt.Errorf("failed to create tachimanga database: %w", err)
	}

	contentsZipPath := filepath.Join(tmpDir, "contents.zip")
	if err := createContentsZip(contentsZipPath, dbPath, backup); err != nil {
		return fmt.Errorf("failed to create contents.zip: %w", err)
	}

	checksum, size, err := computeChecksumAndSize(contentsZipPath)
	if err != nil {
		return fmt.Errorf("failed to compute checksum: %w", err)
	}

	now := time.Now().Unix()
	meta := tmbMeta{
		State:            1,
		ExtInfo:          nil,
		DownloadProgress: 0,
		Version:          tmbVersion,
		CloudBackup:      false,
		CreateAt:         now,
		Checksum:         checksum,
		Size:             size,
		Type:             0,
		UpdateAt:         now,
		BackupID:         0,
		Downloaded:       false,
		RemoteBackup:     false,
		Name:             filepath.Base(outputPath),
	}

	metaBytes, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal meta.json: %w", err)
	}

	metaPath := filepath.Join(tmpDir, "meta.json")
	if err := os.WriteFile(metaPath, metaBytes, 0o644); err != nil {
		return fmt.Errorf("failed to write meta.json: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("failed to create output dir: %w", err)
	}

	if err := createOuterZip(outputPath, metaPath, contentsZipPath); err != nil {
		return fmt.Errorf("failed to create final .tmb archive: %w", err)
	}

	// Generate companion sources.txt and sources.zip alongside the .tmb file
	if err := generateSourcesCompanion(outputPath, backup); err != nil {
		// Non-fatal warning if companion generation fails
		fmt.Printf("Note: failed to generate sources companion: %v\n", err)
	}

	return nil
}

func createDatabase(dbPath string, backup *core.Backup) error {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	if _, err := db.Exec(ddlSchema); err != nil {
		return fmt.Errorf("failed to execute DDL schema: %w", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	nowSec := time.Now().Unix()
	nowMs := time.Now().UnixMilli()
	nowFormatted := time.Now().Format("2006-01-02 15:04:05.000")

	// 1. Insert Migrations
	migStmt, err := tx.Prepare(`INSERT INTO Migrations (version, "name", executed_at) VALUES (?, ?, ?)`)
	if err != nil {
		return err
	}
	defer migStmt.Close()

	for _, m := range migrations {
		if _, err := migStmt.Exec(m.version, m.name, nowFormatted); err != nil {
			return err
		}
	}

	// 2. Insert Official Keiyoushi Repo and Setting
	if _, err := tx.Exec(`
		INSERT INTO Repo (id, "type", "name", meta_url, base_url, homepage, deleted, create_at, update_at, "dirty", "commit_id", "index_url", "badge_label", "signing_key_fingerprint", "contact_discord", "is_legacy", "last_meta_refresh_at")
		VALUES (1, 0, 'Keiyoushi', 'https://raw.githubusercontent.com/keiyoushi/extensions/repo/index.min.json', 'https://raw.githubusercontent.com/keiyoushi/extensions/repo', 'https://keiyoushi.github.io', 0, ?, ?, 1, 0, 'https://github.com/keiyoushi/extensions/raw/repo/index.pb', 'KEI', '9add655a78e96c4ec7a53ef89dccb557cb5d767489fac5e785d671a5a75d4da2', 'https://discord.gg/3FbCpdKbdY', 0, ?)
	`, nowMs, nowMs, nowMs); err != nil {
		return fmt.Errorf("failed to insert default repo: %w", err)
	}

	if _, err := tx.Exec(`
		INSERT OR REPLACE INTO Setting (key, value, create_at, update_at, is_delete)
		VALUES ('UpdateRestrictions', '{"filteredByUpdateStrategy":null,"filteredByMangaStatus":false,"filteredByMangaUnread":false,"filteredByMangaNotStart":false}', ?, ?, 0)
	`, nowMs, nowMs); err != nil {
		return fmt.Errorf("failed to insert default setting: %w", err)
	}

	// 3. Insert Categories
	catStmt, err := tx.Prepare(`INSERT INTO Category ("name", "order", is_default, uuid, create_at, update_at, is_delete, dirty, commit_id) VALUES (?, ?, 0, ?, ?, ?, 0, 1, 0)`)
	if err != nil {
		return err
	}
	defer catStmt.Close()

	orderToCatID := make(map[int64]int64)
	for i, cat := range backup.Categories {
		res, err := catStmt.Exec(cat.Name, cat.Order+1, cat.Name, nowMs, nowMs)
		if err != nil {
			return fmt.Errorf("failed to insert category %s: %w", cat.Name, err)
		}
		catID, _ := res.LastInsertId()
		orderToCatID[cat.Order] = catID
		orderToCatID[int64(i)] = catID
	}

	// 4. Insert Extensions and Sources using Keiyoushi official metadata
	extStmt, err := tx.Prepare(`
		INSERT INTO Extension (
			apk_name, icon_url, "name", pkg_name, version_name, version_code,
			lang, is_nsfw, is_installed, has_update, is_obsolete, class_name,
			repo_id, create_at, update_at, dirty, apk_url, jar_url, content_warning, base_url
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 1, 0, 0, ?, 1, ?, ?, 1, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer extStmt.Close()

	srcStmt, err := tx.Prepare(`INSERT OR REPLACE INTO "Source" (id, "name", lang, extension, is_nsfw) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer srcStmt.Close()

	// Default Local Source extension
	localExtRes, err := extStmt.Exec(
		"localSource", "https://raw.githubusercontent.com/tachiyomiorg/tachiyomi/64ba127e7d43b1d7e6d58a6f5c9b2bd5fe0543f7/app/src/main/res/mipmap-xxxhdpi/ic_local_source.webp",
		"Local Source fake extension", "eu.kanade.tachiyomi.source.local", "1.2", 0,
		"localsourcelang", 0, "eu.kanade.tachiyomi.source.local", nowMs, nowMs, "", "", 0, "",
	)
	if err != nil {
		return err
	}
	localExtID, _ := localExtRes.LastInsertId()
	if _, err := srcStmt.Exec(0, "Local source", "localsourcelang", localExtID, 0); err != nil {
		return err
	}

	// Build complete list of unique sources from backup.Sources and backup.Manga
	allSources := make(map[int64]string)
	for _, src := range backup.Sources {
		allSources[src.SourceID] = src.Name
	}
	for _, m := range backup.Manga {
		if _, exists := allSources[m.Source]; !exists {
			allSources[m.Source] = ""
		}
	}

	pkgToExtID := make(map[string]int64)
	for srcID, srcName := range allSources {
		if srcID == 0 {
			continue
		}
		info := ResolveSourceInfo(srcID, srcName)
		extID, exists := pkgToExtID[info.PackageName]
		if !exists {
			nsfwInt := 0
			if info.IsNSFW {
				nsfwInt = 1
			}
			extRes, err := extStmt.Exec(
				info.APKName, info.IconURL, info.ExtensionName, info.PackageName,
				info.VersionName, info.VersionCode, info.SourceLang, nsfwInt,
				info.PackageName, nowMs, nowMs, info.APKURL, info.JARURL, info.ContentWarning,
				info.HomeURL,
			)
			if err != nil {
				return fmt.Errorf("failed to insert extension %s: %w", info.ExtensionName, err)
			}
			extID, _ = extRes.LastInsertId()
			pkgToExtID[info.PackageName] = extID
		}

		nsfwVal := 0
		if info.IsNSFW {
			nsfwVal = 1
		}
		if _, err := srcStmt.Exec(srcID, info.SourceName, info.SourceLang, extID, nsfwVal); err != nil {
			return fmt.Errorf("failed to insert source %d (%s): %w", srcID, info.SourceName, err)
		}
	}

	// 5. Insert Manga, Chapters, Categories, Trackers, History
	mangaStmt, err := tx.Prepare(`
		INSERT INTO Manga (
			url, title, initialized, artist, author, description, genre, status,
			thumbnail_url, in_library, default_category, in_library_at, "source",
			update_strategy, create_at, update_at, dirty
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 1, ?, ?, ?, ?, ?, ?, 1)
	`)
	if err != nil {
		return err
	}
	defer mangaStmt.Close()

	catMangaStmt, err := tx.Prepare(`INSERT INTO CategoryManga (category, manga) VALUES (?, ?)`)
	if err != nil {
		return err
	}
	defer catMangaStmt.Close()

	chapStmt, err := tx.Prepare(`
		INSERT INTO Chapter (
			url, "name", date_upload, chapter_number, scanlator, "read", bookmark,
			last_page_read, last_read_at, fetched_at, source_order, manga,
			create_at, update_at, dirty
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1)
	`)
	if err != nil {
		return err
	}
	defer chapStmt.Close()

	trackStmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO TrackRecord (
			manga_id, sync_id, remote_id, library_id, title,
			last_chapter_read, total_chapters, status, score, remote_url,
			start_date, finish_date, create_at, update_at, dirty
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1)
	`)
	if err != nil {
		return err
	}
	defer trackStmt.Close()

	histStmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO History (
			manga_id, last_chapter_id, last_read_at, read_duration, sync_buff,
			create_at, update_at, dirty
		) VALUES (?, ?, ?, ?, 0, ?, ?, 1)
	`)
	if err != nil {
		return err
	}
	defer histStmt.Close()

	for _, m := range backup.Manga {
		genreStr := strings.Join(m.Genre, ", ")
		inLibAt := m.DateAdded / 1000
		if inLibAt == 0 {
			inLibAt = nowSec
		}

		updateStrat := "ALWAYS_UPDATE"
		if strings.ToUpper(m.UpdateStrategy) == "ONLY_FETCH_ONCE" {
			updateStrat = "ONLY_FETCH_ONCE"
		}

		defaultCategory := 1
		if len(m.Categories) > 0 {
			defaultCategory = 0
		}

		res, err := mangaStmt.Exec(
			m.URL, m.Title, 1, m.Artist, m.Author, m.Description, genreStr, m.Status,
			m.ThumbnailURL, defaultCategory, inLibAt, m.Source,
			updateStrat, nowMs, nowMs,
		)
		if err != nil {
			return fmt.Errorf("failed to insert manga %s: %w", m.Title, err)
		}

		mangaID, _ := res.LastInsertId()

		// Category assignments
		for _, catOrder := range m.Categories {
			if catID, ok := orderToCatID[catOrder]; ok {
				if _, err := catMangaStmt.Exec(catID, mangaID); err != nil {
					return err
				}
			}
		}

		// Chapters
		chapterURLToID := make(map[string]int64)
		var lastChapterID int64
		for _, ch := range m.Chapters {
			dateFetchSec := ch.DateFetch / 1000
			if dateFetchSec == 0 {
				dateFetchSec = nowSec
			}
			lastReadSec := ch.LastReadAt / 1000

			cRes, err := chapStmt.Exec(
				ch.URL, ch.Name, ch.DateUpload, ch.ChapterNumber, ch.Scanlator,
				ch.Read, ch.Bookmark, ch.LastPageRead, lastReadSec, dateFetchSec,
				ch.SourceOrder, mangaID, nowMs, nowMs,
			)
			if err != nil {
				return fmt.Errorf("failed to insert chapter %s: %w", ch.Name, err)
			}
			cID, _ := cRes.LastInsertId()
			chapterURLToID[ch.URL] = cID
			lastChapterID = cID
		}

		// Tracking
		for _, tr := range m.Tracking {
			if _, err := trackStmt.Exec(
				mangaID, tr.SyncID, tr.MediaID, tr.LibraryID, tr.Title,
				tr.LastChapterRead, tr.TotalChapters, tr.Status, tr.Score, tr.TrackingURL,
				tr.StartedReadingDate, tr.FinishedReadingDate, nowMs, nowMs,
			); err != nil {
				return fmt.Errorf("failed to insert track for %s: %w", m.Title, err)
			}
		}

		// History
		for _, h := range m.History {
			chapID := chapterURLToID[h.ChapterURL]
			if chapID == 0 {
				chapID = lastChapterID
			}
			if chapID == 0 {
				continue
			}
			lastReadSec := h.LastRead / 1000
			if lastReadSec == 0 {
				lastReadSec = nowSec
			}
			readDurationSec := h.ReadDuration / 1000

			if _, err := histStmt.Exec(
				mangaID, chapID, lastReadSec, readDurationSec, nowMs, nowMs,
			); err != nil {
				return fmt.Errorf("failed to insert history for %s: %w", m.Title, err)
			}
		}
	}

	return tx.Commit()
}

func createContentsZip(zipPath, dbPath string, backup *core.Backup) error {
	f, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	// 1. Add tachimanga.db
	dbFile, err := os.Open(dbPath)
	if err != nil {
		return err
	}
	defer dbFile.Close()

	dbEntry, err := zw.Create("tachimanga.db")
	if err != nil {
		return err
	}
	if _, err := io.Copy(dbEntry, dbFile); err != nil {
		return err
	}

	// 2. Add complete standard pref.json
	prefEntry, err := zw.Create("pref.json")
	if err != nil {
		return err
	}
	defaultPref := map[string]interface{}{
		"flutter.themeMode":                           0,
		"flutter.autoBackup":                          false,
		"flutter.autoBackupFrequency":                 0,
		"flutter.dateFormat":                          0,
		"flutter.defaultTab":                          1,
		"flutter.readerMode":                          7,
		"flutter.readerPageLayout":                    0,
		"flutter.libraryDisplayMode":                  0,
		"flutter.libraryShowMangaCount":               true,
		"flutter.tabBarShowLabel":                     true,
		"flutter.deleteDownloadAfterRead":             0,
		"flutter.doubleTapZoomIn":                     true,
		"flutter.showTapZones":                        true,
		"flutter.showStatusBar":                       false,
		"flutter.sourceDisplayMode":                   0,
		"flutter.sourceLanguageFilter": []string{
			"all", "lastUsed", "en", "localsourcelang", "multi",
			"ja", "ko", "zh", "ru", "fr", "es", "id", "vi",
			"pt", "de", "it", "tr", "ar", "pl", "th",
		},
		"flutter.extensionLanguageFilter": []string{
			"installed", "update", "en", "all", "multi",
		},
		"flutter.applePencilSqueezeAction":            1,
		"flutter.applePencilDoubleTapAction":          0,
		"flutter.readerPaddingLandscape":              0,
		"flutter.readerStrictScale":                   true,
		"flutter.readerPageLayoutSkipFirstPage":       false,
		"flutter.autoRefreshTitle":                    false,
		"flutter.userAgentType":                       3,
		"flutter.longPressActionMenu":                 true,
		"flutter.readerScrollIndicator":               false,
		"flutter.themeBlendLevel":                     10,
		"flutter.readerNavigationLayout":              5,
		"flutter.longPressScroll":                     false,
		"flutter.themePureBlackDarkMode":              false,
		"flutter.readerAllowPinchWhenLongPressScroll": true,
		"flutter.downloadTaskInParallel":              1,
		"flutter.swipeRightToGoBackMode":              0,
		"flutter.invertTap":                           false,
		"flutter.keepScreenOnWhileReading":            false,
		"flutter.themeKey":                            "default",
	}
	prefBytes, _ := json.MarshalIndent(defaultPref, "", "  ")
	if _, err := prefEntry.Write(prefBytes); err != nil {
		return err
	}

	// 3. Add standard pref-all.json
	prefAllEntry, err := zw.Create("pref-all.json")
	if err != nil {
		return err
	}
	if _, err := prefAllEntry.Write([]byte("{\n}\n")); err != nil {
		return err
	}

	// 4. Add standard prefs directory entries
	if _, err := zw.Create("prefs/"); err != nil {
		return err
	}
	trackerPlistEntry, err := zw.Create("prefs/suwayomi.tachidesk.tracker.plist")
	if err != nil {
		return err
	}
	trackerPlistBytes := []byte("bplist00\xd1\x01\x02_\x10\x1f/suwayomi/tachidesk/tracker/\xa0\x08\x0b\x2a\x00\x00\x00\x00\x00\x00\x01\x01\x00\x00\x00\x00\x00\x00\x00\x03\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x2b")
	if _, err := trackerPlistEntry.Write(trackerPlistBytes); err != nil {
		return err
	}

	// 5. Add extensions directory entry (clean empty folder for iOS)
	if _, err := zw.Create("extensions/"); err != nil {
		return err
	}

	return nil
}

func computeChecksumAndSize(filePath string) (string, int64, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()

	h := sha1.New()
	size, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}

	return hex.EncodeToString(h.Sum(nil)), size, nil
}

func createOuterZip(outputPath, metaPath, contentsZipPath string) error {
	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	// 1. Add contents.zip FIRST (matching Tachimanga's native zip order)
	contentsFile, err := os.Open(contentsZipPath)
	if err != nil {
		return err
	}
	defer contentsFile.Close()

	contentsEntry, err := zw.Create("contents.zip")
	if err != nil {
		return err
	}
	if _, err := io.Copy(contentsEntry, contentsFile); err != nil {
		return err
	}

	// 2. Add meta.json
	metaFile, err := os.Open(metaPath)
	if err != nil {
		return err
	}
	defer metaFile.Close()

	metaEntry, err := zw.Create("meta.json")
	if err != nil {
		return err
	}
	if _, err := io.Copy(metaEntry, metaFile); err != nil {
		return err
	}

	return nil
}

func generateSourcesCompanion(outputPath string, backup *core.Backup) error {
	baseNoExt := strings.TrimSuffix(outputPath, filepath.Ext(outputPath))
	txtPath := baseNoExt + ".txt"

	// Count manga per source
	mangaCountPerSource := make(map[int64]int)
	for _, m := range backup.Manga {
		mangaCountPerSource[m.Source]++
	}

	allSources := make(map[int64]string)
	for _, src := range backup.Sources {
		allSources[src.SourceID] = src.Name
	}
	for srcID := range mangaCountPerSource {
		if _, exists := allSources[srcID]; !exists {
			allSources[srcID] = ""
		}
	}

	// Sort sources by manga count descending
	type srcEntry struct {
		id    int64
		name  string
		count int
		info  *KeiyoushiSourceInfo
	}
	var entries []srcEntry
	for srcID, srcName := range allSources {
		if srcID == 0 {
			continue
		}
		info := ResolveSourceInfo(srcID, srcName)
		entries = append(entries, srcEntry{
			id:    srcID,
			name:  info.SourceName,
			count: mangaCountPerSource[srcID],
			info:  info,
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].count > entries[j].count
	})

	var sb strings.Builder
	sb.WriteString("================================================================================\n")
	sb.WriteString("                       MANGASTU - SOURCE & EXTENSION LIST                       \n")
	sb.WriteString("================================================================================\n\n")
	sb.WriteString("How to use in Tachimanga (iOS):\n")
	sb.WriteString("1. In Tachimanga, go to: More -> Extension Repos\n")
	sb.WriteString("2. Add the Keiyoushi Extension Repository:\n")
	sb.WriteString("   https://raw.githubusercontent.com/keiyoushi/extensions/repo/index.min.json\n")
	sb.WriteString("3. Go to Browse -> Extensions and install the extensions listed below.\n")
	sb.WriteString("4. Your restored manga will immediately link to the installed extensions!\n\n")
	sb.WriteString("--------------------------------------------------------------------------------\n")
	sb.WriteString(fmt.Sprintf("%-28s | %-20s | %-6s | %-28s\n", "Source Name", "Source ID", "Titles", "Extension to Install"))
	sb.WriteString("--------------------------------------------------------------------------------\n")

	for _, e := range entries {
		sb.WriteString(fmt.Sprintf("%-28s | %-20d | %-6d | %-28s\n", e.name, e.id, e.count, e.info.ExtensionName))
	}

	sb.WriteString("--------------------------------------------------------------------------------\n\n")
	sb.WriteString("Detailed Extension Packages & Direct Download Links:\n\n")
	for _, e := range entries {
		sb.WriteString(fmt.Sprintf("• %s (ID: %d, Titles: %d)\n", e.name, e.id, e.count))
		sb.WriteString(fmt.Sprintf("  Extension: %s (%s v%s)\n", e.info.ExtensionName, e.info.PackageName, e.info.VersionName))
		if e.info.JARURL != "" {
			sb.WriteString(fmt.Sprintf("  JAR URL:   %s\n", e.info.JARURL))
		}
		if e.info.APKURL != "" {
			sb.WriteString(fmt.Sprintf("  APK URL:   %s\n", e.info.APKURL))
		}
		sb.WriteString("\n")
	}

	return os.WriteFile(txtPath, []byte(sb.String()), 0o644)
}
