package output

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/your-moon/gpc/internal/models"
)

// WriteStructuredOutput writes analysis results to a JSON file
func WriteStructuredOutput(results []models.PreloadResult, outputFile string, validationOnly, errorsOnly bool) error {
	// Filter results based on mode
	filteredResults := results
	if errorsOnly {
		// Show only errors
		filteredResults = []models.PreloadResult{}
		for _, result := range results {
			if result.Status == "error" {
				filteredResults = append(filteredResults, result)
			}
		}
	} else if validationOnly {
		// Show validation results (correct and error)
		filteredResults = []models.PreloadResult{}
		for _, result := range results {
			if result.Status == "correct" || result.Status == "error" {
				filteredResults = append(filteredResults, result)
			}
		}
	}

	// Calculate statistics
	total := len(filteredResults)
	correct := 0
	unknown := 0
	errors := 0

	for _, result := range filteredResults {
		switch result.Status {
		case "correct":
			correct++
		case "unknown":
			unknown++
		case "error":
			errors++
		}
	}

	// Calculate accuracy
	accuracy := 0.0
	if total > 0 {
		accuracy = float64(correct) / float64(total) * 100
	}

	// Create analysis result
	analysisResult := models.AnalysisResult{
		TotalPreloads: total,
		Correct:       correct,
		Unknown:       unknown,
		Errors:        errors,
		Accuracy:      accuracy,
		Results:       filteredResults,
	}

	// Write to JSON file
	jsonData, err := json.MarshalIndent(analysisResult, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling JSON: %v", err)
	}

	err = os.WriteFile(outputFile, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("error writing file: %v", err)
	}

	return nil
}

// WriteConsoleOutput writes analysis results to console
func WriteConsoleOutput(results []models.PreloadResult, validationOnly, errorsOnly bool) {
	// Filter results based on mode
	filteredResults := results
	if errorsOnly {
		// Show only errors
		filteredResults = []models.PreloadResult{}
		for _, result := range results {
			if result.Status == "error" {
				filteredResults = append(filteredResults, result)
			}
		}
	} else if validationOnly {
		// Show validation results (correct and error)
		filteredResults = []models.PreloadResult{}
		for _, result := range results {
			if result.Status == "correct" || result.Status == "error" {
				filteredResults = append(filteredResults, result)
			}
		}
	}

	// Calculate statistics
	total := len(filteredResults)
	correct := 0
	unknown := 0
	errors := 0

	for _, result := range filteredResults {
		switch result.Status {
		case "correct":
			correct++
		case "unknown":
			unknown++
		case "error":
			errors++
		}
	}

	// Calculate accuracy
	accuracy := 0.0
	if total > 0 {
		accuracy = float64(correct) / float64(total) * 100
	}

	fmt.Println("🔍 GORM Preload Analysis Results")
	fmt.Println("=================================")

	// Print each result
	for _, result := range filteredResults {
		status := getStatusEmoji(result.Status)
		fmt.Printf("%s %s:%d %s -> %s", status, result.File, result.Line, result.Relation, result.Model)

		if result.Variable != "" {
			fmt.Printf(" (var: %s", result.Variable)
			if result.FindLine > 0 {
				fmt.Printf(", find: %d", result.FindLine)
			}
			fmt.Printf(")")
		}
		fmt.Println()
	}

	// Print summary
	fmt.Println("\n📊 Summary")
	fmt.Println("==========")
	fmt.Printf("Total Preloads: %d\n", total)
	fmt.Printf("✅ Correct:     %d\n", correct)
	fmt.Printf("❓ Unknown:     %d\n", unknown)
	fmt.Printf("❌ Errors:      %d\n", errors)
	fmt.Printf("📈 Accuracy:    %.1f%%\n", accuracy)
}

// getStatusEmoji returns the appropriate emoji for a status
func getStatusEmoji(status string) string {
	switch status {
	case "correct":
		return "✅"
	case "unknown":
		return "❓"
	case "error":
		return "❌"
	default:
		return "❓"
	}
}
