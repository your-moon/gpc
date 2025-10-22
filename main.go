package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// StructInfo holds information about a struct and its fields
type StructInfo struct {
	Name   string
	Fields map[string]string // field name -> field type
}

// PreloadCall represents a Preload call found in the code
type PreloadCall struct {
	File        string
	Line        int
	Relation    string
	Model       string
	LineContent string
	PreloadStr  string // The actual preload string from the line
	Scope       string // The function or scope where this preload call is found
	VarName     string // The variable name if this is part of a variable assignment
}

// GormCall represents a GORM method call (Preload, Find, First, FirstOrCreate)
type GormCall struct {
	File        string
	Line        int
	Method      string // "Preload", "Find", "First", "FirstOrCreate"
	LineContent string
	Scope       string // The function or scope where this call is found
}

// VariableAssignment represents a variable assignment that contains GORM calls
type VariableAssignment struct {
	File        string
	Line        int
	VarName     string // The variable name (e.g., "userDB")
	LineContent string
	Scope       string // The function or scope where this assignment is found
}

// VariableType represents a variable and its actual Go type
type VariableType struct {
	VarName   string // The variable name (e.g., "orders", "currentInvoice")
	TypeName  string // The actual type (e.g., "[]databases.Invoice", "databases.Invoice")
	ModelName string // The extracted model name (e.g., "Invoice")
	Scope     string // The function scope
	File      string // The file path
	Line      int    // The line number
}

// PreloadResult represents a single preload call result
type PreloadResult struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Relation string `json:"relation"`
	Model    string `json:"model"`
	Variable string `json:"variable,omitempty"`
	FindLine int    `json:"find_line,omitempty"`
	Status   string `json:"status"` // "correct", "unknown", "error"
}

// AnalysisResult represents the complete analysis output
type AnalysisResult struct {
	TotalPreloads int             `json:"total_preloads"`
	Correct       int             `json:"correct"`
	Unknown       int             `json:"unknown"`
	Errors        int             `json:"errors"`
	Accuracy      float64         `json:"accuracy"`
	Results       []PreloadResult `json:"results"`
}

var (
	outputFormat string
	outputFile   string
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
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func runChecker(cmd *cobra.Command, args []string) {
	target := args[0]

	// Determine if target is a file or directory
	info, err := os.Stat(target)
	if err != nil {
		fmt.Printf("Error accessing %s: %v\n", target, err)
		os.Exit(1)
	}

	var preloadFiles []string
	var structSearchDir string

	if info.IsDir() {
		// Directory: find preloads in all Go files in this directory
		preloadFiles, err = findGoFiles(target)
		if err != nil {
			fmt.Printf("Error finding Go files: %v\n", err)
			os.Exit(1)
		}
		structSearchDir = target
	} else {
		// File: find preloads only in this file, but structs in parent directory
		preloadFiles = []string{target}
		structSearchDir = filepath.Dir(target)
	}

	// Find all structs in the directory (for validation)
	_, err = findAllStructs(structSearchDir)
	if err != nil {
		fmt.Printf("Error finding structs: %v\n", err)
		os.Exit(1)
	}

	// Find preload calls in specified files
	var preloadCalls []PreloadCall
	var gormCalls []GormCall
	var varAssignments []VariableAssignment
	var variableTypes []VariableType
	for _, file := range preloadFiles {
		filePreloads := findPreloadCalls(file)
		preloadCalls = append(preloadCalls, filePreloads...)

		fileGormCalls := findGormCalls(file)
		gormCalls = append(gormCalls, fileGormCalls...)

		fileVarAssignments := findVariableAssignments(file)
		varAssignments = append(varAssignments, fileVarAssignments...)

		fileVariableTypes := findVariableTypes(file)
		variableTypes = append(variableTypes, fileVariableTypes...)
	}

	// Write structured output to file
	// Write output based on format
	if outputFormat == "json" {
		writeStructuredOutput(preloadCalls, gormCalls, varAssignments, variableTypes)
		fmt.Printf("✅ Analysis complete! Results written to %s\n", outputFile)
	} else {
		writeConsoleOutput(preloadCalls, gormCalls, varAssignments, variableTypes)
	}

	// Print all structs found
	// fmt.Printf("\n=== STRUCTS FOUND ===\n")
	// for name, info := range structs {
	// 	fmt.Printf("Struct: %s\n", name)
	// 	for fieldName, fieldType := range info.Fields {
	// 		fmt.Printf("  %s: %s\n", fieldName, fieldType)
	// 	}
	// 	fmt.Println()
	// }

}

func findGoFiles(dir string) ([]string, error) {
	var goFiles []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, ".go") {
			goFiles = append(goFiles, path)
		}
		return nil
	})
	return goFiles, err
}

func findAllStructs(dir string) (map[string]StructInfo, error) {
	structs := make(map[string]StructInfo)

	goFiles, err := findGoFiles(dir)
	if err != nil {
		return nil, err
	}

	for _, file := range goFiles {
		fileStructs := parseStructsFromFile(file)
		for name, info := range fileStructs {
			structs[name] = info
		}
	}

	return structs, nil
}

func parseStructsFromFile(filename string) map[string]StructInfo {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil
	}

	structs := make(map[string]StructInfo)

	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.TypeSpec:
			if structType, ok := x.Type.(*ast.StructType); ok {
				info := StructInfo{
					Name:   x.Name.Name,
					Fields: make(map[string]string),
				}

				for _, field := range structType.Fields.List {
					for _, name := range field.Names {
						fieldType := getTypeString(field.Type)
						info.Fields[name.Name] = fieldType
					}
				}

				structs[x.Name.Name] = info
			}
		}
		return true
	})

	return structs
}

func findPreloadCalls(filename string) []PreloadCall {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil
	}

	var preloadCalls []PreloadCall
	var currentScope string = "global" // Track current function scope

	// Read the file content to get the actual line text
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil
	}
	lines := strings.Split(string(content), "\n")

	// Find all preload calls in the file
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			// Update current scope when entering a function
			if x.Name != nil {
				currentScope = x.Name.Name
			}
		case *ast.CallExpr:
			if isPreloadCall(x) {
				relation := extractRelation(x)
				// Include all preload calls, even empty strings (for error detection)
				pos := fset.Position(x.Pos())
				lineContent := extractMultiLineContent(lines, pos.Line-1)
				preloadStr := extractPreloadStrFromLine(lineContent)
				preloadCalls = append(preloadCalls, PreloadCall{
					File:        filename,
					Line:        pos.Line,
					Relation:    relation,
					Model:       "Fallback", // Default model - you can implement your own logic
					LineContent: lineContent,
					PreloadStr:  preloadStr,
					Scope:       currentScope,
				})
			}
		}
		return true
	})

	return preloadCalls
}

func extractRelation(call *ast.CallExpr) string {
	if len(call.Args) > 0 {
		if lit, ok := call.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
			return strings.Trim(lit.Value, "\"")
		}
	}
	return ""
}

func extractPreloadStrFromLine(lineContent string) string {
	// Look for patterns like: db.Preload("Driver") or .Preload("Customer")
	// Extract the string inside the Preload() call

	// Find the Preload call in the line
	preloadIndex := strings.Index(lineContent, "Preload(")
	if preloadIndex == -1 {
		return ""
	}

	// Find the opening quote after Preload(
	startIndex := preloadIndex + 8 // len("Preload(")
	for startIndex < len(lineContent) && lineContent[startIndex] != '"' {
		startIndex++
	}

	if startIndex >= len(lineContent) {
		return ""
	}

	// Find the closing quote
	endIndex := startIndex + 1
	for endIndex < len(lineContent) && lineContent[endIndex] != '"' {
		endIndex++
	}

	if endIndex >= len(lineContent) {
		return ""
	}

	// Extract the string between quotes
	return lineContent[startIndex+1 : endIndex]
}

func isPreloadCall(call *ast.CallExpr) bool {
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		return sel.Sel.Name == "Preload"
	}
	return false
}

func isGormCall(call *ast.CallExpr) bool {
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		method := sel.Sel.Name
		return method == "Preload" || method == "Find" || method == "First" || method == "FirstOrCreate"
	}
	return false
}

func getGormMethod(call *ast.CallExpr) string {
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		return sel.Sel.Name
	}
	return ""
}

func findGormCalls(filename string) []GormCall {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil
	}

	var gormCalls []GormCall
	var currentScope string = "global" // Track current function scope

	// Read the file content to get the actual line text
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil
	}
	lines := strings.Split(string(content), "\n")

	// Find all GORM calls in the file
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			// Update current scope when entering a function
			if x.Name != nil {
				currentScope = x.Name.Name
			}
		case *ast.CallExpr:
			if isGormCall(x) {
				method := getGormMethod(x)
				pos := fset.Position(x.Pos())
				lineContent := extractMultiLineContent(lines, pos.Line-1)
				gormCalls = append(gormCalls, GormCall{
					File:        filename,
					Line:        pos.Line,
					Method:      method,
					LineContent: lineContent,
					Scope:       currentScope,
				})
			}
		}
		return true
	})

	return gormCalls
}

func findVariableAssignments(filename string) []VariableAssignment {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil
	}

	var assignments []VariableAssignment
	var currentScope string = "global" // Track current function scope

	// Read the file content to get the actual line text
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil
	}
	lines := strings.Split(string(content), "\n")

	// Find all variable assignments that contain GORM calls
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			// Update current scope when entering a function
			if x.Name != nil {
				currentScope = x.Name.Name
			}
		case *ast.AssignStmt:
			// Handle variable assignments like: userDB := db.Preload("User")
			if x.Tok == token.DEFINE || x.Tok == token.ASSIGN {
				for i, lhs := range x.Lhs {
					if ident, ok := lhs.(*ast.Ident); ok && i < len(x.Rhs) {
						// Check if the right-hand side contains GORM calls
						if containsGormCall(x.Rhs[i]) {
							pos := fset.Position(x.Pos())
							lineContent := extractMultiLineContent(lines, pos.Line-1)
							assignments = append(assignments, VariableAssignment{
								File:        filename,
								Line:        pos.Line,
								VarName:     ident.Name,
								LineContent: lineContent,
								Scope:       currentScope,
							})
						}
					}
				}
			}
		}
		return true
	})

	return assignments
}

func containsGormCall(expr ast.Expr) bool {
	// Check if the expression contains any GORM method calls
	hasGormCall := false
	ast.Inspect(expr, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			if isGormCall(call) {
				hasGormCall = true
				return false // Stop traversing
			}
		}
		return true
	})
	return hasGormCall
}

func findVariableTypes(filename string) []VariableType {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil
	}

	var variableTypes []VariableType
	var currentScope string = "global"

	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			// Update current scope when entering a function
			if x.Name != nil {
				currentScope = x.Name.Name
			}
		case *ast.GenDecl:
			// Handle variable declarations like: var orders []databases.Invoice
			if x.Tok == token.VAR {
				for _, spec := range x.Specs {
					if valueSpec, ok := spec.(*ast.ValueSpec); ok {
						for i, name := range valueSpec.Names {
							var typeName string
							if valueSpec.Type != nil {
								typeName = getTypeString(valueSpec.Type)
							} else if i < len(valueSpec.Values) {
								// Handle cases like: var orders = []databases.Invoice{}
								if call, ok := valueSpec.Values[i].(*ast.CallExpr); ok {
									if composite, ok := call.Fun.(*ast.ArrayType); ok {
										typeName = getTypeString(composite)
									}
								}
							}

							if typeName != "" {
								pos := fset.Position(name.Pos())
								modelName := extractModelFromType(typeName)
								variableTypes = append(variableTypes, VariableType{
									VarName:   name.Name,
									TypeName:  typeName,
									ModelName: modelName,
									Scope:     currentScope,
									File:      filename,
									Line:      pos.Line,
								})
							}
						}
					}
				}
			}
		case *ast.AssignStmt:
			// Handle short variable declarations like: orders := []databases.Invoice{}
			if x.Tok == token.DEFINE {
				for i, lhs := range x.Lhs {
					if ident, ok := lhs.(*ast.Ident); ok && i < len(x.Rhs) {
						typeName := getTypeString(x.Rhs[i])
						if typeName != "" {
							pos := fset.Position(ident.Pos())
							modelName := extractModelFromType(typeName)
							variableTypes = append(variableTypes, VariableType{
								VarName:   ident.Name,
								TypeName:  typeName,
								ModelName: modelName,
								Scope:     currentScope,
								File:      filename,
								Line:      pos.Line,
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

func extractModelFromType(typeName string) string {
	// Extract model name from type declarations like:
	// "[]databases.Invoice" -> "Invoice"
	// "databases.Invoice" -> "Invoice"
	// "[]Invoice" -> "Invoice"
	// "Invoice" -> "Invoice"

	// Remove array prefix if present
	if strings.HasPrefix(typeName, "[]") {
		typeName = typeName[2:]
	}

	// Remove pointer prefix if present
	if strings.HasPrefix(typeName, "*") {
		typeName = typeName[1:]
	}

	// Extract the last part after the last dot (package.Model -> Model)
	if lastDot := strings.LastIndex(typeName, "."); lastDot != -1 {
		return typeName[lastDot+1:]
	}

	return typeName
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
		// Handle cases like []databases.Invoice{}
		if arrayType, ok := x.Fun.(*ast.ArrayType); ok {
			return "[]" + getTypeString(arrayType.Elt)
		}
		return getTypeString(x.Fun)
	case *ast.CompositeLit:
		// Handle cases like databases.Invoice{}
		return getTypeString(x.Type)
	default:
		return ""
	}
}

func determineModelTypes(preloadCalls []PreloadCall, gormCalls []GormCall, varAssignments []VariableAssignment, variableTypes []VariableType) {
	// Create a map of variable assignments for quick lookup
	varMap := make(map[string]VariableAssignment)
	for _, assignment := range varAssignments {
		varMap[assignment.VarName] = assignment
	}

	// Create a map of variable types for quick lookup
	typeMap := make(map[string]VariableType)
	for _, varType := range variableTypes {
		// Use file + scope + variable name as key to handle variables with same name in different scopes/files
		key := varType.File + ":" + varType.Scope + ":" + varType.VarName
		typeMap[key] = varType
	}

	// Group calls by scope for analysis
	scopeGroups := make(map[string][]GormCall)
	for _, call := range gormCalls {
		scopeGroups[call.Scope] = append(scopeGroups[call.Scope], call)
	}

	// Create a map to store model types for each preload call by line
	preloadModels := make(map[string]string) // key: "file:line", value: model

	// Analyze each scope to determine model types
	for _, calls := range scopeGroups {
		// Find Find calls and trace back to determine model types
		for _, call := range calls {
			if call.Method == "Find" {
				model := findModelFromFindCall(call, varMap, calls)
				// Store the model for this specific line
				key := fmt.Sprintf("%s:%d", call.File, call.Line)
				preloadModels[key] = model
			}
		}
	}

	// Print preload calls with their models and variable information
	for _, call := range preloadCalls {
		// Find the variable name and Find call that determined this model
		varName, findCall := findVariableAndFindCall(call, gormCalls, varMap)

		// Determine model from the actual variable type
		var model string
		if varName != "" {
			// Look up the actual type of the variable
			key := call.File + ":" + call.Scope + ":" + varName
			if varType, exists := typeMap[key]; exists {
				model = varType.ModelName
			} else {
				// Fallback to variable name inference if type not found
				model = inferModelFromVariableName(varName)
			}
		}

		// If no model found from variable type, try the old approach
		if model == "" {
			// Try to find the model for this specific line
			key := fmt.Sprintf("%s:%d", call.File, call.Line)
			model = preloadModels[key]

			// If not found, try to find the closest Find call in the same scope
			if model == "" {
				model = findClosestModelForPreload(call, gormCalls, varMap)
			}
		}

		if model == "" {
			model = "Unknown"
		}

		if varName != "" && findCall != "" {
			fmt.Printf("%s:%d: Preload(\"%s\") -> Model: %s (var: %s, find: %s)\n",
				call.File, call.Line, call.Relation, model, varName, findCall)
		} else {
			fmt.Printf("%s:%d: Preload(\"%s\") -> Model: %s\n",
				call.File, call.Line, call.Relation, model)
		}
	}
}

func findVariableAndFindCall(preloadCall PreloadCall, gormCalls []GormCall, varMap map[string]VariableAssignment) (string, string) {
	// First, check if the preload call is part of a method chain that contains a Find/First/FirstOrCreate call on the same line
	if strings.Contains(preloadCall.LineContent, ".Find(") ||
		strings.Contains(preloadCall.LineContent, ".First(") ||
		strings.Contains(preloadCall.LineContent, ".FirstOrCreate(") {
		// Extract variable name from the Find/First/FirstOrCreate call in the same line
		varName := extractVariableNameFromFindCall(preloadCall.LineContent)
		if varName != "" {
			return varName, fmt.Sprintf("line %d", preloadCall.Line)
		}
	}

	// Second, try to find the variable that was assigned to this preload call
	preloadVarName := extractVariableFromPreloadCall(preloadCall.LineContent)

	if preloadVarName != "" {
		// Look for a Find/First/FirstOrCreate call that uses this variable
		for _, call := range gormCalls {
			if (call.Method == "Find" || call.Method == "First" || call.Method == "FirstOrCreate") &&
				call.Scope == preloadCall.Scope && call.File == preloadCall.File {
				// Check if this call uses the same variable
				if strings.Contains(call.LineContent, preloadVarName+".") {
					varName := extractVariableNameFromFindCall(call.LineContent)
					return varName, fmt.Sprintf("line %d", call.Line)
				}
			}
		}
	}

	// If no variable association found, try to find the Find/First/FirstOrCreate call that comes immediately after this preload call
	var nextFindCall *GormCall
	minDistance := 999999

	for _, call := range gormCalls {
		if (call.Method == "Find" || call.Method == "First" || call.Method == "FirstOrCreate") &&
			call.Scope == preloadCall.Scope && call.File == preloadCall.File {
			// Only consider calls that come after the preload call
			if call.Line > preloadCall.Line {
				distance := call.Line - preloadCall.Line
				if distance < minDistance {
					minDistance = distance
					nextFindCall = &call
				}
			}
		}
	}

	if nextFindCall != nil {
		// Extract variable name from the Find call
		varName := extractVariableNameFromFindCall(nextFindCall.LineContent)
		return varName, fmt.Sprintf("line %d", nextFindCall.Line)
	}

	// If no Find/First/FirstOrCreate call found after the preload, fall back to closest call
	var closestFindCall *GormCall
	minDistance = 999999

	for _, call := range gormCalls {
		if (call.Method == "Find" || call.Method == "First" || call.Method == "FirstOrCreate") &&
			call.Scope == preloadCall.Scope && call.File == preloadCall.File {
			distance := abs(call.Line - preloadCall.Line)
			if distance < minDistance {
				minDistance = distance
				closestFindCall = &call
			}
		}
	}

	if closestFindCall != nil {
		// Extract variable name from the Find call
		varName := extractVariableNameFromFindCall(closestFindCall.LineContent)
		return varName, fmt.Sprintf("line %d", closestFindCall.Line)
	}

	return "", ""
}

func extractVariableFromPreloadCall(lineContent string) string {
	// Look for patterns like: orderDB := db.Preload("User") -> orderDB
	// or: userDB := orderDB.Preload("User") -> userDB

	// Find the assignment operator
	assignIndex := strings.Index(lineContent, ":=")
	if assignIndex == -1 {
		assignIndex = strings.Index(lineContent, "=")
		if assignIndex == -1 {
			return ""
		}
	}

	// Find the variable name before the assignment
	startIndex := assignIndex - 1
	for startIndex >= 0 && (lineContent[startIndex] == ' ' || lineContent[startIndex] == '\t') {
		startIndex--
	}

	if startIndex < 0 {
		return ""
	}

	// Find the beginning of the variable name
	endIndex := startIndex + 1
	for startIndex >= 0 && (lineContent[startIndex] != ' ' && lineContent[startIndex] != '\t' && lineContent[startIndex] != '\n') {
		startIndex--
	}
	startIndex++ // Move past the space/tab

	if startIndex >= endIndex {
		return ""
	}

	return lineContent[startIndex:endIndex]
}

func extractVariableNameFromFindCall(lineContent string) string {
	// Look for patterns like: .Find(&orders) -> orders, .First(&user) -> user, .FirstOrCreate(&item) -> item
	var findIndex int
	if strings.Contains(lineContent, ".Find(") {
		findIndex = strings.Index(lineContent, ".Find(")
	} else if strings.Contains(lineContent, ".First(") {
		findIndex = strings.Index(lineContent, ".First(")
	} else if strings.Contains(lineContent, ".FirstOrCreate(") {
		findIndex = strings.Index(lineContent, ".FirstOrCreate(")
	} else {
		return ""
	}

	// Find the opening parenthesis
	var parenIndex int
	if strings.Contains(lineContent, ".Find(") {
		parenIndex = findIndex + 5 // len(".Find(")
	} else if strings.Contains(lineContent, ".First(") {
		parenIndex = findIndex + 6 // len(".First(")
	} else if strings.Contains(lineContent, ".FirstOrCreate(") {
		parenIndex = findIndex + 14 // len(".FirstOrCreate(")
	}

	if parenIndex >= len(lineContent) {
		return ""
	}

	// Look for &variable pattern
	ampIndex := strings.Index(lineContent[parenIndex:], "&")
	if ampIndex == -1 {
		return ""
	}

	ampIndex += parenIndex + 1 // Move past the &

	// Find the end of the variable name
	endIndex := ampIndex
	for endIndex < len(lineContent) && lineContent[endIndex] != ')' && lineContent[endIndex] != ' ' && lineContent[endIndex] != ',' {
		endIndex++
	}

	if endIndex <= ampIndex {
		return ""
	}

	varName := lineContent[ampIndex:endIndex]

	// Remove trailing commas and whitespace
	varName = strings.TrimRight(varName, ", \t")

	return varName
}

func findClosestModelForPreload(preloadCall PreloadCall, gormCalls []GormCall, varMap map[string]VariableAssignment) string {
	// Find the closest Find call in the same scope and file
	var closestFindCall *GormCall
	minDistance := 999999

	for _, call := range gormCalls {
		if call.Method == "Find" && call.Scope == preloadCall.Scope && call.File == preloadCall.File {
			distance := abs(call.Line - preloadCall.Line)
			if distance < minDistance {
				minDistance = distance
				closestFindCall = &call
			}
		}
	}

	if closestFindCall != nil {
		// Get all calls in the same scope for context
		var scopeCalls []GormCall
		for _, call := range gormCalls {
			if call.Scope == preloadCall.Scope {
				scopeCalls = append(scopeCalls, call)
			}
		}
		return findModelFromFindCall(*closestFindCall, varMap, scopeCalls)
	}

	return ""
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func findModelFromFindCall(findCall GormCall, varMap map[string]VariableAssignment, allCalls []GormCall) string {
	// Extract the variable name from the Find call
	// Look for patterns like: userDB.Find(&orders) or db.Find(&trips)

	lineContent := findCall.LineContent

	// Try to extract variable name before .Find()
	// Handle both single-line and multi-line calls with spaces
	dotIndex := strings.Index(lineContent, ".Find(")
	if dotIndex == -1 {
		// Try with spaces: ". Find("
		dotIndex = strings.Index(lineContent, ". Find(")
		if dotIndex == -1 {
			return "Unknown"
		}
	}

	// Find the start of the variable name
	startIndex := dotIndex - 1
	for startIndex >= 0 && (lineContent[startIndex] != ' ' && lineContent[startIndex] != '\t' && lineContent[startIndex] != '\n') {
		startIndex--
	}
	startIndex++ // Move past the space/tab

	if startIndex >= dotIndex {
		return "Unknown"
	}

	// Always infer from the Find argument, not the variable name
	// The variable name doesn't indicate the model type being queried
	return inferModelFromFindArgument(lineContent)
}

func inferModelFromVariableName(varName string) string {
	// Convert variable name to model type
	// user -> User, orders -> Order, trips2 -> Trip, etc.

	// Handle numbered variables like invoices2, trips3, etc.
	if len(varName) > 1 && varName[len(varName)-1] >= '0' && varName[len(varName)-1] <= '9' {
		// Remove the number suffix
		for i := len(varName) - 1; i >= 0; i-- {
			if varName[i] < '0' || varName[i] > '9' {
				varName = varName[:i+1]
				break
			}
		}
	}

	// Handle plural forms
	if strings.HasSuffix(varName, "s") && len(varName) > 1 {
		base := varName[:len(varName)-1]
		return capitalizeFirst(base)
	}

	return capitalizeFirst(varName)
}

func inferModelFromFindArgument(lineContent string) string {
	// Look for patterns like: .Find(&orders) -> Order, .Find(&trips) -> Trip
	// Find the argument to Find()

	findIndex := strings.Index(lineContent, ".Find(")
	if findIndex == -1 {
		return "Unknown"
	}

	// Find the opening parenthesis
	parenIndex := findIndex + 5 // len(".Find(")
	if parenIndex >= len(lineContent) {
		return "Unknown"
	}

	// Look for &variable pattern
	ampIndex := strings.Index(lineContent[parenIndex:], "&")
	if ampIndex == -1 {
		return "Unknown"
	}

	ampIndex += parenIndex + 1 // Move past the &

	// Find the end of the variable name
	endIndex := ampIndex
	for endIndex < len(lineContent) && lineContent[endIndex] != ')' && lineContent[endIndex] != ' ' {
		endIndex++
	}

	if endIndex <= ampIndex {
		return "Unknown"
	}

	varName := lineContent[ampIndex:endIndex]

	// Convert variable name to model type
	// orders -> Order, trips -> Trip, invoices2 -> Invoice, etc.

	// Handle numbered variables like invoices2, trips3, etc.
	if len(varName) > 1 && varName[len(varName)-1] >= '0' && varName[len(varName)-1] <= '9' {
		// Remove the number suffix
		for i := len(varName) - 1; i >= 0; i-- {
			if varName[i] < '0' || varName[i] > '9' {
				varName = varName[:i+1]
				break
			}
		}
	}

	// Handle plural forms
	if strings.HasSuffix(varName, "s") && len(varName) > 1 {
		base := varName[:len(varName)-1]
		return capitalizeFirst(base)
	}

	return capitalizeFirst(varName)
}

func capitalizeFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(string(s[0])) + s[1:]
}

func writeStructuredOutput(preloadCalls []PreloadCall, gormCalls []GormCall, varAssignments []VariableAssignment, variableTypes []VariableType) {
	// Create a map of variable assignments for quick lookup
	varMap := make(map[string]VariableAssignment)
	for _, assignment := range varAssignments {
		varMap[assignment.VarName] = assignment
	}

	// Create a map of variable types for quick lookup
	typeMap := make(map[string]VariableType)
	for _, varType := range variableTypes {
		// Use file + scope + variable name as key to handle variables with same name in different scopes/files
		key := varType.File + ":" + varType.Scope + ":" + varType.VarName
		typeMap[key] = varType
	}

	var results []PreloadResult
	correct := 0
	unknown := 0
	errors := 0

	// Process each preload call
	for _, call := range preloadCalls {
		// Find the variable name and Find call that determined this model
		varName, findCall := findVariableAndFindCall(call, gormCalls, varMap)

		// Determine model from the actual variable type
		var model string
		var status string

		if varName != "" {
			// Look up the actual type of the variable
			key := call.File + ":" + call.Scope + ":" + varName
			if varType, exists := typeMap[key]; exists {
				model = varType.ModelName
				status = "correct"
				correct++
			} else {
				// Fallback to variable name inference if type not found
				model = inferModelFromVariableName(varName)
				if model != "" {
					status = "correct"
					correct++
				} else {
					status = "unknown"
					unknown++
				}
			}
		} else {
			model = "Unknown"
			status = "unknown"
			unknown++
		}

		// Extract find line number if available
		findLine := 0
		if findCall != "" && strings.HasPrefix(findCall, "line ") {
			fmt.Sscanf(findCall, "line %d", &findLine)
		}

		// Create result
		result := PreloadResult{
			File:     call.File,
			Line:     call.Line,
			Relation: call.Relation,
			Model:    model,
			Status:   status,
		}

		if varName != "" {
			result.Variable = varName
		}
		if findLine > 0 {
			result.FindLine = findLine
		}

		results = append(results, result)
	}

	// Calculate accuracy
	total := len(preloadCalls)
	accuracy := 0.0
	if total > 0 {
		accuracy = float64(correct) / float64(total) * 100
	}

	// Create analysis result
	analysisResult := AnalysisResult{
		TotalPreloads: total,
		Correct:       correct,
		Unknown:       unknown,
		Errors:        errors,
		Accuracy:      accuracy,
		Results:       results,
	}

	// Write to JSON file
	jsonData, err := json.MarshalIndent(analysisResult, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		return
	}

	err = os.WriteFile(outputFile, jsonData, 0644)
	if err != nil {
		fmt.Printf("Error writing file: %v\n", err)
		return
	}
}

func writeConsoleOutput(preloadCalls []PreloadCall, gormCalls []GormCall, varAssignments []VariableAssignment, variableTypes []VariableType) {
	// Create a map of variable assignments for quick lookup
	varMap := make(map[string]VariableAssignment)
	for _, assignment := range varAssignments {
		varMap[assignment.VarName] = assignment
	}

	// Create a map of variable types for quick lookup
	typeMap := make(map[string]VariableType)
	for _, varType := range variableTypes {
		// Use file + scope + variable name as key to handle variables with same name in different scopes/files
		key := varType.File + ":" + varType.Scope + ":" + varType.VarName
		typeMap[key] = varType
	}

	correct := 0
	unknown := 0
	errors := 0

	fmt.Println("🔍 GORM Preload Analysis Results")
	fmt.Println("=================================")

	// Process each preload call
	for _, call := range preloadCalls {
		// Find the variable name and Find call that determined this model
		varName, findCall := findVariableAndFindCall(call, gormCalls, varMap)

		// Determine model from the actual variable type
		var model string
		var status string

		if varName != "" {
			// Look up the actual type of the variable
			key := call.File + ":" + call.Scope + ":" + varName
			if varType, exists := typeMap[key]; exists {
				model = varType.ModelName
				status = "✅"
				correct++
			} else {
				// Fallback to variable name inference if type not found
				model = inferModelFromVariableName(varName)
				if model != "" {
					status = "✅"
					correct++
				} else {
					status = "❓"
					unknown++
				}
			}
		} else {
			model = "Unknown"
			status = "❓"
			unknown++
		}

		// Print result
		fmt.Printf("%s %s:%d %s -> %s", status, call.File, call.Line, call.Relation, model)
		if varName != "" {
			fmt.Printf(" (var: %s", varName)
			if findCall != "" {
				fmt.Printf(", %s", findCall)
			}
			fmt.Printf(")")
		}
		fmt.Println()
	}

	// Print summary
	total := len(preloadCalls)
	accuracy := 0.0
	if total > 0 {
		accuracy = float64(correct) / float64(total) * 100
	}

	fmt.Println("\n📊 Summary")
	fmt.Println("==========")
	fmt.Printf("Total Preloads: %d\n", total)
	fmt.Printf("✅ Correct:     %d\n", correct)
	fmt.Printf("❓ Unknown:     %d\n", unknown)
	fmt.Printf("❌ Errors:      %d\n", errors)
	fmt.Printf("📈 Accuracy:    %.1f%%\n", accuracy)
}

// extractMultiLineContent extracts the complete content of a multi-line GORM call
func extractMultiLineContent(lines []string, startLine int) string {
	if startLine < 0 || startLine >= len(lines) {
		return ""
	}

	// Start with the first line
	content := strings.TrimSpace(lines[startLine])

	// If the line ends with a dot, it's likely a multi-line call
	// Continue reading until we find a complete statement
	if strings.HasSuffix(content, ".") {
		// Look for the next few lines to complete the call
		for i := startLine + 1; i < len(lines) && i < startLine+10; i++ {
			nextLine := strings.TrimSpace(lines[i])
			if nextLine == "" {
				continue // Skip empty lines
			}

			content += nextLine

			// Stop if we find a complete statement (ends with semicolon, closing brace, or Find/First/FirstOrCreate)
			if strings.HasSuffix(nextLine, ";") ||
				strings.HasSuffix(nextLine, "}") ||
				strings.Contains(nextLine, ".Find(") ||
				strings.Contains(nextLine, ".First(") ||
				strings.Contains(nextLine, ".FirstOrCreate(") {
				break
			}
		}
	}

	return content
}
