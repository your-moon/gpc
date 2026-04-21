package output

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/your-moon/gpc/internal/models"
)

func WriteStructuredOutput(results []models.PreloadResult, outputFile string, validationOnly, errorsOnly bool) error {
	filtered := filterResults(results, validationOnly, errorsOnly)
	stats := computeStats(filtered)

	analysisResult := models.AnalysisResult{
		Total:   stats.total,
		Valid:   stats.valid,
		Errors:  stats.errors,
		Skipped: stats.skipped,
		Results: filtered,
	}

	data, err := json.MarshalIndent(analysisResult, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}
	return os.WriteFile(outputFile, data, 0644)
}

func WriteConsoleOutput(results []models.PreloadResult, validationOnly, errorsOnly bool) {
	filtered := filterResults(results, validationOnly, errorsOnly)
	stats := computeStats(filtered)

	for _, r := range filtered {
		file := shortenPath(r.File)
		switch r.Status {
		case "error":
			fmt.Fprintf(os.Stderr, "%s:%d: %s not found in %s\n", file, r.Line, r.Relation, r.Model)
		case "skipped":
			fmt.Fprintf(os.Stderr, "%s:%d: skipped (dynamic argument)\n", file, r.Line)
		}
	}

	if stats.errors > 0 {
		fmt.Fprintf(os.Stderr, "\n%d error(s)\n", stats.errors)
		os.Exit(2)
	}

	if !errorsOnly {
		fmt.Fprintf(os.Stdout, "%d preload(s) checked, %d valid", stats.total, stats.valid)
		if stats.skipped > 0 {
			fmt.Fprintf(os.Stdout, ", %d skipped", stats.skipped)
		}
		fmt.Fprintln(os.Stdout)
	}
}

func filterResults(results []models.PreloadResult, validationOnly, errorsOnly bool) []models.PreloadResult {
	if !validationOnly && !errorsOnly {
		return results
	}
	var out []models.PreloadResult
	for _, r := range results {
		if errorsOnly && r.Status == "error" {
			out = append(out, r)
		} else if validationOnly && (r.Status == "valid" || r.Status == "error") {
			out = append(out, r)
		}
	}
	return out
}

type stats struct {
	total, valid, errors, skipped int
}

func computeStats(results []models.PreloadResult) stats {
	var s stats
	s.total = len(results)
	for _, r := range results {
		switch r.Status {
		case "valid":
			s.valid++
		case "error":
			s.errors++
		case "skipped":
			s.skipped++
		}
	}
	return s
}

func shortenPath(path string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return path
	}
	rel, err := filepath.Rel(cwd, path)
	if err != nil {
		return path
	}
	return rel
}
