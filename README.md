<div align="center">

<img src="mangastu-web/public/logo.png" alt="MangaSTU Logo" width="140" />

# MangaSTU
<p align="center">
  A simple tool to <b>convert</b> and <b>merge</b> manga backups between <b>Tachimanga (.tmb)</b> and <b>Tachiyomi / Mihon (.tachibk)</b> without losing your reading progress, categories, or tracking history.
</p>

</div>

## Supported Formats

| Format | Extension | App | Read | Write |
|--------|-----------|-----|------|-------|
| Tachimanga Backup | `.tmb` | Tachimanga (iOS) | ✔ | ✔ |
| Tachiyomi/Komikku Backup | `.tachibk` | Komikku, Mihon, Tachiyomi | ✔ | ✔ |

## Installation

```bash
# From source
go install github.com/SriKousik/mangastu/cmd/mangastu@latest

# Or build locally
git clone https://github.com/SriKousik/mangastu
cd mangastu
go build -o mangastu ./cmd/mangastu/
```

## Usage

### Convert

Convert a backup file from one format to another:

```bash
mangastu convert backup.tmb backup.tachibk
```

### Merge

Merge and deduplicate multiple backup files (in any combination of `.tmb` and `.tachibk`) into a single unified backup:

```bash
# Merge a .tmb and .tachibk into a unified .tachibk
mangastu merge backup1.tmb backup2.tachibk -o merged.tachibk

# Merge into a unified .tmb
mangastu merge backup1.tachibk backup2.tmb -o merged.tmb
```

The merge engine automatically:
- Deduplicates manga across sources and URLs
- Reconciles read status (`read` in either = `read`)
- Preserves highest chapter read progress (`max(LastPageRead)`) and timestamps
- Unifies categories and tracker records (AniList, MAL, MangaUpdates)
- Generates a companion `<name>.txt` listing all sources and Keiyoushi extension links

### List Formats

Show all supported backup formats:

```bash
mangastu list-formats
```

## Data Mapping

Both TMB and Tachibk use the Tachiyomi extension ecosystem with identical source IDs, making conversion lossless for core data:

- **Manga**: title, author, artist, description, genres, status, thumbnail, source, URL
- **Chapters**: name, URL, read state, bookmarks, page progress, upload date
- **Categories**: names and ordering
- **Tracking**: MAL, AniList, Kitsu, MangaUpdates entries
- **History**: reading history with chapter URLs and timestamps

### What's Not Converted

- Extension JAR files (TMB includes these; tachibk does not)
- App-specific preferences (different apps, different settings)
- Source preference plists (format-specific)