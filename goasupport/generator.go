package goasupport

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"strings"

	"github.com/goadesign/goa/design"
	"github.com/goadesign/goa/goagen/codegen"
)

// Generate adds Get`Header`() methods to the XContext objects
func Generate() ([]string, error) {
	var (
		ver    string
		outDir string
	)
	set := flag.NewFlagSet("app", flag.PanicOnError)
	set.String("design", "", "") // Consume design argument so Parse doesn't complain
	set.StringVar(&ver, "version", "", "")
	set.StringVar(&outDir, "out", "", "")
	set.Parse(os.Args[2:])

	// First check compatibility
	if err := codegen.CheckVersion(ver); err != nil {
		return nil, err
	}

	return WriteNames(design.Design, outDir)
}

// RequestContext holds a single goa Request Context object
type RequestContext struct {
	Name    string
	Headers []RequestHeader
}

// RequestHeader holds a single HTTP Header as defined in the design for a Request Context
type RequestHeader struct {
	Name   string
	Header string
	Type   string
}

// ResponseContext holds a single goa Response Context object
type ResponseContext struct {
	Name   string
	Entity Entity
}

// Entity holds a single goa Response entity object that can be used in multiple responses.
type Entity struct {
	AppTypeName    string
	DomainTypeName string
	IsSingle       bool
	IsList         bool
}

func contains(entities []Entity, entity Entity) bool {
	for _, e := range entities {
		if e.AppTypeName == entity.AppTypeName {
			return true
		}
	}
	return false

}

// WriteNames creates the names.txt file.
func WriteNames(api *design.APIDefinition, outDir string) ([]string, error) {
	// Now iterate through the resources to gather their names
	var requestContexts []RequestContext
	var responseContexts []ResponseContext
	var entities []Entity

	api.IterateResources(func(res *design.ResourceDefinition) error {
		res.IterateActions(func(act *design.ActionDefinition) error {
			name := fmt.Sprintf("%v%vContext", codegen.Goify(act.Name, true), codegen.Goify(res.Name, true))
			// look-up headers for conditional request support
			if act.Headers != nil {
				rc := RequestContext{Name: name}
				for header, value := range act.Headers.Type.ToObject() {
					if header != "If-Modified-Since" && header != "If-None-Match" {
						continue
					}
					rc.Headers = append(
						rc.Headers,
						RequestHeader{
							Name:   codegen.Goify(header, true),
							Header: header,
							Type:   codegen.GoTypeRef(value.Type, nil, 0, false),
						})
				}
				requestContexts = append(requestContexts, rc)

				// look-up headers and entity types in responses
				if act.Responses != nil {
					for _, response := range act.Responses {
						if response.Name == design.OK && response.Type != nil {
							if mt, ok := response.Type.(*design.MediaTypeDefinition); ok {
								var entity *Entity
								// lookup conditional request/response headers
								for header := range response.Headers.Type.ToObject() {
									if header == "ETag" || header == "LastModified" {
										// assume that a "list" entities have their name ending with "List"
										// and "single" entities have their name ending with "Single"
										isList := strings.HasSuffix(mt.TypeName, "List")
										var domainTypeName string
										if isList {
											domainTypeName = strings.TrimSuffix(mt.TypeName, "List")
										} else {
											domainTypeName = strings.TrimSuffix(mt.TypeName, "Single")
										}
										entity = &Entity{AppTypeName: mt.TypeName, DomainTypeName: domainTypeName, IsList: isList, IsSingle: !isList}
										break
									}
								}
								// skip if no response header was found
								if entity != nil {
									fmt.Printf("Response context: %s -> entity: %v\n", name, mt.TypeName)
									// for k, v := range m.ToObject() {
									// 	fmt.Printf("%s -> %v\n", k, v)
									// }
									responseContext := ResponseContext{Name: name, Entity: *entity}
									responseContexts = append(responseContexts, responseContext)
									if !contains(entities, *entity) {
										entities = append(entities, *entity)
									}

								}
							}
						}
					}
				}
			}
			return nil
		})
		return nil
	})

	ctxFile := filepath.Join(outDir, "conditional_requests.go")
	ctxWr, err := codegen.SourceFileFor(ctxFile)
	if err != nil {
		panic(err) // bug
	}
	title := fmt.Sprintf("%s: Context Header Methods", api.Context())
	imports := []*codegen.ImportSpec{
		codegen.SimpleImport("bytes"),
		codegen.SimpleImport("crypto/md5"),
		codegen.SimpleImport("encoding/base64"),
		codegen.SimpleImport("strconv"),
		codegen.SimpleImport("time"),
		codegen.SimpleImport("github.com/almighty/almighty-core/configuration"),
	}

	ctxWr.WriteHeader(title, "app", imports)
	// if err := ctxWr.ExecuteTemplate("headerMethods", headerMethods, nil, requestContexts); err != nil {
	// 	return nil, err
	// }
	if err := ctxWr.ExecuteTemplate("constants", constants, nil, requestContexts); err != nil {
		return nil, err
	}
	if err := ctxWr.ExecuteTemplate("cacheControlConfig", cacheControlConfig, nil, requestContexts); err != nil {
		return nil, err
	}
	if err := ctxWr.ExecuteTemplate("conditionalResponseEntity", conditionalResponseEntity, nil, requestContexts); err != nil {
		return nil, err
	}

	for _, responseCtx := range responseContexts {
		if err := ctxWr.ExecuteTemplate("conditional", conditional, nil, responseCtx); err != nil {
			return nil, err
		}
		if err := ctxWr.ExecuteTemplate("modifiedSince", modifiedSince, nil, responseCtx); err != nil {
			return nil, err
		}
		if err := ctxWr.ExecuteTemplate("matchesETag", matchesETag, nil, responseCtx); err != nil {
			return nil, err
		}
		if err := ctxWr.ExecuteTemplate("setLastModified", setLastModified, nil, responseCtx); err != nil {
			return nil, err
		}
		if err := ctxWr.ExecuteTemplate("setETag", setETag, nil, responseCtx); err != nil {
			return nil, err
		}
		if err := ctxWr.ExecuteTemplate("setCacheControl", setCacheControl, nil, responseCtx); err != nil {
			return nil, err
		}
	}
	for _, entity := range entities {
		if err := ctxWr.ExecuteTemplate("getLastModified", getLastModified, nil, entity); err != nil {
			return nil, err
		}
		if err := ctxWr.ExecuteTemplate("getETag", getETag, nil, entity); err != nil {
			return nil, err
		}
	}
	err = ctxWr.FormatCode()
	if err != nil {
		return nil, err
	}
	return []string{ctxFile}, nil
}

const (
	constants = `
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
)`
	conditionalResponseEntity = `
	// ConditionalResponseEntity interface with methods for the response entities
type ConditionalResponseEntity interface {
	GetLastModified() time.Time
	GetETag() string
}`

	cacheControlConfig = `
   type CacheControlConfig func() string 
   `
	conditional = `
{{ $resp := . }}
// Conditional checks if the entity to return changed since the client's last call and returns a "304 Not Modified" response
// or calls the 'nonConditionalCallback' function to carry on.
func (ctx *{{$resp.Name}}) Conditional(entity ConditionalResponseEntity, cacheControlConfig CacheControlConfig, nonConditionalCallback func() error) error {
	lastModified := entity.GetLastModified()
	eTag := entity.GetETag()
	cacheControl := cacheControlConfig()
	ctx.setLastModified(lastModified)
	ctx.setETag(eTag)
	ctx.setCacheControl(cacheControl)
	if !ctx.modifiedSince(lastModified) {
		return ctx.NotModified()
	}
	// check the ETag
	if ctx.matchesETag(eTag) {
		return ctx.NotModified()
	}
	// call the 'nonConditionalCallback' if the entity was modified since the client's last call
	return nonConditionalCallback()
}`

	modifiedSince = `
{{ $resp := . }}
// ModifiedSince returns 'true' if the given 'lastModified' argument is after the context's 'IfModifiedSince' value
func (ctx *{{$resp.Name}}) modifiedSince(lastModified time.Time) bool {
	if ctx.IfModifiedSince != nil {
		ifModifiedSince := ctx.IfModifiedSince.UTC()
		return ifModifiedSince.Before(lastModified.UTC())
	}
	return true
}`

	matchesETag = `
{{ $resp := . }}
// MatchesETag returns 'true' the given 'etag' argument matches with the context's 'IfNoneMatch' value.
func (ctx *{{$resp.Name}}) matchesETag(etag string) bool {
	if ctx.IfNoneMatch != nil && *ctx.IfNoneMatch == etag {
		return true
	}
	return false
}`

	setLastModified = `
{{ $resp := . }}
// SetLastModified sets the 'Last-Modified' header
func (ctx *{{$resp.Name}}) setLastModified(value time.Time) {
	ctx.ResponseData.Header().Set(LastModified, value.String())
}`

	setETag = `
{{ $resp := . }}
// SetETag sets the 'ETag' header
func (ctx *{{$resp.Name}}) setETag(value string) {
	ctx.ResponseData.Header().Set(ETag, value)
}`

	setCacheControl = `
{{ $resp := . }}
// SetCacheControl sets the 'Cache-Control' header
func (ctx *{{$resp.Name}}) setCacheControl(value string) {
	ctx.ResponseData.Header().Set(CacheControl, value)
}`

	getLastModified = `
{{ $entity := . }}
{{ if $entity.IsSingle }}
 // GetLastModified gets the update time for a given element.
func (entity {{$entity.AppTypeName}}) GetLastModified() time.Time {
	var updatedAt time.Time
	if entity.Data.Attributes.UpdatedAt != nil && entity.Data.Attributes.UpdatedAt.After(updatedAt) {
		updatedAt = *entity.Data.Attributes.UpdatedAt
	}
	return updatedAt.Truncate(time.Second).UTC()
}
{{ end }}
{{ if $entity.IsList }}
// GetLastModified gets the update time for a given element.
func (entity {{$entity.AppTypeName}}) GetLastModified() time.Time {
	var updatedAt time.Time
	for _, data := range entity.Data {
		if data.Attributes.UpdatedAt != nil && data.Attributes.UpdatedAt.After(updatedAt) {
			updatedAt = *data.Attributes.UpdatedAt
		}
	}
	return updatedAt.Truncate(time.Second).UTC()
}
{{ end }}`

	getETag = `
{{ $entity := . }}
{{ if $entity.IsSingle }}
// GetETag generates the value to return in the "ETag" HTTP response header, using the data in the given buffer.
// The ETag is the base64-encoded value of the md5 hash of the buffer content
func (entity {{$entity.AppTypeName}}) GetETag() string {
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
{{ end }}
{{ if $entity.IsList }}
// GetETag generates the value to return in the "ETag" HTTP response header, using the data in the given buffer.
// The ETag is the base64-encoded value of the md5 hash of the buffer content
func (entity {{$entity.AppTypeName}}) GetETag() string {
	var buffer bytes.Buffer
	for _, data := range entity.Data {
		buffer.WriteString(data.ID.String())
		buffer.WriteString("-")
		buffer.WriteString(strconv.Itoa(*data.Attributes.Version))
		buffer.WriteString("\n")
	}
	etagData := md5.Sum(buffer.Bytes())
	etag := base64.StdEncoding.EncodeToString(etagData[:])
	return etag
}
{{ end }}`
)
