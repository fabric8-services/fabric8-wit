package main

import (
	"fmt"
	"github.com/ALMighty/almighty-core/app"
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
	service := goa.New("alm")

	// Mount middleware
	service.Use(middleware.RequestID())
	service.Use(middleware.LogRequest(true))
	service.Use(middleware.ErrorHandler(service, true))
	service.Use(middleware.Recover())

	// Mount "version" controller
	c := NewVersionController(service)
	app.MountVersionController(service, c)
	
	// Mount "authtoken" controller
	d := NewLoginController(service)
	app.MountLoginController(service, d)

	fmt.Println("Git Commit SHA: ", Commit)
	fmt.Println("UTC Build Time: ", BuildTime)

	// Start service
	if err := service.ListenAndServe(":8080"); err != nil {
		service.LogError("startup", "err", err)
	}
}
