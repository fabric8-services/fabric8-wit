package main

import (
	"fmt"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/swagger"
	"github.com/goadesign/goa"
	"github.com/goadesign/goa/middleware"
)

var (
	// Commit current build commit set by build script
	Commit = "0"
	// BuildTime set by build script
	BuildTime = "0"
)

func main() {
	// Create service
	service := goa.New("API")

	// Setup middleware
	service.Use(middleware.RequestID())
	service.Use(middleware.LogRequest(true))
	service.Use(middleware.ErrorHandler(service, true))
	service.Use(middleware.Recover())

	// Mount "version" controller
	c := NewVersionController(service)
	app.MountVersionController(service, c)
	// Mount Swagger spec provider controller
	swagger.MountController(service)

	fmt.Println("Git Commit SHA: ", Commit)
	fmt.Println("UTC Build Time: ", BuildTime)

	service.ListenAndServe(":8080")
}
