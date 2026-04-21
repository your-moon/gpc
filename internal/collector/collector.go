package collector

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/packages"

	"github.com/your-moon/gpc/internal/loader"
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

				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				if !terminalMethods[sel.Sel.Name] {
					return true
				}

				if !isGormDBExpr(sel.X, pkg.TypesInfo) {
					return true
				}

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

				// Collect preloads from the inline chain
				preloads := collectPreloads(sel.X, pkg)

				// If no preloads found inline, check if the receiver is a variable
				// that was assigned from a chain containing Preload calls
				if len(preloads) == 0 {
					preloads = collectPreloadsFromVariable(sel.X, file, pkg)
				}

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
// Handles string literals, constants, and clause.Associations.
func resolveStringArg(expr ast.Expr, info *types.Info) (string, bool) {
	// Check for clause.Associations (selector expression)
	if sel, ok := expr.(*ast.SelectorExpr); ok {
		if sel.Sel.Name == "Associations" {
			if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "clause" {
				return "clause.Associations", true
			}
		}
	}

	// Try constant evaluation (handles both literals and const refs)
	tv, ok := info.Types[expr]
	if ok && tv.Value != nil && tv.Value.Kind() == constant.String {
		return constant.StringVal(tv.Value), true
	}
	return "", false
}

// collectPreloadsFromVariable resolves preloads when the receiver is a variable
// e.g., query := db.Preload("User"); query.Find(&orders)
// Also handles struct literals: orm := &QueryBuilder{DB: db.Preload("User")}
func collectPreloadsFromVariable(expr ast.Expr, file *ast.File, pkg *packages.Package) []PreloadInfo {
	ident, ok := expr.(*ast.Ident)
	if !ok {
		return nil
	}

	obj := pkg.TypesInfo.ObjectOf(ident)
	if obj == nil {
		return nil
	}

	var preloads []PreloadInfo
	ast.Inspect(file, func(n ast.Node) bool {
		assign, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}
		for i, lhs := range assign.Lhs {
			lhsIdent, ok := lhs.(*ast.Ident)
			if !ok {
				continue
			}
			if pkg.TypesInfo.ObjectOf(lhsIdent) != obj {
				continue
			}
			if i >= len(assign.Rhs) {
				continue
			}
			rhs := assign.Rhs[i]
			// Direct call chain: query := db.Preload("User")
			if call, ok := rhs.(*ast.CallExpr); ok {
				preloads = append(preloads, collectPreloadsFromCall(call, pkg)...)
			}
			// Struct literal with &: orm := &QueryBuilder{DB: db.Preload("X")}
			if unary, ok := rhs.(*ast.UnaryExpr); ok {
				if comp, ok := unary.X.(*ast.CompositeLit); ok {
					preloads = append(preloads, collectPreloadsFromCompositeLit(comp, pkg)...)
				}
			}
			// Struct literal without &: orm := QueryBuilder{DB: db.Preload("X")}
			if comp, ok := rhs.(*ast.CompositeLit); ok {
				preloads = append(preloads, collectPreloadsFromCompositeLit(comp, pkg)...)
			}
		}
		return true
	})

	return preloads
}

// collectPreloadsFromCompositeLit extracts preloads from struct literal fields
// that are *gorm.DB typed (including embedded fields).
func collectPreloadsFromCompositeLit(comp *ast.CompositeLit, pkg *packages.Package) []PreloadInfo {
	var preloads []PreloadInfo
	for _, elt := range comp.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		// Check if this field's value has type *gorm.DB
		valType := pkg.TypesInfo.TypeOf(kv.Value)
		if valType != nil && isGormDBType(valType) {
			if call, ok := kv.Value.(*ast.CallExpr); ok {
				preloads = append(preloads, collectPreloadsFromCall(call, pkg)...)
			}
		}
	}
	return preloads
}

// collectPreloadsFromCall extracts preloads from a call expression tree.
func collectPreloadsFromCall(call *ast.CallExpr, pkg *packages.Package) []PreloadInfo {
	var preloads []PreloadInfo

	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil
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

	// Recurse into the receiver
	if innerCall, ok := sel.X.(*ast.CallExpr); ok {
		inner := collectPreloadsFromCall(innerCall, pkg)
		preloads = append(inner, preloads...)
	}

	return preloads
}

// isGormDBExpr checks if an expression has type *gorm.DB or a struct embedding *gorm.DB.
func isGormDBExpr(expr ast.Expr, info *types.Info) bool {
	typ := info.TypeOf(expr)
	if typ == nil {
		return false
	}
	return isGormDBType(typ)
}

func isGormDBType(typ types.Type) bool {
	if ptr, ok := typ.(*types.Pointer); ok {
		typ = ptr.Elem()
	}
	named, ok := typ.(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	if obj.Name() == "DB" && obj.Pkg() != nil && obj.Pkg().Path() == gormPkgPath {
		return true
	}
	// Check if the struct embeds *gorm.DB
	st, ok := named.Underlying().(*types.Struct)
	if !ok {
		return false
	}
	for i := 0; i < st.NumFields(); i++ {
		field := st.Field(i)
		if !field.Embedded() {
			continue
		}
		if isGormDBType(field.Type()) {
			return true
		}
	}
	return false
}
