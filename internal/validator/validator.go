package validator

import (
	"strings"

	"github.com/your-moon/gpc/internal/models"
	"github.com/your-moon/gpc/internal/collector"
	"github.com/your-moon/gpc/internal/resolver"
)

// Validate checks all preload chains and returns results.
func Validate(chains []collector.Chain) []models.PreloadResult {
	var results []models.PreloadResult

	for _, chain := range chains {
		model := resolver.Resolve(chain)
		file := chain.File
		pkg := chain.Pkg

		for _, preload := range chain.Preloads {
			line := 0
			if pkg != nil {
				line = pkg.Fset.Position(preload.Pos).Line
			}

			modelName := "Unknown"
			if model != nil {
				if model.Pkg != nil {
					modelName = model.Pkg.Name() + "." + model.Name
				} else {
					modelName = model.Name
				}
			}

			result := models.PreloadResult{
				File:     file,
				Line:     line,
				Relation: preload.Relation,
				Model:    modelName,
			}

			if preload.Dynamic {
				result.Status = "skipped"
				result.Relation = "(dynamic)"
				results = append(results, result)
				continue
			}

			// clause.Associations is always valid in GORM
			if preload.Relation == "clause.Associations" {
				result.Relation = "clause.Associations"
				result.Status = "valid"
				results = append(results, result)
				continue
			}

			// Empty relation is always an error
			if preload.Relation == "" && !preload.Dynamic {
				result.Status = "error"
				results = append(results, result)
				continue
			}

			if model == nil {
				result.Status = "skipped"
				results = append(results, result)
				continue
			}

			// Validate the relation path recursively
			if validatePath(preload.Relation, model) {
				result.Status = "valid"
			} else {
				result.Status = "error"
			}

			results = append(results, result)
		}
	}

	return results
}

// validatePath recursively validates a dotted relation path against a model's struct type.
func validatePath(relation string, model *resolver.Model) bool {
	parts := strings.SplitN(relation, ".", 2)
	fieldName := parts[0]

	fi := resolver.LookupField(model.StructType, fieldName)
	if fi == nil {
		return false
	}

	// If there are more segments, recurse into the field's type
	if len(parts) == 2 {
		if fi.StructType == nil {
			return false
		}
		nextModel := &resolver.Model{
			Name:       fi.Name,
			StructType: fi.StructType,
			Named:      fi.Named,
		}
		if fi.Named != nil && fi.Named.Obj() != nil {
			nextModel.Pkg = fi.Named.Obj().Pkg()
			nextModel.Name = fi.Named.Obj().Name()
		}
		return validatePath(parts[1], nextModel)
	}

	return true
}
