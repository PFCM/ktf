package ktf

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/hcl/v2/hclparse"
)

// TestConvert is a high-level test that just checks all of the testdata
// converts through the Convert function without errors. The convert package
// should have its own more thorough tests to actually check the results are
// correct.
func TestConvert(t *testing.T) {
	files, err := filepath.Glob("./convert/testdata/*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	testFile := func(t *testing.T, filename string, r io.Reader) {
		var out bytes.Buffer
		if err := Convert(r, &out); err != nil {
			t.Fatalf("%s: convert: %v", filename, err)
		}
		// At least make sure the output is valid.
		p := hclparse.NewParser()
		_, err := p.ParseHCL(out.Bytes(), filename)
		if err != nil {
			t.Fatalf("%s: convert -> parse: %v", filename, err)
		}
	}

	// Each file one at a time.
	for _, f := range files {
		t.Run(f, func(t *testing.T) {
			rc, err := os.Open(f)
			if err != nil {
				t.Fatal(err)
			}
			defer rc.Close()
			testFile(t, f, rc)
		})
	}
	// Stick them all together to make sure we're handling mutli-document files.
	t.Run("all-files", func(t *testing.T) {
		var b bytes.Buffer
		for _, f := range files {
			raw, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			b.Write(raw)
			b.WriteString("\n---\n")
		}
		testFile(t, "all-files", bytes.NewReader(b.Bytes()))
	})
}
