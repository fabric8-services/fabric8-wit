package models

import (
	"log"

	"github.com/golang/groupcache"
	"golang.org/x/net/context"
)

const name = "wit"          // cache group name prefix
const cacheBytes = 32 << 10 // 32 KB max per-node memory usage

// WorkItemTypeCache implements a cache of WorkItemTypes
type WorkItemTypeCache struct {
	cache *Cache
}

// Get loads a WIT from the cache. If no WIT found in cache then loads it from DB
func (cache *WorkItemTypeCache) Get(ctx *WorkItemTypeCacheContext, key string, into interface{}) error {
	return cache.cache.Get(ctx, key, into)
}

// NewWorkItemTypeCache constructs WorkItemTypeCache
func NewWorkItemTypeCache() *WorkItemTypeCache {
	cch := NewCache(name, cacheBytes, GetterFunc(
		func(ctx groupcache.Context, key string) (interface{}, error) {
			log.Printf("Work item type %s not found in cache. Loading from DB", key)
			var c *WorkItemTypeCacheContext
			if ctx != nil {
				c = ctx.(*WorkItemTypeCacheContext)
			}
			return c.r.LoadTypeFromDB(c.ctx, key)
		}))

	return &WorkItemTypeCache{cch}
}

// NewWorkItemTypeCacheContext constructs WorkItemTypeCacheContext
func NewWorkItemTypeCacheContext(r *GormWorkItemTypeRepository, ctx context.Context) *WorkItemTypeCacheContext {
	return &WorkItemTypeCacheContext{r, ctx}
}

// WorkItemTypeCacheContext represents work item type context
type WorkItemTypeCacheContext struct {
	r   *GormWorkItemTypeRepository
	ctx context.Context
}
