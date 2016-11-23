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

	_, ok := cache.Get("notexists")
	assert.False(t, ok)
}

func TestReadingWriting(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	_, ok := cache.Get("type1")
	assert.False(t, ok)

	wit := models.WorkItemType{}
	wit.Name = "type1"
	cache.Put(wit)

	cachedWit, ok := cache.Get("type1")
	assert.True(t, ok)
	assert.Equal(t, wit, cachedWit)
}

func TestConcurrentAccess(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	wit := models.WorkItemType{}
	wit.Name = "type1"

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
			cachedWit, ok := cache.Get("type1")
			assert.True(t, ok)
			assert.Equal(t, wit, cachedWit)
		}
	}()
	wg.Wait()
}
