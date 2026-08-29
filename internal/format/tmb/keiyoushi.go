package tmb

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
)

//go:embed keiyoushi_index.json.gz
var embeddedKeiyoushiIndexGz []byte

// KeiyoushiSourceInfo contains all metadata for an extension and source from Keiyoushi.
type KeiyoushiSourceInfo struct {
	SourceID       int64
	SourceName     string
	SourceLang     string
	HomeURL        string
	ExtensionName  string
	PackageName    string
	VersionName    string
	VersionCode    int
	APKName        string
	JARName        string
	APKURL         string
	JARURL         string
	IconURL        string
	ContentWarning int
	IsNSFW         bool
}

type keiyoushiRepoIndex struct {
	Name          string `json:"name"`
	BadgeLabel    string `json:"badgeLabel"`
	SigningKey    string `json:"signingKey"`
	ExtensionList struct {
		Extensions []struct {
			Name           string `json:"name"`
			PackageName    string `json:"packageName"`
			VersionName    string `json:"versionName"`
			VersionCode    string `json:"versionCode"`
			ExtensionLib   string `json:"extensionLib"`
			ContentWarning string `json:"contentWarning"`
			Resources      struct {
				APKURL  string `json:"apkUrl"`
				JARURL  string `json:"jarUrl"`
				IconURL string `json:"iconUrl"`
			} `json:"resources"`
			Sources []struct {
				ID       string `json:"id"`
				Name     string `json:"name"`
				Language string `json:"language"`
				HomeURL  string `json:"homeUrl"`
			} `json:"sources"`
		} `json:"extensions"`
	} `json:"extensionList"`
}

var (
	keiyoushiSourceMap map[int64]*KeiyoushiSourceInfo
	keiyoushiOnce      sync.Once
)

func initKeiyoushi() {
	keiyoushiOnce.Do(func() {
		keiyoushiSourceMap = make(map[int64]*KeiyoushiSourceInfo)

		gzr, err := gzip.NewReader(bytes.NewReader(embeddedKeiyoushiIndexGz))
		if err != nil {
			return
		}
		defer gzr.Close()

		rawJSON, err := io.ReadAll(gzr)
		if err != nil {
			return
		}

		var repo keiyoushiRepoIndex
		if err := json.Unmarshal(rawJSON, &repo); err != nil {
			return
		}

		for _, ext := range repo.ExtensionList.Extensions {
			vCode, _ := strconv.Atoi(ext.VersionCode)
			if vCode == 0 {
				vCode = 1
			}
			vName := ext.VersionName
			if vName == "" {
				vName = "1.4.0"
			}

			apkName := ""
			if ext.Resources.APKURL != "" {
				parts := strings.Split(ext.Resources.APKURL, "/")
				apkName = parts[len(parts)-1]
			}
			if apkName == "" {
				apkName = fmt.Sprintf("%s-v%s.apk", ext.PackageName, vName)
			}

			jarName := ""
			if ext.Resources.JARURL != "" {
				parts := strings.Split(ext.Resources.JARURL, "/")
				jarName = parts[len(parts)-1]
			}
			if jarName == "" {
				jarName = strings.TrimSuffix(apkName, ".apk") + ".jar"
			}

			isNSFW := strings.Contains(strings.ToUpper(ext.ContentWarning), "NSFW")
			contentWarning := 0
			if isNSFW {
				contentWarning = 1
			}

			for _, src := range ext.Sources {
				sid, err := strconv.ParseInt(src.ID, 10, 64)
				if err != nil || sid == 0 {
					continue
				}

				keiyoushiSourceMap[sid] = &KeiyoushiSourceInfo{
					SourceID:       sid,
					SourceName:     src.Name,
					SourceLang:     src.Language,
					HomeURL:        src.HomeURL,
					ExtensionName:  ext.Name,
					PackageName:    ext.PackageName,
					VersionName:    vName,
					VersionCode:    vCode,
					APKName:        apkName,
					JARName:        jarName,
					APKURL:         ext.Resources.APKURL,
					JARURL:         ext.Resources.JARURL,
					IconURL:        ext.Resources.IconURL,
					ContentWarning: contentWarning,
					IsNSFW:         isNSFW,
				}
			}
		}
	})
}

// ResolveSourceInfo resolves a source ID to its full Keiyoushi extension info, or returns a fallback if not in the official repo.
func ResolveSourceInfo(sourceID int64, fallbackName string) *KeiyoushiSourceInfo {
	initKeiyoushi()

	if info, ok := keiyoushiSourceMap[sourceID]; ok {
		return info
	}

	// Fallback for custom or unlisted sources
	cleanName := strings.ToLower(strings.ReplaceAll(fallbackName, " ", ""))
	cleanName = strings.ReplaceAll(cleanName, "-", "")
	cleanName = strings.ReplaceAll(cleanName, "_", "")
	if cleanName == "" {
		cleanName = fmt.Sprintf("source_%d", sourceID)
	}
	displayName := fallbackName
	if displayName == "" {
		displayName = fmt.Sprintf("Source %d", sourceID)
	}

	pkgName := "eu.kanade.tachiyomi.extension.en." + cleanName
	jarName := fmt.Sprintf("tachiyomi-en.%s-v1.4.0.jar", cleanName)
	apkName := fmt.Sprintf("tachiyomi-en.%s-v1.4.0.apk", cleanName)

	return &KeiyoushiSourceInfo{
		SourceID:       sourceID,
		SourceName:     displayName,
		SourceLang:     "en",
		HomeURL:        "",
		ExtensionName:  displayName,
		PackageName:    pkgName,
		VersionName:    "1.4.0",
		VersionCode:    1,
		APKName:        apkName,
		JARName:        jarName,
		APKURL:         "",
		JARURL:         "",
		IconURL:        "https://raw.githubusercontent.com/tachiyomiorg/tachiyomi/64ba127e7d43b1d7e6d58a6f5c9b2bd5fe0543f7/app/src/main/res/mipmap-xxxhdpi/ic_local_source.webp",
		ContentWarning: 0,
		IsNSFW:         false,
	}
}
