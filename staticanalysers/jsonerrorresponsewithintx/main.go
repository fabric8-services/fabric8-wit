package jsonerrorresponsewithintx

import (
	"bytes"

	"github.com/fabric8-services/fabric8-wit/staticanalysers"
)

var logBuf = bytes.Buffer{}
var ilog = staticanalysers.NewIndentLogger(&logBuf, "[staticanalysers/jsonerrorresponsewithintx] ", 0, "  ")

func main() {
	// if len(os.Args) < 2 {
	// 	ilog.Panic("please provide one or more directories to check")
	// }
	// for _, dir := range os.Args[1:] {
	// 	ilog.Println("Inspecting directory", dir)
	// 	ilog.Indent()
	// 	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
	// 		if info.Mode().IsRegular() {
	// 			fset := token.NewFileSet()
	// 			_, err := parseFile(fset, path, nil, parser.AllErrors)
	// 			if err != nil {
	// 				return errs.Wrapf(err, "failed to parse %s", path)
	// 			}
	// 		}
	// 		return nil
	// 	})
	// 	ilog.Outdent()
	// }

	src := `
	package main

	import (
		foo1 "github.com/fabric8-services/fabric8-wit/jsonapi"
		myapp "github.com/fabric8-services/fabric8-wit/application"
	)

	func main() {
		err := myapp.Transactional(c.db, func(appl application.Application) error {
			// this call is not okay
			return foo1.JSONErrorResponse(a, nil)
		})
		// this one is okay
		foo1.JSONErrorResponse(a, nil)
		if err != nil {
			panic(err)
		}
	}`

	errors := findBadCalls("demo.go", src)
	if errors != nil {
		ilog.Println("Errors found")
	}
}
