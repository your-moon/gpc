package preloadcheck

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// StructInfo represents a Go struct and its fields
type StructInfo struct {
	Name   string
	Fields []string
}

// PreloadCall represents a Preload call with its path and context
type PreloadCall struct {
	Path     string
	Line     int
	Filename string
	Context  string
}

// FindCall represents a Find call with its model type
type FindCall struct {
	ModelType string
	Line      int
	Filename  string
	Context   string
}

// Analyzer is the ripgrep-based analyzer
var Analyzer = &analysis.Analyzer{
	Name: "preloadcheck",
	Doc:  "check for mistyped GORM Preload relations using ripgrep (faster and simpler than AST)",
	Run:  runRipgrep,
}

func runRipgrep(pass *analysis.Pass) (interface{}, error) {
	// Get the directory from the first file
	if len(pass.Files) == 0 {
		return nil, nil
	}

	// Extract directory from the first file's position
	firstFile := pass.Files[0]
	pos := pass.Fset.Position(firstFile.Pos())
	directory := pos.Filename
	// Get directory by removing filename
	if lastSlash := strings.LastIndex(directory, "/"); lastSlash != -1 {
		directory = directory[:lastSlash]
	}

	// Parse structs from all files using ripgrep
	structs, err := parseStructsWithRipgrep(directory)
	if err != nil {
		pass.Reportf(0, "ripgrep struct parsing error: %v", err)
		return nil, nil
	}

	// Find Preload and Find calls using ripgrep
	preloadCalls, findCalls, err := findPreloadAndFindCalls(directory)
	if err != nil {
		// Fallback to basic reporting if ripgrep fails
		pass.Reportf(0, "ripgrep error: %v", err)
		return nil, nil
	}

	// Validate preload calls
	validatePreloadCalls(pass, preloadCalls, findCalls, structs)

	return nil, nil
}

func parseStructsWithRipgrep(directory string) (map[string]StructInfo, error) {
	structs := make(map[string]StructInfo)

	// Find all struct definitions using ripgrep
	cmd := exec.Command("rg", "-n", `type\s+(\w+)\s+struct`, directory)
	output, err := cmd.Output()
	if err != nil && err.Error() != "exit status 1" {
		return nil, err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 3)
		if len(parts) >= 3 {
			filename := parts[0]
			lineNum := parts[1]
			content := parts[2]

			// Extract struct name
			re := regexp.MustCompile(`type\s+(\w+)\s+struct`)
			matches := re.FindStringSubmatch(content)
			if len(matches) > 1 {
				structName := matches[1]

				// Parse struct fields from the file
				fields, err := parseStructFields(filename, parseInt(lineNum))
				if err == nil {
					structs[structName] = StructInfo{
						Name:   structName,
						Fields: fields,
					}
				}
			}
		}
	}

	return structs, nil
}

func parseStructFields(filename string, startLine int) ([]string, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	var fields []string
	inStruct := false

	// Start from the struct line and look for fields
	for i := startLine - 1; i < len(lines) && i < startLine+50; i++ { // Look up to 50 lines ahead
		line := strings.TrimSpace(lines[i])

		if strings.Contains(line, "struct") && !inStruct {
			inStruct = true
			continue
		}

		if inStruct {
			if line == "}" {
				break // End of struct
			}

			if line != "" && !strings.HasPrefix(line, "//") {
				// Extract field name
				fieldName := extractFieldName(line)
				if fieldName != "" {
					fields = append(fields, fieldName)
				}
			}
		}
	}

	return fields, nil
}

func extractFieldName(line string) string {
	// Simple field name extraction
	// This handles cases like:
	// - Name string
	// - User User
	// - Items []Item
	// - Driver *Driver

	parts := strings.Fields(line)
	if len(parts) > 0 {
		// Skip if it's a comment or empty line
		if strings.HasPrefix(parts[0], "//") || parts[0] == "" {
			return ""
		}
		return parts[0]
	}
	return ""
}

func findPreloadAndFindCalls(directory string) ([]PreloadCall, []FindCall, error) {
	var preloadCalls []PreloadCall
	var findCalls []FindCall

	// Find Preload calls using ripgrep
	preloadCmd := exec.Command("rg", "-n", `\.Preload\("([^"]+)"\)`, directory)
	preloadOutput, err := preloadCmd.Output()
	if err != nil && err.Error() != "exit status 1" { // ripgrep returns 1 when no matches found
		return nil, nil, err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(preloadOutput)))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 3)
		if len(parts) >= 3 {
			filename := parts[0]
			lineNum := parts[1]
			content := parts[2]

			// Extract preload path using regex
			re := regexp.MustCompile(`\.Preload\("([^"]+)"\)`)
			matches := re.FindStringSubmatch(content)
			if len(matches) > 1 {
				preloadCalls = append(preloadCalls, PreloadCall{
					Path:     matches[1],
					Line:     parseInt(lineNum),
					Filename: filename,
					Context:  content,
				})
			}
		}
	}

	// Find Find calls using ripgrep
	findCmd := exec.Command("rg", "-n", `\.Find\(&[^)]+\)`, directory)
	findOutput, err := findCmd.Output()
	if err != nil && err.Error() != "exit status 1" {
		return nil, nil, err
	}

	scanner = bufio.NewScanner(strings.NewReader(string(findOutput)))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 3)
		if len(parts) >= 3 {
			filename := parts[0]
			lineNum := parts[1]
			content := parts[2]

			// Extract model type using regex
			re := regexp.MustCompile(`\.Find\(&(\w+)\)`)
			matches := re.FindStringSubmatch(content)
			if len(matches) > 1 {
				// Try to find the variable declaration to get the actual type
				modelType := findVariableType(filename, matches[1], parseInt(lineNum))
				findCalls = append(findCalls, FindCall{
					ModelType: modelType,
					Line:      parseInt(lineNum),
					Filename:  filename,
					Context:   content,
				})
			} else {
				// Try alternative pattern for &[]Type
				re = regexp.MustCompile(`\.Find\(&\[\](\w+)\)`)
				matches = re.FindStringSubmatch(content)
				if len(matches) > 1 {
					findCalls = append(findCalls, FindCall{
						ModelType: matches[1],
						Line:      parseInt(lineNum),
						Filename:  filename,
						Context:   content,
					})
				}
			}
		}
	}

	return preloadCalls, findCalls, nil
}

func findVariableType(filename, varName string, findLine int) string {
	// Read the file and look for variable declarations
	content, err := os.ReadFile(filename)
	if err != nil {
		return varName // fallback to variable name
	}

	lines := strings.Split(string(content), "\n")

	// Look backwards from the Find call to find the variable declaration
	for i := findLine - 1; i >= 0 && i >= findLine-10; i-- { // Look up to 10 lines back
		line := strings.TrimSpace(lines[i])

		// Look for variable declaration patterns
		// var variableName []Type
		// var variableName Type
		// variableName := []Type{}
		// variableName := Type{}

		// Pattern 1: var varName []Type
		if strings.HasPrefix(line, "var ") && strings.Contains(line, varName) {
			re := regexp.MustCompile(`var\s+` + varName + `\s+\[\](\w+)`)
			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				return matches[1]
			}

			// Pattern 2: var varName Type
			re = regexp.MustCompile(`var\s+` + varName + `\s+(\w+)`)
			matches = re.FindStringSubmatch(line)
			if len(matches) > 1 {
				return matches[1]
			}
		}

		// Pattern 3: varName := []Type{}
		if strings.Contains(line, varName+" :=") {
			re := regexp.MustCompile(varName + `\s*:=\s*\[\](\w+)\{\}`)
			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				return matches[1]
			}

			// Pattern 4: varName := Type{}
			re = regexp.MustCompile(varName + `\s*:=\s*(\w+)\{\}`)
			matches = re.FindStringSubmatch(line)
			if len(matches) > 1 {
				return matches[1]
			}
		}
	}

	return varName // fallback to variable name
}

func parseInt(s string) int {
	var result int
	fmt.Sscanf(s, "%d", &result)
	return result
}

func validatePreloadCalls(pass *analysis.Pass, preloadCalls []PreloadCall, findCalls []FindCall, structs map[string]StructInfo) {
	for _, preload := range preloadCalls {
		// Find the closest Find call after this Preload call (including same line)
		var closestFind *FindCall
		minDistance := 1000 // Reasonable distance threshold

		for _, find := range findCalls {
			if find.Filename == preload.Filename && find.Line >= preload.Line {
				distance := find.Line - preload.Line
				if distance < minDistance {
					minDistance = distance
					closestFind = &find
				}
			}
		}

		if closestFind == nil {
			pass.Reportf(0, "No Find call found after Preload('%s')", preload.Path)
			continue
		}

		// Get the struct for the model type
		modelType := closestFind.ModelType
		structInfo, exists := structs[modelType]
		if !exists {
			pass.Reportf(0, "Model type '%s' not found for Preload('%s')", modelType, preload.Path)
			continue
		}

		// Validate the preload path
		pathParts := strings.Split(preload.Path, ".")
		if !validatePreloadPath(structInfo, pathParts) {
			pass.Reportf(0, "Invalid preload path '%s' for model '%s'", preload.Path, modelType)
		}
	}
}

func validatePreloadPath(structInfo StructInfo, pathParts []string) bool {
	// Simple validation - check if the first part exists in the struct
	// For now, just check the first field. In a full implementation,
	// we'd need to recursively validate nested paths
	if len(pathParts) == 0 {
		return false
	}

	firstField := pathParts[0]
	for _, field := range structInfo.Fields {
		if field == firstField {
			return true
		}
	}

	return false
}
