package workitemcomments

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
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

type CommentContext struct {
	api            API
	identityHelper IdentityHelper
	space          client.SpaceSingle
	spaceCreated   bool
	iteration      client.IterationSingle
	workItem       client.WorkItem2Single
	comment        client.CommentSingle
	iterationName  string
	spaceName      string
}

func (i *CommentContext) Reset(v interface{}) {
	i.api.Reset()
	i.generateToken()
}

func (i *CommentContext) aUserWithPermissions() error {
	return i.generateToken()
}

func (i *CommentContext) generateToken() error {
	return i.identityHelper.GenerateToken(&i.api)
}

func (i *CommentContext) anExistingSpace() error {
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

func (i *CommentContext) verifySpace() error {
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

func (i *CommentContext) createSpacePayload() *client.CreateSpacePayload {
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

func (i *CommentContext) theUserCreatesANewIterationWithStartDateAndEndDate(startDate string, endDate string) error {
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

func (i *CommentContext) createSpaceIterationPayload(startDate string, endDate string) *client.CreateSpaceIterationsPayload {
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

func (i *CommentContext) aNewIterationShouldBeCreated() error {
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

func (i *CommentContext) anExistingWorkItemExistsInTheProject() error {
	a := i.api
	resp, err := a.c.CreateWorkitem(context.Background(), client.CreateWorkitemPath(), createWorkItemPayload())
	a.resp = resp
	a.err = err
	json.NewDecoder(a.resp.Body).Decode(&i.workItem)
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
						ID:    getSystemBugUUID(),
						Type: "workitemtypes",
					},
				},
			},
			Type: "workitems",
		},
	}
}

func (i *CommentContext) theUserAddsAPlainTextCommentToTheExistingWorkItem() error {
	a := i.api
	workItemID := *i.workItem.Data.ID
	a.resp = nil
	a.body = nil
	a.err = nil
	resp, err := a.c.CreateWorkItemComments(context.Background(), client.CreateWorkItemCommentsPath(workItemID), CreateCommentPayload())
	a.resp = resp
	a.err = err
	json.NewDecoder(a.resp.Body).Decode(&i.comment)
	return nil
}

func CreateCommentPayload() *client.CreateWorkItemCommentsPayload {
	return &client.CreateWorkItemCommentsPayload{
		Data: &client.CreateComment{
			Attributes: &client.CreateCommentAttributes{
				Body: "Test comment",
			},
			Type: "comments",
		},
	}
}

func (i *CommentContext) aNewCommentShouldBeAppendedAgainstTheWorkItem() error {
	createdComment := i.comment
	if createdComment.Data.ID == nil {
		return fmt.Errorf("Expected a comment with ID, but ID was [%s]", createdComment.Data.ID)
	}
	expectedBody := "Test comment"
	actualBody := createdComment.Data.Attributes.Body
	if *actualBody != expectedBody {
		return fmt.Errorf("Expected a comment with body %s, but body was [%s]", expectedBody, actualBody)
	}
	return nil
}

func (i *CommentContext) anExistingWorkItemExistsInTheProjectInAClosedState() error {
	a := i.api
	resp, err := a.c.CreateWorkitem(context.Background(), client.CreateWorkitemPath(), createClosedWorkItemPayload())
	a.resp = resp
	a.err = err
	json.NewDecoder(a.resp.Body).Decode(&a.body)
	mapError := mapstructure.Decode(a.body, &i.workItem)
	if mapError != nil {
		panic(mapError)
	}
	return nil
}

func createClosedWorkItemPayload() *client.CreateWorkitemPayload {
	return &client.CreateWorkitemPayload{
		Data: &client.WorkItem2{
			Attributes: map[string]interface{}{
				workitem.SystemTitle: "Test bug",
				workitem.SystemState: workitem.SystemStateClosed,
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

func (i *CommentContext) theCreatorOfTheCommentMustBeTheSaidUser() error {
	// TODO: Generate an identity for every call to /api/login/generate and verify the identity here against system.creator
	return godog.ErrPending
}
