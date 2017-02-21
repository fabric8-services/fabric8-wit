package workitem_test

import (
	"sync"
	"testing"

	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/workitem"
	"github.com/stretchr/testify/assert"
)

var cache = workitem.NewWorkItemTypeCache()

func TestGetNotExistingTypeReturnsNotOk(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	_, ok := cache.Get("testNotExistingType")
	assert.False(t, ok)
}

func TestGetReturnsPreviouslyPutWIT(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	_, ok := cache.Get("testReadingWriting")
	assert.False(t, ok)

	wit := workitem.WorkItemType{Name: "testReadingWriting"}
	cache.Put(wit)

	cachedWit, ok := cache.Get("testReadingWriting")
	assert.True(t, ok)
	assert.Equal(t, wit, cachedWit)
}

func TestGetReturnNotOkAfterClear(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	c := workitem.NewWorkItemTypeCache()
	c.Put(workitem.WorkItemType{Name: "testClear"})
	_, ok := c.Get("testClear")
	assert.True(t, ok)

	c.Clear()
	_, ok = c.Get("testClear")
	assert.False(t, ok)
}

func TestNoFailuresWithConcurrentMapReadAndMapWrite(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	wit := workitem.WorkItemType{Name: "testConcurrentAccess"}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 3000; i++ {
			cache.Put(wit)
		}
	}()
	cache.Put(wit)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			cachedWit, ok := cache.Get("testConcurrentAccess")
			assert.True(t, ok)
			assert.Equal(t, wit, cachedWit)
		}
	}()
	wg.Wait()
}
