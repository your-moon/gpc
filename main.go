package main

import (
	"golang.org/x/tools/go/analysis/singlechecker"

	preloadcheck "github.com/your-moon/gpc/pkg/preloadcheck"
)

func main() {
	singlechecker.Main(preloadcheck.Analyzer)
}
