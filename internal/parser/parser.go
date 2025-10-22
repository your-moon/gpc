package parser

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/your-moon/gpc/internal/debug"
	"github.com/your-moon/gpc/internal/models"
)

// FindGoFiles finds all Go files in a directory
func FindGoFiles(dir string) ([]string, error) {
	debug.PassHeader("FILE DISCOVERY")
	debug.Info("Scanning directory: %s", dir)

	var goFiles []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			debug.Error("Error accessing %s: %v", path, err)
			return err
		}

		if !info.IsDir() && strings.HasSuffix(path, ".go") {
			goFiles = append(goFiles, path)
			debug.Verbose("Found Go file: %s", path)
		}

		return nil
	})

	debug.PassFooter("FILE DISCOVERY", len(goFiles))
	debug.Stats("File Discovery", map[string]interface{}{
		"Directory":      dir,
		"Go Files Found": len(goFiles),
		"Error":          err != nil,
	})

	return goFiles, err
}

// FindAllStructs finds all struct definitions in a directory
func FindAllStructs(dir string) (map[string]models.StructInfo, error) {
	structs := make(map[string]models.StructInfo)

	goFiles, err := FindGoFiles(dir)
	if err != nil {
		return nil, err
	}

	for _, file := range goFiles {
		fileStructs := ParseStructsFromFile(file)
		for name, info := range fileStructs {
			// Make struct names unique by including package path
			// This prevents overwriting when multiple structs have the same name
			uniqueName := getUniqueStructName(name, file)
			structs[uniqueName] = info
		}
	}

	return structs, nil
}

// getUniqueStructName creates a unique name for a struct by including the package path
func getUniqueStructName(structName, filePath string) string {
	// Extract package name from file path
	// e.g., "/path/to/databases/payment.go" -> "databases"
	// e.g., "/path/to/services/superapp/struct.go" -> "superapp"
	dir := filepath.Dir(filePath)
	packageName := filepath.Base(dir)

	// Create unique name: package.struct
	return packageName + "." + structName
}

// ParseStructsFromFile parses struct definitions from a single file
func ParseStructsFromFile(filename string) map[string]models.StructInfo {
	structs := make(map[string]models.StructInfo)

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return structs
	}

	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.TypeSpec:
			if structType, ok := x.Type.(*ast.StructType); ok {
				fields := make(map[string]string)

				for _, field := range structType.Fields.List {
					if len(field.Names) > 0 {
						fieldName := field.Names[0].Name
						fieldType := getFieldType(field.Type)
						fields[fieldName] = fieldType

					}
				}

				structs[x.Name.Name] = models.StructInfo{
					Name:   x.Name.Name,
					Fields: fields,
				}
			}
		}
		return true
	})

	return structs
}

// getFieldType extracts the type string from an AST field type
func getFieldType(expr ast.Expr) string {
	switch x := expr.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.SelectorExpr:
		return getFieldType(x.X) + "." + x.Sel.Name
	case *ast.StarExpr:
		return "*" + getFieldType(x.X)
	case *ast.ArrayType:
		return "[]" + getFieldType(x.Elt)
	default:
		return ""
	}
}

// FindPreloadCalls finds all Preload calls in a file
func FindPreloadCalls(filename string) []models.PreloadCall {
	debug.PassHeader("PRELOAD CALL PARSING")
	debug.Info("Parsing preload calls in: %s", filename)

	var preloadCalls []models.PreloadCall

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		debug.Error("Failed to parse file %s: %v", filename, err)
		return preloadCalls
	}

	// Read file content for line-by-line analysis
	content, err := os.ReadFile(filename)
	if err != nil {
		debug.Error("Failed to read file %s: %v", filename, err)
		return preloadCalls
	}
	lines := strings.Split(string(content), "\n")

	debug.Verbose("File has %d lines", len(lines))

	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.CallExpr:
			if isPreloadCall(x) {
				line := fset.Position(x.Pos()).Line
				relation := extractRelation(x)
				preloadStr := extractPreloadStrFromLine(lines[line-1])

				// Find the function scope
				scope := findFunctionScope(node, x.Pos())

				// Get multi-line content if this is part of a multi-line statement
				fullContent := extractMultiLineContent(lines, line-1)

				debug.Item(len(preloadCalls), "Preload call found at line %d: %s -> %s (scope: %s)",
					line, relation, preloadStr, scope)
				debug.Indent(1, "Line content: %s", lines[line-1])
				if fullContent != lines[line-1] {
					debug.Indent(1, "Multi-line content: %s", fullContent)
				}

				preloadCalls = append(preloadCalls, models.PreloadCall{
					File:        filename,
					Line:        line,
					Relation:    relation,
					LineContent: fullContent, // Use full multi-line content
					PreloadStr:  preloadStr,
					Scope:       scope,
				})
			}
		}
		return true
	})

	debug.PassFooter("PRELOAD CALL PARSING", len(preloadCalls))
	debug.Stats("Preload Parsing", map[string]interface{}{
		"File":                filename,
		"Preload Calls Found": len(preloadCalls),
		"Lines in File":       len(lines),
	})

	return preloadCalls
}

// FindGormCalls finds all GORM method calls in a file
func FindGormCalls(filename string) []models.GormCall {
	var gormCalls []models.GormCall

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return gormCalls
	}

	// Read file content for line-by-line analysis
	content, err := os.ReadFile(filename)
	if err != nil {
		return gormCalls
	}
	lines := strings.Split(string(content), "\n")

	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.CallExpr:
			if isGormCall(x) {
				line := fset.Position(x.Pos()).Line
				method := getGormMethod(x)

				// Find the function scope
				scope := findFunctionScope(node, x.Pos())

				gormCalls = append(gormCalls, models.GormCall{
					File:        filename,
					Line:        line,
					Method:      method,
					LineContent: lines[line-1],
					Scope:       scope,
				})
			}
		}
		return true
	})

	return gormCalls
}

// FindVariableAssignments finds variable assignments that might be used in GORM calls
func FindVariableAssignments(filename string) []models.VariableAssignment {
	var assignments []models.VariableAssignment

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return assignments
	}

	// Read file content for line-by-line analysis
	content, err := os.ReadFile(filename)
	if err != nil {
		return assignments
	}
	lines := strings.Split(string(content), "\n")

	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.AssignStmt:
			if len(x.Lhs) == 1 && len(x.Rhs) == 1 {
				if ident, ok := x.Lhs[0].(*ast.Ident); ok {
					if containsGormCall(x.Rhs[0]) {
						line := fset.Position(x.Pos()).Line
						scope := findFunctionScope(node, x.Pos())

						assignments = append(assignments, models.VariableAssignment{
							VarName:     ident.Name,
							AssignedTo:  getExpressionString(x.Rhs[0]),
							Line:        line,
							File:        filename,
							Scope:       scope,
							LineContent: lines[line-1],
						})
					}
				}
			}
		}
		return true
	})

	return assignments
}

// FindVariableTypes finds variable type declarations in a file
func FindVariableTypes(filename string) []models.VariableType {
	var variableTypes []models.VariableType

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return variableTypes
	}

	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.DeclStmt:
			if genDecl, ok := x.Decl.(*ast.GenDecl); ok {
				for _, spec := range genDecl.Specs {
					if valueSpec, ok := spec.(*ast.ValueSpec); ok {
						for i, name := range valueSpec.Names {
							var typeName string
							if valueSpec.Type != nil {
								typeName = getTypeString(valueSpec.Type)
							} else if len(valueSpec.Values) > i {
								typeName = getTypeString(valueSpec.Values[i])
							}

							if typeName != "" {
								line := fset.Position(x.Pos()).Line
								scope := findFunctionScope(node, x.Pos())
								packageName, modelName := extractPackageAndModelFromType(typeName)

								variableTypes = append(variableTypes, models.VariableType{
									VarName:     name.Name,
									TypeName:    typeName,
									PackageName: packageName,
									ModelName:   modelName,
									Scope:       scope,
									File:        filename,
									Line:        line,
								})
							}
						}
					}
				}
			}
		case *ast.AssignStmt:
			if x.Tok.String() == ":=" {
				for i, lhs := range x.Lhs {
					if ident, ok := lhs.(*ast.Ident); ok {
						var typeName string
						if len(x.Rhs) > i {
							typeName = getTypeString(x.Rhs[i])
						}

						if typeName != "" {
							line := fset.Position(x.Pos()).Line
							scope := findFunctionScope(node, x.Pos())
							packageName, modelName := extractPackageAndModelFromType(typeName)

							variableTypes = append(variableTypes, models.VariableType{
								VarName:     ident.Name,
								TypeName:    typeName,
								PackageName: packageName,
								ModelName:   modelName,
								Scope:       scope,
								File:        filename,
								Line:        line,
							})
						}
					}
				}
			}
		}
		return true
	})

	return variableTypes
}

// Helper functions

func isPreloadCall(call *ast.CallExpr) bool {
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		return sel.Sel.Name == "Preload"
	}
	return false
}

func isGormCall(call *ast.CallExpr) bool {
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		method := sel.Sel.Name
		return method == "Find" || method == "First" || method == "FirstOrCreate" ||
			method == "Take" || method == "Last" || method == "Save" || method == "Create" ||
			method == "Update" || method == "Delete" || method == "Count" || method == "Scan"
	}
	return false
}

func getGormMethod(call *ast.CallExpr) string {
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		return sel.Sel.Name
	}
	return ""
}

func extractRelation(call *ast.CallExpr) string {
	if len(call.Args) > 0 {
		if lit, ok := call.Args[0].(*ast.BasicLit); ok {
			return strings.Trim(lit.Value, "\"")
		}
		// Handle clause.Associations - this is a special GORM constant
		if ident, ok := call.Args[0].(*ast.Ident); ok {
			if ident.Name == "Associations" {
				return "clause.Associations" // Special marker for GORM's clause.Associations
			}
		}
		// Handle clause.Associations when it's a selector expression
		if sel, ok := call.Args[0].(*ast.SelectorExpr); ok {
			if sel.Sel.Name == "Associations" {
				return "clause.Associations" // Special marker for GORM's clause.Associations
			}
		}
	}
	return ""
}

func extractPreloadStrFromLine(lineContent string) string {
	// Find the Preload call in the line
	preloadIndex := strings.Index(lineContent, ".Preload(")
	if preloadIndex == -1 {
		return ""
	}

	// Find the opening parenthesis
	openParen := preloadIndex + 8 // len(".Preload(")
	parenCount := 1
	endIndex := openParen

	for endIndex < len(lineContent) && parenCount > 0 {
		if lineContent[endIndex] == '(' {
			parenCount++
		} else if lineContent[endIndex] == ')' {
			parenCount--
		}
		endIndex++
	}

	if parenCount == 0 {
		return lineContent[preloadIndex:endIndex]
	}

	return ""
}

func containsGormCall(expr ast.Expr) bool {
	switch x := expr.(type) {
	case *ast.CallExpr:
		return isGormCall(x) || isPreloadCall(x)
	case *ast.SelectorExpr:
		return containsGormCall(x.X)
	case *ast.BinaryExpr:
		return containsGormCall(x.X) || containsGormCall(x.Y)
	default:
		return false
	}
}

func getExpressionString(expr ast.Expr) string {
	switch x := expr.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.SelectorExpr:
		return getExpressionString(x.X) + "." + x.Sel.Name
	case *ast.CallExpr:
		return getExpressionString(x.Fun) + "()"
	default:
		return ""
	}
}

func getTypeString(expr ast.Expr) string {
	switch x := expr.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.SelectorExpr:
		return getTypeString(x.X) + "." + x.Sel.Name
	case *ast.ArrayType:
		return "[]" + getTypeString(x.Elt)
	case *ast.StarExpr:
		return "*" + getTypeString(x.X)
	case *ast.CallExpr:
		if arrayType, ok := x.Fun.(*ast.ArrayType); ok {
			return "[]" + getTypeString(arrayType.Elt)
		}
		return getTypeString(x.Fun)
	case *ast.CompositeLit:
		return getTypeString(x.Type)
	default:
		return ""
	}
}

func extractModelFromType(typeName string) string {
	_, modelName := extractPackageAndModelFromType(typeName)
	return modelName
}

// extractPackageAndModelFromType extracts both package and model information from a type string
func extractPackageAndModelFromType(typeName string) (string, string) {
	// Remove array and pointer prefixes
	cleanType := strings.TrimPrefix(typeName, "[]")
	cleanType = strings.TrimPrefix(cleanType, "*")

	// Extract package and model information
	if lastDot := strings.LastIndex(cleanType, "."); lastDot != -1 {
		packageName := cleanType[:lastDot]
		modelName := cleanType[lastDot+1:]
		return packageName, modelName
	}

	// No package qualifier, return empty package and the type as model
	return "", cleanType
}

func findFunctionScope(node *ast.File, pos token.Pos) string {
	var funcName string

	ast.Inspect(node, func(n ast.Node) bool {
		if funcDecl, ok := n.(*ast.FuncDecl); ok {
			if funcDecl.Pos() <= pos && pos <= funcDecl.End() {
				if funcDecl.Name != nil {
					funcName = funcDecl.Name.Name
				}
			}
		}
		return true
	})

	if funcName == "" {
		return "global"
	}
	return funcName
}

// extractMultiLineContent extracts the full content of a multi-line statement
func extractMultiLineContent(lines []string, startLine int) string {
	if startLine < 0 || startLine >= len(lines) {
		return ""
	}

	// Start with the current line
	content := strings.TrimSpace(lines[startLine])

	// If the line doesn't end with a semicolon or closing parenthesis, it might be multi-line
	if !strings.HasSuffix(content, ";") && !strings.HasSuffix(content, ")") && !strings.HasSuffix(content, "}") {
		// Look ahead to find the complete statement
		for i := startLine + 1; i < len(lines); i++ {
			line := strings.TrimSpace(lines[i])
			if line == "" {
				continue // Skip empty lines
			}

			// Stop if we hit a comment line or a new statement
			if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "/*") {
				break
			}

			// Stop if we hit a closing brace that's not part of the statement
			if line == "}" && !strings.Contains(content, "{") {
				break
			}

			content += " " + line

			// Stop if we find a complete statement
			if strings.HasSuffix(line, ";") || strings.HasSuffix(line, ")") || strings.HasSuffix(line, "}") {
				break
			}
		}
	}

	return content
}
