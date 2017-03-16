package controller

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"strconv"
	"time"

	"github.com/almighty/almighty-core/app"
)

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
	OK(entity interface{}) error
	NotModified() error
	ModifiedSince(lastModified time.Time) bool
	MatchesETag(eTag string) bool
	SetHeader(name, value string)
}

type ConditionalRequestEntity interface {
	GetLastModified() time.Time
	GenerateETag() string
}

func conditional(ctx ConditionalRequestContext, entity ConditionalRequestEntity, maxAge string) error {
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
	return ctx.OK(entity)
}

// WorkItemTypeSingle an alias to app.WorkItemTypeSingle
type WorkItemTypeSingle app.WorkItemTypeSingle

// GetLastModified gets the update time for a given element.
func (entity WorkItemTypeSingle) GetLastModified() time.Time {
	var updatedAt time.Time
	if entity.Data.Attributes.UpdatedAt != nil {
		updatedAt = entity.Data.Attributes.UpdatedAt.Truncate(time.Second).UTC()
	}
	return updatedAt
}

// GenerateETag generates the value to return in the "ETag" HTTP response header, using the data in the given buffer.
// The ETag is the base64-encoded value of the md5 hash of the buffer content
func (entity WorkItemTypeSingle) GenerateETag() string {
	var buffer bytes.Buffer
	// build a block of text for the given type with one <id>-<version>
	buffer.WriteString(entity.Data.ID.String())
	buffer.WriteString("-")
	buffer.WriteString(strconv.Itoa(entity.Data.Attributes.Version))
	buffer.WriteString("\n")
	etagData := md5.Sum(buffer.Bytes())
	etag := base64.StdEncoding.EncodeToString(etagData[:])
	return etag
}

// ShowWorkitemtypeContext an alias to app.ShowWorkitemtypeContext
type ShowWorkitemtypeContext struct {
	app.ShowWorkitemtypeContext
}

// NotModified returns the `304 Not Modified` response with an empty body
func (ctx ShowWorkitemtypeContext) NotModified() error {
	return ctx.ShowWorkitemtypeContext.NotModified()
}

// OK returns the `200 OK` response with the given entity in the body
func (ctx ShowWorkitemtypeContext) OK(entity interface{}) error {
	res := app.WorkItemTypeSingle(entity.(WorkItemTypeSingle))
	return ctx.ShowWorkitemtypeContext.OK(&res)
}

// ModifiedSince returns `true` if the context's `If-Modified-Since` header is `nil` or before the given `lastModified` argument.
func (ctx ShowWorkitemtypeContext) ModifiedSince(lastModified time.Time) bool {
	if ctx.IfModifiedSince != nil {
		ifModifiedSince := ctx.IfModifiedSince.UTC()
		return ifModifiedSince.Before(lastModified)
	}
	return true
}

// MatchesETag returns `true` the given `etag` matches with the context's `If-None-Match` header.
func (ctx ShowWorkitemtypeContext) MatchesETag(etag string) bool {
	if ctx.IfNoneMatch != nil && *ctx.IfNoneMatch == etag {
		return true
	}
	return false
}

// SetHeader sets the header with the given name and value.
func (ctx ShowWorkitemtypeContext) SetHeader(name, value string) {
	ctx.ResponseData.Header().Set(name, value)
}

// ---------

// WorkItemTypeList an alias to app.WorkItemTypeList
type WorkItemTypeList app.WorkItemTypeList

// GetLastModified gets the update time for a given element.
func (entity WorkItemTypeList) GetLastModified() time.Time {
	var updatedAt time.Time
	for _, workitemtypeData := range entity.Data {
		if workitemtypeData.Attributes.UpdatedAt != nil && workitemtypeData.Attributes.UpdatedAt.After(updatedAt) {
			updatedAt = *workitemtypeData.Attributes.UpdatedAt
		}
	}
	return updatedAt.Truncate(time.Second).UTC()
}

// GenerateETag generates the value to return in the "ETag" HTTP response header, using the data in the given buffer.
// The ETag is the base64-encoded value of the md5 hash of the buffer content
func (entity WorkItemTypeList) GenerateETag() string {
	var buffer bytes.Buffer
	for _, workitemtypeData := range entity.Data {
		buffer.WriteString(workitemtypeData.ID.String())
		buffer.WriteString("-")
		buffer.WriteString(strconv.Itoa(workitemtypeData.Attributes.Version))
		buffer.WriteString("\n")
	}
	etagData := md5.Sum(buffer.Bytes())
	etag := base64.StdEncoding.EncodeToString(etagData[:])
	return etag
}

// ListWorkitemtypeContext an alias to app.ListWorkitemtypeContext
type ListWorkitemtypeContext struct {
	app.ListWorkitemtypeContext
}

// NotModified returns the `304 Not Modified` response with an empty body
func (ctx ListWorkitemtypeContext) NotModified() error {
	return ctx.ListWorkitemtypeContext.NotModified()
}

// OK returns the `200 OK` response with the given entity in the body
func (ctx ListWorkitemtypeContext) OK(entity interface{}) error {
	res := app.WorkItemTypeList(entity.(WorkItemTypeList))
	return ctx.ListWorkitemtypeContext.OK(&res)
}

// ModifiedSince returns `true` if the
func (ctx ListWorkitemtypeContext) ModifiedSince(lastModified time.Time) bool {
	if ctx.IfModifiedSince != nil {
		ifModifiedSince := ctx.IfModifiedSince.UTC()
		return ifModifiedSince.Before(lastModified.UTC())
	}
	return true
}

// MatchesETag returns `true` the given `etag` matches with the context's `If-None-Match` header.
func (ctx ListWorkitemtypeContext) MatchesETag(etag string) bool {
	if ctx.IfNoneMatch != nil && *ctx.IfNoneMatch == etag {
		return true
	}
	return false
}

// SetHeader sets the header with the given name and value
func (ctx ListWorkitemtypeContext) SetHeader(name, value string) {
	ctx.ResponseData.Header().Set(name, value)
}

// ----

// WorkItemLinkTypeSingle an alias to app.WorkItemLinkTypeSingle
type WorkItemLinkTypeSingle app.WorkItemLinkTypeSingle

// GetLastModified gets the update time for a given element.
func (entity WorkItemLinkTypeSingle) GetLastModified() time.Time {
	var updatedAt time.Time
	if entity.Data.Attributes.UpdatedAt != nil && entity.Data.Attributes.UpdatedAt.After(updatedAt) {
		updatedAt = *entity.Data.Attributes.UpdatedAt
	}
	return updatedAt.Truncate(time.Second).UTC()
}

// GenerateETag generates the value to return in the "ETag" HTTP response header, using the data in the given buffer.
// The ETag is the base64-encoded value of the md5 hash of the buffer content
func (entity WorkItemLinkTypeSingle) GenerateETag() string {
	var buffer bytes.Buffer
	// build a block of text for the given type with one <id>-<version>
	buffer.WriteString(entity.Data.ID.String())
	buffer.WriteString("-")
	buffer.WriteString(strconv.Itoa(*entity.Data.Attributes.Version))
	buffer.WriteString("\n")
	etagData := md5.Sum(buffer.Bytes())
	etag := base64.StdEncoding.EncodeToString(etagData[:])
	return etag
}

// WorkItemLinkTypeList an alias to app.WorkItemLinkTypeList
type WorkItemLinkTypeList app.WorkItemLinkTypeList

// GetLastModified gets the update time for a given element.
func (entity WorkItemLinkTypeList) GetLastModified() time.Time {
	var updatedAt time.Time
	for _, workitemlinktypeData := range entity.Data {
		if workitemlinktypeData.Attributes.UpdatedAt != nil && workitemlinktypeData.Attributes.UpdatedAt.After(updatedAt) {
			updatedAt = *workitemlinktypeData.Attributes.UpdatedAt
		}
	}
	return updatedAt.Truncate(time.Second).UTC()
}

// GenerateETag generates the value to return in the "ETag" HTTP response header, using the data in the given buffer.
// The ETag is the base64-encoded value of the md5 hash of the buffer content
func (entity WorkItemLinkTypeList) GenerateETag() string {
	var buffer bytes.Buffer
	for _, workitemtypeData := range entity.Data {
		// build a block of text for the given type with one <id>-<version>
		buffer.WriteString(workitemtypeData.ID.String())
		buffer.WriteString("-")
		buffer.WriteString(strconv.Itoa(*workitemtypeData.Attributes.Version))
		buffer.WriteString("\n")
	}
	etagData := md5.Sum(buffer.Bytes())
	etag := base64.StdEncoding.EncodeToString(etagData[:])
	return etag
}

// ListSourceLinkTypesWorkitemtypeContext an alias to app.ListSourceLinkTypesWorkitemtypeContext
type ListSourceLinkTypesWorkitemtypeContext struct {
	app.ListSourceLinkTypesWorkitemtypeContext
}

// NotModified returns the `304 Not Modified` response with an empty body
func (ctx ListSourceLinkTypesWorkitemtypeContext) NotModified() error {
	return ctx.ListSourceLinkTypesWorkitemtypeContext.NotModified()
}

// OK returns the `200 OK` response with the given entity in the body
func (ctx ListSourceLinkTypesWorkitemtypeContext) OK(entity interface{}) error {
	res := app.WorkItemLinkTypeList(entity.(WorkItemLinkTypeList))
	return ctx.ListSourceLinkTypesWorkitemtypeContext.OK(&res)
}

// ModifiedSince returns `true` if the
func (ctx ListSourceLinkTypesWorkitemtypeContext) ModifiedSince(lastModified time.Time) bool {
	if ctx.IfModifiedSince != nil {
		ifModifiedSince := ctx.IfModifiedSince.UTC()
		return ifModifiedSince.Before(lastModified.UTC())
	}
	return true
}

// MatchesETag returns `true` the given `etag` matches with the context's `If-None-Match` header.
func (ctx ListSourceLinkTypesWorkitemtypeContext) MatchesETag(etag string) bool {
	if ctx.IfNoneMatch != nil && *ctx.IfNoneMatch == etag {
		return true
	}
	return false
}

// SetHeader sets the header with the given name and value
func (ctx ListSourceLinkTypesWorkitemtypeContext) SetHeader(name, value string) {
	ctx.ResponseData.Header().Set(name, value)
}

// ListTargetLinkTypesWorkitemtypeContext an alias to app.ListTargetLinkTypesWorkitemtypeContext
type ListTargetLinkTypesWorkitemtypeContext struct {
	app.ListTargetLinkTypesWorkitemtypeContext
}

// NotModified returns the `304 Not Modified` response with an empty body
func (ctx ListTargetLinkTypesWorkitemtypeContext) NotModified() error {
	return ctx.ListTargetLinkTypesWorkitemtypeContext.NotModified()
}

// OK returns the `200 OK` response with the given entity in the body
func (ctx ListTargetLinkTypesWorkitemtypeContext) OK(entity interface{}) error {
	res := app.WorkItemLinkTypeList(entity.(WorkItemLinkTypeList))
	return ctx.ListTargetLinkTypesWorkitemtypeContext.OK(&res)
}

// ModifiedSince returns `true` if the
func (ctx ListTargetLinkTypesWorkitemtypeContext) ModifiedSince(lastModified time.Time) bool {
	if ctx.IfModifiedSince != nil {
		ifModifiedSince := ctx.IfModifiedSince.UTC()
		return ifModifiedSince.Before(lastModified.UTC())
	}
	return true
}

// MatchesETag returns `true` the given `etag` matches with the context's `If-None-Match` header.
func (ctx ListTargetLinkTypesWorkitemtypeContext) MatchesETag(etag string) bool {
	if ctx.IfNoneMatch != nil && *ctx.IfNoneMatch == etag {
		return true
	}
	return false
}

// SetHeader sets the header with the given name and value
func (ctx ListTargetLinkTypesWorkitemtypeContext) SetHeader(name, value string) {
	ctx.ResponseData.Header().Set(name, value)
}
