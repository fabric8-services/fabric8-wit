package spaces

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/almighty/almighty-core/client"
	goaclient "github.com/goadesign/goa/client"
	"github.com/goadesign/goa/uuid"
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

func (a *API) parseErrorResponse(operationMessage string) error {
	json.NewDecoder(a.resp.Body).Decode(&a.body)
	errors := a.body["errors"].([]interface{})
	if len(errors) == 1 {
		firstError := errors[0].(map[string]interface{})
		errorDetail := firstError["detail"]
		return fmt.Errorf("%v due to: %v", operationMessage, errorDetail)
	}
	var buffer bytes.Buffer
	for _, error := range errors {
		errorInstance := error.(map[string]interface{})
		buffer.WriteString(errorInstance["detail"].(string))
		buffer.WriteString("\n")
	}
	return fmt.Errorf("%v due to: %v", operationMessage, buffer.String())
}

type IdentityHelper struct {
	savedToken string
}

func (i *IdentityHelper) GenerateToken(a *API) error {
	resp, err := a.c.ShowStatus(context.Background(), client.GenerateLoginPath())
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

type SpaceContext struct {
	api            API
	identityHelper IdentityHelper
	space          client.SpaceSingle
	spaces         client.SpaceList
	spaceName      string
}

func (s *SpaceContext) CleanupDatabase() {
	s.api.Reset()
	s.generateToken()

	a := s.api
	a.c.Dump = true
	// TODO handle paginated responses when too many spaces exist
	listResp, listErr := a.c.ListSpace(context.Background(), client.ListSpacePath(), nil, nil)
	defer listResp.Body.Close()

	if listResp.StatusCode != http.StatusOK {
		panic(fmt.Errorf("Failed to list spaces"))
	}
	if listErr != nil {
		panic(fmt.Errorf("Failed to list spaces: %v", listErr))
	}

	// TODO Determine why Decoder.decode fails locally,
	// and ioutil.ReadAll with json.Unmarshal is required
	listRespBody, listRespBodyErr := ioutil.ReadAll(listResp.Body)
	if listRespBodyErr == nil {
		allSpaces := new(client.SpaceList)
		json.Unmarshal(listRespBody, &allSpaces)

		// Delete each space individually
		s.api.Reset()
		s.generateToken()
		for _, aSpace := range allSpaces.Data {
			itrSpaceID := (*aSpace).ID.String()
			deleteResp, deleteErr := a.c.DeleteSpace(context.Background(), client.DeleteSpacePath(itrSpaceID))
			if deleteResp.StatusCode != http.StatusOK {
				panic(fmt.Errorf("Failed to delete space %v, due to error: %v", itrSpaceID, deleteResp.StatusCode))
			}
			if deleteErr != nil {
				panic(fmt.Errorf("Failed to delete space %v: %v", itrSpaceID, deleteErr))
			}
		}
	} else {
		panic(listRespBodyErr)
	}
}

func (s *SpaceContext) Reset(v interface{}) {
	s.api.Reset()
	s.generateToken()
}

func (s *SpaceContext) aUserWithPermissions() error {
	return s.generateToken()
}

func (s *SpaceContext) generateToken() error {
	return s.identityHelper.GenerateToken(&s.api)
}

func (s *SpaceContext) theUserCreatesANewSpace(spaceName string) error {
	a := s.api
	resp, err := a.c.CreateSpace(context.Background(), client.CreateSpacePath(), s.createSpacePayload(spaceName))
	a.resp = resp
	a.err = err
	if a.resp.StatusCode != http.StatusCreated {
		return a.parseErrorResponse("Failed to create space")
	}
	dec := json.NewDecoder(a.resp.Body)
	if err := dec.Decode(&s.space); err == io.EOF {
		return nil
	} else if err != nil {
		panic(err)
	}
	return nil
}

func (s *SpaceContext) createSpacePayload(spaceName string) *client.CreateSpacePayload {
	s.spaceName = spaceName
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
	if len(createdSpace.Data.ID) < 1 {
		return fmt.Errorf("Expected a space with ID, but ID was [%s]", createdSpace.Data.ID)
	}
	expectedTitle := s.spaceName
	actualTitle := createdSpace.Data.Attributes.Name
	if *actualTitle != expectedTitle {
		return fmt.Errorf("Expected a space with title %s, but title was [%s]", expectedTitle, *actualTitle)
	}

	return nil
}

func (s *SpaceContext) aSpaceAlreadyExistsWithTheSameUserAsOwner(spaceName string) error {
	a := s.api
	resp, err := a.c.ListSpace(context.Background(), client.ListSpacePath(), nil, nil)
	a.resp = resp
	a.err = err
	if a.resp.StatusCode != http.StatusOK {
		return a.parseErrorResponse("Failed to list spaces")
	}
	dec := json.NewDecoder(a.resp.Body)
	if err := dec.Decode(&s.spaces); err == io.EOF {
		for _, aSpace := range s.spaces.Data {
			var itrSpaceName string
			itrSpaceName = *aSpace.Attributes.Name
			if itrSpaceName == spaceName {
				return nil
			}
		}
		return fmt.Errorf("Failed to locate space with name: %v under current user", spaceName)
	} else if err != nil {
		panic(err)
	}
	return nil
}

func (s *SpaceContext) aNewSpaceShouldNotBeCreated() error {
	createdSpace := s.space
	if *createdSpace.Data.ID != (uuid.UUID{}) {
		return fmt.Errorf("A space with name %v, was created when it should not be.", createdSpace.Data.Attributes.Name)
	}

	return nil
}

func (s *SpaceContext) aSpaceAlreadyExistsWithADifferentUserAsOwner(spaceName string) error {
	a := s.api
	resp, err := a.c.ListSpace(context.Background(), client.ListSpacePath(), nil, nil)
	a.resp = resp
	a.err = err
	if a.resp.StatusCode != http.StatusOK {
		return a.parseErrorResponse("Failed to list spaces")
	}
	dec := json.NewDecoder(a.resp.Body)
	if err := dec.Decode(&s.spaces); err == io.EOF {
		for _, aSpace := range s.spaces.Data {
			var itrSpaceName string
			itrSpaceName = *aSpace.Attributes.Name
			// TODO Verify the owner of the space is different from current identity of token
			if itrSpaceName == spaceName {
				return nil
			}
		}
		return fmt.Errorf("Failed to locate space with name: %v under current user", spaceName)
	} else if err != nil {
		panic(err)
	}
	return nil
}
