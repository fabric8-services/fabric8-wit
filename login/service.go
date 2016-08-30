package login

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

// Service defines the basic entrypoint required to perform a remote oauth login
type Service interface {
	Perform(ctx *app.AuthorizeLoginContext) error
}

// NewGitHubOAuth creates a new login.Service capable of using GitHub for authorization
func NewGitHubOAuth(config *oauth2.Config, repository account.IdentityRepository) Service {
	return &gitHubOAuth{
		config:     config,
		repository: repository,
	}
}

type gitHubOAuth struct {
	config     *oauth2.Config
	repository account.IdentityRepository
}

var stateReferer = map[string]string{}

func (gh *gitHubOAuth) Perform(ctx *app.AuthorizeLoginContext) error {
	state := ctx.Params.Get("state")
	code := ctx.Params.Get("code")
	if code != "" {
		// After redirect from oauth provider

		// validate known state
		var referer string
		defer func() {
			delete(stateReferer, state)
		}()

		if referer = stateReferer[state]; referer == "" || state == "" {
			return ctx.Unauthorized()
		}

		token, err := gh.config.Exchange(ctx, code)
		if err != nil {
			fmt.Println(err)
			return ctx.Unauthorized()
		}

		emails, err := gh.getUserEmails(ctx, token)
		// locate identity
		user, err := gh.getUser(ctx, token)
		// register emails in User table

		fmt.Println(emails)
		fmt.Println(user)

		// generate token

		ctx.ResponseData.Header().Set("Location", referer)
		fmt.Println("Redirect to referer: ", referer)
		cookie := http.Cookie{Name: "almighty", Value: "weee", Domain: "localhost"}
		http.SetCookie(ctx.ResponseWriter, &cookie)
		return ctx.TemporaryRedirect()
	}

	// First time access, redirect to oauth provider

	// store referer id to state to match later
	referer := ctx.RequestData.Header.Get("Referer")
	fmt.Println("Got Request from: ", referer)
	state = uuid.NewV4().String()
	stateReferer[state] = referer

	redirectURL := gh.config.AuthCodeURL(state, oauth2.AccessTypeOnline)
	ctx.ResponseData.Header().Set("Location", redirectURL)
	return ctx.TemporaryRedirect()
}

func (gh gitHubOAuth) getUserEmails(ctx context.Context, token *oauth2.Token) ([]ghEmail, error) {
	client := gh.config.Client(ctx, token)
	resp, err := client.Get("https://api.github.com/user/emails")
	if err != nil {
		return nil, err
	}

	var emails []ghEmail
	json.NewDecoder(resp.Body).Decode(&emails)
	return emails, nil
}

func (gh gitHubOAuth) getUser(ctx context.Context, token *oauth2.Token) (*ghUser, error) {
	client := gh.config.Client(ctx, token)
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return nil, err
	}

	var user ghUser
	json.NewDecoder(resp.Body).Decode(&user)
	return &user, nil
}

// ghEmail represents the needed response from api.github.com/user/emails
type ghEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

// ghUser represents the needed response from api.github.com/user
type ghUser struct {
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}
