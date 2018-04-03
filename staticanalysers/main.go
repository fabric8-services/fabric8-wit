package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"os"

	"github.com/fabric8-services/fabric8-wit/staticanalysers/indentlogger"
	"github.com/fabric8-services/fabric8-wit/staticanalysers/jsonerrorresponsewithintx"
)

var logBuf = bytes.Buffer{}
var ilog = indentlogger.New(os.Stdout, "[staticanalysers] ", 0, "  ")

// Flags to enable analysers (feel free to add your own)
var useJSONErrorResponseWithinTX = flag.Bool("useJSONErrorResponseWithinTX", true, "when set, check for unwanted calls within transactions")

// main houses custom programs that can inspect Go source code and look out for
// a structure or call sequence which we like to avoid for good reasons. Each
// analyser should be equipped with examples and contain a documentation to
// explain why it was written in the first place. The framework that main()
// provides is straightforward and can easily be extended.
func main() {
	analysisErrors := []error{}
	defer func() {
		if len(analysisErrors) > 0 {
			ilog.Printf("ERRORS DETECTED: %d", len(analysisErrors))
			for idx, e := range analysisErrors {
				ilog.Printf("[ERROR #%3d]: %s", idx, e)
			}
		} else {
			ilog.Print("No errors detected.")
		}
		// flush the buffer at the end of this function
		fmt.Print(&logBuf)

		// Let exit code equal the number of errors found
		os.Exit(len(analysisErrors))
	}()

	if len(os.Args) < 2 {
		ilog.Fatal("Please provide one or more directories to check")
	}

	// ilog.Print("Analysers:")
	// ilog.Println("- JSONErrorResponseWithinTX: ", *useJSONErrorResponseWithinTX)

	for _, dir := range os.Args[1:] {
		// ilog.Println("Parsing directory", dir)
		// ilog.Indent()
		fset := token.NewFileSet()
		pkgs, err := parser.ParseDir(fset, dir, nil, parser.PackageClauseOnly)
		if err != nil {
			ilog.Panic(err)
		}
		for _, pkg := range pkgs {
			// ilog.Println("Checking package", pkg.Name)
			// ilog.Indent()
			for f := range pkg.Files {
				// ilog.Println("Checking file", f)
				// ilog.Indent()

				// Run checkers
				if *useJSONErrorResponseWithinTX {
					analysisErrors = append(analysisErrors, jsonerrorresponsewithintx.FindBadCalls(f, nil, false)...)
				}

				// ilog.Outdent()
			}
			// ilog.Outdent()
		}
		// ilog.Outdent()
	}
}
