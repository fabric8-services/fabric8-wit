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
}

func TestJSONAPICompliance(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &JSONComplianceTestSuite{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (s *JSONComplianceTestSuite) NewSecuredSpaceController(identity account.Identity) (*goa.Service, *SpaceController) {
	svc := testsupport.ServiceAsUser("Space-Service", identity)
	return svc, NewSpaceController(svc, gormapplication.NewGormDB(s.DB), s.Configuration, &DummyResourceManager{})
}

func (s *JSONComplianceTestSuite) TestListSpaces() {
	// given
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.Identities(1), tf.Spaces(1, func(fxt *tf.TestFixture, idx int) error {
		fxt.Spaces[idx].OwnerId = fxt.Identities[0].ID
		return nil
	}))

	svc, ctrl := s.NewSecuredSpaceController(*fxt.Identities[0])
	// when
	_, spaceList := test.ListSpaceOK(s.T(), svc.Context, svc, ctrl, nil, nil, nil, nil)
	// then
	jsonSpaceList, err := json.MarshalIndent(spaceList, "", "  ")
	require.Nil(s.T(), err)
	s.T().Logf("JSON response:\n%s\n", string(jsonSpaceList))
	schemaLocation, err := filepath.Abs("./test-files/json_api_schema/json_api_schema.json")
	require.Nil(s.T(), err)
	schemaLoader := gojsonschema.NewReferenceLoader(fmt.Sprintf("file://%s", schemaLocation))
	schema, err := gojsonschema.NewSchema(schemaLoader)
	if err != nil {
		s.T().Logf("Error while loading the schema from %s: %s", schemaLocation, err.Error())
	}
	require.Nil(s.T(), err)
	documentLoader := gojsonschema.NewBytesLoader(jsonSpaceList)
	result, err := schema.Validate(documentLoader)
	require.Nil(s.T(), err)
	if result.Valid() {
		s.T().Logf("The document is valid\n")
	} else {
		s.T().Logf("The document is not valid. see errors :\n")
		for _, desc := range result.Errors() {
			s.T().Logf("- %s\n", desc)
		}
	}
}
