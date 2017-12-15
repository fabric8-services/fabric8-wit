package controller_test

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app/test"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/goadesign/goa"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/xeipuuv/gojsonschema"
)

type JSONComplianceTestSuite struct {
	gormtestsupport.DBTestSuite
	svc           goa.Service
	jsonapiSchema gojsonschema.Schema
}

func TestJSONAPICompliance(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &JSONComplianceTestSuite{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (s *JSONComplianceTestSuite) SetupSuite() {
	s.DBTestSuite.SetupSuite()
	schemaLocation, err := filepath.Abs("./test-files/json_api_schema/json_api_schema.json")
	require.NoError(s.T(), err)
	schemaLoader := gojsonschema.NewReferenceLoader(fmt.Sprintf("file://%s", schemaLocation))
	schema, err := gojsonschema.NewSchema(schemaLoader)
	if err != nil {
		s.T().Logf("Error while loading the schema from %s: %s", schemaLocation, err.Error())
	}
	s.jsonapiSchema = *schema

}

func (s *JSONComplianceTestSuite) Validate(response interface{}) {
	marshalledResponse, err := json.MarshalIndent(response, "", "  ")
	require.NoError(s.T(), err)
	s.T().Logf("JSON response:\n%s\n", string(marshalledResponse))
	require.NoError(s.T(), err)
	documentLoader := gojsonschema.NewBytesLoader(marshalledResponse)
	result, err := s.jsonapiSchema.Validate(documentLoader)
	require.NoError(s.T(), err)
	if result.Valid() {
		s.T().Logf("The document is valid\n")
	} else {
		s.T().Logf("The document is not valid. see errors :\n")
		for _, desc := range result.Errors() {
			s.T().Logf("- %s\n", desc)
		}
	}
}

func NewService(identity account.Identity) *goa.Service {
	return testsupport.ServiceAsUser("JSON-Compliance-Service", identity)
}

func (s *JSONComplianceTestSuite) TestListSpaces() {
	// given
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.Identities(1), tf.Spaces(1, func(fxt *tf.TestFixture, idx int) error {
		fxt.Spaces[idx].OwnerID = fxt.Identities[0].ID
		return nil
	}))
	svc := NewService(*fxt.Identities[0])
	spaceCtrl := NewSpaceController(svc, gormapplication.NewGormDB(s.DB), s.Configuration, &DummyResourceManager{})
	// when
	_, spaceList := test.ListSpaceOK(s.T(), svc.Context, svc, spaceCtrl, nil, nil, nil, nil)
	// then
	s.Validate(spaceList)
}

func (s *JSONComplianceTestSuite) TestSearchCodebases() {
	s.T().Run("Single match", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(s.T(), s.DB,
			tf.Identities(1, tf.SetIdentityUsernames("spaceowner")),
			tf.Codebases(2, func(fxt *tf.TestFixture, idx int) error {
				fxt.Codebases[idx].URL = fmt.Sprintf("http://foo.com/single/%d", idx)
				return nil
			}),
		)
		svc := NewService(*fxt.Identities[0])
		searchCtrl := NewSearchController(svc, gormapplication.NewGormDB(s.DB), s.Configuration)
		// when
		_, codebaseList := test.CodebasesSearchOK(t, nil, svc, searchCtrl, nil, nil, "http://foo.com/single/0")
		// then
		s.Validate(codebaseList)
	})

	s.T().Run("Multi-match", func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(s.T(), s.DB,
			tf.Identities(1, tf.SetIdentityUsernames("spaceowner")),
			tf.Spaces(2),
			tf.Codebases(2, func(fxt *tf.TestFixture, idx int) error {
				fxt.Codebases[idx].URL = fmt.Sprintf("http://foo.com/multi/0") // both codebases have the same URL...
				fxt.Codebases[idx].SpaceID = fxt.Spaces[idx].ID                // ... but they belong to different spaces
				return nil
			}),
		)
		svc := NewService(*fxt.Identities[0])
		searchCtrl := NewSearchController(svc, gormapplication.NewGormDB(s.DB), s.Configuration)
		// when
		_, codebaseList := test.CodebasesSearchOK(t, nil, svc, searchCtrl, nil, nil, "http://foo.com/multi/0")
		// then
		// then
		s.Validate(codebaseList)
	})
}
