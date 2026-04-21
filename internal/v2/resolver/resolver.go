package resolver

import (
	"go/types"

	"github.com/your-moon/gpc/internal/v2/collector"
)

// Model holds the resolved model type information.
type Model struct {
	Name       string         // struct name (e.g., "Order")
	Pkg        *types.Package // package the struct belongs to
	StructType *types.Struct  // the underlying struct type
	Named      *types.Named   // the named type
}

// Resolve determines the model type from a chain's terminal call argument.
func Resolve(chain collector.Chain) *Model {
	if chain.Terminal == nil || chain.Terminal.Arg == nil {
		return nil
	}

	info := chain.Pkg.TypesInfo
	argType := info.TypeOf(chain.Terminal.Arg)
	if argType == nil {
		return nil
	}

	return extractModel(argType)
}

// extractModel unwraps pointer/slice/array types to find the underlying named struct.
func extractModel(typ types.Type) *Model {
	typ = deref(typ)

	switch t := typ.(type) {
	case *types.Named:
		underlying := t.Underlying()
		if st, ok := underlying.(*types.Struct); ok {
			return &Model{
				Name:       t.Obj().Name(),
				Pkg:        t.Obj().Pkg(),
				StructType: st,
				Named:      t,
			}
		}
		return extractModel(underlying)
	case *types.Slice:
		return extractModel(t.Elem())
	case *types.Array:
		return extractModel(t.Elem())
	case *types.Pointer:
		return extractModel(t.Elem())
	}

	return nil
}

// deref removes one layer of pointer indirection (for &variable).
func deref(typ types.Type) types.Type {
	if ptr, ok := typ.(*types.Pointer); ok {
		return ptr.Elem()
	}
	return typ
}

// FieldInfo holds resolved information about a struct field.
type FieldInfo struct {
	Name       string
	Type       types.Type
	StructType *types.Struct // non-nil if the field type is a struct
	Named      *types.Named  // non-nil if the field has a named type
}

// LookupField finds a field by name in a struct, including promoted (embedded) fields.
func LookupField(st *types.Struct, name string) *FieldInfo {
	for i := 0; i < st.NumFields(); i++ {
		field := st.Field(i)
		if field.Name() == name {
			fi := &FieldInfo{
				Name: field.Name(),
				Type: field.Type(),
			}
			underlying := unwrapToStruct(field.Type())
			if underlying != nil {
				fi.StructType = underlying.st
				fi.Named = underlying.named
			}
			return fi
		}
	}

	// Check embedded (promoted) fields
	for i := 0; i < st.NumFields(); i++ {
		field := st.Field(i)
		if !field.Embedded() {
			continue
		}
		embedded := unwrapToStruct(field.Type())
		if embedded != nil {
			if result := LookupField(embedded.st, name); result != nil {
				return result
			}
		}
	}

	return nil
}

type structInfo struct {
	st    *types.Struct
	named *types.Named
}

func unwrapToStruct(typ types.Type) *structInfo {
	typ = derefAll(typ)

	switch t := typ.(type) {
	case *types.Slice:
		typ = derefAll(t.Elem())
	case *types.Array:
		typ = derefAll(t.Elem())
	}

	if named, ok := typ.(*types.Named); ok {
		if st, ok := named.Underlying().(*types.Struct); ok {
			return &structInfo{st: st, named: named}
		}
	}
	if st, ok := typ.(*types.Struct); ok {
		return &structInfo{st: st}
	}
	return nil
}

func derefAll(typ types.Type) types.Type {
	for {
		ptr, ok := typ.(*types.Pointer)
		if !ok {
			return typ
		}
		typ = ptr.Elem()
	}
}
