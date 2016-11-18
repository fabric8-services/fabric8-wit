package models

import (
	"encoding/json"

	"github.com/golang/groupcache"
)

// GetterFunc is a function to load a value if it's not available in the cache
type GetterFunc func(ctx groupcache.Context, key string) (interface{}, error)

// NewCache constructs Cache
func NewCache(name string, cacheBytes int64, fn GetterFunc) *Cache {
	cch := groupcache.NewGroup(name, cacheBytes, groupcache.GetterFunc(
		func(ctx groupcache.Context, key string, dest groupcache.Sink) error {
			strct, err := fn(ctx, key)
			if err != nil {
				return err
			}

			bytes, err := json.Marshal(strct)
			if err != nil {
				return err
			}

			dest.SetBytes(bytes)

			return nil
		}))

	return &Cache{cch}
}

// Cache represents a cache
type Cache struct {
	group *groupcache.Group
}

// Get loads a value from cache
func (cache *Cache) Get(ctx groupcache.Context, key string, into interface{}) error {
	var data []byte
	if err := cache.group.Get(ctx, key, groupcache.AllocatingByteSliceSink(&data)); err != nil {
		return err
	}

	if err := json.Unmarshal(data, &into); err != nil {
		return err
	}

	return nil
}
