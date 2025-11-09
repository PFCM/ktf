// package gen contains the generated code for the converters (and the necessary helpers).
package gen

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
)

// specs is the registry of all of the generated ConverterSpecs. It should only
// be accessed via register or FindSpec.
var specs map[string]ConverterSpec

func register(name string, spec ConverterSpec) {
	if specs == nil {
		specs = make(map[string]ConverterSpec)
	}
	if _, ok := specs[name]; ok {
		panic(fmt.Sprintf("attempt to register a spec with name %q for the second time", name))
	}
	specs[name] = spec
}

func toBool(in any) (cty.Value, error) {
	b, ok := in.(bool)
	if !ok {
		return cty.Value{}, fmt.Errorf("expected bool, got %T (value %v)", in, in)
	}
	return cty.BoolVal(b), nil
}

func toInt(in any) (cty.Value, error) {
	switch v := in.(type) {
	case int:
		return cty.NumberIntVal(int64(v)), nil
	case int32:
		return cty.NumberIntVal(int64(v)), nil
	case int64:
		return cty.NumberIntVal(v), nil
	case uint:
		return cty.NumberUIntVal(uint64(v)), nil
	case uint32:
		return cty.NumberUIntVal(uint64(v)), nil
	case uint64:
		return cty.NumberUIntVal(v), nil
	case float64:
		// TODO: check the conversion is clean?
		return cty.NumberIntVal(int64(v)), nil
	default:
		return cty.Value{}, fmt.Errorf("expected some kind of int, got %T (value %v)", v, v)
	}
}

func toFloat(a any) (cty.Value, error) {
	switch v := a.(type) {
	case float32:
		return cty.NumberFloatVal(float64(v)), nil
	case float64:
		return cty.NumberFloatVal(v), nil
	default:
		return cty.Value{}, fmt.Errorf("expected some kind of float, got %T (value %v)", v, v)
	}
}

func toString(a any) (cty.Value, error) {
	switch v := a.(type) {
	case string:
		return cty.StringVal(v), nil
	case float64, bool:
		// Sometimes people don't quote things that are expected to be
		// strings.
		return cty.StringVal(fmt.Sprint(v)), nil
	default:
		return cty.Value{}, fmt.Errorf("expected string, got %T (value %v)", a, a)
	}
}

func toStringMap(in any) (cty.Value, error) {
	m, ok := in.(map[string]any)
	if !ok {
		return cty.Value{}, fmt.Errorf("expected map[string]any, got %T (value %v)", in, in)
	}

	out := make(map[string]cty.Value, len(m))
	for k, v := range m {
		s, ok := v.(string)
		if !ok {
			return cty.Value{}, fmt.Errorf("expected string, got: %T (value: %v)", v, v)
		}
		out[k] = cty.StringVal(s)
	}
	return cty.MapVal(out), nil
}
