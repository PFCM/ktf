// package ktf converts kubernetes yaml manifests to terraform resources.
package ktf

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/pfcm/ktf/convert"
	"github.com/pfcm/ktf/resource"
)

// Convert attempts to read yaml from in and convert it to HCL terraform
// resources, which will be written to out.
func Convert(in io.Reader, out io.Writer) error {
	var (
		d = yaml.NewYAMLOrJSONDecoder(in, 1024)
		f = hclwrite.NewEmptyFile()
		b = f.Body()
	)
	for {
		r := resource.New()
		if err := d.Decode(&r); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return err
		}
		if r.IsEmpty() {
			continue
		}
		block, err := convert.Convert(r)
		if err != nil {
			return fmt.Errorf("converting resource %+v/%v: %w", r.TypeKey, r.Metadata.Name, err)
		}
		b.AppendBlock(block)
	}
	var buf bytes.Buffer
	if _, err := f.WriteTo(&buf); err != nil {
		return err
	}
	formatted := hclwrite.Format(buf.Bytes())

	_, err := out.Write(formatted)
	return err
}
