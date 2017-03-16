package goaheadermethods

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/goadesign/goa/design"
	"github.com/goadesign/goa/goagen/codegen"
)

// Generate adds Get`Header`() methods to the XContext objects
func Generate() ([]string, error) {
	var (
		ver    string
		outDir string
	)
	set := flag.NewFlagSet("app", flag.PanicOnError)
	set.String("design", "", "") // Consume design argument so Parse doesn't complain
	set.StringVar(&ver, "version", "", "")
	set.StringVar(&outDir, "out", "", "")
	set.Parse(os.Args[2:])

	// First check compatibility
	if err := codegen.CheckVersion(ver); err != nil {
		return nil, err
	}

	return WriteNames(design.Design, outDir)
}

// RequestContext holds a single goa Request Context object
type RequestContext struct {
	Name    string
	Headers []RequestHeader
}

// RequestHeader holds a single HTTP Header as defined in the design for a Request Context
type RequestHeader struct {
	Name   string
	Header string
	Type   string
}

// WriteNames creates the names.txt file.
func WriteNames(api *design.APIDefinition, outDir string) ([]string, error) {
	// Now iterate through the resources to gather their names
	var rcs []RequestContext
	api.IterateResources(func(res *design.ResourceDefinition) error {
		res.IterateActions(func(act *design.ActionDefinition) error {
			if act.Headers != nil {

				name := fmt.Sprintf("%v%vContext", codegen.Goify(act.Name, true), codegen.Goify(res.Name, true))
				rc := RequestContext{Name: name}
				for header, value := range act.Headers.Type.ToObject() {
					rc.Headers = append(
						rc.Headers,
						RequestHeader{
							Name:   codegen.Goify(header, true),
							Header: header,
							Type:   codegen.GoTypeRef(value.Type, nil, 0, false),
						})
				}
				rcs = append(rcs, rc)
			}
			return nil
		})
		return nil
	})

	ctxFile := filepath.Join(outDir, "context_headers.go")
	ctxWr, err := codegen.SourceFileFor(ctxFile)
	if err != nil {
		panic(err) // bug
	}
	title := fmt.Sprintf("%s: Contex Header Methods", api.Context())
	imports := []*codegen.ImportSpec{
		codegen.SimpleImport("fmt"),
		codegen.SimpleImport("net/http"),
		codegen.SimpleImport("strconv"),
		codegen.SimpleImport("strings"),
		codegen.SimpleImport("time"),
		codegen.SimpleImport("unicode/utf8"),
	}

	ctxWr.WriteHeader(title, "app", imports)
	if err := ctxWr.ExecuteTemplate("headerMethods", headerMethods, nil, rcs); err != nil {
		return nil, err
	}
	err = ctxWr.FormatCode()
	if err != nil {
		return nil, err
	}
	return []string{ctxFile}, nil
}

const (
	headerMethods = `
{{ range $req := . }}{{ range $head := $req.Headers }}
// Get{{ $head.Name }} return the HTTP Header {{ $head.Header }}
func (ctx *{{$req.Name}}) Get{{ $head.Name }}() *{{ $head.Type }} {
	return ctx.{{ $head.Name }}
}{{ end }}
{{ end }}
`
)
