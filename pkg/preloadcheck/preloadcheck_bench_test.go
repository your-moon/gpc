package preloadcheck

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/analysis"
)

func BenchmarkAnalyzer(b *testing.B) {
	code := `package test

type Address struct { City string }
type Profile struct { Address Address }
type User struct { Profile Profile }
type Order struct { User User }

func test() {
	var orders []Order
	_ = orders
}`

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", code, 0)
	if err != nil {
		b.Fatal(err)
	}

	conf := types.Config{}
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}

	pkg, err := conf.Check("test", fset, []*ast.File{f}, info)
	if err != nil {
		b.Fatal(err)
	}

	pass := &analysis.Pass{
		Fset:      fset,
		Files:     []*ast.File{f},
		TypesInfo: info,
		Pkg:       pkg,
		Report:    func(d analysis.Diagnostic) {},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = run(pass)
	}
}

func BenchmarkCheckPreloadPath(b *testing.B) {
	code := `package test
type Address struct { City string }
type Profile struct { Address Address }
type User struct { Profile Profile }
type Order struct { User User }`

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", code, 0)
	if err != nil {
		b.Fatal(err)
	}

	conf := types.Config{}
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}

	pkg, err := conf.Check("test", fset, []*ast.File{f}, info)
	if err != nil {
		b.Fatal(err)
	}

	obj := pkg.Scope().Lookup("Order")
	if obj == nil {
		b.Fatal("Order type not found")
	}

	pass := &analysis.Pass{
		Fset:      fset,
		Files:     []*ast.File{f},
		TypesInfo: info,
		Pkg:       pkg,
	}

	path := []string{"User", "Profile", "Address"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = checkPreloadPath(pass, obj.Type(), path)
	}
}

func BenchmarkCheckPreloadPathDeep(b *testing.B) {
	// Test with deeper nesting
	code := `package test
type L10 struct { Value string }
type L9 struct { L10 L10 }
type L8 struct { L9 L9 }
type L7 struct { L8 L8 }
type L6 struct { L7 L7 }
type L5 struct { L6 L6 }
type L4 struct { L5 L5 }
type L3 struct { L4 L4 }
type L2 struct { L3 L3 }
type L1 struct { L2 L2 }
type Root struct { L1 L1 }`

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", code, 0)
	if err != nil {
		b.Fatal(err)
	}

	conf := types.Config{}
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}

	pkg, err := conf.Check("test", fset, []*ast.File{f}, info)
	if err != nil {
		b.Fatal(err)
	}

	obj := pkg.Scope().Lookup("Root")
	if obj == nil {
		b.Fatal("Root type not found")
	}

	pass := &analysis.Pass{
		Fset:      fset,
		Files:     []*ast.File{f},
		TypesInfo: info,
		Pkg:       pkg,
	}

	path := []string{"L1", "L2", "L3", "L4", "L5", "L6", "L7", "L8", "L9", "L10"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = checkPreloadPath(pass, obj.Type(), path)
	}
}
