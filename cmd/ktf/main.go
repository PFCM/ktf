// binary ktf converts kubernetes yaml to terraform.
package main

import (
	"flag"
	"io"
	"log"
	"os"

	"github.com/pfcm/ktf"
)

var (
	inputFileFlag  = flag.String("in", "-", "`path` of a kubernetes yaml manifest to convert, or \"-\" to read from stdin")
	outputFileFlag = flag.String("out", "-", "`path` at which to write output, or \"-\" to write to stdout. If the file already exists, it will be overwritten")
)

func main() {
	flag.Parse()

	var input io.ReadCloser
	if *inputFileFlag == "-" {
		input = os.Stdin
	} else {
		i, err := os.Open(*inputFileFlag)
		if err != nil {
			log.Fatal(err)
		}
		input = i
	}

	var output io.WriteCloser
	if *outputFileFlag == "-" {
		output = os.Stdout
	} else {
		o, err := os.Create(*outputFileFlag)
		if err != nil {
			log.Fatal(err)
		}
		output = o
	}

	if err := ktf.Convert(input, output); err != nil {
		log.Fatal(err)
	}
}
