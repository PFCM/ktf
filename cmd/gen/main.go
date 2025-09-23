// binary gen generates converters from the resource definition. The code is
// intended to go directly in the convert package.
package main

import (
	"bytes"
	_ "embed"
	"flag"
	"fmt"
	"go/format"
	"io"
	"iter"
	"log"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/template"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pfcm/it"
	"github.com/pfcm/ktf/resource"
	"github.com/pfcm/terraform-provider-kubernetes/v2/kubernetes"
)

var (
	outputDirFlag   = flag.String("output-dir", "", "`directory` to hold results")
	resourcesFlag   = flag.String("resources", "", "`Comma separated list` of resources to generate (in the terraform_provider_kubernetes underscore format). If empty, will generate everything exported by the provide (except kubernetes_manifest)")
	skipAliasesFlag = flag.Bool("skip-aliases", false, "if true, only generates versioned resources. Only applicable if -resources=\"\"")
	packageNameFlag = flag.String("go-package", "gen", "go package name for the generated code")
)

func main() {
	flag.Parse()
	if *outputDirFlag != "" {
		os.MkdirAll(*outputDirFlag, 077)
	}

	allResources := kubernetes.Provider().ResourcesMap

	var names iter.Seq[string]
	if *resourcesFlag == "" {
		names = maps.Keys(allResources)
		if *skipAliasesFlag {
			names = it.Filter(names, func(name string) bool {
				// TODO: some resources only exist without versions
				pieces := strings.Split(name, "_")
				return strings.HasPrefix(pieces[len(pieces)-1], "v")
			})
		}
	} else {
		names = slices.Values(strings.Split(*resourcesFlag, ","))
	}
	for name := range names {
		schema, ok := allResources[name]
		if !ok {
			log.Fatalf("unknown resource %q", name)
		}
		var b bytes.Buffer
		if err := generate(&b, *packageNameFlag, name, schema); err != nil {
			log.Fatalf("generating %q: %v", name, err)
		}
		out, err := format.Source(b.Bytes())
		if err != nil {
			log.Fatalf("formatting generated code: %v\ncode:\n%s", err, b.String())
		}
		if err := os.WriteFile(
			filepath.Join(*outputDirFlag, name+".gen.go"),
			out, 0777,
		); err != nil {
			log.Fatal(err)
		}
	}
}

func generate(w io.Writer, packageName, name string, resource *schema.Resource) error {
	data := struct {
		Package       string
		Name          string
		SchemaVersion int
		Schema        map[string]*schema.Schema
		Blocks        []blockSpec
	}{
		Package:       packageName,
		Name:          name,
		SchemaVersion: resource.SchemaVersion,
		Blocks:        collectBlockSpecs(name, resource.Schema),
	}

	return specTmpl.Execute(w, data)
}

type blockSpec struct {
	Name string // unique reference for the block

	Attributes map[string]valueType

	// Names of sub-blocks.
	Blocks map[string]string
	// TODO: things like description, optional etc? Probably not necessary,
	// we're not trying to validate the config, just convert it.
}

type valueType struct {
	First  schema.ValueType
	Second schema.ValueType // only set if first is list or set
}

func (vt valueType) IsList() bool {
	// TODO: sets are probably special?
	return vt.First == schema.TypeList || vt.First == schema.TypeSet
}

func collectBlockSpecs(name string, r map[string]*schema.Schema) []blockSpec {
	type todoBlock struct {
		name     string
		schema   map[string]*schema.Schema
		maxItems int
	}
	var (
		blockSpecs []blockSpec
		todo       = []todoBlock{{name: resource.ToCamel(name), schema: r}}
	)
	for len(todo) > 0 {
		i := len(todo) - 1
		c := todo[i]
		todo = todo[:i]

		var (
			attrs  = make(map[string]valueType)
			blocks = make(map[string]string)
		)
		for name, s := range c.schema {
			if s.Computed && !s.Optional {
				// Read-only.
				continue
			}
			switch t := s.Type; t {
			case schema.TypeList, schema.TypeSet:
				// Could be nested block, if the value type is not simple.
				switch e := s.Elem.(type) {
				case *schema.Schema:
					// Simple type.
					attrs[name] = valueType{First: t, Second: e.Type}
				case *schema.Resource:
					// Nested block.
					childName := fmt.Sprintf("%s_%s", c.name, resource.ToCamel(name))
					blocks[name] = childName
					todo = append(todo, todoBlock{
						name:   childName,
						schema: e.Schema,
					})
				default:
					panic(fmt.Errorf("impossible type %+v", e))
				}
			default:
				// Must be an attribute.
				attrs[name] = valueType{First: t}
			}
		}
		blockSpecs = append(blockSpecs, blockSpec{
			Name:       c.name,
			Attributes: attrs,
			Blocks:     blocks,
			// TODO: pass min and max down? Build this when we push maybe?
		})
	}
	return blockSpecs
}

//go:embed spec.tmpl
var rawSpecTmpl string

var specTmpl = template.Must(template.New("spec").Funcs(template.FuncMap{
	"valueFunc": func(in schema.ValueType) (string, error) {
		f, ok := map[schema.ValueType]string{
			schema.TypeBool:   "toBool",
			schema.TypeInt:    "toInt",
			schema.TypeFloat:  "toFloat",
			schema.TypeString: "toString",
			schema.TypeMap:    "toStringMap",
		}[in]
		if !ok {
			return "", fmt.Errorf("unknown ValueType %v", in)
		}
		return f, nil
	},
	"first": func(bs []blockSpec) blockSpec { return bs[0] },
}).Parse(rawSpecTmpl))
