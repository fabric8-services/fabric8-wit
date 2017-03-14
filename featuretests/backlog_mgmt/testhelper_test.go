package backlogmgmt

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/DATA-DOG/godog"
	"github.com/almighty/almighty-core/client"
	"github.com/almighty/almighty-core/workitem"
	goaclient "github.com/goadesign/goa/client"
	"github.com/mitchellh/mapstructure"
	"github.com/satori/go.uuid"
	goauuid "github.com/goadesign/goa/uuid"
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

type BacklogContext struct {
	api            API
	identityHelper IdentityHelper
	space          client.SpaceSingle
	spaceCreated   bool
	iteration      client.IterationSingle
	workItem       client.WorkItem2Single
	iterationName  string
	spaceName      string
}

func (i *BacklogContext) Reset(v interface{}) {
	i.api.Reset()
	i.generateToken()
}

func (i *BacklogContext) aUserWithPermissions() error {
	return i.generateToken()
}

func (i *BacklogContext) generateToken() error {
	return i.identityHelper.GenerateToken(&i.api)
}

func (i *BacklogContext) anExistingSpace() error {
	i.generateToken()
	if !i.spaceCreated {
		a := i.api
		resp, err := a.c.CreateSpace(context.Background(), client.CreateSpacePath(), i.createSpacePayload())
		log.Printf("resp: %+v\n", resp)
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

func (i *BacklogContext) verifySpace() error {
	fmt.Printf("\ni = %+v\n", i)
	fmt.Printf("\ni.space = %+v\n", i.space)
	if len(i.space.Data.ID) < 1 {
		return fmt.Errorf("Expected a space with ID, but ID was [%s]", i.space.Data.ID)
	}
	expectedTitle := i.spaceName
	fmt.Printf("\ni.space (2) = %+v\n", i.space)
	actualTitle := i.space.Data.Attributes.Name
	if *actualTitle != expectedTitle {
		return fmt.Errorf("Expected a space with title %s, but title was [%s]", expectedTitle, *actualTitle)
	}
	i.spaceCreated = true
	return nil
}

func (i *BacklogContext) createSpacePayload() *client.CreateSpacePayload {
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

func (i *BacklogContext) theUserCreatesANewIterationWithStartDateAndEndDate(startDate string, endDate string) error {
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

func (i *BacklogContext) createSpaceIterationPayload(startDate string, endDate string) *client.CreateSpaceIterationsPayload {
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

func (i *BacklogContext) aNewIterationShouldBeCreated() error {
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

func (i *BacklogContext) theUserAddsAnItemToTheBacklogWithTitleAndDescription() error {
	a := i.api
	resp, err := a.c.CreateWorkitem(context.Background(), client.CreateWorkitemPath(), createWorkItemPayload())
	a.resp = resp
	a.err = err
	json.NewDecoder(a.resp.Body).Decode(&a.body)
	mapError := mapstructure.Decode(a.body, &i.workItem)
	if mapError != nil {
		panic(mapError)
	}
	return nil
}

func createWorkItemPayload() *client.CreateWorkitemPayload {
	return &client.CreateWorkitemPayload{
		Data: &client.WorkItem2{
			Attributes: map[string]interface{}{
				workitem.SystemTitle: "Test bug",
				workitem.SystemState: workitem.SystemStateNew,
			},
			Relationships: &client.WorkItemRelationships{
				BaseType: &client.RelationBaseType{
					Data: &client.BaseTypeData{
						ID:   getSystemBugUUID(),
						Type: "workitemtypes",
					},
				},
			},
			Type: "workitems",
		},
	}
}

func getSystemBugUUID() goauuid.UUID {
	val, err := goauuid.FromString(workitem.SystemBug.String())
	if err != nil {
		panic(err)
	} else {
		return val
	}
}

func (i *BacklogContext) aNewWorkItemShouldBeCreatedInTheBacklog() error {

	createdWorkItem := i.workItem
	if len(*createdWorkItem.Data.ID) < 1 {
		return fmt.Errorf("Expected a work item with ID, but ID was [%s]", createdWorkItem.Data.ID)
	}
	expectedTitle := "Test bug"
	actualTitle := createdWorkItem.Data.Attributes["system.title"]
	if actualTitle != expectedTitle {
		return fmt.Errorf("Expected a work item with title %s, but title was [%s]", expectedTitle, actualTitle)
	}
	expectedState := "new"
	actualState := createdWorkItem.Data.Attributes["system.state"]
	if expectedState != actualState {
		return fmt.Errorf("Expected a work item with state %s, but state was [%s]", expectedState, actualState)
	}

	return nil
}

func (i *BacklogContext) theCreatorOfTheWorkItemMustBeTheSaidUser() error {
	// TODO: Generate an identity for every call to /api/login/generate and verify the identity here against system.creator
	return godog.ErrPending
}
