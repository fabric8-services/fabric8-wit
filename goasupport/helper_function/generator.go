package helperfunctions

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/goadesign/goa/design"
	"github.com/goadesign/goa/goagen/codegen"
	errs "github.com/pkg/errors"
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
	if err := set.Parse(os.Args[2:]); err != nil {
		return nil, errs.WithStack(err)
	}
	// First check compatibility
	if err := codegen.CheckVersion(ver); err != nil {
		return nil, err
	}
	return writeFunctions(design.Design, outDir)
}

// WriteNames creates the names.txt file.
func writeFunctions(api *design.APIDefinition, outDir string) ([]string, error) {
	ctxFile := filepath.Join(outDir, "helper_functions.go")
	ctxWr, err := codegen.SourceFileFor(ctxFile)
	if err != nil {
		panic(err) // bug
	}
	title := fmt.Sprintf("%s: Helper functions - See goasupport/helper_function/generator.go", api.Context())
	imports := []*codegen.ImportSpec{
		codegen.NewImport("uuid", "github.com/satori/go.uuid"),
		codegen.NewImport("", "github.com/fabric8-services/fabric8-wit/ptr"),
	}
	if err := ctxWr.WriteHeader(title, "app", imports); err != nil {
		return nil, errs.WithStack(err)
	}
	if err := ctxWr.ExecuteTemplate("newSpaceRelation", newSpaceRelation, nil, nil); err != nil {
		return nil, err
	}
	if err := ctxWr.ExecuteTemplate("newSpaceTemplateRelation", newSpaceTemplateRelation, nil, nil); err != nil {
		return nil, err
	}
	return []string{ctxFile}, nil
}

const (
	newSpaceRelation = `
func NewSpaceRelation(id uuid.UUID, relatedURL string) *RelationSpaces {
	return &RelationSpaces{
		Data: &RelationSpacesData{
			Type: ptr.String("spaces"),
			ID:   &id,
		},
		Links: &GenericLinks{
			Self: &relatedURL,
			Related: &relatedURL,
		},
	}
}
`

	newSpaceTemplateRelation = `
func NewSpaceTemplateRelation(id uuid.UUID, relatedURL string) *SpaceTemplateRelation {
	return &SpaceTemplateRelation{
		Data: &SpaceTemplateRelationData{
			Type: "spacetemplates",
			ID:   id,
		},
		Links: &GenericLinks{
			Self: &relatedURL,
			Related: &relatedURL,
		},
	}
}
`
)
