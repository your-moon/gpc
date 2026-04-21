package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/your-moon/gpc/internal/models"
	"github.com/your-moon/gpc/internal/output"
	"github.com/your-moon/gpc/internal/engine"
)

var (
	outputFormat   string
	outputFile     string
	validationOnly bool
	errorsOnly     bool
)

var rootCmd = &cobra.Command{
	Use:   "gpc [file or directory]",
	Short: "GORM Preload Checker - validates GORM Preload() calls",
	Long: `A static analysis tool for GORM that detects typos and invalid relation names in Preload() calls.

When you specify a file, it will:
- Find preload calls only in that file
- Find struct definitions in the entire directory (for validation)

When you specify a directory, it will:
- Find preload calls in all Go files in that directory
- Find struct definitions in the entire directory`,
	Args: cobra.ExactArgs(1),
	Run:  runChecker,
}

func init() {
	rootCmd.Flags().StringVarP(&outputFormat, "output", "o", "console", "Output format: console (default) or json")
	rootCmd.Flags().StringVarP(&outputFile, "file", "f", "gpc_results.json", "Output file for json format")
	rootCmd.Flags().BoolVarP(&validationOnly, "validation-only", "V", false, "Show only validation results (errors and correct relations)")
	rootCmd.Flags().BoolVarP(&errorsOnly, "errors-only", "e", false, "Show only error results")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func runChecker(cmd *cobra.Command, args []string) {
	target := args[0]

	info, err := os.Stat(target)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	var dir string
	var filterFile string
	if info.IsDir() {
		dir = target
	} else {
		dir = filepath.Dir(target)
		absPath, _ := filepath.Abs(target)
		filterFile = absPath
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	results, err := engine.Analyze(absDir)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Filter to target file if specified
	if filterFile != "" {
		var filtered []models.PreloadResult
		for _, r := range results {
			if r.File == filterFile {
				filtered = append(filtered, r)
			}
		}
		results = filtered
	}

	if outputFormat == "json" {
		err = output.WriteStructuredOutput(results, outputFile, validationOnly, errorsOnly)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Analysis complete! Results written to %s\n", outputFile)
	} else {
		output.WriteConsoleOutput(results, validationOnly, errorsOnly)
	}
}
