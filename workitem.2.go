package main

import (
	"fmt"
	"log"
	"strconv"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/login"
	query "github.com/almighty/almighty-core/query/simple"
	"github.com/almighty/almighty-core/workitem"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
)

const (
	pageSizeDefault = 20
	pageSizeMax     = 100

	APIStringTypeAssignee     = "identities"
	APIStringTypeWorkItem     = "workitems"
	APIStringTypeWorkItemType = "workitemtypes"
)

// Workitem2Controller implements the workitem.2 resource.
type Workitem2Controller struct {
	*goa.Controller
	db application.DB
}

// NewWorkitem2Controller creates a workitem.2 controller.
func NewWorkitem2Controller(service *goa.Service, db application.DB) *Workitem2Controller {
	if db == nil {
		panic("db must not be nil")
	}
	return &Workitem2Controller{Controller: service.NewController("WorkitemController"), db: db}
}

func buildAbsoluteURL(req *goa.RequestData) string {
	scheme := "http"
	if req.TLS != nil { // isHTTPS
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s%s", scheme, req.Host, req.URL.Path)
}

func setPagingLinks(links *app.PagingLinks, path string, resultLen, offset, limit, count int) {

	// prev link
	if offset > 0 && count > 0 {
		var prevStart int
		// we do have a prev link
		if offset <= count {
			prevStart = offset - limit
		} else {
			// the first range that intersects the end of the useful range
			prevStart = offset - (((offset-count)/limit)+1)*limit
		}
		realLimit := limit
		if prevStart < 0 {
			// need to cut the range to start at 0
			realLimit = limit + prevStart
			prevStart = 0
		}
		prev := fmt.Sprintf("%s?page[offset]=%d&page[limit]=%d", path, prevStart, realLimit)
		links.Prev = &prev
	}

	// next link
	nextStart := offset + resultLen
	if nextStart < count {
		// we have a next link
		next := fmt.Sprintf("%s?page[offset]=%d&page[limit]=%d", path, nextStart, limit)
		links.Next = &next
	}

	// first link
	var firstEnd int
	if offset > 0 {
		firstEnd = offset % limit // this is where the second page starts
	} else {
		// offset == 0, first == current
		firstEnd = limit
	}
	first := fmt.Sprintf("%s?page[offset]=%d&page[limit]=%d", path, 0, firstEnd)
	links.First = &first

	// last link
	var lastStart int
	if offset < count {
		// advance some pages until touching the end of the range
		lastStart = offset + (((count - offset - 1) / limit) * limit)
	} else {
		// retreat at least one page until covering the range
		lastStart = offset - ((((offset - count) / limit) + 1) * limit)
	}
	realLimit := limit
	if lastStart < 0 {
		// need to cut the range to start at 0
		realLimit = limit + lastStart
		lastStart = 0
	}
	last := fmt.Sprintf("%s?page[offset]=%d&page[limit]=%d", path, lastStart, realLimit)
	links.Last = &last
}

// List runs the list action.
// Prev and Next links will be present only when there actually IS a next or previous page.
// Last will always be present. Total Item count needs to be computed from the "Last" link.
func (c *Workitem2Controller) List(ctx *app.ListWorkitem2Context) error {
	// Workitem2Controller_List: start_implement

	exp, err := query.Parse(ctx.Filter)
	if err != nil {
		jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrBadRequest(fmt.Sprintf("could not parse filter: %s", err.Error())))
		return ctx.BadRequest(jerrors)
	}
	var offset int
	var limit int

	if ctx.PageOffset == nil {
		offset = 0
	} else {
		offsetValue, err := strconv.Atoi(*ctx.PageOffset)
		if err != nil {
			offset = 0
		} else {
			offset = offsetValue
		}
	}
	if offset < 0 {
		offset = 0
	}

	if ctx.PageLimit == nil {
		limit = pageSizeDefault
	} else {
		limit = *ctx.PageLimit
	}

	if limit <= 0 {
		limit = pageSizeDefault
	} else if limit > pageSizeMax {
		limit = pageSizeMax
	}

	return application.Transactional(c.db, func(tx application.Application) error {
		result, tc, err := tx.WorkItems().List(ctx.Context, exp, &offset, &limit)
		count := int(tc)
		if err != nil {
			switch err := err.(type) {
			case errors.BadParameterError:
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrBadRequest(fmt.Sprintf("Error listing work items: %s", err.Error())))
				return ctx.BadRequest(jerrors)
			default:
				log.Printf("Error listing work items: %s", err.Error())
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrInternal(fmt.Sprintf("Error listing work items: %s", err.Error())))
				return ctx.InternalServerError(jerrors)
			}
		}

		response := app.WorkItem2List{
			Links: &app.PagingLinks{},
			Meta:  &app.WorkItemListResponseMeta{TotalCount: count},
			Data:  c.ConvertWorkItemToJSONAPIArray(result),
		}

		setPagingLinks(response.Links, buildAbsoluteURL(ctx.RequestData), len(result), offset, limit, count)

		return ctx.OK(&response)
	})

	// Workitem2Controller_List: end_implement
}

// Update does PATCH workitem
func (c *Workitem2Controller) Update(ctx *app.UpdateWorkitem2Context) error {
	return application.Transactional(c.db, func(appl application.Application) error {

		if ctx.Payload == nil || ctx.Payload.Data == nil || ctx.Payload.Data.ID == nil {
			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(errors.NewBadParameterError("data.id", nil))
			return ctx.NotFound(jerrors)
		}

		wi, err := appl.WorkItems().Load(ctx, *ctx.Payload.Data.ID)
		if err != nil {
			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrNotFound(fmt.Sprintf("Error updating work item: %s", err.Error())))
			return ctx.NotFound(jerrors)
		}
		err = c.ConvertJSONAPIToWorkItem(*ctx.Payload.Data, wi)
		if err != nil {
			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrBadRequest(fmt.Sprintf("Error updating work item: %s", err.Error())))
			return ctx.BadRequest(jerrors)
		}
		wi, err = appl.WorkItems().Save(ctx, *wi)
		if err != nil {
			switch err := err.(type) {
			case errors.BadParameterError:
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrBadRequest(fmt.Sprintf("Error updating work item: %s", err.Error())))
				return ctx.BadRequest(jerrors)
			case errors.NotFoundError:
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrNotFound(err.Error()))
				return ctx.NotFound(jerrors)
			case errors.VersionConflictError:
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrBadRequest(fmt.Sprintf("Error updating work item: %s", err.Error())))
				return ctx.BadRequest(jerrors)
			default:
				log.Printf("Error updating work items: %s", err.Error())
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrInternal(err.Error()))
				return ctx.InternalServerError(jerrors)
			}
		}

		wi2 := c.ConvertWorkItemToJSONAPI(wi)
		resp := &app.WorkItem2Single{
			Data: wi2,
			Links: &app.WorkItemLinks{
				Self: buildAbsoluteURL(ctx.RequestData),
			},
		}

		return ctx.OK(resp)
	})
}

// Create does POST workitem
func (c *Workitem2Controller) Create(ctx *app.CreateWorkitem2Context) error {
	currentUser, err := login.ContextIdentity(ctx)
	if err != nil {
		jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrUnauthorized(err.Error()))
		return ctx.Unauthorized(jerrors)
	}

	var wit *string
	if ctx.Payload.Data != nil && ctx.Payload.Data.Relationships != nil && ctx.Payload.Data.Relationships.BaseType != nil {
		if ctx.Payload.Data.Relationships.BaseType.Data != nil {
			wit = &ctx.Payload.Data.Relationships.BaseType.Data.ID
		}
	}
	if wit == nil { // TODO Figure out path source etc. Should be a required relation
		jerrors, _ := jsonapi.ErrorToJSONAPIErrors(errors.NewBadParameterError("data.relationships.basetype.data.id", nil))
		return ctx.BadRequest(jerrors)

	}

	wi := app.WorkItem{
		Fields: make(map[string]interface{}),
	}
	c.ConvertJSONAPIToWorkItem(*ctx.Payload.Data, &wi)

	return application.Transactional(c.db, func(appl application.Application) error {

		wi, err := appl.WorkItems().Create(ctx, *wit, wi.Fields, currentUser)
		if err != nil {
			switch err := err.(type) {
			case errors.BadParameterError:
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrBadRequest(fmt.Sprintf("Error updating work item: %s", err.Error())))
				return ctx.BadRequest(jerrors)
			case errors.VersionConflictError:
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrBadRequest(fmt.Sprintf("Error updating work item: %s", err.Error())))
				return ctx.BadRequest(jerrors)
			default:
				log.Printf("Error updating work items: %s", err.Error())
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrInternal(err.Error()))
				return ctx.InternalServerError(jerrors)
			}
		}

		wi2 := c.ConvertWorkItemToJSONAPI(wi)
		resp := &app.WorkItem2Single{
			Data: wi2,
			Links: &app.WorkItemLinks{
				Self: buildAbsoluteURL(ctx.RequestData),
			},
		}
		ctx.ResponseData.Header().Set("Location", app.Workitem2Href(wi2.ID))
		return ctx.Created(resp)
	})
}

// Show does GET workitem
func (c *Workitem2Controller) Show(ctx *app.ShowWorkitem2Context) error {
	return application.Transactional(c.db, func(appl application.Application) error {

		wi, err := appl.WorkItems().Load(ctx, ctx.ID)
		if err != nil {
			switch err := err.(type) {
			case errors.NotFoundError:
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrNotFound(err.Error()))
				return ctx.NotFound(jerrors)
			case errors.BadParameterError:
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrBadRequest(fmt.Sprintf("Error updating work item: %s", err.Error())))
				return ctx.BadRequest(jerrors)
			case errors.VersionConflictError:
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrBadRequest(fmt.Sprintf("Error updating work item: %s", err.Error())))
				return ctx.BadRequest(jerrors)
			default:
				log.Printf("Error updating work items: %s", err.Error())
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrInternal(err.Error()))
				return ctx.InternalServerError(jerrors)
			}
		}

		wi2 := c.ConvertWorkItemToJSONAPI(wi)
		resp := &app.WorkItem2Single{
			Data: wi2,
			Links: &app.WorkItemLinks{
				Self: buildAbsoluteURL(ctx.RequestData),
			},
		}
		return ctx.OK(resp)
	})
}

// Delete does DELETE workitem
func (c *Workitem2Controller) Delete(ctx *app.DeleteWorkitem2Context) error {
	return application.Transactional(c.db, func(appl application.Application) error {

		err := appl.WorkItems().Delete(ctx, ctx.ID)
		if err != nil {
			switch err := err.(type) {
			case errors.NotFoundError:
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrNotFound(err.Error()))
				return ctx.NotFound(jerrors)
			case errors.BadParameterError:
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrBadRequest(fmt.Sprintf("Error updating work item: %s", err.Error())))
				return ctx.BadRequest(jerrors)
			case errors.VersionConflictError:
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrBadRequest(fmt.Sprintf("Error updating work item: %s", err.Error())))
				return ctx.BadRequest(jerrors)
			default:
				log.Printf("Error updating work items: %s", err.Error())
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrInternal(err.Error()))
				return ctx.InternalServerError(jerrors)
			}
		}
		return ctx.OK([]byte{})
	})
}

// ConvertWorkItemToJSONAPIArray is responsible for converting given []WorkItem model object into a
// response resource object by jsonapi.org specifications
func (c *Workitem2Controller) ConvertWorkItemToJSONAPIArray(wis []*app.WorkItem) []*app.WorkItem2 {
	ops := []*app.WorkItem2{}
	for _, wi := range wis {
		ops = append(ops, c.ConvertWorkItemToJSONAPI(wi))
	}
	return ops
}

// ConvertWorkItemToJSONAPI is responsible for converting given WorkItem model object into a
// response resource object by jsonapi.org specifications
func (c *Workitem2Controller) ConvertWorkItemToJSONAPI(wi *app.WorkItem) *app.WorkItem2 {
	// construct default values from input WI

	op := &app.WorkItem2{
		ID:   &wi.ID,
		Type: APIStringTypeWorkItem,
		Attributes: map[string]interface{}{
			"version": wi.Version,
		},
		Relationships: &app.WorkItemRelationships{
			BaseType: &app.RelationBaseType{
				Data: &app.BaseTypeData{
					ID:   wi.Type,
					Type: APIStringTypeWorkItemType,
				},
			},
		},
	}
	// Move fields into Relationships or Attributes as needed
	for name, val := range wi.Fields {
		switch name {
		case workitem.SystemAssignee:
			if val != nil {
				valStr := val.(string)
				op.Relationships.Assignee = &app.RelationAssignee{
					Data: &app.AssigneeData{
						ID:   valStr,
						Type: APIStringTypeAssignee,
					},
				}
			}
		default:
			op.Attributes[name] = val
		}
	}
	if op.Relationships.Assignee == nil {
		op.Relationships.Assignee = &app.RelationAssignee{Data: nil}
	}

	return op
}

// ConvertJSONAPIToWorkItem is responsible for converting given WorkItem model object into a
// response resource object by jsonapi.org specifications
func (c *Workitem2Controller) ConvertJSONAPIToWorkItem(source app.WorkItem2, target *app.WorkItem) error {
	// construct default values from input WI

	var version = -1
	if source.Attributes["version"] != nil {
		v, err := strconv.Atoi(fmt.Sprintf("%v", source.Attributes["version"]))
		if err != nil {
			return errors.NewBadParameterError("data.attributes.version", source.Attributes["version"])
		}
		version = v
	}
	target.Version = version

	if source.Relationships != nil && source.Relationships.Assignee != nil {
		if source.Relationships.Assignee.Data == nil {
			delete(target.Fields, workitem.SystemAssignee)
		} else {
			uuidStr := source.Relationships.Assignee.Data.ID
			assigneeUUID, err := uuid.FromString(uuidStr)
			if err != nil {
				return errors.NewBadParameterError("data.relationships.assignee.data.id", uuidStr)
			}
			ok := c.db.Identities().ValidIdentity(context.Background(), assigneeUUID)
			if !ok {
				return errors.NewBadParameterError("data.relationships.assignee.data.id", uuidStr)
			}

			target.Fields[workitem.SystemAssignee] = source.Relationships.Assignee.Data.ID
		}
	}
	for key, val := range source.Attributes {
		target.Fields[key] = val
	}
	return nil
}
