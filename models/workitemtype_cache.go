package models

import (
	"fmt"
	"log"

	"github.com/golang/groupcache"
	"golang.org/x/net/context"
)

const namePrefix = "wit"    // cache group name prefix
const cacheBytes = 32 << 10 // 32 KB max per-node memory usage

// WorkItemTypeCache implements a cache of WorkItemTypes
type WorkItemTypeCache struct {
	cache *Cache
}

// NewWorkItemTypeCache constructs WorkItemTypeCache
func NewWorkItemTypeCache(r *GormWorkItemTypeRepository) *WorkItemTypeCache {
	// One cache group per GormWorkItemTypeRepository instance
	name := namePrefix + fmt.Sprintf("%p", r)
	cch := NewCache(name, cacheBytes, GetterFunc(
		func(ctx groupcache.Context, key string) (interface{}, error) {
			log.Printf("Work item type %s not found in cache. Loading from DB", key)
			var c context.Context
			if ctx != nil {
				c = ctx.(context.Context)
			}
			return r.LoadTypeFromDB(c, key)
		}))

	return &WorkItemTypeCache{cch}
}

// Get loads a WIT from the cache. If no WIT found in cache then loads it from DB
func (cache *WorkItemTypeCache) Get(ctx context.Context, key string, into interface{}) error {
	return cache.cache.Get(ctx, key, into)
}
