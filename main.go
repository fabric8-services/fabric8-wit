package main

import (
	"fmt"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/swagger"
	token "github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa"
	"github.com/goadesign/goa/middleware"
	"github.com/goadesign/goa/middleware/security/jwt"
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

	publicKey, err := token.ParseRSAPublicKeyFromPEM([]byte(RSAPublicKey))
	if err != nil {
		panic(err)
	}
	app.ConfigureJWTSecurity(service, jwt.New(publicKey, nil))

	// Mount "version" controller
	c := NewVersionController(service)
	app.MountVersionController(service, c)

	// Mount "authtoken" controller
	d := NewLoginController(service)
	app.MountLoginController(service, d)

	// Mount Swagger spec provider controller
	swagger.MountController(service)

	fmt.Println("Git Commit SHA: ", Commit)
	fmt.Println("UTC Build Time: ", BuildTime)

	service.ListenAndServe(":8080")
}
