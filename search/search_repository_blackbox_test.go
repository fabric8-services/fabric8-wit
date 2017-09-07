package search_test

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/migration"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/search"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestRunSearchRepositoryBlackboxTest(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &searchRepositoryBlackboxTest{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

type searchRepositoryBlackboxTest struct {
	gormtestsupport.DBTestSuite
	clean      func()
	searchRepo *search.GormSearchRepository
}

// SetupSuite overrides the DBTestSuite's function but calls it before doing anything else
func (s *searchRepositoryBlackboxTest) SetupSuite() {
	s.DBTestSuite.SetupSuite()
	ctx := migration.NewMigrationContext(context.Background())
	s.DBTestSuite.PopulateDBTestSuite(ctx)
}

func (s *searchRepositoryBlackboxTest) SetupTest() {
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
	s.searchRepo = search.NewGormSearchRepository(s.DB)
}

func (s *searchRepositoryBlackboxTest) TearDownTest() {
	s.clean()
}

func (s *searchRepositoryBlackboxTest) getTestFixture() *tf.TestFixture {
	return tf.NewTestFixture(s.T(), s.DB,
		tf.WorkItemTypes(3, func(fxt *tf.TestFixture, idx int) error {
			wit := fxt.WorkItemTypes[idx]
			wit.ID = uuid.NewV4()
			switch idx {
			case 0:
				wit.Name = "base"
				wit.Path = workitem.LtreeSafeID(wit.ID)
			case 1:
				wit.Name = "sub1"
				wit.Path = fxt.WorkItemTypes[0].Path + workitem.GetTypePathSeparator() + workitem.LtreeSafeID(wit.ID)
			case 2:
				wit.Name = "sub2"
				wit.Path = fxt.WorkItemTypes[0].Path + workitem.GetTypePathSeparator() + workitem.LtreeSafeID(wit.ID)
			}
			return nil
		}),
		tf.WorkItems(2, func(fxt *tf.TestFixture, idx int) error {
			wi := fxt.WorkItems[idx]
			switch idx {
			case 0:
				wi.Type = fxt.WorkItemTypes[1].ID
				wi.Fields[workitem.SystemTitle] = "Test TestRestrictByType"
				wi.Fields[workitem.SystemState] = "closed"
			case 1:
				wi.Type = fxt.WorkItemTypes[2].ID
				wi.Fields[workitem.SystemTitle] = "Test TestRestrictByType 2"
				wi.Fields[workitem.SystemState] = "closed"
			}
			return nil
		}),
	)
}

func (s *searchRepositoryBlackboxTest) TestRestrictByType() {
	// given
	req := &http.Request{Host: "localhost"}
	params := url.Values{}
	ctx := goa.NewContext(context.Background(), nil, req, params)

	res, count, err := s.searchRepo.SearchFullText(ctx, "TestRestrictByType", nil, nil, nil)
	require.Nil(s.T(), err)
	require.True(s.T(), count == uint64(len(res))) // safety check for many, many instances of bogus search results.

	// when
	testFxt := s.getTestFixture()

	base := testFxt.WorkItemTypes[0]
	sub1 := testFxt.WorkItemTypes[1]
	sub2 := testFxt.WorkItemTypes[2]
	wi1 := testFxt.WorkItems[0]
	wi2 := testFxt.WorkItems[1]

	res, count, err = s.searchRepo.SearchFullText(ctx, "TestRestrictByType", nil, nil, nil)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), uint64(2), count)
	assert.Equal(s.T(), res[0].Fields["system.order"], wi2.Fields["system.order"])
	assert.Equal(s.T(), res[1].Fields["system.order"], wi1.Fields["system.order"])

	res, count, err = s.searchRepo.SearchFullText(ctx, "TestRestrictByType type:"+sub1.ID.String(), nil, nil, nil)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), uint64(1), count)
	if count == 1 {
		assert.Equal(s.T(), wi1.ID, res[0].ID)
		assert.Equal(s.T(), res[0].Fields["system.order"], wi1.Fields["system.order"])
	}

	res, count, err = s.searchRepo.SearchFullText(ctx, "TestRestrictByType type:"+sub2.ID.String(), nil, nil, nil)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), uint64(1), count)
	if count == 1 {
		assert.Equal(s.T(), wi2.ID, res[0].ID)
		assert.Equal(s.T(), res[0].Fields["system.order"], wi2.Fields["system.order"])
	}

	res, count, err = s.searchRepo.SearchFullText(ctx, "TestRestrictByType type:"+base.ID.String(), nil, nil, nil)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), uint64(2), count)
	assert.Equal(s.T(), res[0].Fields["system.order"], wi2.Fields["system.order"])
	assert.Equal(s.T(), res[1].Fields["system.order"], wi1.Fields["system.order"])

	res, count, err = s.searchRepo.SearchFullText(ctx, "TestRestrictByType type:"+sub2.ID.String()+" type:"+sub1.ID.String(), nil, nil, nil)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), uint64(2), count)
	assert.Equal(s.T(), res[0].Fields["system.order"], wi2.Fields["system.order"])
	assert.Equal(s.T(), res[1].Fields["system.order"], wi1.Fields["system.order"])

	res, count, err = s.searchRepo.SearchFullText(ctx, "TestRestrictByType type:"+base.ID.String()+" type:"+sub1.ID.String(), nil, nil, nil)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), uint64(2), count)
	assert.Equal(s.T(), res[0].Fields["system.order"], wi2.Fields["system.order"])
	assert.Equal(s.T(), res[1].Fields["system.order"], wi1.Fields["system.order"])

	_, count, err = s.searchRepo.SearchFullText(ctx, "TRBTgorxi type:"+base.ID.String(), nil, nil, nil)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), uint64(0), count)
}

func (s *searchRepositoryBlackboxTest) TestFilterCount() {
	// given
	req := &http.Request{Host: "localhost"}
	params := url.Values{}
	ctx := goa.NewContext(context.Background(), nil, req, params)
	notexistspace := "5f734617-472e-5dab-ab8d-e038345724b2"
	fs1 := fmt.Sprintf(`{"$AND": [{"space": "%s"}]}`, notexistspace)
	res, count, err := s.searchRepo.Filter(ctx, fs1, nil, nil, nil)
	require.Nil(s.T(), err)
	require.True(s.T(), count == uint64(len(res))) // safety check for many, many instances of bogus search results.

	// when
	testFxt := s.getTestFixture()

	// then
	fs2 := fmt.Sprintf(`{"$AND": [{"space": "%s"}]}`, testFxt.Spaces[0].ID)
	start := 3
	res, count, err = s.searchRepo.Filter(ctx, fs2, nil, &start, nil)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), uint64(2), count)
	assert.Equal(s.T(), 0, len(res))

	res, count, err = s.searchRepo.Filter(ctx, fs2, nil, nil, nil)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), uint64(2), count)
	assert.Equal(s.T(), 2, len(res))

}
