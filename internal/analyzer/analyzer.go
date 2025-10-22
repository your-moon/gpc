package analyzer

import (
	"fmt"
	"strings"

	"github.com/your-moon/gpc/internal/debug"
	"github.com/your-moon/gpc/internal/models"
)

// AnalyzePreloads analyzes preload calls and determines their associated models
func AnalyzePreloads(preloadCalls []models.PreloadCall, gormCalls []models.GormCall, varAssignments []models.VariableAssignment, variableTypes []models.VariableType) []models.PreloadResult {
	debug.PassHeader("MODEL ANALYSIS")
	debug.Info("Analyzing %d preload calls with %d gorm calls, %d variable assignments, %d variable types",
		len(preloadCalls), len(gormCalls), len(varAssignments), len(variableTypes))

	// Create lookup maps
	varMap := make(map[string]models.VariableAssignment)
	for _, assignment := range varAssignments {
		varMap[assignment.VarName] = assignment
		debug.Verbose("Variable assignment: %s -> %s (line %d)",
			assignment.VarName, assignment.AssignedTo, assignment.Line)
	}

	typeMap := make(map[string]models.VariableType)
	for _, varType := range variableTypes {
		// Use file + scope + variable name as key to handle variables with same name in different scopes/files
		key := varType.File + ":" + varType.Scope + ":" + varType.VarName
		typeMap[key] = varType
		debug.Verbose("Variable type: %s -> %s (model: %s, scope: %s)",
			varType.VarName, varType.TypeName, varType.ModelName, varType.Scope)
	}

	var results []models.PreloadResult

	// Process each preload call
	for i, call := range preloadCalls {
		debug.Section(fmt.Sprintf("Processing Preload Call %d", i+1))
		debug.Info("Preload: %s at line %d in scope %s", call.Relation, call.Line, call.Scope)
		debug.Indent(1, "Line content: %s", call.LineContent)

		// Find the variable name and Find call that determined this model
		varName, findCall := findVariableAndFindCall(call, gormCalls, varMap)
		debug.Indent(1, "Found variable: '%s', find call: '%s'", varName, findCall)

		// Determine model from the actual variable type
		var model string
		var status string

		if varName != "" {
			// Look up the actual type of the variable
			key := call.File + ":" + call.Scope + ":" + varName
			debug.Indent(1, "Looking up type with key: %s", key)

			if varType, exists := typeMap[key]; exists {
				model = varType.ModelName
				status = "correct"
				debug.Indent(2, "Found type: %s -> model: %s (CORRECT)", varType.TypeName, model)
			} else {
				// Fallback to variable name inference if type not found
				model = inferModelFromVariableName(varName)
				if model != "" {
					status = "correct"
					debug.Indent(2, "Inferred from variable name: %s -> %s (CORRECT)", varName, model)
				} else {
					status = "unknown"
					debug.Indent(2, "Could not infer model from variable name: %s (UNKNOWN)", varName)
				}
			}
		} else {
			model = "Unknown"
			status = "unknown"
			debug.Indent(1, "No variable found (UNKNOWN)")
		}

		// Extract find line number if available
		findLine := 0
		if findCall != "" && strings.HasPrefix(findCall, "line ") {
			fmt.Sscanf(findCall, "line %d", &findLine)
		}

		// Create result
		result := models.PreloadResult{
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

		debug.Indent(1, "Result: %s -> %s (%s)", call.Relation, model, status)
		results = append(results, result)
	}

	debug.PassFooter("MODEL ANALYSIS", len(results))

	// Calculate and display statistics
	correct := 0
	unknown := 0
	errors := 0
	for _, result := range results {
		switch result.Status {
		case "correct":
			correct++
		case "unknown":
			unknown++
		case "error":
			errors++
		}
	}

	debug.Stats("Analysis Results", map[string]interface{}{
		"Total Preloads": len(results),
		"Correct":        correct,
		"Unknown":        unknown,
		"Errors":         errors,
		"Accuracy":       fmt.Sprintf("%.1f%%", float64(correct)/float64(len(results))*100),
	})

	return results
}

// findVariableAndFindCall finds the variable name and Find call associated with a preload call
func findVariableAndFindCall(preloadCall models.PreloadCall, gormCalls []models.GormCall, varMap map[string]models.VariableAssignment) (string, string) {
	// Normalize the line content by removing extra spaces for better pattern matching
	normalizedContent := strings.ReplaceAll(preloadCall.LineContent, " .", ".")
	normalizedContent = strings.ReplaceAll(normalizedContent, ". ", ".")

	// First, check if the preload call is part of a method chain that contains a Find/First/FirstOrCreate call on the same line
	if strings.Contains(normalizedContent, ".Find(") ||
		strings.Contains(normalizedContent, ".First(") ||
		strings.Contains(normalizedContent, ".FirstOrCreate(") {
		// Extract variable name from the Find/First/FirstOrCreate call in the same line
		varName := extractVariableNameFromFindCall(normalizedContent)
		if varName != "" {
			return varName, fmt.Sprintf("line %d", preloadCall.Line)
		}
	}

	// Check if there's a variable assigned to this preload call
	assignedVar := extractVariableFromPreloadCall(preloadCall.LineContent)
	if assignedVar != "" {
		// Look for a Find/First/FirstOrCreate call that uses this variable
		for _, call := range gormCalls {
			if call.Scope == preloadCall.Scope && call.File == preloadCall.File {
				if (call.Method == "Find" || call.Method == "First" || call.Method == "FirstOrCreate") &&
					strings.Contains(call.LineContent, assignedVar) {
					return assignedVar, fmt.Sprintf("line %d", call.Line)
				}
			}
		}
	}

	// Look for the closest Find/First/FirstOrCreate call after this preload
	closestCall := findClosestModelForPreload(preloadCall, gormCalls, varMap)
	if closestCall != "" {
		return closestCall, "closest call"
	}

	return "", ""
}

// extractVariableFromPreloadCall extracts the variable name from a line containing a preload call and an assignment
func extractVariableFromPreloadCall(lineContent string) string {
	// Look for patterns like: varDB := db.Preload("User")
	// or: orderDB = db.Preload("User")

	// Check for := assignment
	if assignIndex := strings.Index(lineContent, ":="); assignIndex != -1 {
		beforeAssign := strings.TrimSpace(lineContent[:assignIndex])
		if strings.Contains(beforeAssign, "db.Preload") || strings.Contains(beforeAssign, ".Preload") {
			// Extract variable name before :=
			parts := strings.Fields(beforeAssign)
			if len(parts) > 0 {
				return parts[len(parts)-1]
			}
		}
	}

	// Check for = assignment
	if assignIndex := strings.Index(lineContent, "="); assignIndex != -1 {
		beforeAssign := strings.TrimSpace(lineContent[:assignIndex])
		if strings.Contains(beforeAssign, "db.Preload") || strings.Contains(beforeAssign, ".Preload") {
			// Extract variable name before =
			parts := strings.Fields(beforeAssign)
			if len(parts) > 0 {
				return parts[len(parts)-1]
			}
		}
	}

	return ""
}

// extractVariableNameFromFindCall extracts the variable name from a Find/First/FirstOrCreate call
func extractVariableNameFromFindCall(lineContent string) string {
	// Look for patterns like: .Find(&variable) or .First(&variable)

	// Find the method call
	var method string
	if strings.Contains(lineContent, ".Find(") {
		method = ".Find("
	} else if strings.Contains(lineContent, ".First(") {
		method = ".First("
	} else if strings.Contains(lineContent, ".FirstOrCreate(") {
		method = ".FirstOrCreate("
	} else {
		return ""
	}

	// Find the start of the method call
	methodIndex := strings.Index(lineContent, method)
	if methodIndex == -1 {
		return ""
	}

	// Find the & symbol after the method call
	ampIndex := strings.Index(lineContent[methodIndex:], "&")
	if ampIndex == -1 {
		return ""
	}
	ampIndex += methodIndex + 1 // Adjust for the slice

	// Find the end of the variable name
	endIndex := ampIndex
	for endIndex < len(lineContent) && lineContent[endIndex] != ')' && lineContent[endIndex] != ' ' && lineContent[endIndex] != ',' {
		endIndex++
	}

	if endIndex > ampIndex {
		varName := lineContent[ampIndex:endIndex]
		// Remove trailing commas and whitespace
		varName = strings.TrimRight(varName, ", \t")
		return varName
	}

	return ""
}

// findClosestModelForPreload finds the closest model for a preload call
func findClosestModelForPreload(preloadCall models.PreloadCall, gormCalls []models.GormCall, varMap map[string]models.VariableAssignment) string {
	var closestCall models.GormCall
	minDistance := 999999

	for _, call := range gormCalls {
		if call.Scope == preloadCall.Scope && call.File == preloadCall.File {
			if call.Method == "Find" || call.Method == "First" || call.Method == "FirstOrCreate" {
				distance := abs(call.Line - preloadCall.Line)
				if distance < minDistance {
					minDistance = distance
					closestCall = call
				}
			}
		}
	}

	if minDistance < 999999 {
		// Try to extract variable name from the closest call
		varName := extractVariableNameFromFindCall(closestCall.LineContent)
		if varName != "" {
			return varName
		}
	}

	return ""
}

// abs returns the absolute value of an integer
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// inferModelFromVariableName infers the model name from a variable name
func inferModelFromVariableName(varName string) string {
	// Remove common prefixes and suffixes
	name := varName
	if strings.HasSuffix(name, "s") && len(name) > 1 {
		name = name[:len(name)-1] // Remove plural 's'
	}
	if strings.HasSuffix(name, "List") {
		name = name[:len(name)-4] // Remove 'List' suffix
	}
	if strings.HasSuffix(name, "Items") {
		name = name[:len(name)-5] // Remove 'Items' suffix
	}

	// Capitalize first letter
	if len(name) > 0 {
		return strings.ToUpper(string(name[0])) + name[1:]
	}

	return ""
}
