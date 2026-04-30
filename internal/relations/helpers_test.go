package relations

import (
	"testing"

	"github.com/your-moon/gpc/internal/collector"
	"github.com/your-moon/gpc/internal/loader"
	"github.com/your-moon/gpc/internal/testutil"
)

func loadAndCollect(t *testing.T, files map[string]string) []collector.Chain {
	t.Helper()
	dir := testutil.CreateTestModule(t, files)
	result, err := loader.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return collector.Collect(result)
}
