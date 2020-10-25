package jsonerrorresponsewithintx

import (
	"github.com/fabric8-services/fabric8-wit/staticanalysers/indentlogger"
)

func Example_findBadCalls_false_positive() {
	ilog = indentlogger.New(&logBuf, "", 0, "  ")
	src := `
	package main

	import (
		foo1 "github.com/fabric8-services/fabric8-wit/jsonapi"
		myapp "github.com/fabric8-services/fabric8-wit/application"
	)

	func main() {
		err := myapp.Transactional(c.db, func(appl application.Application) error {
			// this call is not okay
			return nil
		})
		// this one is okay
		foo1.JSONErrorResponse(a, nil)
		if err != nil {
			panic(err)
		}
	}`

	FindBadCalls("bad_call.go", src, true)
	// Output:
	// Scanning file bad_call.go
	//   Scanning imports
	//     Package "github.com/fabric8-services/fabric8-wit/jsonapi" imported as "foo1"
	//     Package "github.com/fabric8-services/fabric8-wit/application" imported as "myapp"
	//   Find calls to foo1.JSONErrorResponse
	//     found call to foo1.JSONErrorResponse at bad_call.go:12:11
	//     found call to foo1.JSONErrorResponse at bad_call.go:15:3
	//   Traversing up AST to find transactional contexts
	//     ERROR: foo1.JSONErrorResponse called at bad_call.go:12:11 from within Transactional which was started at bad_call.go:10:10
}

func Example_findBadCalls_bad_call() {
	ilog = indentlogger.New(&logBuf, "", 0, "  ")
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

	FindBadCalls("bad_call.go", src, true)
	// Output:
	// Scanning file bad_call.go
	//   Scanning imports
	//     Package "github.com/fabric8-services/fabric8-wit/jsonapi" imported as "foo1"
	//     Package "github.com/fabric8-services/fabric8-wit/application" imported as "myapp"
	//   Find calls to foo1.JSONErrorResponse
	//     found call to foo1.JSONErrorResponse at bad_call.go:12:11
	//     found call to foo1.JSONErrorResponse at bad_call.go:15:3
	//   Traversing up AST to find transactional contexts
	//     ERROR: foo1.JSONErrorResponse called at bad_call.go:12:11 from within Transactional which was started at bad_call.go:10:10
}

func Example_findBadCalls_bad_call_deeply_nested() {
	src := `
	package main

	import (
		foo1 "github.com/fabric8-services/fabric8-wit/jsonapi"
		"github.com/fabric8-services/fabric8-wit/application"
	)

	func main() {
		err := application.Transactional(c.db, func(appl application.Application) error {
			// this call is not okay
			arr := []int{0,1,2,3,4}
			return func() error {
				for i := range arr {
					return foo1.JSONErrorResponse(a, nil)
				}
			}()
		})
		// this one is okay
		foo1.JSONErrorResponse(a, nil)
		if err != nil {
			panic(err)
		}
	}`

	FindBadCalls("bad_call_deeply_nested.go", src, true)
	// Output:
	// Scanning file bad_call_deeply_nested.go
	//   Scanning imports
	//     Package "github.com/fabric8-services/fabric8-wit/jsonapi" imported as "foo1"
	//     Package "github.com/fabric8-services/fabric8-wit/application" imported as "application"
	//   Find calls to foo1.JSONErrorResponse
	//     found call to foo1.JSONErrorResponse at bad_call_deeply_nested.go:15:13
	//     found call to foo1.JSONErrorResponse at bad_call_deeply_nested.go:20:3
	//   Traversing up AST to find transactional contexts
	//     ERROR: foo1.JSONErrorResponse called at bad_call_deeply_nested.go:15:13 from within Transactional which was started at bad_call_deeply_nested.go:10:10
}

func Example_findBadCalls_no_problem() {
	src := `
	package main

	import (
		"github.com/fabric8-services/fabric8-wit/jsonapi"
		"github.com/fabric8-services/fabric8-wit/application"
	)

	func main() {
		jsonapi.JSONErrorResponse(a, nil)
		err := myapp.Transactional(c.db, func(appl application.Application) error {
			return nil
		})
		if err != nil {
			panic(err)
		}
	}`

	FindBadCalls("no_problem.go", src, true)
	// Output:
	// Scanning file no_problem.go
	//   Scanning imports
	//     Package "github.com/fabric8-services/fabric8-wit/jsonapi" imported as "jsonapi"
	//     Package "github.com/fabric8-services/fabric8-wit/application" imported as "application"
	//   Find calls to jsonapi.JSONErrorResponse
	//     found call to jsonapi.JSONErrorResponse at no_problem.go:10:3
	//   Traversing up AST to find transactional contexts
}

func Example_findBadCalls_application_not_imported() {
	src := `
	package main

	import (
		"github.com/fabric8-services/fabric8-wit/jsonapi"
	)

	func main() {
		jsonapi.JSONErrorResponse(a, nil)
	}`

	FindBadCalls("application_not_imported.go", src, true)
	// Output:
	// Scanning file application_not_imported.go
	//   Scanning imports
	//     Package "github.com/fabric8-services/fabric8-wit/jsonapi" imported as "jsonapi"
	//   Skipping file application_not_imported.go because package "" is not imported
}

func Example_findBadCalls_jsonapi_not_imported() {
	src := `
	package main

	import (
		"github.com/fabric8-services/fabric8-wit/application"
	)

	func main() {
		err := myapp.Transactional(c.db, func(appl application.Application) error {
			return nil
		})
		if err != nil {
			panic(err)
		}
	}`

	FindBadCalls("jsonapi_not_imported.go", src, true)
	// Output:
	// Scanning file jsonapi_not_imported.go
	//   Scanning imports
	//     Package "github.com/fabric8-services/fabric8-wit/application" imported as "application"
	//   Skipping file jsonapi_not_imported.go because package "github.com/fabric8-services/fabric8-wit/jsonapi" is not imported
}

func Example_findBadCalls_parse_error() {
	src := `
	package main

	func main() 
	}`

	FindBadCalls("parse_error.go", src, true)
	// Output:
	// Scanning file parse_error.go
	//   ERROR: failed to parse file "parse_error.go" with: parse_error.go:5:2: expected declaration, found '}'
}
