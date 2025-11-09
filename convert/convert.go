// package convert holds all of the converters that take something parsed out of
// a yaml file turn it into an hcl block.
package convert

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/pfcm/ktf/convert/gen"
	"github.com/pfcm/ktf/resource"
)

func Convert(r resource.Resource) (*hclwrite.Block, error) {
	spec, ok := gen.FindSpec(r.TypeKey)
	if !ok {
		return convertToManifest(r)
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

func convertToManifest(r resource.Resource) (*hclwrite.Block, error) {
	name := resource.ToSnake(strings.Join([]string{r.Kind, r.Metadata.Name}, "__"))
	name = strings.ReplaceAll(name, ".", "_")
	b := hclwrite.NewBlock("resource", []string{"kubernetes_manifest", name})

	tokens, err := manifestDataTokens(r)
	if err != nil {
		return nil, err
	}
	b.Body().SetAttributeRaw("manifest", tokens)

	return b, nil
}

func manifestDataTokens(r resource.Resource) (hclwrite.Tokens, error) {
	var tokens hclwrite.Tokens

	emitSingle := func(b byte, spacesBefore int) error {
		var t hclsyntax.TokenType
		switch b {
		case '{':
			t = hclsyntax.TokenOBrace
		case '}':
			t = hclsyntax.TokenCBrace
		case '[':
			t = hclsyntax.TokenOBrack
		case ']':
			t = hclsyntax.TokenCBrack
		case '=':
			t = hclsyntax.TokenEqual
		case ',':
			t = hclsyntax.TokenComma
		case '\n':
			t = hclsyntax.TokenNewline
		default:
			return fmt.Errorf("unknown single char token: %q", b)
		}
		tokens = append(tokens, &hclwrite.Token{
			Type:         t,
			Bytes:        []byte{b},
			SpacesBefore: spacesBefore,
		})
		return nil
	}
	emitString := func(s string) {
		tokens = append(tokens, &hclwrite.Token{
			Type:  hclsyntax.TokenQuotedLit,
			Bytes: fmt.Appendf(nil, "%q", s),
		})
	}
	emitFloat64 := func(f float64) {
		tokens = append(tokens, &hclwrite.Token{
			Type:  hclsyntax.TokenNumberLit,
			Bytes: fmt.Append(nil, f),
		})
	}
	emitBool := func(b bool) {
		tokens = append(tokens, &hclwrite.Token{
			Type:  hclsyntax.TokenStringLit, // or Ident?
			Bytes: fmt.Append(nil, b),
		})
	}

	// forward declarations so these closures can mutually recurse.
	var (
		emitValue func(any) error
		emitMap   func(map[string]any) error
		emitList  func([]any) error
		emitAttr  func(string, any) error
	)
	emitValue = func(a any) error {
		switch v := a.(type) {
		case string:
			emitString(v)
		case float64:
			emitFloat64(v)
		case bool:
			emitBool(v)
		case map[string]any:
			return emitMap(v)
		case []any:
			return emitList(v)
		default:
			return fmt.Errorf("unhandled type in manifest: %T (value: %v)", v, v)
		}
		return nil
	}
	emitMap = func(m map[string]any) error {
		if err := emitSingle('{', 1); err != nil {
			return err
		}
		if err := emitSingle('\n', 0); err != nil {
			return err
		}

		names := slices.Collect(maps.Keys(m))
		slices.Sort(names)
		for _, name := range names {
			if err := emitAttr(name, m[name]); err != nil {
				return err
			}
		}

		return emitSingle('}', 1)
	}
	emitList = func(l []any) error {
		if err := emitSingle('[', 1); err != nil {
			return err
		}

		for i, v := range l {
			if err := emitValue(v); err != nil {
				return err
			}
			if i != len(l)-1 {
				emitSingle(',', 0)
			}
		}

		return emitSingle(']', 1)
	}
	emitAttr = func(name string, value any) error {
		// "name" = <value>
		emitString(name)
		emitSingle('=', 1)

		if err := emitValue(value); err != nil {
			return err
		}
		return emitSingle('\n', 0)
	}

	if err := emitValue(r.ToMap()); err != nil {
		return nil, err
	}
	return tokens, nil
}
