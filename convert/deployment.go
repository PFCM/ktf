package convert

import (
	"fmt"
	"maps"
	"slices"

	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/pfcm/ktf/resource"
)

func deployment(r resource.Resource) (*hclwrite.Block, error) {
	b := hclwrite.NewBlock("resource", []string{
		"kubernetes_deployment",
		resource.ToSnake(r.Metadata.Name),
	})
	// if err := appendMetadata(b, r.Metadata); err != nil {
	// 	return nil, err
	// }

	leftovers := keySet(r.Spec)
	// Attributes first. See
	// https://github.com/hashicorp/terraform-provider-kubernetes/blob/v2.38.0/kubernetes/resource_kubernetes_deployment_v1.go#L59
	// for what's what.
	// spec := b.Body().AppendNewBlock("spec", nil)
	// for name, toCty := range map[string]func(any) (cty.Value, error){} {
	// 	v, ok := r.Spec[name]
	// 	if !ok {
	// 		continue
	// 	}

	// 	delete(leftovers, name)
	// }
	// Now nested blocks.

	if len(leftovers) != 0 {
		// TODO: just log?
		return nil, fmt.Errorf("unknown keys in deployment.spec: %v", slices.Collect(maps.Keys(leftovers)))
	}
	return b, nil
}

func keySet[K comparable, V any](m map[K]V) map[K]bool {
	out := make(map[K]bool, len(m))
	for k := range m {
		out[k] = true
	}
	return out
}
