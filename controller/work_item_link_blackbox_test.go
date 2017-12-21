package controller_test

import (
	"bytes"
	"fmt"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/rest"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	testtoken "github.com/fabric8-services/fabric8-wit/test/token"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/fabric8-services/fabric8-wit/workitem/link"

	"github.com/goadesign/goa"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type workItemLinkSuite struct {
	gormtestsupport.DBTestSuite
	testDir string
}

func TestSuiteWorkItemLinks(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, new(workItemLinkSuite))
}

func (s *workItemLinkSuite) SetupTest() {
	s.DBTestSuite.SetupTest()
	s.testDir = filepath.Join("test-files", "work_item_link")
}

// CreateWorkItemLinkCategory creates a work item link category
func newCreateWorkItemLinkCategoryPayload(name string) *app.CreateWorkItemLinkCategoryPayload {
	description := "This work item link category is managed by an admin user."
	// Use the goa generated code to create a work item link category
	return &app.CreateWorkItemLinkCategoryPayload{
		Data: &app.WorkItemLinkCategoryData{
			Type: link.EndpointWorkItemLinkCategories,
			Attributes: &app.WorkItemLinkCategoryAttributes{
				Name:        &name,
				Description: &description,
			},
		},
	}
}

// CreateWorkItem defines a work item link
func newCreateWorkItemPayload(spaceID uuid.UUID, workItemType uuid.UUID, title string) *app.CreateWorkitemsPayload {
	spaceRelatedURL := rest.AbsoluteURL(&http.Request{Host: "api.service.domain.org"}, app.SpaceHref(spaceID.String()))
	witRelatedURL := rest.AbsoluteURL(&http.Request{Host: "api.service.domain.org"}, app.WorkitemtypeHref(spaceID.String(), workItemType))
	payload := app.CreateWorkitemsPayload{
		Data: &app.WorkItem{
			Attributes: map[string]interface{}{
				workitem.SystemTitle: title,
				workitem.SystemState: workitem.SystemStateClosed,
			},
			Relationships: &app.WorkItemRelationships{
				BaseType: &app.RelationBaseType{
					Data: &app.BaseTypeData{
						ID:   workItemType,
						Type: "workitemtypes",
					},
					Links: &app.GenericLinks{
						Self:    &witRelatedURL,
						Related: &witRelatedURL,
					},
				},
				Space: app.NewSpaceRelation(spaceID, spaceRelatedURL),
			},
			Type: "workitems",
		},
	}
	return &payload
}

// CreateWorkItemLinkType defines a work item link type
func newCreateWorkItemLinkTypePayload(name string, categoryID, spaceID uuid.UUID) *app.CreateWorkItemLinkTypePayload {
	description := "Specify that one bug blocks another one."
	lt := link.WorkItemLinkType{
		Name:           name,
		Description:    &description,
		Topology:       link.TopologyNetwork,
		ForwardName:    "forward name string for " + name,
		ReverseName:    "reverse name string for " + name,
		LinkCategoryID: categoryID,
		SpaceID:        spaceID,
	}
	reqLong := &http.Request{Host: "api.service.domain.org"}
	payload := ConvertWorkItemLinkTypeFromModel(reqLong, lt)
	// The create payload is required during creation. Simply copy data over.
	return &app.CreateWorkItemLinkTypePayload{
		Data: payload.Data,
	}
}

// newCreateWorkItemLinkPayload returns the payload to create a work item link
func newCreateWorkItemLinkPayload(sourceID, targetID, linkTypeID uuid.UUID) *app.CreateWorkItemLinkPayload {
	lt := link.WorkItemLink{
		SourceID:   sourceID,
		TargetID:   targetID,
		LinkTypeID: linkTypeID,
	}
	payload := ConvertLinkFromModel(&http.Request{Host: "api.service.domain.org"}, lt)
	// The create payload is required during creation. Simply copy data over.
	return &app.CreateWorkItemLinkPayload{
		Data: payload.Data,
	}
}

// newUpdateWorkItemLinkPayload returns the payload to update a work item link
func newUpdateWorkItemLinkPayload(linkID, sourceID, targetID, linkTypeID uuid.UUID) *app.UpdateWorkItemLinkPayload {
	lt := link.WorkItemLink{
		ID:         linkID,
		SourceID:   sourceID,
		TargetID:   targetID,
		LinkTypeID: linkTypeID,
	}
	payload := ConvertLinkFromModel(&http.Request{Host: "api.service.domain.org"}, lt)
	// The create payload is required during creation. Simply copy data over.
	return &app.UpdateWorkItemLinkPayload{
		Data: payload.Data,
	}
}

func (s *workItemLinkSuite) SecuredController(identity account.Identity) (*goa.Service, *WorkItemLinkController) {
	svc := testsupport.ServiceAsUser("WorkItemLink-Service", identity)
	return svc, NewWorkItemLinkController(svc, gormapplication.NewGormDB(s.DB), s.Configuration)
}

func (s *workItemLinkSuite) TestCreate() {
	resetFn := s.DisableGormCallbacks()
	defer resetFn()
	s.T().Run(http.StatusText(http.StatusOK), func(t *testing.T) {
		// helper function used in all ok-cases
		createOK := func(t *testing.T, fxt *tf.TestFixture, svc *goa.Service, ctrl *WorkItemLinkController) {
			// when
			createPayload := newCreateWorkItemLinkPayload(fxt.WorkItems[0].ID, fxt.WorkItems[1].ID, fxt.WorkItemLinkTypes[0].ID)
			res, workItemLink := test.CreateWorkItemLinkCreated(t, svc.Context, svc, ctrl, createPayload)
			// then
			require.NotNil(t, workItemLink)

			// ensure some relations are included in the response
			expectedIDs := map[uuid.UUID]struct{}{
				fxt.WorkItemLinkCategories[0].ID: {},
				fxt.Spaces[0].ID:                 {},
				fxt.WorkItemLinkTypes[0].ID:      {},
				fxt.WorkItems[0].ID:              {},
				fxt.WorkItems[1].ID:              {},
			}
			for _, obj := range workItemLink.Included {
				var id uuid.UUID
				switch v := obj.(type) {
				case *app.WorkItemLinkCategoryData:
					id = *v.ID
				case *app.Space:
					id = *v.ID
				case *app.WorkItemLinkTypeData:
					id = *v.ID
				case *app.WorkItem:
					id = *v.ID
				default:
					t.Errorf("object of unknown type included in work item link list response: %T", obj)
				}
				_, ok := expectedIDs[id]
				if ok {
					delete(expectedIDs, id)
				}
			}
			require.Empty(t, 0, expectedIDs, "these elements where missing from the included objects: %+v", expectedIDs)

			compareWithGoldenUUIDAgnostic(t, filepath.Join(s.testDir, "create", "ok.golden.json"), workItemLink)
			res.Header().Set("Etag", "0icd7ov5CqwDXN6Fx9z18g==") // overwrite Etag to always match
			compareWithGoldenUUIDAgnostic(t, filepath.Join(s.testDir, "create", "ok.headers.golden.json"), res)
		}

		t.Run("as space owner", func(t *testing.T) {
			fxt := tf.NewTestFixture(t, s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(4), tf.WorkItemLinkTypes(1))
			svc, ctrl := s.SecuredController(*fxt.Identities[0])
			createOK(t, fxt, svc, ctrl)
		})
		t.Run("as space collaborator", func(t *testing.T) {
			fxt := tf.NewTestFixture(t, s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(4), tf.WorkItemLinkTypes(1), tf.Identities(2, tf.SetIdentityUsernames("owner", "collaborator")))
			svc := testsupport.ServiceAsSpaceUser("TestWorkItem-Service", *fxt.IdentityByUsername("collaborator"), &TestSpaceAuthzService{*fxt.IdentityByUsername("collaborator"), ""})
			ctrl := NewWorkItemLinkController(svc, gormapplication.NewGormDB(s.DB), s.Configuration)
			createOK(t, fxt, svc, ctrl)
		})
		t.Run("as non-owner and non-collaborator", func(t *testing.T) {
			fxt := tf.NewTestFixture(t, s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(4), tf.WorkItemLinkTypes(1), tf.Identities(2, tf.SetIdentityUsernames("alice", "bob")))
			svc := testsupport.ServiceAsUser("TestWorkItem-Service", *fxt.IdentityByUsername("bob"))
			ctrl := NewWorkItemLinkController(svc, gormapplication.NewGormDB(s.DB), s.Configuration)
			createOK(t, fxt, svc, ctrl)
		})
	})
	s.T().Run(http.StatusText(http.StatusUnauthorized), func(t *testing.T) {
		t.Run("as not logged in user", func(t *testing.T) {
			// given
			fxt := tf.NewTestFixture(t, s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(4), tf.WorkItemLinkTypes(1))
			svc := goa.New("TestUnauthorizedCreateWorkItemLink-Service")
			ctrl := NewWorkItemLinkController(svc, gormapplication.NewGormDB(s.DB), s.Configuration)
			// when/then
			createPayload := newCreateWorkItemLinkPayload(fxt.WorkItems[0].ID, fxt.WorkItems[1].ID, fxt.WorkItemLinkTypes[0].ID)
			_, _ = test.CreateWorkItemLinkUnauthorized(t, svc.Context, svc, ctrl, createPayload)
		})
	})
	s.T().Run(http.StatusText(http.StatusConflict), func(t *testing.T) {
		t.Run("regression test for issue #586", func(t *testing.T) {
			// given
			fxt := tf.NewTestFixture(t, s.DB,
				tf.CreateWorkItemEnvironment(),
				tf.WorkItemLinks(2),
				tf.WorkItemLinkTypes(1, tf.SetTopologies(link.TopologyNetwork)))
			svc, ctrl := s.SecuredController(*fxt.Identities[0])
			// then the same link cannot be created again
			createPayload := newCreateWorkItemLinkPayload(fxt.WorkItemLinks[0].SourceID, fxt.WorkItemLinks[0].TargetID, fxt.WorkItemLinks[0].LinkTypeID)
			_, _ = test.CreateWorkItemLinkConflict(t, svc.Context, svc, ctrl, createPayload)
		})
	})
	s.T().Run(http.StatusText(http.StatusBadRequest), func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(2), tf.WorkItemLinkTypes(1))
		svc, ctrl := s.SecuredController(*fxt.Identities[0])
		t.Run("invalid type id", func(t *testing.T) {
			createPayload := newCreateWorkItemLinkPayload(fxt.WorkItems[0].ID, fxt.WorkItems[1].ID, uuid.Nil)
			_, _ = test.CreateWorkItemLinkBadRequest(t, svc.Context, svc, ctrl, createPayload)
		})
	})

	s.T().Run(http.StatusText(http.StatusNotFound), func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(2), tf.WorkItemLinkTypes(1))
		svc, ctrl := s.SecuredController(*fxt.Identities[0])
		t.Run("not existing link type id", func(t *testing.T) {
			createPayload := newCreateWorkItemLinkPayload(fxt.WorkItems[0].ID, fxt.WorkItems[1].ID, uuid.NewV4())
			// when then
			_, _ = test.CreateWorkItemLinkNotFound(t, svc.Context, svc, ctrl, createPayload)
		})
		t.Run("not existing source", func(t *testing.T) {
			createPayload := newCreateWorkItemLinkPayload(uuid.NewV4(), fxt.WorkItems[1].ID, fxt.WorkItemLinkTypes[0].ID)
			// when then
			_, _ = test.CreateWorkItemLinkNotFound(t, svc.Context, svc, ctrl, createPayload)
		})
		t.Run("not existing target", func(t *testing.T) {
			createPayload := newCreateWorkItemLinkPayload(fxt.WorkItems[0].ID, uuid.NewV4(), fxt.WorkItemLinkTypes[0].ID)
			// when then
			_, _ = test.CreateWorkItemLinkNotFound(t, svc.Context, svc, ctrl, createPayload)
		})
	})

}

func (s *workItemLinkSuite) TestDelete() {
	s.T().Run(http.StatusText(http.StatusOK), func(t *testing.T) {
		t.Run("as space owner", func(t *testing.T) {
			// given
			fxt := tf.NewTestFixture(t, s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItemLinks(1))
			svc, ctrl := s.SecuredController(*fxt.Identities[0])
			// when
			_ = test.DeleteWorkItemLinkOK(t, svc.Context, svc, ctrl, fxt.WorkItemLinks[0].ID)
			// then verify that the link really was deleted
			_, _ = test.ShowWorkItemLinkNotFound(t, svc.Context, svc, ctrl, fxt.WorkItemLinks[0].ID, nil, nil)
		})
		t.Run("as space collaborator", func(t *testing.T) {
			fxt := tf.NewTestFixture(t, s.DB, tf.WorkItemLinks(1), tf.Identities(2, tf.SetIdentityUsernames("owner", "collaborator")))
			svc := testsupport.ServiceAsSpaceUser("TestWorkItem-Service", *fxt.IdentityByUsername("collaborator"), &TestSpaceAuthzService{*fxt.IdentityByUsername("collaborator"), ""})
			ctrl := NewWorkItemLinkController(svc, gormapplication.NewGormDB(s.DB), s.Configuration)
			// when
			test.DeleteWorkItemLinkOK(s.T(), svc.Context, svc, ctrl, fxt.WorkItemLinks[0].ID)
			// then verify that the link really was deleted
			_, _ = test.ShowWorkItemLinkNotFound(t, svc.Context, svc, ctrl, fxt.WorkItemLinks[0].ID, nil, nil)
		})
	})
	s.T().Run(http.StatusText(http.StatusForbidden), func(t *testing.T) {
		t.Run("not as space collaborator", func(t *testing.T) {
			// given
			fxt := tf.NewTestFixture(t, s.DB, tf.WorkItemLinks(1), tf.Identities(2, tf.SetIdentityUsernames("owner", "collaborator")))
			svc := testsupport.ServiceAsSpaceUser("svc", *fxt.IdentityByUsername("collaborator"), &TestSpaceAuthzService{*fxt.IdentityByUsername("owner"), ""})
			ctrl := NewWorkItemLinkController(svc, gormapplication.NewGormDB(s.DB), s.Configuration)
			// when
			test.DeleteWorkItemLinkForbidden(t, svc.Context, svc, ctrl, fxt.WorkItemLinks[0].ID)
			// then verify the link still exists
			_, _ = test.ShowWorkItemLinkOK(t, svc.Context, svc, ctrl, fxt.WorkItemLinks[0].ID, nil, nil)
		})
	})
	s.T().Run(http.StatusText(http.StatusUnauthorized), func(t *testing.T) {
		t.Run("as not logged in user", func(t *testing.T) {
			// given
			fxt := tf.NewTestFixture(t, s.DB, tf.WorkItemLinks(1))
			svc := goa.New("TestUnauthorizedDeleteWorkItemLink-Service")
			ctrl := NewWorkItemLinkController(svc, gormapplication.NewGormDB(s.DB), s.Configuration)
			// when/then
			_, _ = test.DeleteWorkItemLinkUnauthorized(t, svc.Context, svc, ctrl, fxt.WorkItemLinks[0].ID)
		})
	})
	s.T().Run(http.StatusText(http.StatusNotFound), func(t *testing.T) {
		// given
		fxt := tf.NewTestFixture(t, s.DB, tf.CreateWorkItemEnvironment())
		svc, ctrl := s.SecuredController(*fxt.Identities[0])
		// when/then
		_, _ = test.DeleteWorkItemLinkNotFound(t, svc.Context, svc, ctrl, uuid.NewV4())
	})
}

func (s *workItemLinkSuite) TestShow() {
	s.T().Run(http.StatusText(http.StatusOK), func(t *testing.T) {
		t.Run("normal", func(t *testing.T) {
			resetFn := s.DisableGormCallbacks()
			defer resetFn()

			// given
			fxt := tf.NewTestFixture(t, s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItemLinks(1))
			svc, ctrl := s.SecuredController(*fxt.Identities[0])
			// when
			res, l := test.ShowWorkItemLinkOK(t, svc.Context, svc, ctrl, fxt.WorkItemLinks[0].ID, nil, nil)
			// then
			assertResponseHeaders(t, res)
			actual, err := ConvertLinkToModel(*l)
			require.NoError(t, err)
			require.Equal(t, fxt.WorkItemLinks[0].SourceID, actual.SourceID)
			require.Equal(t, fxt.WorkItemLinks[0].TargetID, actual.TargetID)
			require.Equal(t, fxt.WorkItemLinks[0].LinkTypeID, actual.LinkTypeID)
			compareWithGoldenUUIDAgnostic(t, filepath.Join(s.testDir, "show", "ok.golden.json"), l)
			res.Header().Set("Etag", "0icd7ov5CqwDXN6Fx9z18g==") // overwrite Etag to always match
			compareWithGoldenUUIDAgnostic(t, filepath.Join(s.testDir, "show", "ok.headers.golden.json"), res)
		})
		t.Run("using expired IfModifiedSince header", func(t *testing.T) {
			// given
			fxt := tf.NewTestFixture(t, s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItemLinks(1))
			svc, ctrl := s.SecuredController(*fxt.Identities[0])
			// when
			ifModifiedSince := app.ToHTTPTime(fxt.WorkItemLinks[0].UpdatedAt.Add(-1 * time.Hour))
			res, l := test.ShowWorkItemLinkOK(t, svc.Context, svc, ctrl, fxt.WorkItemLinks[0].ID, &ifModifiedSince, nil)
			// then
			assertResponseHeaders(t, res)
			actual, err := ConvertLinkToModel(*l)
			require.NoError(t, err)
			require.Equal(t, fxt.WorkItemLinks[0].SourceID, actual.SourceID)
			require.Equal(t, fxt.WorkItemLinks[0].TargetID, actual.TargetID)
			require.Equal(t, fxt.WorkItemLinks[0].LinkTypeID, actual.LinkTypeID)
		})
		t.Run("using expired IfNoneMatch header", func(t *testing.T) {
			// given
			fxt := tf.NewTestFixture(t, s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItemLinks(1))
			svc, ctrl := s.SecuredController(*fxt.Identities[0])
			// when
			ifNoneMatch := "foo"
			res, l := test.ShowWorkItemLinkOK(t, svc.Context, svc, ctrl, fxt.WorkItemLinks[0].ID, nil, &ifNoneMatch)
			// then
			assertResponseHeaders(t, res)
			actual, err := ConvertLinkToModel(*l)
			require.NoError(t, err)
			require.Equal(t, fxt.WorkItemLinks[0].SourceID, actual.SourceID)
			require.Equal(t, fxt.WorkItemLinks[0].TargetID, actual.TargetID)
			require.Equal(t, fxt.WorkItemLinks[0].LinkTypeID, actual.LinkTypeID)
		})
	})
	s.T().Run(http.StatusText(http.StatusNotModified), func(t *testing.T) {
		t.Run("using IfModifiedSince header", func(t *testing.T) {
			// given
			fxt := tf.NewTestFixture(t, s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItemLinks(1))
			svc, ctrl := s.SecuredController(*fxt.Identities[0])
			// when
			ifModifiedSince := app.ToHTTPTime(fxt.WorkItemLinks[0].UpdatedAt)
			res := test.ShowWorkItemLinkNotModified(t, svc.Context, svc, ctrl, fxt.WorkItemLinks[0].ID, &ifModifiedSince, nil)
			// then
			assertResponseHeaders(t, res)
		})
		t.Run("using IfNoneMatch header", func(t *testing.T) {
			// given
			fxt := tf.NewTestFixture(t, s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItemLinks(1))
			svc, ctrl := s.SecuredController(*fxt.Identities[0])
			// when
			ifNoneMatch := app.GenerateEntityTag(*fxt.WorkItemLinks[0])
			res := test.ShowWorkItemLinkNotModified(t, svc.Context, svc, ctrl, fxt.WorkItemLinks[0].ID, nil, &ifNoneMatch)
			// then
			assertResponseHeaders(t, res)
		})
	})
	s.T().Run(http.StatusText(http.StatusNotFound), func(t *testing.T) {
		t.Run("not existing link", func(t *testing.T) {
			// given
			fxt := tf.NewTestFixture(t, s.DB, tf.CreateWorkItemEnvironment())
			svc, ctrl := s.SecuredController(*fxt.Identities[0])
			// when
			_, jerrs := test.ShowWorkItemLinkNotFound(t, svc.Context, svc, ctrl, uuid.NewV4(), nil, nil)
			ignoreMe := "IGNOREME"
			jerrs.Errors[0].ID = &ignoreMe
			compareWithGoldenUUIDAgnostic(t, filepath.Join(s.testDir, "show", "not_found.errors.golden.json"), jerrs)
		})
	})
}

func (s *workItemLinkSuite) TestList() {
	resetFn := s.DisableGormCallbacks()
	defer resetFn()
	// given
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItemLinks(1))
	svc, _ := s.SecuredController(*fxt.Identities[0])
	relCtrl := NewWorkItemRelationshipsLinksController(svc, gormapplication.NewGormDB(s.DB), s.Configuration)

	s.T().Run(http.StatusText(http.StatusOK), func(t *testing.T) {
		t.Run("for /api/workitems/:id/relationships/links", func(t *testing.T) {
			// when
			_, links := test.ListWorkItemRelationshipsLinksOK(t, svc.Context, svc, relCtrl, fxt.WorkItemLinks[0].SourceID, nil, nil)
			for i, obj := range links.Included {
				switch t := obj.(type) {
				case *app.WorkItem:
					t.Attributes[workitem.SystemNumber] = i
					t.Attributes[workitem.SystemOrder] = float64(i)
				}
			}
			// then
			compareWithGoldenUUIDAgnostic(t, filepath.Join(s.testDir, "list", "ok.golden.json"), links)
		})
	})
	s.T().Run(http.StatusText(http.StatusNotFound), func(t *testing.T) {
		t.Run("for /api/workitems/:id/relationships/links", func(t *testing.T) {
			// when
			_, _ = test.ListWorkItemRelationshipsLinksNotFound(t, svc.Context, svc, relCtrl, uuid.NewV4(), nil, nil)
		})
	})
}

func (s *workItemLinkSuite) getWorkItemLinkTestDataFunc() func(t *testing.T) []testSecureAPI {
	return func(t *testing.T) []testSecureAPI {
		privatekey := testtoken.PrivateKey()
		differentPrivatekey, err := jwt.ParseRSAPrivateKeyFromPEM(([]byte(RSADifferentPrivateKeyTest)))
		require.NoError(t, err, "Could not parse private key")
		createWorkItemLinkPayloadString := bytes.NewBuffer([]byte(`
		{
			"data": {
				"attributes": {
					"version": 0
				},
				"id": "40bbdd3d-8b5d-4fd6-ac90-7236b669af04",
				"relationships": {
					"link_type": {
						"data": {
						"id": "6c5610be-30b2-4880-9fec-81e4f8e4fd76",
						"type": "workitemlinktypes"
						}
					},
					"source": {
						"data": {
						"id": "1234",
						"type": "workitems"
						}
					},
					"target": {
						"data": {
						"id": "1234",
						"type": "workitems"
						}
					}
				},
				"type": "workitemlinks"
			}
		}
  		`))

		testWorkItemLinksAPI := []testSecureAPI{
			// Create Work Item API with different parameters
			{
				method:             http.MethodPost,
				url:                endpointWorkItemLinks,
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            createWorkItemLinkPayloadString,
				jwtToken:           getExpiredAuthHeader(t, privatekey),
			}, {
				method:             http.MethodPost,
				url:                endpointWorkItemLinks,
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            createWorkItemLinkPayloadString,
				jwtToken:           getMalformedAuthHeader(t, privatekey),
			}, {
				method:             http.MethodPost,
				url:                endpointWorkItemLinks,
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            createWorkItemLinkPayloadString,
				jwtToken:           getValidAuthHeader(t, differentPrivatekey),
			}, {
				method:             http.MethodPost,
				url:                endpointWorkItemLinks,
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            createWorkItemLinkPayloadString,
				jwtToken:           "",
			},
			// Delete Work Item API with different parameters
			{
				method:             http.MethodDelete,
				url:                endpointWorkItemLinks + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            nil,
				jwtToken:           getExpiredAuthHeader(t, privatekey),
			}, {
				method:             http.MethodDelete,
				url:                endpointWorkItemLinks + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            nil,
				jwtToken:           getMalformedAuthHeader(t, privatekey),
			}, {
				method:             http.MethodDelete,
				url:                endpointWorkItemLinks + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            nil,
				jwtToken:           getValidAuthHeader(t, differentPrivatekey),
			}, {
				method:             http.MethodDelete,
				url:                endpointWorkItemLinks + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            nil,
				jwtToken:           "",
			},
			// Try fetching a random work item link
			// We do not have security on GET hence this should return 404 not found
			{
				method:             http.MethodGet,
				url:                endpointWorkItemLinks + "/fc591f38-a805-4abd-bfce-2460e49d8cc4",
				expectedStatusCode: http.StatusNotFound,
				expectedErrorCode:  jsonapi.ErrorCodeNotFound,
				payload:            nil,
				jwtToken:           "",
			},
		}
		return testWorkItemLinksAPI
	}
}

// This test case will check authorized access to Create/Update/Delete APIs
func (s *workItemLinkSuite) TestUnauthorizeWorkItemLinkCUD() {
	UnauthorizeCreateUpdateDeleteTest(s.T(), s.getWorkItemLinkTestDataFunc(), func() *goa.Service {
		return goa.New("TestUnauthorizedCreateWorkItemLink-Service")
	}, func(service *goa.Service) error {
		controller := NewWorkItemLinkController(service, gormapplication.NewGormDB(s.DB), s.Configuration)
		app.MountWorkItemLinkController(service, controller)
		return nil
	})
}

// The work item ID will be used to construct /api/workitems/:id/relationships/links endpoints
func (s *workItemLinkSuite) getWorkItemRelationshipLinksTestData() func(t *testing.T) []testSecureAPI {
	return func(t *testing.T) []testSecureAPI {
		testWorkItemLinksAPI := []testSecureAPI{
			// Get links for non existing work item
			{
				method:             http.MethodGet,
				url:                fmt.Sprintf(endpointWorkItemRelationshipsLinks, "7c73067d-be4f-4e7a-bf1d-644dabb90a5c"),
				expectedStatusCode: http.StatusNotFound,
				expectedErrorCode:  jsonapi.ErrorCodeNotFound,
				payload:            nil,
				jwtToken:           "",
			},
		}
		return testWorkItemLinksAPI
	}
}

func (s *workItemLinkSuite) TestUnauthorizeWorkItemRelationshipsLinksCUD() {
	UnauthorizeCreateUpdateDeleteTest(s.T(), s.getWorkItemRelationshipLinksTestData(), func() *goa.Service {
		return goa.New("TestUnauthorizedCreateWorkItemRelationshipsLinks-Service")
	}, func(service *goa.Service) error {
		controller := NewWorkItemRelationshipsLinksController(service, gormapplication.NewGormDB(s.DB), s.Configuration)
		app.MountWorkItemRelationshipsLinksController(service, controller)
		return nil
	})
}
