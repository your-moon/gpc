package preloadcheck

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "preloadcheck",
	Doc:  "check for mistyped GORM Preload relations (including nested)",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	// Track model types by looking for .Find calls first
	modelTypes := make(map[ast.Node]types.Type)

	// First pass: find all .Find calls and their model types
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "Find" {
				return true
			}

			if len(call.Args) > 0 {
				arg := call.Args[0]
				var modelType types.Type

				if unary, ok := arg.(*ast.UnaryExpr); ok && unary.Op == token.AND {
					// Handle &variable case
					if ident, ok := unary.X.(*ast.Ident); ok {
						if obj := pass.TypesInfo.ObjectOf(ident); obj != nil {
							modelType = obj.Type()
						}
					}
				} else if ident, ok := arg.(*ast.Ident); ok {
					// Handle direct variable case
					if obj := pass.TypesInfo.ObjectOf(ident); obj != nil {
						modelType = obj.Type()
					}
				}

				if modelType != nil {
					modelTypes[call] = modelType
				}
			}

			return true
		})
	}

	// Second pass: check Preload calls
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "Preload" {
				return true
			}

			if len(call.Args) == 0 {
				return true
			}

			lit, ok := call.Args[0].(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}

			preloadPath := strings.Trim(lit.Value, "\"")
			parts := strings.Split(preloadPath, ".")

			// Find the associated model type
			modelType := findModelTypeForPreload(pass, call, modelTypes)
			if modelType == nil {
				return true
			}

			if err := checkPreloadPath(pass, modelType, parts); err != "" {
				pass.Reportf(call.Pos(), "invalid preload: %s", err)
			}

			return true
		})
	}
	return nil, nil
}

// Find the type of the model being used in .Find(&model)
func findModelType(pass *analysis.Pass, call *ast.CallExpr) types.Type {
	// For now, we'll use a simplified approach that looks for common patterns
	// In a real implementation, you'd need to track the full call chain

	// Look for the receiver of the Preload call
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		// Get the type of the receiver (the GORM DB instance)
		receiverType := pass.TypesInfo.TypeOf(sel.X)
		if receiverType != nil {
			// For now, we'll return a placeholder type
			// In practice, you'd need to track the model type through the call chain
			// This is a limitation of the current implementation
			return receiverType
		}
	}

	return nil
}

// Find the model type for a Preload call by looking for nearby Find calls
func findModelTypeForPreload(pass *analysis.Pass, preloadCall *ast.CallExpr, modelTypes map[ast.Node]types.Type) types.Type {
	// This is a simplified approach - in practice you'd need to track the call chain
	// For now, we'll look for any Find call in the same function
	// This is a limitation of the current implementation

	// Look for the most recent Find call in the same scope
	for findCall, modelType := range modelTypes {
		// Simple heuristic: if the Find call is before the Preload call, use it
		if findCall.Pos() < preloadCall.Pos() {
			return modelType
		}
	}

	return nil
}

// Recursively verify the preload path through struct fields
func checkPreloadPath(pass *analysis.Pass, t types.Type, parts []string) string {
	named, ok := t.(*types.Named)
	if !ok {
		return "not a named type"
	}

	structType, ok := named.Underlying().(*types.Struct)
	if !ok {
		return named.Obj().Name() + " is not a struct"
	}

	for i, part := range parts {
		var fieldType types.Type
		found := false
		for j := 0; j < structType.NumFields(); j++ {
			f := structType.Field(j)
			if f.Name() == part {
				fieldType = f.Type()
				found = true
				break
			}
		}
		if !found {
			return strings.Join(parts[:i+1], ".") + " not found in " + named.Obj().Name()
		}

		// unwrap pointer
		if ptr, ok := fieldType.(*types.Pointer); ok {
			fieldType = ptr.Elem()
		}

		// if last part — done
		if i == len(parts)-1 {
			return ""
		}

		named, ok = fieldType.(*types.Named)
		if !ok {
			return strings.Join(parts[:i+1], ".") + " is not a named struct"
		}

		structType, ok = named.Underlying().(*types.Struct)
		if !ok {
			return strings.Join(parts[:i+1], ".") + " is not a struct"
		}
	}

	return ""
}
