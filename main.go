package main

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/almighty/almighty-core/app"
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
	// Development enables certain dev only features, like auto token generation
	Development = false
)

func main() {

	flag.BoolVar(&Development, "dev", false, "Enable development related features, e.g. token generation endpoint")
	flag.Parse()

	// Create service
	service := goa.New("alm")

	// Mount middleware
	service.Use(middleware.RequestID())
	service.Use(middleware.LogRequest(true))
	service.Use(middleware.ErrorHandler(service, true))
	service.Use(middleware.Recover())

	publicKey, err := token.ParseRSAPublicKeyFromPEM([]byte(RSAPublicKey))
	if err != nil {
		panic(err)
	}
	app.UseJWTMiddleware(service, jwt.New(publicKey, nil, app.NewJWTSecurity()))

	// Mount "login" controller
	c := NewLoginController(service)
	app.MountLoginController(service, c)
	// Mount "version" controller
	c2 := NewVersionController(service)
	app.MountVersionController(service, c2)

	fmt.Println("Git Commit SHA: ", Commit)
	fmt.Println("UTC Build Time: ", BuildTime)
	fmt.Println("Dev mode:       ", Development)

	http.Handle("/api/", service.Mux)
	http.Handle("/", http.FileServer(assetFS()))
	http.Handle("/favicon.ico", http.NotFoundHandler())

	// Start http
	if err := http.ListenAndServe(":8080", nil); err != nil {
		service.LogError("startup", "err", err)
	}

}
