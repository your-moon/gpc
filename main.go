package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/your-moon/gpc/internal/debug"
	"github.com/your-moon/gpc/internal/service"
)

var (
	outputFormat string
	outputFile   string
	debugMode    bool
	verboseMode  bool
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
	rootCmd.Flags().BoolVarP(&debugMode, "debug", "d", false, "Enable debug output")
	rootCmd.Flags().BoolVarP(&verboseMode, "verbose", "v", false, "Enable verbose output")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func runChecker(cmd *cobra.Command, args []string) {
	target := args[0]

	// Set debug modes
	debug.SetDebugMode(debugMode)
	debug.SetVerboseMode(verboseMode)

	// Create service instance
	svc := service.NewService(outputFormat, outputFile)

	// Run analysis
	if err := svc.AnalyzeTarget(target); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
