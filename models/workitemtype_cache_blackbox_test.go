package models_test

import (
	"sync"
	"testing"

	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/resource"
	"github.com/stretchr/testify/assert"
)

var cache = models.NewWorkItemTypeCache()

func TestNotExistingType(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	_, ok := cache.Get("testNotExistingType")
	assert.False(t, ok)
}

func TestReadingWriting(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	_, ok := cache.Get("testReadingWriting")
	assert.False(t, ok)

	wit := models.WorkItemType{Name: "testReadingWriting"}
	cache.Put(wit)

	cachedWit, ok := cache.Get("testReadingWriting")
	assert.True(t, ok)
	assert.Equal(t, wit, cachedWit)
}

func TestClear(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	c := models.NewWorkItemTypeCache()
	c.Put(models.WorkItemType{Name: "testClear"})
	_, ok := c.Get("testClear")
	assert.True(t, ok)

	c.Clear()
	_, ok = c.Get("testClear")
	assert.False(t, ok)
}

func TestDelete(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	c := models.NewWorkItemTypeCache()
	c.Put(models.WorkItemType{Name: "toDelete"})
	c.Put(models.WorkItemType{Name: "toPersist"})
	_, ok := c.Get("toDelete")
	assert.True(t, ok)
	_, ok = c.Get("toPersist")
	assert.True(t, ok)

	c.Delete("toDelete")
	_, ok = c.Get("toDelete")
	assert.False(t, ok)
	_, ok = c.Get("toPersist")
	assert.True(t, ok)
}

func TestConcurrentAccess(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	wit := models.WorkItemType{Name: "testConcurrentAccess"}

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
