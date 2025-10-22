package validator

import (
	"fmt"
	"strings"

	"github.com/your-moon/gpc/internal/debug"
	"github.com/your-moon/gpc/internal/models"
)

// ValidatePreloadRelations validates preload relations against actual struct definitions
func ValidatePreloadRelations(results []models.PreloadResult, allStructs map[string]models.StructInfo) []models.PreloadResult {
	debug.PassHeader("STRUCT VALIDATION")
	debug.Info("Validating %d preload results against %d struct definitions",
		len(results), len(allStructs))

	// Create a map of struct names for quick lookup
	structMap := make(map[string]models.StructInfo)
	for name, info := range allStructs {
		structMap[name] = info
		debug.Verbose("Found struct: %s with %d fields", name, len(info.Fields))
	}

	// Validate each preload result
	for i, result := range results {
		debug.Section(fmt.Sprintf("Validating Preload %d", i+1))
		debug.Info("Preload: %s -> %s (model: %s)", result.Relation, result.Model, result.Model)

		// Extract the base model name (remove package prefix if present)
		baseModel := extractBaseModelName(result.Model)
		debug.Indent(1, "Base model: %s", baseModel)

		// Check if the model struct exists
		if structInfo, exists := structMap[baseModel]; exists {
			debug.Indent(1, "Model struct found: %s", baseModel)

			// Check if the relation field exists in the struct
			if validateRelationInStruct(result.Relation, structInfo) {
				debug.Indent(2, "✅ Relation '%s' found in struct %s", result.Relation, baseModel)
				results[i].Status = "correct"
			} else {
				debug.Indent(2, "❌ Relation '%s' NOT found in struct %s", result.Relation, baseModel)
				results[i].Status = "error"
			}
		} else {
			debug.Indent(1, "❌ Model struct '%s' not found", baseModel)
			results[i].Status = "unknown"
		}
	}

	debug.PassFooter("STRUCT VALIDATION", len(results))
	return results
}

// extractBaseModelName extracts the base model name from a potentially package-qualified name
func extractBaseModelName(modelName string) string {
	// Remove package prefix if present (e.g., "databases.Invoice" -> "Invoice")
	if lastDot := strings.LastIndex(modelName, "."); lastDot != -1 {
		return modelName[lastDot+1:]
	}
	return modelName
}

// validateRelationInStruct checks if a relation exists in a struct
func validateRelationInStruct(relation string, structInfo models.StructInfo) bool {
	// Handle nested relations (e.g., "User.Profile")
	parts := strings.Split(relation, ".")
	if len(parts) == 1 {
		// Simple relation - check if field exists
		_, exists := structInfo.Fields[parts[0]]
		return exists
	}

	// Nested relation - check if the first part exists and is a struct type
	firstField, exists := structInfo.Fields[parts[0]]
	if !exists {
		return false
	}

	// For nested relations, we would need to recursively check the nested struct
	// For now, we'll assume the first part exists and is valid
	// TODO: Implement recursive struct validation for nested relations
	debug.Indent(3, "Nested relation '%s' - first part '%s' found with type '%s'",
		relation, parts[0], firstField)

	return true
}

// GetStructStatistics returns statistics about the structs found
func GetStructStatistics(allStructs map[string]models.StructInfo) map[string]interface{} {
	stats := make(map[string]interface{})

	totalStructs := len(allStructs)
	totalFields := 0

	for _, structInfo := range allStructs {
		totalFields += len(structInfo.Fields)
	}

	stats["total_structs"] = totalStructs
	stats["total_fields"] = totalFields

	if totalStructs > 0 {
		stats["avg_fields_per_struct"] = float64(totalFields) / float64(totalStructs)
	} else {
		stats["avg_fields_per_struct"] = 0.0
	}

	return stats
}
