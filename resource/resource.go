// package resource holds the intermediate type for partially parsed resources.
// It is in its own package mostly just to avoid a circular import.
package resource

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Resource is a generic resource, partially decoded so that we can figure
// out how best to marshal it.
type Resource struct {
	TypeKey

	Metadata PartialMetadata
	Raw      map[string]any // everything, including the keys pulled out above
}

func New() Resource {
	return Resource{
		Raw: make(map[string]any),
	}
}

// IsEmpty reports if a Resource has any fields set at all. People often seem to
// put completely empty documents in their yaml for some reason.
func (r Resource) IsEmpty() bool {
	return len(r.Raw) == 0
}

// UnmarshalJSON unmarshals a Resource, pulling out the metadata that's useful
// for deciding how to turn it into a terraform resource and leaving everything
// else in Remainder.
func (r *Resource) UnmarshalJSON(raw []byte) error {
	typeMeta := struct {
		TypeKey
		Metadata PartialMetadata `json:"metadata"`
	}{}
	if err := json.Unmarshal(raw, &typeMeta); err != nil {
		return err
	}
	all := make(map[string]any)
	if err := json.Unmarshal(raw, &all); err != nil {
		return err
	}
	r.TypeKey = typeMeta.TypeKey
	r.Metadata = typeMeta.Metadata
	r.Raw = all
	return nil
}

// TypeKey contains enough to identify the type of a resource/object, used to
// look up the converter for it.
type TypeKey struct {
	APIVersion string `json:"apiVersion"` // the apiVersion from the manifest.
	Kind       string // the kind from  the manifest.
}

// PartialMetadata is a partially kubernetes ObjectMeta block, to make it easy
// to access fields that are usually need while generating resources, such as
// metadata and name.
type PartialMetadata struct {
	Name, Namespace string
	Meta            map[string]any
}

// UnmarshalJSON partially decodes a metadata block.
func (p *PartialMetadata) UnmarshalJSON(raw []byte) error {
	p.Meta = make(map[string]any)
	nameNamespace := struct {
		Name      string
		Namespace string
	}{}
	if err := json.Unmarshal(raw, &nameNamespace); err != nil {
		return err
	}
	if nameNamespace.Name == "" {
		return fmt.Errorf("missing \"name\" in %q", raw)
	}
	p.Name = nameNamespace.Name
	p.Namespace = nameNamespace.Namespace

	// Unmarshal it again, straight into the map to pick up everything else.
	if err := json.Unmarshal(raw, &p.Meta); err != nil {
		return err
	}
	// Probably don't need these twice.
	// TODO: check capitalisation?
	delete(p.Meta, "name")
	delete(p.Meta, "namespace")
	return nil
}

// ToSnake converts a string to the usual terraform snake_case by:
// - inserting a _ before any transition from lower case to upper case
// - replacing any - with _
// - lowercasing everything
func ToSnake(in string) string {
	var (
		b         strings.Builder
		prev      rune
		prevUpper = true
	)
	for _, r := range in {
		upper := unicode.IsUpper(r)
		switch {
		case r == '-':
			b.WriteByte('_')
		case !prevUpper && upper && prev != '-' && prev != '_':
			b.WriteByte('_')
			fallthrough
		default:
			b.WriteRune(unicode.ToLower(r))
		}
		prevUpper = upper
		prev = r
	}
	return b.String()
}

// ToCamel converts a string from snake_case to camelCase (always beginning with
// lower case).
func ToCamel(in string) string {
	pieces := strings.Split(in, "_")
	if len(pieces) <= 1 {
		return in
	}
	c := cases.Title(language.English)
	for i := range pieces {
		if i == 0 {
			continue
		}
		pieces[i] = c.String(pieces[i])
	}
	return strings.Join(pieces, "")
}
