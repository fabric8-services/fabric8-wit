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

	"github.com/goadesign/goa"
	"github.com/stretchr/testify/assert"
)

func TestPagingLinks(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	svc := goa.New("TestPaginLinks-Service")
	assert.NotNil(t, svc)
	db := testsupport.NewMockDB()
	controller := NewWorkitemController(svc, db)

	repo := db.WorkItems().(*testsupport.WorkItemRepository)
	pagingTest := createPagingTest(t, svc.Context, controller, repo, space.SystemSpace.String(), 13)
	pagingTest(2, 5, "page[offset]=0&page[limit]=2", "page[offset]=12&page[limit]=5", "page[offset]=0&page[limit]=2", "page[offset]=7&page[limit]=5")
	pagingTest(10, 3, "page[offset]=0&page[limit]=1", "page[offset]=10&page[limit]=3", "page[offset]=7&page[limit]=3", "")
	pagingTest(0, 4, "page[offset]=0&page[limit]=4", "page[offset]=12&page[limit]=4", "", "page[offset]=4&page[limit]=4")
	pagingTest(4, 8, "page[offset]=0&page[limit]=4", "page[offset]=12&page[limit]=8", "page[offset]=0&page[limit]=4", "page[offset]=12&page[limit]=8")

	pagingTest(16, 14, "page[offset]=0&page[limit]=2", "page[offset]=2&page[limit]=14", "page[offset]=2&page[limit]=14", "")
	pagingTest(16, 18, "page[offset]=0&page[limit]=16", "page[offset]=0&page[limit]=16", "page[offset]=0&page[limit]=16", "")

	pagingTest(3, 50, "page[offset]=0&page[limit]=3", "page[offset]=3&page[limit]=50", "page[offset]=0&page[limit]=3", "")
	pagingTest(0, 50, "page[offset]=0&page[limit]=50", "page[offset]=0&page[limit]=50", "", "")

	pagingTest = createPagingTest(t, svc.Context, controller, repo, space.SystemSpace.String(), 0)
	pagingTest(2, 5, "page[offset]=0&page[limit]=2", "page[offset]=0&page[limit]=2", "", "")
}

func TestPagingErrors(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	svc := goa.New("TestPaginErrors-Service")
	db := testsupport.NewMockDB()
	controller := NewWorkitemController(svc, db)
	repo := db.WorkItems().(*testsupport.WorkItemRepository)
	repo.ListReturns(makeWorkItems(100), uint64(100), nil)

	var offset string = "-1"
	var limit int = 2
	_, result := test.ListWorkitemOK(t, context.Background(), nil, controller, space.SystemSpace.String(), nil, nil, nil, nil, nil, nil, &limit, &offset)
	if !strings.Contains(*result.Links.First, "page[offset]=0") {
		assert.Fail(t, "Offset is negative", "Expected offset to be %d, but was %s", 0, *result.Links.First)
	}

	offset = "0"
	limit = 0
	_, result = test.ListWorkitemOK(t, context.Background(), nil, controller, space.SystemSpace.String(), nil, nil, nil, nil, nil, nil, &limit, &offset)
	if !strings.Contains(*result.Links.First, "page[limit]=20") {
		assert.Fail(t, "Limit is 0", "Expected limit to be default size %d, but was %s", 20, *result.Links.First)
	}

	offset = "0"
	limit = -1
	_, result = test.ListWorkitemOK(t, context.Background(), nil, controller, space.SystemSpace.String(), nil, nil, nil, nil, nil, nil, &limit, &offset)
	if !strings.Contains(*result.Links.First, "page[limit]=20") {
		assert.Fail(t, "Limit is negative", "Expected limit to be default size %d, but was %s", 20, *result.Links.First)
	}

	offset = "-3"
	limit = -1
	_, result = test.ListWorkitemOK(t, context.Background(), nil, controller, space.SystemSpace.String(), nil, nil, nil, nil, nil, nil, &limit, &offset)
	if !strings.Contains(*result.Links.First, "page[limit]=20") {
		assert.Fail(t, "Limit is negative", "Expected limit to be default size %d, but was %s", 20, *result.Links.First)
	}
	if !strings.Contains(*result.Links.First, "page[offset]=0") {
		assert.Fail(t, "Offset is negative", "Expected offset to be %d, but was %s", 0, *result.Links.First)
	}

	offset = "ALPHA"
	limit = 40
	_, result = test.ListWorkitemOK(t, context.Background(), nil, controller, space.SystemSpace.String(), nil, nil, nil, nil, nil, nil, &limit, &offset)
	if !strings.Contains(*result.Links.First, "page[limit]=40") {
		assert.Fail(t, "Limit is within range", "Expected limit to be size %d, but was %s", 40, *result.Links.First)
	}
	if !strings.Contains(*result.Links.First, "page[offset]=0") {
		assert.Fail(t, "Offset is negative", "Expected offset to be %d, but was %s", 0, *result.Links.First)
	}
}

func TestPagingLinksHasAbsoluteURL(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	svc := goa.New("TestPaginAbsoluteURL-Service")
	db := testsupport.NewMockDB()
	controller := NewWorkitemController(svc, db)

	offset := "10"
	limit := 10

	repo := db.WorkItems().(*testsupport.WorkItemRepository)
	repo.ListReturns(makeWorkItems(10), uint64(100), nil)

	_, result := test.ListWorkitemOK(t, context.Background(), nil, controller, space.SystemSpace.String(), nil, nil, nil, nil, nil, nil, &limit, &offset)
	if !strings.HasPrefix(*result.Links.First, "http://") {
		assert.Fail(t, "Not Absolute URL", "Expected link %s to contain absolute URL but was %s", "First", *result.Links.First)
	}
	if !strings.HasPrefix(*result.Links.Last, "http://") {
		assert.Fail(t, "Not Absolute URL", "Expected link %s to contain absolute URL but was %s", "Last", *result.Links.Last)
	}
	if !strings.HasPrefix(*result.Links.Prev, "http://") {
		assert.Fail(t, "Not Absolute URL", "Expected link %s to contain absolute URL but was %s", "Prev", *result.Links.Prev)
	}
	if !strings.HasPrefix(*result.Links.Next, "http://") {
		assert.Fail(t, "Not Absolute URL", "Expected link %s to contain absolute URL but was %s", "Next", *result.Links.Next)
	}
}

func TestPagingDefaultAndMaxSize(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	svc := goa.New("TestPaginSize-Service")
	db := testsupport.NewMockDB()
	controller := NewWorkitemController(svc, db)

	offset := "0"
	var limit int
	repo := db.WorkItems().(*testsupport.WorkItemRepository)
	repo.ListReturns(makeWorkItems(10), uint64(100), nil)

	_, result := test.ListWorkitemOK(t, context.Background(), nil, controller, space.SystemSpace.String(), nil, nil, nil, nil, nil, nil, nil, &offset)
	if !strings.Contains(*result.Links.First, "page[limit]=20") {
		assert.Fail(t, "Limit is nil", "Expected limit to be default size %d, got %v", 20, *result.Links.First)
	}
	limit = 1000
	_, result = test.ListWorkitemOK(t, context.Background(), nil, controller, space.SystemSpace.String(), nil, nil, nil, nil, nil, nil, &limit, &offset)
	if !strings.Contains(*result.Links.First, "page[limit]=100") {
		assert.Fail(t, "Limit is more than max", "Expected limit to be %d, got %v", 100, *result.Links.First)
	}

	limit = 50
	_, result = test.ListWorkitemOK(t, context.Background(), nil, controller, space.SystemSpace.String(), nil, nil, nil, nil, nil, nil, &limit, &offset)
	if !strings.Contains(*result.Links.First, "page[limit]=50") {
		assert.Fail(t, "Limit is within range", "Expected limit to be %d, got %v", 50, *result.Links.First)
	}
}
