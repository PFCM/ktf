package gen

import (
	"cmp"
	"iter"
	"maps"
	"slices"
	"strings"

	"github.com/zclconf/go-cty/cty"

	"github.com/pfcm/ktf/resource"
)

// ConverterSpec contains instructions for how to convert a resource,
// essentially the list of fields with types. Generally it should be generated
// by cmd/gen.
type ConverterSpec struct {
	ResourceName string
	Attributes   map[string]func(any) (cty.Value, error)
	// TODO: is this a good idea, or should we just list the names and have
	// a way to look them all up?
	Blocks map[string]ConverterSpec
}

// FindSpec tries to find the ConverterSpec for the given type key.
func FindSpec(tk resource.TypeKey) (ConverterSpec, bool) {
	group, version, ok := strings.Cut(tk.APIVersion, "/")
	if !ok {
		version = group
	}
	name := "kubernetes_" + resource.ToSnake(tk.Kind)
	if spec, ok := specs[name+"_"+version]; ok {
		return spec, ok
	}
	// Might be present unversioned?
	spec, ok := specs[name]
	return spec, ok
}

func (cs ConverterSpec) IterAttrs() iter.Seq2[string, func(any) (cty.Value, error)] {
	return sortedMapIter(cs.Attributes)
}

func (cs ConverterSpec) IterBlocks() iter.Seq2[string, ConverterSpec] {
	return sortedMapIter(cs.Blocks)
}

func sortedMapIter[M ~map[K]V, K cmp.Ordered, V any](m M) iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		keys := slices.Collect(maps.Keys(m))
		slices.Sort(keys)
		for _, k := range keys {
			if !yield(k, m[k]) {
				return
			}
		}
	}
}
