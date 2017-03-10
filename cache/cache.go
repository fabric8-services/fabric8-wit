package cache

import (
	"time"
)

// CacheRequestDirectives the cache directives in the HTTP requests.
type CacheRequestDirectives struct {
	IfModifiedSince *time.Time
	ETag            *string
}
