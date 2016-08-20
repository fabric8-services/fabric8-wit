package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/user"
	"time"

	"golang.org/x/net/context"

	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/transaction"
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

const (
	// DBMaxRetryAttempts is the number of times alm server will attempt to open a connection to the database before it gives up
	DBMaxRetryAttempts int = 50
)

func main() {
	printUserInfo()

	var dbHost string

	flag.BoolVar(&Development, "dev", false, "Enable development related features, e.g. token generation endpoint")
	flag.StringVar(&dbHost, "dbhost", "", "The hostname of the db server")
	flag.Parse()

	if len(dbHost) == 0 {
		dbHost = os.Getenv("DBHOST")
	}

	if len(dbHost) == 0 {
		dbHost = "localhost"
	}

	var db *gorm.DB
	var err error
	for i := 1; i <= DBMaxRetryAttempts; i++ {
		fmt.Printf("Opening DB connection attempt %d of %d\n", i, DBMaxRetryAttempts)
		db, err = gorm.Open("postgres", fmt.Sprintf("host=%s user=postgres password=mysecretpassword sslmode=disable", dbHost))
		if err != nil {
			time.Sleep(time.Second)
		} else {
			defer db.Close()
			break
		}
	}
	if err != nil {
		panic("Could not open connection to database")
	}

	// Migrate the schema
	ts := models.NewGormTransactionSupport(db)
	witRepo := models.NewWorkItemTypeRepository(ts)
	wiRepo := models.NewWorkItemRepository(ts, witRepo)

	if err := transaction.Do(ts, func() error {
		return migration.Perform(context.Background(), ts.TX(), witRepo)
	}); err != nil {
		panic(err.Error())
	}

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
	loginCtrl := NewLoginController(service)
	app.MountLoginController(service, loginCtrl)
	// Mount "version" controller
	versionCtrl := NewVersionController(service)
	app.MountVersionController(service, versionCtrl)

	// Mount "workitem" controller
	workitemCtrl := NewWorkitemController(service, wiRepo, ts)
	app.MountWorkitemController(service, workitemCtrl)

	// Mount "workitemtype" controller
	workitemtypeCtrl := NewWorkitemtypeController(service, witRepo, ts)
	app.MountWorkitemtypeController(service, workitemtypeCtrl)

	// Mount "tracker" controller
	repo2 := models.NewTrackerRepository(ts)
	c5 := NewTrackerController(service, repo2, ts)
	app.MountTrackerController(service, c5)

	fmt.Println("Git Commit SHA: ", Commit)
	fmt.Println("UTC Build Time: ", BuildTime)
	fmt.Println("Dev mode:       ", Development)

	http.Handle("/api/", service.Mux)
	http.Handle("/", http.FileServer(assetFS()))
	http.Handle("/favicon.ico", http.NotFoundHandler())
	http.HandleFunc("/healthcheck", func(w http.ResponseWriter, r *http.Request) {
		_, err := db.DB().Exec("select 1")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	})

	// Start http
	if err := http.ListenAndServe(":8080", nil); err != nil {
		service.LogError("startup", "err", err)
	}

}

func printUserInfo() {
	u, err := user.Current()
	if err != nil {
		fmt.Printf("Failed to get current user: %s", err.Error())
	} else {
		fmt.Printf("Running as user name \"%s\" with UID %s.\n", u.Username, u.Uid)
		/*
			g, err := user.LookupGroupId(u.Gid)
			if err != nil {
				fmt.Printf("Failed to lookup group: %", err.Error())
			} else {
				fmt.Printf("Running with group \"%s\" with GID %s.\n", g.Name, g.Gid)
			}
		*/
	}
}
