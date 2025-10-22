package service

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/your-moon/gpc/internal/analyzer"
	"github.com/your-moon/gpc/internal/debug"
	"github.com/your-moon/gpc/internal/models"
	"github.com/your-moon/gpc/internal/output"
	"github.com/your-moon/gpc/internal/parser"
	"github.com/your-moon/gpc/internal/validator"
)

// Service handles the main analysis workflow
type Service struct {
	outputFormat   string
	outputFile     string
	validationOnly bool
	errorsOnly     bool
}

// NewService creates a new service instance
func NewService(outputFormat, outputFile string, validationOnly, errorsOnly bool) *Service {
	return &Service{
		outputFormat:   outputFormat,
		outputFile:     outputFile,
		validationOnly: validationOnly,
		errorsOnly:     errorsOnly,
	}
}

// AnalyzeTarget analyzes a file or directory for GORM preload calls
func (s *Service) AnalyzeTarget(target string) error {
	// Determine if target is a file or directory
	info, err := os.Stat(target)
	if err != nil {
		return err
	}

	var preloadFiles []string
	var structSearchDir string

	if info.IsDir() {
		// Directory: find preloads in all Go files in this directory
		preloadFiles, err = parser.FindGoFiles(target)
		if err != nil {
			return err
		}
		structSearchDir = target
	} else {
		// File: find preloads only in this file, but structs in parent directory
		preloadFiles = []string{target}
		structSearchDir = getParentDir(target)
	}

	// Find all structs in the directory (for validation)
	debug.Info("Searching for structs in directory: %s", structSearchDir)
	allStructs, err := parser.FindAllStructs(structSearchDir)
	if err != nil {
		debug.Error("Failed to find structs: %v", err)
		return err
	}
	debug.Info("Found %d structs", len(allStructs))

	// Find preload calls in specified files
	var preloadCalls []models.PreloadCall
	var gormCalls []models.GormCall
	var varAssignments []models.VariableAssignment
	var variableTypes []models.VariableType

	for _, file := range preloadFiles {
		filePreloads := parser.FindPreloadCalls(file)
		preloadCalls = append(preloadCalls, filePreloads...)

		fileGormCalls := parser.FindGormCalls(file)
		gormCalls = append(gormCalls, fileGormCalls...)

		fileVarAssignments := parser.FindVariableAssignments(file)
		varAssignments = append(varAssignments, fileVarAssignments...)

		fileVariableTypes := parser.FindVariableTypes(file)
		variableTypes = append(variableTypes, fileVariableTypes...)
	}

	// Analyze the preload calls
	results := analyzer.AnalyzePreloads(preloadCalls, gormCalls, varAssignments, variableTypes)

	// Validate preload relations against struct definitions
	results = validator.ValidatePreloadRelations(results, allStructs)

	// Write output based on format
	if s.outputFormat == "json" {
		err = output.WriteStructuredOutput(results, s.outputFile, s.validationOnly, s.errorsOnly)
		if err != nil {
			return err
		}
		fmt.Printf("✅ Analysis complete! Results written to %s\n", s.outputFile)
	} else {
		output.WriteConsoleOutput(results, s.validationOnly, s.errorsOnly)
	}

	return nil
}

// getParentDir returns the parent directory of a file path
func getParentDir(filePath string) string {
	// Use filepath.Dir for proper cross-platform path handling
	parent := filepath.Dir(filePath)
	if parent == "." {
		return "."
	}
	return parent
}
