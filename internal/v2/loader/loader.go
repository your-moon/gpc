package loader

import (
	"fmt"

	"golang.org/x/tools/go/packages"
)

// Result holds the loaded packages with type information.
type Result struct {
	Packages []*packages.Package
}

// Load loads all Go packages in the given directory with full type information.
func Load(dir string) (*Result, error) {
	cfg := &packages.Config{
		Mode: packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo |
			packages.NeedName | packages.NeedFiles | packages.NeedImports | packages.NeedDeps,
		Dir: dir,
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, fmt.Errorf("loading packages: %w", err)
	}

	// Check for package-level errors
	var errs []error
	for _, pkg := range pkgs {
		for _, e := range pkg.Errors {
			errs = append(errs, fmt.Errorf("%s: %s", pkg.PkgPath, e.Msg))
		}
	}
	if len(errs) > 0 {
		return nil, fmt.Errorf("package errors: %v", errs[0])
	}

	return &Result{Packages: pkgs}, nil
}
