// package lex does a lexical analysis on stdin, and prints the resulting list
// of tokens.
package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func main() {
	in, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}

	tokens, diag := hclsyntax.LexConfig(in, "<stdin>", hcl.Pos{})
	if diag.HasErrors() {
		log.Fatal(diag)
	}
	for _, token := range tokens {
		fmt.Printf("%v (%q)\n", token.Type, token.Bytes)
	}
}
