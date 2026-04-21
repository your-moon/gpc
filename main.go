package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/your-moon/gpc/internal/engine"
	"github.com/your-moon/gpc/internal/models"
	"github.com/your-moon/gpc/internal/output"
)

var (
	outputFormat   string
	outputFile     string
	validationOnly bool
	errorsOnly     bool
)

var rootCmd = &cobra.Command{
	Use:   "gpc [directory or file]",
	Short: "Static analysis tool for GORM Preload() calls",
	Long:  "Validates relation names in GORM Preload() calls using type-checked analysis.",
	Args:  cobra.ExactArgs(1),
	Run:   run,
}

func init() {
	rootCmd.Flags().StringVarP(&outputFormat, "format", "o", "text", "Output format: text or json")
	rootCmd.Flags().StringVarP(&outputFile, "file", "f", "", "Write JSON output to file (implies -o json)")
	rootCmd.Flags().BoolVarP(&validationOnly, "valid", "V", false, "Show only validated results (valid and errors)")
	rootCmd.Flags().BoolVarP(&errorsOnly, "errors-only", "e", false, "Show only errors")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) {
	target := args[0]

	info, err := os.Stat(target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gpc: %v\n", err)
		os.Exit(1)
	}

	var dir, filterFile string
	if info.IsDir() {
		dir = target
	} else {
		dir = filepath.Dir(target)
		filterFile, _ = filepath.Abs(target)
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gpc: %v\n", err)
		os.Exit(1)
	}

	results, err := engine.Analyze(absDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gpc: %v\n", err)
		os.Exit(1)
	}

	if filterFile != "" {
		var filtered []models.PreloadResult
		for _, r := range results {
			if r.File == filterFile {
				filtered = append(filtered, r)
			}
		}
		results = filtered
	}

	if outputFile != "" {
		outputFormat = "json"
	}

	if outputFormat == "json" {
		dest := outputFile
		if dest == "" {
			dest = "gpc_results.json"
		}
		if err := output.WriteStructuredOutput(results, dest, validationOnly, errorsOnly); err != nil {
			fmt.Fprintf(os.Stderr, "gpc: %v\n", err)
			os.Exit(1)
		}
	} else {
		output.WriteConsoleOutput(results, validationOnly, errorsOnly)
	}
}
