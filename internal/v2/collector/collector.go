package collector

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/packages"

	"github.com/your-moon/gpc/internal/v2/loader"
)

// PreloadInfo holds info about a single .Preload("X") call.
type PreloadInfo struct {
	Relation string // resolved string value, empty if dynamic
	Dynamic  bool   // true if argument is not a resolvable constant
	Pos      token.Pos
}

// TerminalCall holds info about the terminal call (.Find, .First, etc.)
type TerminalCall struct {
	Method string    // "Find", "First", "FirstOrCreate", etc.
	Arg    ast.Expr  // the &variable argument
	Pos    token.Pos
}

// Chain represents a Preload chain ending in a terminal call.
type Chain struct {
	Preloads []PreloadInfo
	Terminal *TerminalCall
	File     string
	Pkg      *packages.Package
}

var terminalMethods = map[string]bool{
	"Find": true, "First": true, "FirstOrCreate": true,
	"Take": true, "Last": true, "Scan": true,
}

const gormPkgPath = "gorm.io/gorm"

// Collect walks all packages and extracts Preload chains.
func Collect(result *loader.Result) []Chain {
	var chains []Chain

	for _, pkg := range result.Packages {
		for _, file := range pkg.Syntax {
			fileName := pkg.Fset.Position(file.Pos()).Filename
			ast.Inspect(file, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}

				// Look for terminal calls (Find, First, etc.)
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				if !terminalMethods[sel.Sel.Name] {
					return true
				}

				// Check the receiver chain is gorm.DB
				if !isGormDBExpr(sel.X, pkg.TypesInfo) {
					return true
				}

				// Extract the terminal call
				var terminal *TerminalCall
				if len(call.Args) > 0 {
					terminal = &TerminalCall{
						Method: sel.Sel.Name,
						Arg:    call.Args[0],
						Pos:    call.Pos(),
					}
				} else {
					return true
				}

				// Walk backward through the chain to collect Preload calls
				preloads := collectPreloads(sel.X, pkg)

				if len(preloads) > 0 {
					chains = append(chains, Chain{
						Preloads: preloads,
						Terminal: terminal,
						File:     fileName,
						Pkg:      pkg,
					})
				}

				return true
			})
		}
	}

	return chains
}

// collectPreloads walks the method chain backward collecting all .Preload() calls.
func collectPreloads(expr ast.Expr, pkg *packages.Package) []PreloadInfo {
	var preloads []PreloadInfo
	cur := expr

	for {
		call, ok := cur.(*ast.CallExpr)
		if !ok {
			break
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			break
		}

		if sel.Sel.Name == "Preload" && len(call.Args) > 0 {
			pi := PreloadInfo{Pos: call.Pos()}
			relation, ok := resolveStringArg(call.Args[0], pkg.TypesInfo)
			if ok {
				pi.Relation = relation
			} else {
				pi.Dynamic = true
			}
			preloads = append(preloads, pi)
		}

		cur = sel.X
	}

	// Reverse so order matches source order (outermost first)
	for i, j := 0, len(preloads)-1; i < j; i, j = i+1, j-1 {
		preloads[i], preloads[j] = preloads[j], preloads[i]
	}

	return preloads
}

// resolveStringArg resolves a call argument to a string value.
// Handles string literals and constants.
func resolveStringArg(expr ast.Expr, info *types.Info) (string, bool) {
	// Try constant evaluation (handles both literals and const refs)
	tv, ok := info.Types[expr]
	if ok && tv.Value != nil && tv.Value.Kind() == constant.String {
		return constant.StringVal(tv.Value), true
	}
	return "", false
}

// isGormDBExpr checks if an expression has type *gorm.DB.
func isGormDBExpr(expr ast.Expr, info *types.Info) bool {
	typ := info.TypeOf(expr)
	if typ == nil {
		return false
	}
	// Unwrap pointer
	if ptr, ok := typ.(*types.Pointer); ok {
		typ = ptr.Elem()
	}
	named, ok := typ.(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	return obj.Name() == "DB" && obj.Pkg() != nil && obj.Pkg().Path() == gormPkgPath
}
