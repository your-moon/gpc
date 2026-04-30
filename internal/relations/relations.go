// Package relations verifies GORM Preload relation paths against the
// inferred model's Go type information.
package relations

import (
	"github.com/your-moon/gpc/internal/collector"
	"github.com/your-moon/gpc/internal/models"
)

// Verify resolves the model for each chain and verifies every relation
// path against that model's type graph.
func Verify(chains []collector.Chain) []models.PreloadResult {
	var results []models.PreloadResult
	for _, chain := range chains {
		m := resolveModel(chain)
		for _, p := range chain.Preloads {
			results = append(results, verifyPreload(chain, m, p))
		}
	}
	return results
}

func verifyPreload(chain collector.Chain, m *model, p collector.PreloadInfo) models.PreloadResult {
	res := models.PreloadResult{
		File:     chain.File,
		Line:     p.Line,
		Relation: p.Relation,
		Model:    modelDisplay(m),
	}

	if p.Dynamic {
		res.Status = "skipped"
		res.Relation = "(dynamic)"
		return res
	}
	if p.Relation == "clause.Associations" {
		res.Status = "valid"
		return res
	}
	if p.Relation == "" {
		res.Status = "error"
		return res
	}
	if m == nil {
		res.Status = "skipped"
		return res
	}

	if m.walk(p.Relation).ok {
		res.Status = "valid"
	} else {
		res.Status = "error"
	}
	return res
}

func modelDisplay(m *model) string {
	if m == nil {
		return "Unknown"
	}
	if m.pkg != nil {
		return m.pkg.Name() + "." + m.name
	}
	return m.name
}
