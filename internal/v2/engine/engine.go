package engine

import (
	"github.com/your-moon/gpc/internal/models"
	"github.com/your-moon/gpc/internal/v2/collector"
	"github.com/your-moon/gpc/internal/v2/loader"
	"github.com/your-moon/gpc/internal/v2/validator"
)

// Analyze runs the full v2 analysis pipeline on the given directory.
func Analyze(dir string) ([]models.PreloadResult, error) {
	result, err := loader.Load(dir)
	if err != nil {
		return nil, err
	}

	chains := collector.Collect(result)

	results := validator.Validate(chains)

	return results, nil
}
