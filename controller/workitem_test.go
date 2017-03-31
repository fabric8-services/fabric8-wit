package controller_test

import (
	"context"
	"strings"
	"testing"

	"github.com/almighty/almighty-core/app/test"
	. "github.com/almighty/almighty-core/controller"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/space"
	testsupport "github.com/almighty/almighty-core/test"

	"github.com/almighty/almighty-core/configuration"
	"github.com/goadesign/goa"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestSuitePaging(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	suite.Run(t, &TestPagingSuite{})
}

type TestPagingSuite struct {
	suite.Suite
	config     configuration.ConfigurationData
	controller *WorkitemController
	repo       *testsupport.WorkItemRepository
	svc        *goa.Service
}

func (s *TestPagingSuite) SetupSuite() {
	config, err := configuration.NewConfigurationData("../config.yaml")
	require.Nil(s.T(), err)
	s.config = *config
	s.svc = goa.New("TestPaginLinks-Service")
	assert.NotNil(s.T(), s.svc)
	db := testsupport.NewMockDB()
	s.controller = NewWorkitemController(s.svc, db, &s.config)
	s.repo = db.WorkItems().(*testsupport.WorkItemRepository)
}

func (s *TestPagingSuite) TestPagingLinks() {
	pagingTest := createPagingTest(s.T(), s.svc.Context, s.controller, s.repo, space.SystemSpace.String(), 13)
	pagingTest(2, 5, "page[offset]=0&page[limit]=2", "page[offset]=12&page[limit]=5", "page[offset]=0&page[limit]=2", "page[offset]=7&page[limit]=5")
	pagingTest(10, 3, "page[offset]=0&page[limit]=1", "page[offset]=10&page[limit]=3", "page[offset]=7&page[limit]=3", "")
	pagingTest(0, 4, "page[offset]=0&page[limit]=4", "page[offset]=12&page[limit]=4", "", "page[offset]=4&page[limit]=4")
	pagingTest(4, 8, "page[offset]=0&page[limit]=4", "page[offset]=12&page[limit]=8", "page[offset]=0&page[limit]=4", "page[offset]=12&page[limit]=8")

	pagingTest(16, 14, "page[offset]=0&page[limit]=2", "page[offset]=2&page[limit]=14", "page[offset]=2&page[limit]=14", "")
	pagingTest(16, 18, "page[offset]=0&page[limit]=16", "page[offset]=0&page[limit]=16", "page[offset]=0&page[limit]=16", "")

	pagingTest(3, 50, "page[offset]=0&page[limit]=3", "page[offset]=3&page[limit]=50", "page[offset]=0&page[limit]=3", "")
	pagingTest(0, 50, "page[offset]=0&page[limit]=50", "page[offset]=0&page[limit]=50", "", "")

	pagingTest = createPagingTest(s.T(), s.svc.Context, s.controller, s.repo, space.SystemSpace.String(), 0)
	pagingTest(2, 5, "page[offset]=0&page[limit]=2", "page[offset]=0&page[limit]=2", "", "")
}

func (s *TestPagingSuite) TestPagingErrors() {
	s.repo.ListReturns(makeWorkItems(100), uint64(100), nil)

	var offset string = "-1"
	var limit int = 2
	_, result := test.ListWorkitemOK(s.T(), context.Background(), nil, s.controller, space.SystemSpace.String(), nil, nil, nil, nil, nil, nil, &limit, &offset, nil, nil)
	if !strings.Contains(*result.Links.First, "page[offset]=0") {
		assert.Fail(s.T(), "Offset is negative", "Expected offset to be %d, but was %s", 0, *result.Links.First)
	}

	offset = "0"
	limit = 0
	_, result = test.ListWorkitemOK(s.T(), context.Background(), nil, s.controller, space.SystemSpace.String(), nil, nil, nil, nil, nil, nil, &limit, &offset, nil, nil)
	if !strings.Contains(*result.Links.First, "page[limit]=20") {
		assert.Fail(s.T(), "Limit is 0", "Expected limit to be default size %d, but was %s", 20, *result.Links.First)
	}

	offset = "0"
	limit = -1
	_, result = test.ListWorkitemOK(s.T(), context.Background(), nil, s.controller, space.SystemSpace.String(), nil, nil, nil, nil, nil, nil, &limit, &offset, nil, nil)
	if !strings.Contains(*result.Links.First, "page[limit]=20") {
		assert.Fail(s.T(), "Limit is negative", "Expected limit to be default size %d, but was %s", 20, *result.Links.First)
	}

	offset = "-3"
	limit = -1
	_, result = test.ListWorkitemOK(s.T(), context.Background(), nil, s.controller, space.SystemSpace.String(), nil, nil, nil, nil, nil, nil, &limit, &offset, nil, nil)
	if !strings.Contains(*result.Links.First, "page[limit]=20") {
		assert.Fail(s.T(), "Limit is negative", "Expected limit to be default size %d, but was %s", 20, *result.Links.First)
	}
	if !strings.Contains(*result.Links.First, "page[offset]=0") {
		assert.Fail(s.T(), "Offset is negative", "Expected offset to be %d, but was %s", 0, *result.Links.First)
	}

	offset = "ALPHA"
	limit = 40
	_, result = test.ListWorkitemOK(s.T(), context.Background(), nil, s.controller, space.SystemSpace.String(), nil, nil, nil, nil, nil, nil, &limit, &offset, nil, nil)
	if !strings.Contains(*result.Links.First, "page[limit]=40") {
		assert.Fail(s.T(), "Limit is within range", "Expected limit to be size %d, but was %s", 40, *result.Links.First)
	}
	if !strings.Contains(*result.Links.First, "page[offset]=0") {
		assert.Fail(s.T(), "Offset is negative", "Expected offset to be %d, but was %s", 0, *result.Links.First)
	}
}

func (s *TestPagingSuite) TestPagingLinksHasAbsoluteURL() {
	// given
	offset := "10"
	limit := 10
	s.repo.ListReturns(makeWorkItems(10), uint64(100), nil)
	// when
	_, result := test.ListWorkitemOK(s.T(), context.Background(), nil, s.controller, space.SystemSpace.String(), nil, nil, nil, nil, nil, nil, &limit, &offset, nil, nil)
	// then
	if !strings.HasPrefix(*result.Links.First, "http://") {
		assert.Fail(s.T(), "Not Absolute URL", "Expected link %s to contain absolute URL but was %s", "First", *result.Links.First)
	}
	if !strings.HasPrefix(*result.Links.Last, "http://") {
		assert.Fail(s.T(), "Not Absolute URL", "Expected link %s to contain absolute URL but was %s", "Last", *result.Links.Last)
	}
	if !strings.HasPrefix(*result.Links.Prev, "http://") {
		assert.Fail(s.T(), "Not Absolute URL", "Expected link %s to contain absolute URL but was %s", "Prev", *result.Links.Prev)
	}
	if !strings.HasPrefix(*result.Links.Next, "http://") {
		assert.Fail(s.T(), "Not Absolute URL", "Expected link %s to contain absolute URL but was %s", "Next", *result.Links.Next)
	}
}

func (s *TestPagingSuite) TestPagingDefaultAndMaxSize() {
	// given
	offset := "0"
	var limit int
	s.repo.ListReturns(makeWorkItems(10), uint64(100), nil)
	// when
	_, result := test.ListWorkitemOK(s.T(), context.Background(), nil, s.controller, space.SystemSpace.String(), nil, nil, nil, nil, nil, nil, nil, &offset, nil, nil)
	// then
	if !strings.Contains(*result.Links.First, "page[limit]=20") {
		assert.Fail(s.T(), "Limit is nil", "Expected limit to be default size %d, got %v", 20, *result.Links.First)
	}
	// when
	limit = 1000
	_, result = test.ListWorkitemOK(s.T(), context.Background(), nil, s.controller, space.SystemSpace.String(), nil, nil, nil, nil, nil, nil, &limit, &offset, nil, nil)
	// then
	if !strings.Contains(*result.Links.First, "page[limit]=100") {
		assert.Fail(s.T(), "Limit is more than max", "Expected limit to be %d, got %v", 100, *result.Links.First)
	}
	// when
	limit = 50
	_, result = test.ListWorkitemOK(s.T(), context.Background(), nil, s.controller, space.SystemSpace.String(), nil, nil, nil, nil, nil, nil, &limit, &offset, nil, nil)
	// then
	if !strings.Contains(*result.Links.First, "page[limit]=50") {
		assert.Fail(s.T(), "Limit is within range", "Expected limit to be %d, got %v", 50, *result.Links.First)
	}
}
