package preloadcheck

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"golang.org/x/tools/go/analysis"
)

func TestAnalyzer(t *testing.T) {
	// Test analyzer properties
	if Analyzer == nil {
		t.Fatal("Analyzer is nil")
	}

	if Analyzer.Name != "preloadcheck" {
		t.Errorf("Expected analyzer name 'preloadcheck', got '%s'", Analyzer.Name)
	}

	if Analyzer.Doc == "" {
		t.Error("Analyzer doc should not be empty")
	}

	if Analyzer.Run == nil {
		t.Error("Analyzer Run function should not be nil")
	}
}

func TestValidatePreloadPath(t *testing.T) {
	tests := []struct {
		name      string
		structInfo StructInfo
		path      []string
		wantValid bool
	}{
		{
			name: "valid single level",
			structInfo: StructInfo{
				Name:   "Order",
				Fields: []string{"User"},
			},
			path:      []string{"User"},
			wantValid: true,
		},
		{
			name: "invalid field name",
			structInfo: StructInfo{
				Name:   "Order",
				Fields: []string{"User"},
			},
			path:      []string{"Customer"},
			wantValid: false,
		},
		{
			name: "empty path",
			structInfo: StructInfo{
				Name:   "Order",
				Fields: []string{"User"},
			},
			path:      []string{},
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := validatePreloadPath(tt.structInfo, tt.path)
			if valid != tt.wantValid {
				t.Errorf("validatePreloadPath() = %v, want %v", valid, tt.wantValid)
			}
		})
	}
}

func TestAnalyzerRun(t *testing.T) {
	// Test that the analyzer runs without crashing
	fset := token.NewFileSet()
	code := `package test

type User struct { Name string }
type Order struct { User User }

func test() {
	var orders []Order
	_ = orders
}`

	f, err := parser.ParseFile(fset, "test.go", code, 0)
	if err != nil {
		t.Fatalf("Failed to parse code: %v", err)
	}

	pass := &analysis.Pass{
		Fset:  fset,
		Files: []*ast.File{f},
		Report: func(d analysis.Diagnostic) {},
	}

	_, err = runRipgrep(pass)
	if err != nil {
		t.Errorf("Analyzer run failed: %v", err)
	}
}
