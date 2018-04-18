package helperfunctions

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/goadesign/goa/design"
	"github.com/goadesign/goa/goagen/codegen"
)

// Generate adds method to support conditional queries
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
	return writeFunctions(design.Design, outDir)
}

// WriteNames creates the names.txt file.
func writeFunctions(api *design.APIDefinition, outDir string) ([]string, error) {
	ctxFile := filepath.Join(outDir, "jsonapi_errors_stringer.go")
	ctxWr, err := codegen.SourceFileFor(ctxFile)
	if err != nil {
		panic(err) // bug
	}
	title := fmt.Sprintf("%s: String functions for JSONAPI Errors - See goasupport/jsonapi_errors_stringer/generator.go", api.Context())
	imports := []*codegen.ImportSpec{
		codegen.SimpleImport("fmt"),
		codegen.SimpleImport("github.com/davecgh/go-spew/spew"),
	}
	ctxWr.WriteHeader(title, "app", imports)
	if err := ctxWr.ExecuteTemplate("jsonAPIErrorsStringer", jsonAPIErrorsStringer, nil, nil); err != nil {
		return nil, err
	}
	return []string{ctxFile}, nil
}

const (
	jsonAPIErrorsStringer = `
// String implements the Stringer interface for JSONAPIErrors
func (mt JSONAPIErrors) String() string {
	if mt.Errors == nil {
		return ""
	}
	res := fmt.Sprintf("%d JSONAPI Error(s):\n", len(mt.Errors))
	for i, e := range mt.Errors {
		res += fmt.Sprintf("[ERROR No. %3d]: %s\n", i, e)
	}
	return res
}

// String implements the Stringer interface for a JSONAPIError
func (ut JSONAPIError) String() string {
	return fmt.Sprintf(` + "`" + `
		Code:    %[1]s
		Detail:  %[2]s
		ID:      %[3]s
		Links:   %[4]s
		Meta:    %[5]s
		Source:  %[6]s
		Status:  %[7]s
		Title:   %[8]s
	` + "`" + `, spew.Sdump(ut.Code),
		spew.Sdump(ut.Detail),
		spew.Sdump(ut.ID),
		spew.Sdump(ut.Links),
		spew.Sdump(ut.Meta),
		spew.Sdump(ut.Source),
		spew.Sdump(ut.Status),
		spew.Sdump(ut.Title))
}
`
)
