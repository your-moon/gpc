package relations

import (
	"go/types"
	"strings"
)

// walkResult records whether a relation path resolved end-to-end and,
// if not, where it broke.
//
//   - ok=true:  failedAt = -1, parent = nil
//   - ok=false: failedAt = index of the first segment that didn't resolve,
//     parent = the named struct type the failing segment was looked up in
//     (nil when the segment's parent is an anonymous struct or unknown)
type walkResult struct {
	ok       bool
	failedAt int
	parent   *types.Named
}

// walk traverses a dotted relation path through the model's struct fields,
// descending one segment at a time.
func (m *model) walk(path string) walkResult {
	parts := strings.Split(path, ".")
	cur := m
	for i, seg := range parts {
		fi := lookupField(cur.structType, seg)
		if fi == nil {
			return walkResult{ok: false, failedAt: i, parent: cur.named}
		}
		if i == len(parts)-1 {
			break
		}
		if fi.structType == nil {
			return walkResult{ok: false, failedAt: i, parent: cur.named}
		}
		cur = nextModel(fi)
	}
	return walkResult{ok: true, failedAt: -1}
}

// nextModel builds the model for the next segment from a resolved field.
func nextModel(fi *fieldInfo) *model {
	next := &model{
		name:       fi.name,
		structType: fi.structType,
		named:      fi.named,
	}
	if fi.named != nil && fi.named.Obj() != nil {
		next.pkg = fi.named.Obj().Pkg()
		next.name = fi.named.Obj().Name()
	}
	return next
}
