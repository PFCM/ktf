// package convert holds all of the converters that take something parsed out of
// a yaml file turn it into an hcl block.
package convert

import (
	"fmt"
	"maps"
	"slices"

	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/pfcm/ktf/convert/gen"
	"github.com/pfcm/ktf/resource"
)

// // Converter translates a resource to a single HCL block.
// type Converter func(resource.Resource) (*hclwrite.Block, error)

// // converters is the big map of TypeKey to converter. All new converters need to
// // be added in here. To get a converter, use getConverter, which supplies the
// // appropriate default.
// var converters = map[resource.TypeKey]Converter{
// 	{APIVersion: "v1", Kind: "Namespace"}:           namespace,
// 	{APIVersion: "core/v1", Kind: "Namespace"}:      namespace,
// 	{APIVersion: "v1", Kind: "ServiceAccount"}:      serviceAccount,
// 	{APIVersion: "core/v1", Kind: "ServiceAccount"}: serviceAccount,
// 	{APIVersion: "apps/v1", Kind: "Deployment"}: func(r resource.Resource) (*hclwrite.Block, error) {
// 		return convertFromSpec(kubernetesDeploymentV1, "kubernetes_deployment_v1", r)
// 	},
// }

// // Get returns the best converter for the given TypeKey.
// func Get(tk resource.TypeKey) Converter {
// 	if c, ok := converters[tk]; ok {
// 		return c
// 	}
// 	return toManifest
// }

func Convert(r resource.Resource) (*hclwrite.Block, error) {
	spec, ok := gen.FindSpec(r.TypeKey)
	if !ok {
		// TODO: fallback to manifest
		return nil, fmt.Errorf("no converter registered for %v", r.TypeKey)
	}
	return convertFromSpec(spec, spec.ResourceName, r)
}

func convertFromSpec(spec gen.ConverterSpec, resourceName string, r resource.Resource) (*hclwrite.Block, error) {
	b := hclwrite.NewBlock("resource", []string{resourceName, resource.ToSnake(r.Metadata.Name)})
	md := maps.Clone(r.Metadata.Meta)
	md["name"] = r.Metadata.Name
	if r.Metadata.Namespace != "" {
		md["namespace"] = r.Metadata.Namespace
	}
	raw := map[string]any{
		"metadata": md,
	}
	if _, ok := spec.Blocks["spec"]; ok {
		raw["spec"] = maps.Clone(r.Spec)
	}

	if err := writeFromSpec(spec, b, raw); err != nil {
		return nil, err
	}
	return b, nil
}

func writeFromSpec(spec gen.ConverterSpec, b *hclwrite.Block, data map[string]any) error {
	var (
		leftovers = keySet(data)
		body      = b.Body()
	)
	for name, toVal := range spec.IterAttrs() {
		// spec.Attributes will be snake_case, but data comes from the
		// normal yaml and will be camelCase.
		camelName := resource.ToCamel(name)
		v, ok := data[camelName]
		if !ok {
			continue
		}
		delete(leftovers, camelName)

		val, err := toVal(v)
		if err != nil {
			return err
		}
		body.SetAttributeValue(name, val)
	}
	for name, subSpec := range spec.IterBlocks() {
		camelName := resource.ToCamel(name)
		v, ok := data[camelName]
		if !ok {
			v2, ok := data[camelName+"s"]
			if ok {
				v = v2
				camelName = camelName + "s"
			} else {
				continue
			}
		}
		delete(leftovers, camelName)

		var subData []map[string]any
		switch t := v.(type) {
		case map[string]any:
			subData = []map[string]any{t}
		case []any:
			for _, a := range t {
				sd, ok := a.(map[string]any)
				if !ok {
					return fmt.Errorf("unexpected type in list for %q: %T (value %v)", name, a, a)
				}
				subData = append(subData, sd)
			}
		default:
			return fmt.Errorf("unexpected type for %q: %T (value %v)", name, v, v)
		}
		for _, sd := range subData {
			subBlock := body.AppendNewBlock(name, nil)
			if err := writeFromSpec(subSpec, subBlock, sd); err != nil {
				return fmt.Errorf("writing %q: %w", name, err)
			}
		}
	}

	if len(leftovers) != 0 {
		return fmt.Errorf("leftover keys: %v", slices.Collect(maps.Keys(leftovers)))
	}
	return nil
}

// serviceAccount converts a service account.
// func serviceAccount(r resource.Resource) (*hclwrite.Block, error) {
// 	b := hclwrite.NewBlock("resource", []string{
// 		"kubernetes_service_account",
// 		ToSnake(r.Metadata.Name),
// 	})
// 	if err := appendMetadata(b, r.Metadata); err != nil {
// 		return nil, err
// 	}
// 	return b, nil
// }

// // namespace converts a namespace resource, which is just about the simplest
// // possible.
// func namespace(r resource.Resource) (*hclwrite.Block, error) {
// 	b := hclwrite.NewBlock("resource", []string{"kubernetes_namespace", ToSnake(r.Metadata.Name)})
// 	if err := appendMetadata(b, r.Metadata); err != nil {
// 		return nil, err
// 	}
// 	return b, nil
// }

// func appendMetadata(b *hclwrite.Block, p resource.PartialMetadata) error {
// 	md := b.Body().AppendNewBlock("metadata", nil).Body()

// 	md.SetAttributeValue("name", cty.StringVal(p.Name))
// 	if p.Namespace != "" {
// 		md.SetAttributeValue("namespace", cty.StringVal(p.Namespace))
// 	}
// 	// These fields and their types (give or take the conversion to cty)
// 	// come directly from
// 	// https://github.com/hashicorp/terraform-provider-kubernetes/blob/v2.38.0/kubernetes/schema_metadata.go
// 	// TODO: does the order matter?
// 	for name, toCty := range map[string]func(any) (cty.Value, error){
// 		"annotations": toStringMap,
// 		"labels":      toStringMap,
// 	} {
// 		val, ok := p.Meta[name]
// 		if !ok {
// 			continue
// 		}
// 		v, err := toCty(val)
// 		if err != nil {
// 			return err
// 		}
// 		md.SetAttributeValue(name, v)
// 	}
// 	return nil
// }

// // toManifest is the generic fallback converter that produces a block containing
// // a kubernetes_manifest resource.
// // TODO: make a proper name if it has a generatable name
// func toManifest(r resource.Resource) (*hclwrite.Block, error) {
// 	nameElems := []string{r.Metadata.Namespace, r.Metadata.Name, r.Kind}
// 	if r.Metadata.Namespace == "" {
// 		nameElems = nameElems[1:]
// 	}
// 	name := ToSnake(strings.Join(nameElems, "_"))

// 	// The kubernetes_manifest resource is kind of cursed, because the
// 	// "manifest" attribute is just a huge map, with values that might be
// 	// anything. This is very tricky to represent in any of terraform's
// 	// surprisingly numerous and nearly identical type systems.
// 	return nil, fmt.Errorf("no implemented (%s)", name)
// }
