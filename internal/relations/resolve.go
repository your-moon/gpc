package relations

import (
	"go/types"

	"github.com/your-moon/gpc/internal/collector"
)

// model is the resolved struct a Preload chain loads into.
type model struct {
	name       string
	pkg        *types.Package
	structType *types.Struct
	named      *types.Named
}

// fieldInfo describes one resolved field on a struct.
type fieldInfo struct {
	name       string
	typ        types.Type
	structType *types.Struct // non-nil if the field's type unwraps to a struct
	named      *types.Named  // non-nil if the field's type is named
}

// resolveModel determines the model from a chain's terminal call argument.
func resolveModel(chain collector.Chain) *model {
	if chain.Terminal == nil || chain.Terminal.Arg == nil || chain.Pkg == nil {
		return nil
	}
	argType := chain.Pkg.TypesInfo.TypeOf(chain.Terminal.Arg)
	if argType == nil {
		return nil
	}
	return extractModel(argType)
}

// extractModel unwraps pointer/slice/array types to find the underlying named struct.
func extractModel(typ types.Type) *model {
	typ = deref(typ)
	switch t := typ.(type) {
	case *types.Named:
		if st, ok := t.Underlying().(*types.Struct); ok {
			return &model{
				name:       t.Obj().Name(),
				pkg:        t.Obj().Pkg(),
				structType: st,
				named:      t,
			}
		}
		return extractModel(t.Underlying())
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

// lookupField finds a field by name in a struct, including promoted (embedded) fields.
func lookupField(st *types.Struct, name string) *fieldInfo {
	for i := 0; i < st.NumFields(); i++ {
		field := st.Field(i)
		if field.Name() == name {
			fi := &fieldInfo{name: field.Name(), typ: field.Type()}
			if u := unwrapToStruct(field.Type()); u != nil {
				fi.structType = u.st
				fi.named = u.named
			}
			return fi
		}
	}
	for i := 0; i < st.NumFields(); i++ {
		field := st.Field(i)
		if !field.Embedded() {
			continue
		}
		if u := unwrapToStruct(field.Type()); u != nil {
			if found := lookupField(u.st, name); found != nil {
				return found
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
