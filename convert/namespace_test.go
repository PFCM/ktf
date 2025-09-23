package convert

import (
	_ "embed"
)

//go:embed testdata/namespace.yaml
// var validNamespace []byte

// func TestNamespace(t *testing.T) {
// 	// The only real way for the namespace to go wrong is missing keys in the yaml, so
// 	// there's not much to gain by trying to find cases where it fails.
// 	want := []byte(`resource "kubernetes_namespace" "namespace_test" {
// 	metadata {
// 		name = "namespace-test"
// 		annotations = {
// 			"complex.annotation.test.com/something" = "v19+blarg"
// 			some-annotation = "0.1"
// 		}
// 		labels = {
// 			big-label = "haha\nha\nhaha\n"
// 			llaabbel = "yep"
// 		}
// 	}
// }
// `)

// 	d := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(validNamespace), 1024)
// 	r := resource.New()
// 	if err := d.Decode(&r); err != nil {
// 		t.Fatal(err)
// 	}

// 	got, err := namespace(r)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	f := hclwrite.NewEmptyFile()
// 	f.Body().AppendBlock(got)
// 	var b bytes.Buffer
// 	if _, err := f.WriteTo(&b); err != nil {
// 		t.Fatal(err)
// 	}

// 	// TODO: relying on the formatting is a bit unpleasant
// 	gotFmted := hclwrite.Format(b.Bytes())
// 	wantFmted := hclwrite.Format(want)

// 	if d := cmp.Diff(gotFmted, wantFmted); d != "" {
// 		t.Errorf("unexpected conversion result (-got, +want):\n%v", d)
// 	}
// }
