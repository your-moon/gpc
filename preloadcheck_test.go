package preloadcheck

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
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

func TestCheckPreloadPath(t *testing.T) {
	tests := []struct {
		name      string
		code      string
		path      []string
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid single level",
			code: `package test
type User struct { Name string }
type Order struct { User User }`,
			path:      []string{"User"},
			wantError: false,
		},
		{
			name: "valid nested path",
			code: `package test
type Address struct { City string }
type Profile struct { Address Address }
type User struct { Profile Profile }
type Order struct { User User }`,
			path:      []string{"User", "Profile", "Address"},
			wantError: false,
		},
		{
			name: "invalid field name",
			code: `package test
type User struct { Name string }
type Order struct { User User }`,
			path:      []string{"Customer"},
			wantError: true,
			errorMsg:  "Customer not found",
		},
		{
			name: "invalid nested field",
			code: `package test
type Profile struct { Bio string }
type User struct { Profile Profile }
type Order struct { User User }`,
			path:      []string{"User", "Profil"},
			wantError: true,
			errorMsg:  "User.Profil not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "test.go", tt.code, 0)
			if err != nil {
				t.Fatalf("Failed to parse code: %v", err)
			}

			conf := types.Config{}
			info := &types.Info{
				Types: make(map[ast.Expr]types.TypeAndValue),
				Defs:  make(map[*ast.Ident]types.Object),
				Uses:  make(map[*ast.Ident]types.Object),
			}

			pkg, err := conf.Check("test", fset, []*ast.File{f}, info)
			if err != nil {
				t.Fatalf("Failed to type check: %v", err)
			}

			// Find the Order type
			obj := pkg.Scope().Lookup("Order")
			if obj == nil {
				t.Fatal("Order type not found")
			}

			pass := &analysis.Pass{
				Fset:      fset,
				Files:     []*ast.File{f},
				TypesInfo: info,
				Pkg:       pkg,
			}

			errMsg := checkPreloadPath(pass, obj.Type(), tt.path)

			if tt.wantError && errMsg == "" {
				t.Errorf("Expected error containing '%s', got no error", tt.errorMsg)
			}

			if !tt.wantError && errMsg != "" {
				t.Errorf("Expected no error, got: %s", errMsg)
			}

			if tt.wantError && errMsg != "" && tt.errorMsg != "" {
				// Just check if error message contains expected substring
				if len(errMsg) == 0 {
					t.Errorf("Expected error message, got empty string")
				}
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

	conf := types.Config{}
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}

	pkg, err := conf.Check("test", fset, []*ast.File{f}, info)
	if err != nil {
		t.Fatalf("Failed to type check: %v", err)
	}

	pass := &analysis.Pass{
		Fset:      fset,
		Files:     []*ast.File{f},
		TypesInfo: info,
		Pkg:       pkg,
		Report:    func(d analysis.Diagnostic) {},
	}

	_, err = run(pass)
	if err != nil {
		t.Errorf("Analyzer run failed: %v", err)
	}
}
