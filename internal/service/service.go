package service

import (
	"fmt"
	"os"

	"github.com/your-moon/gpc/internal/analyzer"
	"github.com/your-moon/gpc/internal/models"
	"github.com/your-moon/gpc/internal/output"
	"github.com/your-moon/gpc/internal/parser"
)

// Service handles the main analysis workflow
type Service struct {
	outputFormat string
	outputFile   string
}

// NewService creates a new service instance
func NewService(outputFormat, outputFile string) *Service {
	return &Service{
		outputFormat: outputFormat,
		outputFile:   outputFile,
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
	_, err = parser.FindAllStructs(structSearchDir)
	if err != nil {
		return err
	}

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

	// Write output based on format
	if s.outputFormat == "json" {
		err = output.WriteStructuredOutput(results, s.outputFile)
		if err != nil {
			return err
		}
		fmt.Printf("✅ Analysis complete! Results written to %s\n", s.outputFile)
	} else {
		output.WriteConsoleOutput(results)
	}

	return nil
}

// getParentDir returns the parent directory of a file path
func getParentDir(filePath string) string {
	// Simple implementation - in a real scenario you might want to use filepath.Dir
	lastSlash := -1
	for i := len(filePath) - 1; i >= 0; i-- {
		if filePath[i] == '/' {
			lastSlash = i
			break
		}
	}
	if lastSlash > 0 {
		return filePath[:lastSlash]
	}
	return "."
}
