package iterations

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/almighty/almighty-core/client"
	goaclient "github.com/goadesign/goa/client"
	"github.com/satori/go.uuid"
	"golang.org/x/net/context"
)

type API struct {
	c    *client.Client
	resp *http.Response
	err  error
	body map[string]interface{}
}

func (a *API) Reset() {
	a.c = nil
	a.resp = nil
	a.err = nil

	a.c = client.New(goaclient.HTTPClientDoer(http.DefaultClient))
	a.c.Host = "localhost:8080"
}

type IdentityHelper struct {
	savedToken string
}

func (i *IdentityHelper) GenerateToken(a *API) error {
	resp, err := a.c.ShowStatus(context.Background(), client.GenerateLoginPath())
	a.resp = resp
	a.err = err

	defer a.resp.Body.Close()
	htmlData, err := ioutil.ReadAll(a.resp.Body)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var keys []map[string] interface{}
	json.Unmarshal(htmlData, &keys)
	token := fmt.Sprint(keys[0]["token"].(map[string] interface{})["access_token"])
	if token == "" {
		return fmt.Errorf("Failed to obtain a login token")
	}
	i.savedToken = token

	a.c.SetJWTSigner(&goaclient.APIKeySigner{
		SignQuery: false,
		KeyName:   "Authorization",
		KeyValue:  i.savedToken,
		Format:    "Bearer %s",
	})

	userResp, userErr := a.c.ShowUser(context.Background(), client.ShowUserPath())
	var user map[string]interface{}
	json.NewDecoder(userResp.Body).Decode(&user)

	if userErr != nil {
		fmt.Printf("Error: %s", userErr)
	}

	return nil
}

func (i *IdentityHelper) Reset() {
	i.savedToken = ""
}

type IterationContext struct {
	api            API
	identityHelper IdentityHelper
	space          client.SpaceSingle
	spaceCreated   bool
	iteration      client.IterationSingle
	iterationName  string
	spaceName      string
}

func (i *IterationContext) Reset(v interface{}) {
	i.api.Reset()
	i.generateToken()
}

func (i *IterationContext) aUserWithPermissions() error {
	return i.generateToken()
}

func (i *IterationContext) generateToken() error {
	return i.identityHelper.GenerateToken(&i.api)
}

func (i *IterationContext) anExistingSpace() error {
	if i.spaceCreated == false {
		a := i.api
		resp, err := a.c.CreateSpace(context.Background(), client.CreateSpacePath(), i.createSpacePayload())
		a.resp = resp
		a.err = err
		dec := json.NewDecoder(a.resp.Body)
		if err := dec.Decode(&i.space); err == io.EOF {
			return i.verifySpace()
		} else if err != nil {
			panic(err)
		}
		return i.verifySpace()
	}
	return nil
}

func (i *IterationContext) verifySpace() error {
	if len(i.space.Data.ID) < 1 {
		return fmt.Errorf("Expected a space with ID, but ID was [%s]", i.space.Data.ID)
	}
	expectedTitle := i.spaceName
	actualTitle := i.space.Data.Attributes.Name
	if *actualTitle != expectedTitle {
		return fmt.Errorf("Expected a space with title %s, but title was [%s]", expectedTitle, *actualTitle)
	}
	i.spaceCreated = true
	return nil
}

func (i *IterationContext) createSpacePayload() *client.CreateSpacePayload {
	i.spaceName = "Test space" + uuid.NewV4().String()
	return &client.CreateSpacePayload{
		Data: &client.Space{
			Attributes: &client.SpaceAttributes{
				Name: &i.spaceName,
			},
			Type: "spaces",
		},
	}
}

func (i *IterationContext) theUserCreatesANewIterationWithStartDateAndEndDate(startDate string, endDate string) error {
	a := i.api
	spaceID := i.space.Data.ID.String()
	resp, err := a.c.CreateSpaceIterations(context.Background(), client.CreateSpaceIterationsPath(spaceID), i.createSpaceIterationPayload(startDate, endDate))
	a.resp = resp
	a.err = err
	dec := json.NewDecoder(a.resp.Body)
	if err := dec.Decode(&i.iteration); err == io.EOF {
		return nil
	} else if err != nil {
		panic(err)
	}
	return nil
}

func (i *IterationContext) createSpaceIterationPayload(startDate string, endDate string) *client.CreateSpaceIterationsPayload {
	iterationName := "Test iteration"
	i.iterationName = iterationName
	const longForm = "2006-01-02"
	t1, _ := time.Parse(longForm, startDate)
	t2, _ := time.Parse(longForm, endDate)
	return &client.CreateSpaceIterationsPayload{
		Data: &client.Iteration{
			Attributes: &client.IterationAttributes{
				Name:    &iterationName,
				StartAt: &t1,
				EndAt:   &t2,
			},
			Type: "iterations",
		},
	}
}

func (i *IterationContext) aNewIterationShouldBeCreated() error {
	createdIteration := i.iteration
	if len(createdIteration.Data.ID) < 1 {
		return fmt.Errorf("Expected an iteration with ID, but ID was [%s]", createdIteration.Data.ID)
	}
	expectedName := i.iterationName
	actualName := createdIteration.Data.Attributes.Name
	if *actualName != expectedName {
		return fmt.Errorf("Expected a space with title %s, but title was [%s]", expectedName, *actualName)
	}

	return nil
}
