package spaces

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

	// Option 1 - Extract the 1st token from the html Data in the reponse
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

type SpaceContext struct {
	api Api
	identityHelper IdentityHelper
	space client.SpaceSingle
	spaceName string
}

func (s *SpaceContext) Reset(v interface{}) {
	s.api.Reset()
	s.generateToken()
}

func (i *SpaceContext) aUserWithPermissions() error {
	return i.generateToken()
}

func (s *SpaceContext) generateToken() error {
	err := s.identityHelper.GenerateToken(&s.api)
	if err != nil {
		return err
	}

	return nil
}

func (s *SpaceContext) theUserCreatesANewSpace() error {
	a := s.api
	resp, err := a.c.CreateSpace(context.Background(), client.CreateSpacePath(), s.createSpacePayload())
	a.resp = resp
	a.err = err
	dec := json.NewDecoder(a.resp.Body)
	if err := dec.Decode(&s.space); err == io.EOF {
		return nil
	} else if err != nil {
		panic(err)
	}
	return nil
}

func (s *SpaceContext) createSpacePayload() *client.CreateSpacePayload {
	s.spaceName = "Test space" + uuid.NewV4().String()
	return &client.CreateSpacePayload{
		Data: &client.Space{
			Attributes: &client.SpaceAttributes{
				Name: &s.spaceName,
			},
			Type: "spaces",
		},
	}
}

func (s *SpaceContext) aNewSpaceShouldBeCreated() error {
	createdSpace := s.space
	if len(createdSpace.Data.ID ) < 1 {
		return fmt.Errorf("Expected a space with ID, but ID was [%s]", createdSpace.Data.ID)
	}
	expectedTitle :=  s.spaceName
	actualTitle := createdSpace.Data.Attributes.Name
	if *actualTitle != expectedTitle {
		return fmt.Errorf("Expected a space with title %s, but title was [%s]", expectedTitle , *actualTitle)
	}

	return nil
}