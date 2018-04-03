package jsonerrorresponsewithintx

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path"

	"github.com/fabric8-services/fabric8-wit/staticanalysers/indentlogger"
	errs "github.com/pkg/errors"
)

const (
	jsonapiPackage             = "github.com/fabric8-services/fabric8-wit/jsonapi"
	applicationPackage         = "github.com/fabric8-services/fabric8-wit/application"
	transactionalFuncName      = "Transactional"
	jsonErrorResponseFunceName = "JSONErrorResponse"
)

// TODO(kwk): Make thread-safe when needed
var logBuf = bytes.Buffer{}
var ilog = indentlogger.New(&logBuf, "[staticanalysers/jsonerrorresponsewithintx] ", 0, "  ")

// FindBadCalls searches for calls to jsonapi.JSONErrorResponses that are being
// made from within an application.Transactional call. It prints the logs and
// returns each error that it prints as well.
func FindBadCalls(filename string, src interface{}, outputOnNoError bool) (badCallErrors []error) {
	ilog.Printf("Scanning file %s\n", filename)
	ilog.Indent()
	defer func() {
		ilog.Outdent()

		// flush the buffer at the end of this function for examples to pick it up
		if len(badCallErrors) > 0 || outputOnNoError {
			fmt.Print(&logBuf)
		}
		logBuf = bytes.Buffer{}
	}()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, src, parser.AllErrors)
	if err != nil {
		err = errs.Wrapf(err, "failed to parse file \"%s\" with", filename)
		ilog.Println("ERROR:", err)
		return []error{err}
	}

	jsonAPIImportedAs, applicationImportedAs := findImports(file)
	if jsonAPIImportedAs == "" {
		ilog.Printf(`Skipping file %s because package "%s" is not imported`, filename, jsonapiPackage)
		return nil
	}
	if applicationImportedAs == "" {
		ilog.Printf(`Skipping file %s because package "%s" is not imported`, filename, applicationImportedAs)
		return nil
	}

	// Get all function declarations as entry points to the file
	funcDecls := []*ast.FuncDecl{}
	for _, d := range file.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok {
			funcDecls = append(funcDecls, fn)
		}
	}

	// Find all calls to jsonapi.JSONErrorResponse and remember the path that
	// let to the call in the visitor struct.
	badFuncName := jsonErrorResponseFunceName
	badPkgName := jsonAPIImportedAs
	ilog.Printf("Find calls to %s.%s", badPkgName, badFuncName)
	ilog.Indent()
	ancestors := []ast.Node{}
	foundCallsAndAncestors := map[ast.Node][]ast.Node{}
	for _, funcDecl := range funcDecls {
		ast.Inspect(funcDecl, func(node ast.Node) bool {
			if isCallExprTo(node, badPkgName, badFuncName) {
				ilog.Printf("found call to %s.%s at %s", badPkgName, badFuncName, fset.Position(node.Pos()))
				// remember parents for each call
				foundCallsAndAncestors[node] = ancestors
				// reset ancestors
				ancestors = []ast.Node{}
				return false
			}
			// remember current node as parent node
			ancestors = append(ancestors, node)
			return true
		})
	}
	ilog.Outdent()

	ilog.Println("Traversing up AST to find transactional contexts")
	errors := []error{}
	for node, ancestors := range foundCallsAndAncestors {
		ilog.Indent()
		for _, ancestor := range ancestors {
			if isSelectorExprTo(ancestor, applicationImportedAs, transactionalFuncName) {
				err := errs.Errorf("%s.%s called at %s from within %s which was started at %s", badPkgName, badFuncName, fset.Position(node.Pos()), transactionalFuncName, fset.Position(ancestor.Pos()))
				ilog.Println("ERROR:", err)
				errors = append(errors, err)
			}
		}
		ilog.Outdent()
	}

	if len(errors) > 0 {
		return errors
	}
	return nil
}

// isSelectorExprTo returns true if the given node is a selector expression to
// pkgName.funcName; otherwise false is returned
func isSelectorExprTo(node ast.Node, pkgName, funcName string) bool {
	if selectorExpr, ok := node.(*ast.SelectorExpr); ok {
		if identPkg, ok := selectorExpr.X.(*ast.Ident); ok {
			if identPkg.Name == pkgName && selectorExpr.Sel.Name == funcName {
				return true
			}
		}
	}
	return false
}

// isCallExprTo returns true if the given node is a function call to
// pkgName.funcName; otherwise false is returned
func isCallExprTo(node ast.Node, pkgName, funcName string) bool {
	if callExpr, ok := node.(*ast.CallExpr); ok {
		return isSelectorExprTo(callExpr.Fun, pkgName, funcName)
	}
	return false
}

// findImports returns the local aliases under which the
// "github.com/fabric8-services/fabric8-wit/jsonapi" and
// "github.com/fabric8-services/fabric8-wit/application" are imported. If they
// are just imported without an alias we return the packages base name.
func findImports(file *ast.File) (jsonAPIImportedAs string, applicationImportedAs string) {
	ilog.Println("Scanning imports")
	ilog.Indent()
	defer ilog.Outdent()

	for _, imp := range file.Imports {
		if imp.Path.Value == "\""+jsonapiPackage+"\"" {
			// Do we import the jsonapi package with a local alias?
			if imp.Name != nil {
				jsonAPIImportedAs = imp.Name.Name
			} else {
				jsonAPIImportedAs = path.Base(jsonapiPackage)
			}
			ilog.Printf(`Package "%s" imported as "%s"`, jsonapiPackage, jsonAPIImportedAs)
		} else if imp.Path.Value == "\""+applicationPackage+"\"" {
			// Do we import the application package with a local alias?
			if imp.Name != nil {
				applicationImportedAs = imp.Name.Name
			} else {
				applicationImportedAs = path.Base(applicationPackage)
			}
			ilog.Printf(`Package "%s" imported as "%s"`, applicationPackage, applicationImportedAs)
		}
	}

	return jsonAPIImportedAs, applicationImportedAs
}
