package controller

import "time"

const (
	// IfModifiedSince the "If-Modified-Since" HTTP request header name
	IfModifiedSince = "If-Modified-Since"
	// LastModified the "Last-Modified" HTTP response header name
	LastModified = "Last-Modified"
	// IfNoneMatch the "If-None-Match" HTTP request header name
	IfNoneMatch = "If-None-Match"
	// ETag the "ETag" HTTP response header name
	// should be ETag but GOA will convert it to "Etag" when setting the header.
	// Plus, RFC 2616 specifies that header names are case insensitive:
	// https://www.w3.org/Protocols/rfc2616/rfc2616-sec4.html#sec4.2
	ETag = "Etag"
	// CacheControl the "Cache-Control" HTTP response header name
	CacheControl = "Cache-Control"
	// MaxAge the "max-age" HTTP response header value
	MaxAge = "max-age"
)

// ConditionalRequestContext interface with methods for conditional requests contexts
type ConditionalRequestContext interface {
	GenericOK(entity interface{}) error
	NotModified() error
	ModifiedSince(lastModified time.Time) bool
	MatchesETag(eTag string) bool
	SetHeader(name, value string)
}

// ConditionalRequestEntity interface with methods for conditional response entity
type ConditionalResponseEntity interface {
	GetLastModified() time.Time
	GenerateETag() string
}

func conditional(ctx ConditionalRequestContext, entity ConditionalResponseEntity, maxAge string) error {
	// check the "If-Modified-Since header against the last update timestamp"
	// HTTP header does not include microseconds, so we need to ignore them in the "updated_at" record field.
	lastModified := entity.GetLastModified()
	if !ctx.ModifiedSince(lastModified) {
		return ctx.NotModified()
	}
	// check the ETag
	eTag := entity.GenerateETag()
	if ctx.MatchesETag(eTag) {
		return ctx.NotModified()
	}
	// return the work item type along with conditional query and caching headers
	ctx.SetHeader(LastModified, lastModified.String())
	ctx.SetHeader(ETag, eTag)
	ctx.SetHeader(CacheControl, MaxAge+"="+maxAge)
	return ctx.GenericOK(entity)
}
