package convert

import (
	_ "embed"
	"testing"
)

//go:embed testdata/serviceaccount.yaml
var validServiceAccount []byte

func TestServiceAccount(t *testing.T) {
	t.Fatal("TODO")
}
