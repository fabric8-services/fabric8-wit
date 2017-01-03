package main

import (
	"github.com/DATA-DOG/godog"
	"github.com/almighty/almighty-core/client"
	"net/http"
	goaclient "github.com/goadesign/goa/client"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"golang.org/x/net/context"
	"encoding/json"
	"io"
	"github.com/satori/go.uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/almighty/almighty-core/workitem"
)

type Api struct {
	c    *client.Client
	resp *http.Response
	err  error
	body map[string]interface{}
	space client.SpaceSingle
	iteration client.IterationSingle
	workItem client.WorkItem2Single
	comment client.CommentSingle
	spaceName string
	iterationName string
}

func (a *Api) newScenario(i interface{}) {
	a.c = nil
	a.resp = nil
	a.err = nil

	a.c = client.New(goaclient.HTTPClientDoer(http.DefaultClient))
	a.c.Host = "localhost:8080"
}

var savedToken string

func (a *Api) anExistingSpace() error {
	if a.space.Data == nil {
		a.theUserCreatesANewSpace()
	}
	return nil
}

func (a *Api) aUserWithPermissions() error {
	resp, err := a.c.ShowStatus(context.Background(), "api/login/generate")
	a.resp = resp
	a.err = err

	// Option 1 - Extarct the 1st token from the html Data in the reponse
	defer a.resp.Body.Close()
	htmlData, err := ioutil.ReadAll(a.resp.Body)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	//fmt.Println("[[[", string(htmlData), "]]]")
	lastBin := strings.LastIndex(string(htmlData), "\"},{\"token\":\"")
	//fmt.Printf("The token to use is: %v\n", string(htmlData)[11:lastBin])

	// Option 2 - Extract the 1st token from JSON in the response
	lastBin = strings.LastIndex(string(htmlData), ",")
	//fmt.Printf("The token to use is: %v\n", string(htmlData)[1:lastBin])

	// TODO - Extract the token from the JSON map read from the html Data in the response
	byt := []byte(string(htmlData)[1:lastBin])
	var keys map[string]interface{}
	json.Unmarshal(byt, &keys)
	savedToken = fmt.Sprint(keys["token"])

	//key := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJmdWxsTmFtZSI6IlRlc3QgRGV2ZWxvcGVyIiwiaW1hZ2VVUkwiOiIiLCJ1dWlkIjoiNGI4Zjk0YjUtYWQ4OS00NzI1LWI1ZTUtNDFkNmJiNzdkZjFiIn0.ML2N_P2qm-CMBliUA1Mqzn0KKAvb9oVMbyynVkcyQq3myumGeCMUI2jy56KPuwIHySv7i-aCUl4cfIjG-8NCuS4EbFSp3ja0zpsv1UDyW6tr-T7jgAGk-9ALWxcUUEhLYSnxJoEwZPQUFNTWLYGWJiIOgM86__OBQV6qhuVwjuMlikYaHIKPnetCXqLTMe05YGrbxp7xgnWMlk9tfaxgxAJF5W6WmOlGaRg01zgvoxkRV-2C6blimddiaOlK0VIsbOiLQ04t9QA8bm9raLWX4xOkXN4ubpdsobEzcJaTD7XW0pOeWPWZY2cXCQulcAxfIy6UmCXA14C07gyuRs86Rw" // call api to get key
	a.c.SetJWTSigner(&goaclient.APIKeySigner{
		SignQuery: false,
		KeyName:   "Authorization",
		KeyValue:  savedToken,
		Format:    "Bearer %s",
	})


	userResp, userErr := a.c.ShowUser(context.Background(), "/api/user")
	var user map[string]interface{}
	json.NewDecoder(userResp.Body).Decode(&user)

	if userErr != nil {
		fmt.Printf("Error: %s", userErr)
	}

	return nil
}

func (a *Api) theUserCreatesANewSpace() error {
	resp, err := a.c.CreateSpace(context.Background(), "/api/spaces", a.createSpacePayload())
	a.resp = resp
	a.err = err
	dec := json.NewDecoder(a.resp.Body)
	if err := dec.Decode(&a.space); err == io.EOF {
		return nil
	} else if err != nil {
		panic(err)
	}
	return nil
}

func (a *Api) createSpacePayload() *client.CreateSpacePayload {
	a.spaceName = "Test space" + uuid.NewV4().String()
	return &client.CreateSpacePayload{
		Data: &client.Space{
			Attributes: &client.SpaceAttributes{
				Name: &a.spaceName,
			},
			Type: "spaces",
		},
	}
}

func (a *Api) aNewSpaceShouldBeCreated() error {
	createdSpace := a.space
	if len(createdSpace.Data.ID ) < 1 {
		return fmt.Errorf("Expected a space with ID, but ID was [%s]", createdSpace.Data.ID)
	}
	expectedTitle :=  a.spaceName
	actualTitle := createdSpace.Data.Attributes.Name
	if *actualTitle != expectedTitle {
		return fmt.Errorf("Expected a space with title %s, but title was [%s]", expectedTitle , *actualTitle)
	}

	return nil
}

func (a *Api) anExistingWorkItemExistsInTheProject() error {
	resp, err := a.c.CreateWorkitem(context.Background(), "/api/workitems", createWorkItemPayload())
	a.resp = resp
	a.err = err
	json.NewDecoder(a.resp.Body).Decode(&a.workItem)
	return nil
}

func (a *Api) theUserAddsAPlainTextCommentToTheExistingWorkItem() error {
	workItemId := a.workItem.Data.ID
	path := fmt.Sprintf("/api/workitems/%v/comments", *workItemId)
	a.resp = nil
	a.body = nil
	a.err = nil
	resp, err := a.c.CreateWorkItemComments(context.Background(), path, CreateCommentPayload())
	a.resp = resp
	a.err = err
	json.NewDecoder(a.resp.Body).Decode(&a.comment)
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

func (a *Api) aNewCommentShouldBeAppendedAgainstTheWorkItem() error {
	createdComment := a.comment
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

func (a *Api) anExistingWorkItemExistsInTheProjectInAClosedState() error {
	resp, err := a.c.CreateWorkitem(context.Background(), "/api/workitems", createClosedWorkItemPayload())
	a.resp = resp
	a.err = err
	json.NewDecoder(a.resp.Body).Decode(&a.body)
	mapError := mapstructure.Decode(a.body, &a.workItem)
	if mapError != nil {
		panic(mapError)
	}
	return nil
}

func createClosedWorkItemPayload() *client.CreateWorkitemPayload {
	return &client.CreateWorkitemPayload{
		Data: &client.WorkItem2{
			Attributes: map[string]interface{}{
				workitem.SystemTitle:   "Test bug",
				workitem.SystemState:   workitem.SystemStateClosed,
			},
			Relationships: &client.WorkItemRelationships{
				BaseType: &client.RelationBaseType{
					Data: &client.BaseTypeData{
						ID:   workitem.SystemBug,
						Type: "workitemtypes",
					},
				},
			},
			Type: "workitems",
		},
	}
}

func (a *Api) theCreatorOfTheCommentMustBeTheSaidUser() error {
	// TODO: Generate an identity for every call to /api/login/generate and verify the identity here against system.creator
	return godog.ErrPending
}

func (a *Api) theUserAddsAnItemToTheBacklogWithTitleAndDescription() error {
	resp, err := a.c.CreateWorkitem(context.Background(), "/api/workitems", createWorkItemPayload())
	a.resp = resp
	a.err = err
	json.NewDecoder(a.resp.Body).Decode(&a.body)
	mapError := mapstructure.Decode(a.body, &a.workItem)
	if mapError != nil {
		panic(mapError)
	}
	return nil
}

func createWorkItemPayload() *client.CreateWorkitemPayload {
	return &client.CreateWorkitemPayload{
		Data: &client.WorkItem2{
			Attributes: map[string]interface{}{
				workitem.SystemTitle:   "Test bug",
				workitem.SystemState:   workitem.SystemStateNew,
			},
			Relationships: &client.WorkItemRelationships{
				BaseType: &client.RelationBaseType{
					Data: &client.BaseTypeData{
						ID:   workitem.SystemBug,
						Type: "workitemtypes",
					},
				},
			},
			Type: "workitems",
		},
	}
}

func (a *Api) aNewWorkItemShouldBeCreatedInTheBacklog() error {

	createdWorkItem := a.workItem
	if len(*createdWorkItem.Data.ID ) < 1 {
		return fmt.Errorf("Expected a work item with ID, but ID was [%s]", createdWorkItem.Data.ID)
	}
	expectedTitle :=  "Test bug"
	actualTitle := createdWorkItem.Data.Attributes["system.title"]
	if actualTitle != expectedTitle {
		return fmt.Errorf("Expected a work item with title %s, but title was [%s]", expectedTitle , actualTitle)
	}
	expectedState :=  "new"
	actualState := createdWorkItem.Data.Attributes["system.state"]
	if expectedState != actualState {
		return fmt.Errorf("Expected a work item with state %s, but state was [%s]", expectedState, actualState)
	}

	return nil
}

func (a *Api) theCreatorOfTheWorkItemMustBeTheSaidUser() error {
	// TODO: Generate an identity for every call to /api/login/generate and verify the identity here against system.creator
	return godog.ErrPending
}

func (a *Api) theUserCreatesANewIteration() error {
	spaceIterationsPath := fmt.Sprintf("/api/spaces/%v/iterations", a.space.Data.ID)
	resp, err := a.c.CreateSpaceIterations(context.Background(), spaceIterationsPath, a.createSpaceIterationPayload())
	a.resp = resp
	a.err = err
	dec := json.NewDecoder(a.resp.Body)
	fmt.Println("Decoding space iteration")
	if err := dec.Decode(&a.iteration); err == io.EOF {
		return nil
	} else if err != nil {
		panic(err)
	}
	return nil
}

func (a *Api) createSpaceIterationPayload() *client.CreateSpaceIterationsPayload {
	iterationName := "Test iteration"
	a.iterationName = iterationName
	return &client.CreateSpaceIterationsPayload{
		Data: &client.Iteration{
			Attributes: &client.IterationAttributes{
				Name: &iterationName,
			},
			Type: "iterations",
		},
	}
}

func (a *Api) aNewIterationShouldBeCreated() error {
	createdIteration := a.iteration
	if len(createdIteration.Data.ID ) < 1 {
		return fmt.Errorf("Expected an iteration with ID, but ID was [%s]", createdIteration.Data.ID)
	}
	expectedName :=  a.iterationName
	actualName := createdIteration.Data.Attributes.Name
	if *actualName != expectedName {
		return fmt.Errorf("Expected a space with title %s, but title was [%s]", expectedName , *actualName)
	}

	return nil
}

func FeatureContext(s *godog.Suite) {
	a := Api{}

	s.BeforeScenario(a.newScenario)

	s.Step(`^a user with permissions to create spaces,$`, a.aUserWithPermissions)
	s.Step(`^the user creates a new space,$`, a.theUserCreatesANewSpace)
	s.Step(`^a new space should be created\.$`, a.aNewSpaceShouldBeCreated)

	s.Step(`^an existing space,$`, a.anExistingSpace)
	s.Step(`^a user with permissions to add items to backlog,$`, a.aUserWithPermissions)
	s.Step(`^the user adds an item to the backlog with title and description,$`, a.theUserAddsAnItemToTheBacklogWithTitleAndDescription)
	s.Step(`^a new work item with a space-unique ID should be created in the backlog$`, a.aNewWorkItemShouldBeCreatedInTheBacklog)
	s.Step(`^the creator of the work item must be the said user\.$`, a.theCreatorOfTheWorkItemMustBeTheSaidUser)

	s.Step(`^a user with permissions to create iterations in a space,$`, a.aUserWithPermissions)
	s.Step(`^the user creates a new iteration,$`, a.theUserCreatesANewIteration)
	s.Step(`^a new iteration should be created\.$`, a.aNewIterationShouldBeCreated)

	s.Step(`^a user with permissions to comment on work items,$`, a.aUserWithPermissions)
	s.Step(`^an existing work item exists in the space$`, a.anExistingWorkItemExistsInTheProject)
	s.Step(`^the user adds a plain text comment to the existing work item,$`, a.theUserAddsAPlainTextCommentToTheExistingWorkItem)
	s.Step(`^a new comment should be appended against the work item$`, a.aNewCommentShouldBeAppendedAgainstTheWorkItem)
	s.Step(`^the creator of the comment must be the said user\.$`, a.theCreatorOfTheCommentMustBeTheSaidUser)
	s.Step(`^an existing work item exists in the space in a closed state$`, a.anExistingWorkItemExistsInTheProjectInAClosedState)
}
