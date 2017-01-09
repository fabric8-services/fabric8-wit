package workitem_comments

import (
	"github.com/almighty/almighty-core/client"
	"net/http"
	goaclient "github.com/goadesign/goa/client"
	"fmt"
	"os"
	"strings"
	"io/ioutil"
	"golang.org/x/net/context"
	"encoding/json"
	"github.com/satori/go.uuid"
	"io"
	"time"
	"github.com/mitchellh/mapstructure"
	"github.com/almighty/almighty-core/workitem"
	"github.com/DATA-DOG/godog"
)

type Api struct {
	c    *client.Client
	resp *http.Response
	err  error
	body map[string]interface{}
}

func (a *Api) Reset() {
	a.c = nil
	a.resp = nil
	a.err = nil

	a.c = client.New(goaclient.HTTPClientDoer(http.DefaultClient))
	a.c.Host = "localhost:8080"
}

type IdentityHelper struct {
	savedToken string
}

func (i *IdentityHelper) GenerateToken(a *Api) error {
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
	token := fmt.Sprint(keys["token"])
	if token == "" {
		return fmt.Errorf("Failed to obtain a login token")
	}
	i.savedToken = token

	//key := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJmdWxsTmFtZSI6IlRlc3QgRGV2ZWxvcGVyIiwiaW1hZ2VVUkwiOiIiLCJ1dWlkIjoiNGI4Zjk0YjUtYWQ4OS00NzI1LWI1ZTUtNDFkNmJiNzdkZjFiIn0.ML2N_P2qm-CMBliUA1Mqzn0KKAvb9oVMbyynVkcyQq3myumGeCMUI2jy56KPuwIHySv7i-aCUl4cfIjG-8NCuS4EbFSp3ja0zpsv1UDyW6tr-T7jgAGk-9ALWxcUUEhLYSnxJoEwZPQUFNTWLYGWJiIOgM86__OBQV6qhuVwjuMlikYaHIKPnetCXqLTMe05YGrbxp7xgnWMlk9tfaxgxAJF5W6WmOlGaRg01zgvoxkRV-2C6blimddiaOlK0VIsbOiLQ04t9QA8bm9raLWX4xOkXN4ubpdsobEzcJaTD7XW0pOeWPWZY2cXCQulcAxfIy6UmCXA14C07gyuRs86Rw" // call api to get key
	a.c.SetJWTSigner(&goaclient.APIKeySigner{
		SignQuery: false,
		KeyName:   "Authorization",
		KeyValue:  i.savedToken,
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

func (i *IdentityHelper) Reset() {
	i.savedToken = ""
}

type CommentContext struct {
	api Api
	identityHelper IdentityHelper
	space client.SpaceSingle
	spaceCreated bool
	iteration client.IterationSingle
	workItem client.WorkItem2Single
	comment client.CommentSingle
	iterationName string
	spaceName string
}

func (i *CommentContext) Reset(v interface{}) {
	i.api.Reset()
	i.generateToken()
}

func (i *CommentContext) aUserWithPermissions() error {
	return i.generateToken()
}

func (i *CommentContext) generateToken() error {
	err := i.identityHelper.GenerateToken(&i.api)
	if err != nil {
		return err
	}

	return nil
}

func (c *CommentContext) anExistingSpace() error {
	if c.spaceCreated == false {
		a := c.api
		resp, err := a.c.CreateSpace(context.Background(), client.CreateSpacePath(), c.createSpacePayload())
		a.resp = resp
		a.err = err
		dec := json.NewDecoder(a.resp.Body)
		if err := dec.Decode(&c.space); err == io.EOF {
			return c.verifySpace()
		} else if err != nil {
			panic(err)
		}
		return c.verifySpace()
	}
	return nil
}

func (c *CommentContext) verifySpace() error {
	if len(c.space.Data.ID) < 1 {
		return fmt.Errorf("Expected a space with ID, but ID was [%s]", c.space.Data.ID)
	}
	expectedTitle :=  c.spaceName
	actualTitle := c.space.Data.Attributes.Name
	if *actualTitle != expectedTitle {
		return fmt.Errorf("Expected a space with title %s, but title was [%s]", expectedTitle , *actualTitle)
	}
	c.spaceCreated = true
	return nil
}

func (c *CommentContext) createSpacePayload() *client.CreateSpacePayload {
	c.spaceName = "Test space" + uuid.NewV4().String()
	return &client.CreateSpacePayload{
		Data: &client.Space{
			Attributes: &client.SpaceAttributes{
				Name: &c.spaceName,
			},
			Type: "spaces",
		},
	}
}

func (c *CommentContext) theUserCreatesANewIterationWithStartDateAndEndDate(startDate string, endDate string) error {
	a := c.api
	spaceIterationsPath := fmt.Sprintf("/api/spaces/%v/iterations", c.space.Data.ID)
	resp, err := a.c.CreateSpaceIterations(context.Background(), spaceIterationsPath, c.createSpaceIterationPayload(startDate, endDate))
	a.resp = resp
	a.err = err
	dec := json.NewDecoder(a.resp.Body)
	if err := dec.Decode(&c.iteration); err == io.EOF {
		return nil
	} else if err != nil {
		panic(err)
	}
	return nil
}

func (c *CommentContext) createSpaceIterationPayload(startDate string, endDate string) *client.CreateSpaceIterationsPayload {
	iterationName := "Test iteration"
	c.iterationName = iterationName
	const longForm = "2006-01-02"
	t1, _ := time.Parse(longForm, startDate)
	t2, _ := time.Parse(longForm, endDate)
	return &client.CreateSpaceIterationsPayload{
		Data: &client.Iteration{
			Attributes: &client.IterationAttributes{
				Name: &iterationName,
				StartAt: &t1,
				EndAt: &t2,
			},
			Type: "iterations",
		},
	}
}

func (c *CommentContext) aNewIterationShouldBeCreated() error {
	createdIteration := c.iteration
	if len(createdIteration.Data.ID ) < 1 {
		return fmt.Errorf("Expected an iteration with ID, but ID was [%s]", createdIteration.Data.ID)
	}
	expectedName :=  c.iterationName
	actualName := createdIteration.Data.Attributes.Name
	if *actualName != expectedName {
		return fmt.Errorf("Expected a space with title %s, but title was [%s]", expectedName , *actualName)
	}

	return nil
}

func (c *CommentContext) anExistingWorkItemExistsInTheProject() error {
	a := c.api
	resp, err := a.c.CreateWorkitem(context.Background(), "/api/workitems", createWorkItemPayload())
	a.resp = resp
	a.err = err
	json.NewDecoder(a.resp.Body).Decode(&c.workItem)
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

func (c *CommentContext) theUserAddsAPlainTextCommentToTheExistingWorkItem() error {
	a := c.api
	workItemId := c.workItem.Data.ID
	path := fmt.Sprintf("/api/workitems/%v/comments", *workItemId)
	a.resp = nil
	a.body = nil
	a.err = nil
	resp, err := a.c.CreateWorkItemComments(context.Background(), path, CreateCommentPayload())
	a.resp = resp
	a.err = err
	json.NewDecoder(a.resp.Body).Decode(&c.comment)
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

func (a *CommentContext) aNewCommentShouldBeAppendedAgainstTheWorkItem() error {
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

func (c *CommentContext) anExistingWorkItemExistsInTheProjectInAClosedState() error {
	a := c.api
	resp, err := a.c.CreateWorkitem(context.Background(), "/api/workitems", createClosedWorkItemPayload())
	a.resp = resp
	a.err = err
	json.NewDecoder(a.resp.Body).Decode(&a.body)
	mapError := mapstructure.Decode(a.body, &c.workItem)
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

func (c *CommentContext) theCreatorOfTheCommentMustBeTheSaidUser() error {
	// TODO: Generate an identity for every call to /api/login/generate and verify the identity here against system.creator
	return godog.ErrPending
}