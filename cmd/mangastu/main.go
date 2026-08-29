package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/misfit/mangastu/internal/convert"
	"github.com/misfit/mangastu/internal/format"
	"github.com/misfit/mangastu/internal/format/tachibk"
	"github.com/misfit/mangastu/internal/format/tmb"
	"github.com/misfit/mangastu/internal/inspect"
	"github.com/misfit/mangastu/internal/merger"
	"github.com/misfit/mangastu/internal/server"
)

const (
	version = "0.1.0"
	banner  = `
  ███╗   ███╗ █████╗ ███╗   ██╗ ██████╗  █████╗ ███████╗████████╗██╗   ██╗
  ████╗ ████║██╔══██╗████╗  ██║██╔════╝ ██╔══██╗██╔════╝╚══██╔══╝██║   ██║
  ██╔████╔██║███████║██╔██╗ ██║██║  ███╗███████║███████╗   ██║   ██║   ██║
  ██║╚██╔╝██║██╔══██║██║╚██╗██║██║   ██║██╔══██║╚════██║   ██║   ██║   ██║
  ██║ ╚═╝ ██║██║  ██║██║ ╚████║╚██████╔╝██║  ██║███████║   ██║   ╚██████╔╝
  ╚═╝     ╚═╝╚═╝  ╚═╝╚═╝  ╚═══╝ ╚═════╝ ╚═╝  ╚═╝╚══════╝   ╚═╝    ╚═════╝

  Manga Backup Format Bridge v%s
`
)

func newRegistry() *format.Registry {
	reg := format.NewRegistry()
	reg.Register(&tmb.TMB{})
	reg.Register(&tachibk.TachiBK{})
	return reg
}

func main() {
	registry := newRegistry()

	rootCmd := &cobra.Command{
		Use:   "mangastu",
		Short: "MangaSTU — Manga Backup Format Bridge",
		Long: fmt.Sprintf(banner, version) + `
  Convert manga backup files between different app formats.
  Supported formats: .tmb (Tachimanga), .tachibk (Komikku/Mihon)`,
	}

	// === convert command ===
	convertCmd := &cobra.Command{
		Use:   "convert <input> <output>",
		Short: "Convert a backup file from one format to another",
		Long: `Convert a manga backup file between supported formats.

Examples:
  mangastu convert backup.tmb backup.tachibk
  mangastu convert my_backup.tachibk output.tmb`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			inputPath := args[0]
			outputPath := args[1]

			// Validate input exists
			if _, err := os.Stat(inputPath); os.IsNotExist(err) {
				return fmt.Errorf("input file not found: %s", inputPath)
			}

			conv := convert.New(registry)

			fmt.Printf("Reading %s...\n", filepath.Base(inputPath))
			start := time.Now()

			result, err := conv.Convert(inputPath, outputPath)
			if err != nil {
				return err
			}

			elapsed := time.Since(start)

			fmt.Printf("Found %d manga\n", result.MangaCount)
			fmt.Printf("Found %d chapters\n", result.ChapterCount)
			fmt.Println()
			fmt.Printf("Converting %s → %s...\n", result.InputFormat, result.OutputFormat)
			fmt.Println()

			// Progress bar (simple)
			printProgressBar(100)

			fmt.Println()
			fmt.Printf("Done in %s\n", elapsed.Round(time.Millisecond))
			fmt.Printf("Output: %s\n", outputPath)

			// Show file size
			if stat, err := os.Stat(outputPath); err == nil {
				fmt.Printf("Size:   %s\n", formatSize(stat.Size()))
			}

			if len(result.Warnings) > 0 {
				fmt.Printf("\nWarnings (%d):\n", len(result.Warnings))
				for _, w := range result.Warnings {
					fmt.Printf("  %s\n", w.String())
				}
			}

			return nil
		},
	}

	// === inspect command ===
	inspectCmd := &cobra.Command{
		Use:   "inspect <file>",
		Short: "Show detailed information about a backup file",
		Long: `Inspect a backup file and display a detailed summary of its contents.

Examples:
  mangastu inspect backup.tmb
  mangastu inspect backup.tachibk`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]

			if _, err := os.Stat(path); os.IsNotExist(err) {
				return fmt.Errorf("file not found: %s", path)
			}

			inspector := inspect.New(registry)
			result, err := inspector.Inspect(path)
			if err != nil {
				return err
			}

			fmt.Println(result.Summary)
			return nil
		},
	}

	// === validate command ===
	validateCmd := &cobra.Command{
		Use:   "validate <file>",
		Short: "Validate a backup file for integrity and convertibility",
		Long: `Validate a backup file and report any issues found.

Examples:
  mangastu validate backup.tmb
  mangastu validate backup.tachibk`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]

			if _, err := os.Stat(path); os.IsNotExist(err) {
				return fmt.Errorf("file not found: %s", path)
			}

			inspector := inspect.New(registry)
			output, err := inspector.Validate(path)
			if err != nil {
				return err
			}

			fmt.Println(output)
			return nil
		},
	}

	// === merge command ===
	var mergeOutput string
	mergeCmd := &cobra.Command{
		Use:   "merge <input1> <input2>... [-o output] / <output>",
		Short: "Merge and deduplicate multiple backup files into a single backup",
		Long: `Merge multiple backup files (in any combination of .tmb and .tachibk) into a single unified backup.
Deduplicates manga across backups, reconciles read progress, chapter status, tracking, categories, and history.

Examples:
  mangastu merge backup1.tmb backup2.tachibk -o merged.tachibk
  mangastu merge backup1.tachibk backup2.tmb merged.tmb`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			var inputPaths []string
			outputPath := mergeOutput

			if outputPath == "" {
				if len(args) < 3 {
					return fmt.Errorf("please specify an output file with -o <output> or as the last argument")
				}
				inputPaths = args[:len(args)-1]
				outputPath = args[len(args)-1]
			} else {
				inputPaths = args
			}

			// Validate all input files exist
			for _, p := range inputPaths {
				if _, err := os.Stat(p); os.IsNotExist(err) {
					return fmt.Errorf("input file not found: %s", p)
				}
			}

			m := merger.New(registry)
			fmt.Printf("Merging %d backup files...\n", len(inputPaths))
			for i, p := range inputPaths {
				fmt.Printf("  [%d] %s\n", i+1, filepath.Base(p))
			}
			fmt.Println()

			report, err := m.Merge(inputPaths, outputPath, merger.MergeOptions{})
			if err != nil {
				return err
			}

			printProgressBar(100)
			fmt.Println()
			fmt.Printf("Done in %s\n", report.Duration.Round(time.Millisecond))
			fmt.Printf("Output:          %s\n", outputPath)
			if stat, err := os.Stat(outputPath); err == nil {
				fmt.Printf("Size:            %s\n", formatSize(stat.Size()))
			}
			fmt.Println()
			fmt.Printf("Merge Summary:\n")
			fmt.Printf("  Total Inputs:  %d manga across %d files\n", report.TotalInputManga, len(report.InputFiles))
			fmt.Printf("  Unique Manga:  %d\n", report.UniqueManga)
			fmt.Printf("  Merged Overlap:%d manga reconciled\n", report.MergedManga)
			fmt.Printf("  Chapters:      %d (%d read)\n", report.TotalChapters, report.ReadChapters)
			fmt.Printf("  Categories:    %d\n", report.CategoriesCount)
			fmt.Printf("  Sources:       %d\n", report.SourcesCount)
			fmt.Printf("  Trackers:      %d\n", report.TrackersCount)
			fmt.Printf("  History:       %d\n", report.HistoryCount)

			return nil
		},
	}
	mergeCmd.Flags().StringVarP(&mergeOutput, "output", "o", "", "Output path for merged backup (.tmb or .tachibk)")

	// === list-formats command ===
	listFormatsCmd := &cobra.Command{
		Use:   "list-formats",
		Short: "List all supported backup formats",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Supported formats:")
			fmt.Println()
			for _, f := range registry.List() {
				fmt.Printf("  %-30s %s\n", f.Name(), strings.Join(f.Extensions(), ", "))
			}
			fmt.Println()
		},
	}

	// === version command ===
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version number",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("mangastu v%s\n", version)
		},
	}

	// === serve command ===
	var servePort int
	var staticDir string
	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Start HTTP web server and REST API",
		Long: `Start the MangaSTU HTTP production server serving the web UI and REST API.

Examples:
  mangastu serve
  mangastu serve --port 8080 --static-dir ./dist`,
		RunE: func(cmd *cobra.Command, args []string) error {
			srv := server.New(servePort, staticDir)
			return srv.Start()
		},
	}
	serveCmd.Flags().IntVarP(&servePort, "port", "p", 8080, "Port to listen on")
	serveCmd.Flags().StringVarP(&staticDir, "static-dir", "s", "./dist", "Directory containing built frontend web assets")

	rootCmd.AddCommand(convertCmd, mergeCmd, inspectCmd, validateCmd, listFormatsCmd, versionCmd, serveCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func printProgressBar(percent int) {
	width := 40
	filled := width * percent / 100
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	fmt.Printf("\r%s %d%%", bar, percent)
	fmt.Println()
}

func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d bytes", bytes)
	}
}
