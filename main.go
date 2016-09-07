package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/user"
	"time"

	"golang.org/x/net/context"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"

	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/login"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/remoteworkitem"
	"github.com/almighty/almighty-core/token"
	"github.com/almighty/almighty-core/transaction"
	jwtGo "github.com/dgrijalva/jwt-go"
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
	var scheduler *remoteworkitem.Scheduler

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

	identityRepository := account.NewIdentityRepository(db)
	userRepository := account.NewUserRepository(db)

	if err := transaction.Do(ts, func() error {
		return migration.Perform(context.Background(), ts.TX(), witRepo)
	}); err != nil {
		panic(err.Error())
	}

	// Scheduler to fetch and import remote tracker items
	scheduler = remoteworkitem.NewScheduler(db)
	defer scheduler.Stop()
	scheduler.ScheduleAllQueries()

	// Create service
	service := goa.New("alm")

	// Mount middleware
	service.Use(middleware.RequestID())
	service.Use(middleware.LogRequest(true))
	service.Use(middleware.ErrorHandler(service, true))
	service.Use(middleware.Recover())

	publicKey, err := jwtGo.ParseRSAPublicKeyFromPEM([]byte(token.RSAPublicKey))
	if err != nil {
		panic(err)
	}
	app.UseJWTMiddleware(service, jwt.New(publicKey, nil, app.NewJWTSecurity()))

	// Mount "login" controller
	oauth := &oauth2.Config{
		ClientID:     "875da0d2113ba0a6951d",
		ClientSecret: "2fe6736e90a9283036a37059d75ac0c82f4f5288",
		Scopes:       []string{"user:email"},
		Endpoint:     github.Endpoint,
	}
	tokenManager := token.NewManager(token.RSAPrivateKey, token.RSAPublicKey)
	loginService := login.NewGitHubOAuth(oauth, identityRepository, userRepository, tokenManager)
	loginCtrl := NewLoginController(service, loginService)
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

	ts2 := remoteworkitem.NewGormTransactionSupport(db)

	// Mount "tracker" controller
	repo2 := remoteworkitem.NewTrackerRepository(ts2)
	c5 := NewTrackerController(service, repo2, ts2, scheduler)
	app.MountTrackerController(service, c5)

	// Mount "trackerquery" controller
	repo3 := remoteworkitem.NewTrackerQueryRepository(ts2)
	c6 := NewTrackerqueryController(service, repo3, ts2, scheduler)
	app.MountTrackerqueryController(service, c6)

	// Mount "user" controller
	userCtrl := NewUserController(service, identityRepository)
	app.MountUserController(service, userCtrl)

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
