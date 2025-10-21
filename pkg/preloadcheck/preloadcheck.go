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
	// Track call chains to better associate Preload calls with their models
	callChains := make(map[ast.Node][]ast.Node)

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
							// For slice types, get the element type
							if slice, ok := modelType.(*types.Slice); ok {
								modelType = slice.Elem()
							}
						}
					}
				} else if ident, ok := arg.(*ast.Ident); ok {
					// Handle direct variable case
					if obj := pass.TypesInfo.ObjectOf(ident); obj != nil {
						modelType = obj.Type()
						// For slice types, get the element type
						if slice, ok := modelType.(*types.Slice); ok {
							modelType = slice.Elem()
						}
					}
				}

				if modelType != nil {
					modelTypes[call] = modelType
					// Track the call chain leading to this Find call
					callChains[call] = buildCallChain(call)
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
			modelType := findModelTypeForPreload(pass, call, modelTypes, callChains)
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

// Build a call chain for a given call expression
func buildCallChain(call *ast.CallExpr) []ast.Node {
	chain := []ast.Node{call}

	// Walk up the call chain to find the root receiver
	current := call
	for {
		if sel, ok := current.Fun.(*ast.SelectorExpr); ok {
			chain = append([]ast.Node{sel}, chain...)
			if callExpr, ok := sel.X.(*ast.CallExpr); ok {
				current = callExpr
			} else {
				break
			}
		} else {
			break
		}
	}

	return chain
}

// Find the model type for a Preload call by looking for nearby Find calls
func findModelTypeForPreload(pass *analysis.Pass, preloadCall *ast.CallExpr, modelTypes map[ast.Node]types.Type, callChains map[ast.Node][]ast.Node) types.Type {
	// Method 1: Look for Find calls that are part of the same method chain
	// This is the most accurate approach
	if modelType := findModelInSameChain(pass, preloadCall, modelTypes); modelType != nil {
		return modelType
	}

	// Method 2: Fallback to closest Find call (with very small distance)
	if modelType := findClosestModel(pass, preloadCall, modelTypes); modelType != nil {
		return modelType
	}

	return nil
}

// Method 1: Find model type by analyzing the actual method chain
func findModelInSameChain(pass *analysis.Pass, preloadCall *ast.CallExpr, modelTypes map[ast.Node]types.Type) types.Type {
	// Look for the pattern: db.Preload().Preload().Find()
	// This is the most common GORM pattern

	// Get the receiver of the Preload call
	if sel, ok := preloadCall.Fun.(*ast.SelectorExpr); ok {
		receiver := sel.X

		// Look for Find calls that use the same receiver and come after this Preload
		for findCall, modelType := range modelTypes {
			if findCall.Pos() > preloadCall.Pos() {
				if findCallExpr, ok := findCall.(*ast.CallExpr); ok {
					if findSel, ok := findCallExpr.Fun.(*ast.SelectorExpr); ok {
						if sameReceiver(pass, receiver, findSel.X) {
							// Check if they're close enough to be part of the same chain
							distance := findCall.Pos() - preloadCall.Pos()
							if distance < 1000 { // Reasonable distance for method chaining
								return modelType
							}
						}
					}
				}
			}
		}
	}

	return nil
}

// Method 2: Find the closest Find call as fallback
func findClosestModel(pass *analysis.Pass, preloadCall *ast.CallExpr, modelTypes map[ast.Node]types.Type) types.Type {
	var closestFindCall ast.Node
	var closestModelType types.Type
	var closestDistance token.Pos = ^token.Pos(0) // Max value

	for findCall, modelType := range modelTypes {
		if findCall.Pos() > preloadCall.Pos() {
			distance := findCall.Pos() - preloadCall.Pos()
			// Use a very small distance threshold (e.g., 20 characters)
			// This helps avoid picking up Find calls from other parts of the function
			if distance < 20 && distance < closestDistance {
				closestDistance = distance
				closestFindCall = findCall
				closestModelType = modelType
			}
		}
	}

	if closestFindCall != nil {
		return closestModelType
	}

	return nil
}

// Method 1: Find model type by looking for Find calls with the same receiver
func findModelWithSameReceiver(pass *analysis.Pass, preloadCall *ast.CallExpr, modelTypes map[ast.Node]types.Type) types.Type {
	// Get the receiver of the Preload call
	if sel, ok := preloadCall.Fun.(*ast.SelectorExpr); ok {
		receiver := sel.X

		// Look for Find calls that use the same receiver
		for findCall, modelType := range modelTypes {
			if findCall.Pos() > preloadCall.Pos() {
				if findCallExpr, ok := findCall.(*ast.CallExpr); ok {
					if findSel, ok := findCallExpr.Fun.(*ast.SelectorExpr); ok {
						if sameReceiver(pass, receiver, findSel.X) {
							// Check if they're close enough to be part of the same chain
							distance := findCall.Pos() - preloadCall.Pos()
							if distance < 1000 { // Reasonable distance for method chaining
								return modelType
							}
						}
					}
				}
			}
		}
	}

	return nil
}

// Method 2: Find model type by looking in the same statement/block
func findModelInSameStatement(pass *analysis.Pass, preloadCall *ast.CallExpr, modelTypes map[ast.Node]types.Type) types.Type {
	// Simple approach: Look for the closest Find call that comes after this Preload
	// This works well for most GORM usage patterns
	var closestFindCall ast.Node
	var closestModelType types.Type
	var closestDistance token.Pos = ^token.Pos(0)

	for findCall, modelType := range modelTypes {
		if findCall.Pos() > preloadCall.Pos() {
			distance := findCall.Pos() - preloadCall.Pos()
			// Use a reasonable distance threshold (e.g., 100 characters)
			// This covers most method chaining scenarios
			if distance < 100 && distance < closestDistance {
				closestDistance = distance
				closestFindCall = findCall
				closestModelType = modelType
			}
		}
	}

	if closestFindCall != nil {
		return closestModelType
	}

	return nil
}

// Find the containing statement/block for a given node
func findContainingStatement(pass *analysis.Pass, node ast.Node) ast.Node {
	// This is a simplified implementation
	// In practice, you'd traverse up the AST to find the containing statement
	return nil
}

// Method 2: Find model type by looking in the same function scope
func findModelInSameFunction(pass *analysis.Pass, preloadCall *ast.CallExpr, modelTypes map[ast.Node]types.Type) types.Type {
	// Find the function that contains this Preload call
	funcNode := findContainingFunction(pass, preloadCall)
	if funcNode == nil {
		return nil
	}

	// Look for Find calls in the same function that come after this Preload
	var closestFindCall ast.Node
	var closestModelType types.Type
	var closestDistance token.Pos = ^token.Pos(0)

	for findCall, modelType := range modelTypes {
		if findCall.Pos() > preloadCall.Pos() && isInFunction(pass, findCall, funcNode) {
			distance := findCall.Pos() - preloadCall.Pos()
			if distance < closestDistance {
				closestDistance = distance
				closestFindCall = findCall
				closestModelType = modelType
			}
		}
	}

	if closestFindCall != nil {
		return closestModelType
	}

	return nil
}

// Check if two AST nodes represent the same receiver
func sameReceiver(pass *analysis.Pass, node1, node2 ast.Node) bool {
	// Simple comparison - in practice you might need more sophisticated analysis
	// This is a basic implementation that works for most common cases

	// If both are identifiers, compare their names
	if ident1, ok1 := node1.(*ast.Ident); ok1 {
		if ident2, ok2 := node2.(*ast.Ident); ok2 {
			return ident1.Name == ident2.Name
		}
	}

	// If both are selector expressions, compare the full path
	if sel1, ok1 := node1.(*ast.SelectorExpr); ok1 {
		if sel2, ok2 := node2.(*ast.SelectorExpr); ok2 {
			return sameReceiver(pass, sel1.X, sel2.X) && sel1.Sel.Name == sel2.Sel.Name
		}
	}

	return false
}

// Check if two call chains share the same root receiver
func shareSameRoot(pass *analysis.Pass, chain1, chain2 []ast.Node) bool {
	if len(chain1) == 0 || len(chain2) == 0 {
		return false
	}

	// Get the root receiver from each chain
	root1 := chain1[0]
	root2 := chain2[0]

	// Compare the root receivers
	return sameReceiver(pass, root1, root2)
}

// Helper functions for proper AST analysis

// Find the parent statement that contains the given node
func findParentStatement(pass *analysis.Pass, node ast.Node) ast.Node {
	// Traverse up the AST to find the containing statement
	for _, file := range pass.Files {
		var parent ast.Node
		ast.Inspect(file, func(n ast.Node) bool {
			if n == node {
				return false // Stop traversal
			}
			// Check if this node contains our target node
			if containsNode(n, node) {
				parent = n
			}
			return true
		})
		if parent != nil {
			return parent
		}
	}
	return nil
}

// Check if a parent node contains a child node
func containsNode(parent, child ast.Node) bool {
	found := false
	ast.Inspect(parent, func(n ast.Node) bool {
		if n == child {
			found = true
			return false
		}
		return true
	})
	return found
}

// Find Find calls within the same statement
func findFindInStatement(pass *analysis.Pass, statement ast.Node, preloadCall *ast.CallExpr, modelTypes map[ast.Node]types.Type) types.Type {
	// Look for Find calls in the same statement that come after the Preload call
	var closestFindCall ast.Node
	var closestModelType types.Type
	var closestDistance token.Pos = ^token.Pos(0)

	for findCall, modelType := range modelTypes {
		if findCall.Pos() > preloadCall.Pos() {
			// Check if the Find call is within the same statement
			if isInStatement(pass, findCall, statement) {
				distance := findCall.Pos() - preloadCall.Pos()
				if distance < closestDistance {
					closestDistance = distance
					closestFindCall = findCall
					closestModelType = modelType
				}
			}
		}
	}

	if closestFindCall != nil {
		return closestModelType
	}

	return nil
}

// Find the function that contains the given node
func findContainingFunction(pass *analysis.Pass, node ast.Node) ast.Node {
	// This is a simplified implementation
	// In practice, you'd traverse up the AST to find the containing function
	return nil
}

// Check if a node is within a specific function
func isInFunction(pass *analysis.Pass, node ast.Node, funcNode ast.Node) bool {
	// This is a simplified implementation
	// In practice, you'd check if the node is within the function's body
	return true
}

// Check if a node is within a specific statement
func isInStatement(pass *analysis.Pass, node ast.Node, statement ast.Node) bool {
	// This is a simplified implementation
	// In practice, you'd check if the node is within the statement's scope
	return true
}

// Check if two nodes are in the same function
func sameFunction(pass *analysis.Pass, node1, node2 ast.Node) bool {
	// This is a simplified check - in practice you'd want to traverse up the AST
	// to find the containing function for each node
	// For now, we'll assume they're in the same function if they're in the same file
	return true
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
