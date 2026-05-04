package engine

import (
	"github.com/your-moon/gpc/internal/collector"
	"github.com/your-moon/gpc/internal/loader"
	"github.com/your-moon/gpc/internal/models"
	"github.com/your-moon/gpc/internal/relations"
)

// Analyze loads packages in dir, collects Preload chains, and verifies them.
func Analyze(dir string) ([]models.PreloadResult, error) {
	result, err := loader.Load(dir)
	if err != nil {
		return nil, err
	}

	chains := collector.Collect(result)

	return relations.Verify(chains), nil
}
