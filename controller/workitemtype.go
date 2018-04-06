package controller

import (
	"fmt"
	"net/http"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/ptr"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/fabric8-services/fabric8-wit/space"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/goadesign/goa"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

const (
	sourceLinkTypesRouteEnd = "/source-link-types"
	targetLinkTypesRouteEnd = "/target-link-types"
)

const APIWorkItemTypes = "workitemtypes"

// WorkitemtypeController implements the workitemtype resource.
type WorkitemtypeController struct {
	*goa.Controller
	db     application.DB
	config WorkItemControllerConfiguration
}

type WorkItemControllerConfiguration interface {
	GetCacheControlWorkItemTypes() string
	GetCacheControlWorkItemType() string
}

// NewWorkitemtypeController creates a workitemtype controller.
func NewWorkitemtypeController(service *goa.Service, db application.DB, config WorkItemControllerConfiguration) *WorkitemtypeController {
	return &WorkitemtypeController{
		Controller: service.NewController("WorkitemtypeController"),
		db:         db,
		config:     config,
	}
}

// Show runs the show action.
func (c *WorkitemtypeController) Show(ctx *app.ShowWorkitemtypeContext) error {
	err := application.Transactional(c.db, func(appl application.Application) error {
		witModel, err := appl.WorkItemTypes().Load(ctx.Context, ctx.SpaceID, ctx.WitID)
		if err != nil {
			return err
		}
		return ctx.ConditionalRequest(*witModel, c.config.GetCacheControlWorkItemType, func() error {
			witData := ConvertWorkItemTypeFromModel(ctx.Request, witModel)
			wit := &app.WorkItemTypeSingle{Data: &witData}
			return ctx.OK(wit)
		})
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	return nil
}

// List runs the list action
func (c *WorkitemtypeController) List(ctx *app.ListWorkitemtypeContext) error {
	log.Debug(ctx, map[string]interface{}{"space_id": ctx.SpaceID}, "Listing work item types per space")
	start, limit, err := parseLimit(ctx.Page)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, "Could not parse paging"))
	}
	witModels := []workitem.WorkItemType{}
	err = application.Transactional(c.db, func(appl application.Application) error {
		witModelsOrig, err := appl.WorkItemTypes().List(ctx.Context, ctx.SpaceID, start, &limit)
		if err != nil {
			return errs.Wrap(err, "Error listing work item types")
		}
		// Remove "planneritem" from the list of WITs
		for _, wit := range witModelsOrig {
			if wit.ID != workitem.SystemPlannerItem {
				witModels = append(witModels, wit)
			}
		}
		return nil
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	return ctx.ConditionalEntities(witModels, c.config.GetCacheControlWorkItemTypes, func() error {
		// TEMP!!!!! Until Space Template can setup a Space, redirect to SystemSpace WITs if non are found
		// for the space.
		err = application.Transactional(c.db, func(appl application.Application) error {
			if len(witModels) == 0 {
				witModels, err = appl.WorkItemTypes().List(ctx.Context, space.SystemSpace, start, &limit)
				if err != nil {
					return errs.Wrap(err, "Error listing work item types")
				}
			}
			return nil
		})
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		// convert from model to app
		result := &app.WorkItemTypeList{}
		result.Data = make([]*app.WorkItemTypeData, len(witModels))
		for index, value := range witModels {
			wit := ConvertWorkItemTypeFromModel(ctx.Request, &value)
			result.Data[index] = &wit
		}
		return ctx.OK(result)
	})
}

// ConvertWorkItemTypeFromModel converts from models to app representation
func ConvertWorkItemTypeFromModel(request *http.Request, t *workitem.WorkItemType) app.WorkItemTypeData {
	spaceSelfURL := rest.AbsoluteURL(request, app.SpaceHref(t.SpaceID.String()))
	var converted = app.WorkItemTypeData{
		Type: "workitemtypes",
		ID:   ptr.UUID(t.ID),
		Attributes: &app.WorkItemTypeAttributes{
			CreatedAt:   ptr.Time(t.CreatedAt.UTC()),
			UpdatedAt:   ptr.Time(t.UpdatedAt.UTC()),
			Version:     &t.Version,
			Description: t.Description,
			Icon:        t.Icon,
			Name:        t.Name,
			Fields:      map[string]*app.FieldDefinition{},
		},
		Relationships: &app.WorkItemTypeRelationships{
			Space: app.NewSpaceRelation(t.SpaceID, spaceSelfURL),
		},
	}
	for name, def := range t.Fields {
		ct := ConvertFieldTypeFromModel(def.Type)
		converted.Attributes.Fields[name] = &app.FieldDefinition{
			Required:    def.Required,
			Label:       def.Label,
			Description: def.Description,
			Type:        &ct,
		}
	}
	// TODO(kwk): Replaces this temporary static hack with a more dynamic solution
	getGuidedChildTypes := func(witIDs ...uuid.UUID) *app.RelationGenericList {
		res := &app.RelationGenericList{
			Data: make([]*app.GenericData, len(witIDs)),
		}
		for i, id := range witIDs {
			res.Data[i] = &app.GenericData{
				ID:   ptr.String(id.String()),
				Type: ptr.String(APIWorkItemTypes),
				// Links: &app.GenericLinks{
				// 	Related: strPtr(rest.AbsoluteURL(request, app.WorkitemtypeHref(t.SpaceID, id.String()))),
				// },
			}
		}
		return res
	}
	switch t.ID {
	case workitem.SystemScenario, workitem.SystemFundamental, workitem.SystemPapercuts:
		converted.Relationships.GuidedChildTypes = getGuidedChildTypes(workitem.SystemExperience, workitem.SystemValueProposition)
	case workitem.SystemExperience, workitem.SystemValueProposition:
		converted.Relationships.GuidedChildTypes = getGuidedChildTypes(workitem.SystemFeature, workitem.SystemBug)
	case workitem.SystemFeature:
		converted.Relationships.GuidedChildTypes = getGuidedChildTypes(workitem.SystemTask, workitem.SystemBug)
	case workitem.SystemBug:
		converted.Relationships.GuidedChildTypes = getGuidedChildTypes(workitem.SystemTask)
	}
	return converted
}

// converts the field type from modesl to app representation
func ConvertFieldTypeFromModel(t workitem.FieldType) app.FieldType {
	result := app.FieldType{}
	result.Kind = string(t.GetKind())
	switch t2 := t.(type) {
	case workitem.ListType:
		result.ComponentType = ptr.String(string(t2.ComponentType.GetKind()))
	case workitem.EnumType:
		result.BaseType = ptr.String(string(t2.BaseType.GetKind()))
		result.Values = t2.Values
	}

	return result
}

func ConvertFieldTypeToModel(t app.FieldType) (workitem.FieldType, error) {
	kind, err := workitem.ConvertStringToKind(t.Kind)
	if err != nil {
		return nil, errs.WithStack(err)
	}
	switch *kind {
	case workitem.KindList:
		componentType, err := workitem.ConvertAnyToKind(*t.ComponentType)
		if err != nil {
			return nil, errs.WithStack(err)
		}
		if !componentType.IsSimpleType() {
			return nil, fmt.Errorf("Component type is not list type: %T", componentType)
		}
		return workitem.ListType{workitem.SimpleType{*kind}, workitem.SimpleType{*componentType}}, nil
	case workitem.KindEnum:
		bt, err := workitem.ConvertAnyToKind(*t.BaseType)
		if err != nil {
			return nil, errs.WithStack(err)
		}
		if !bt.IsSimpleType() {
			return nil, fmt.Errorf("baseType type is not list type: %T", bt)
		}
		baseType := workitem.SimpleType{*bt}

		values := t.Values
		converted, err := workitem.ConvertList(func(ft workitem.FieldType, element interface{}) (interface{}, error) {
			return ft.ConvertToModel(element)
		}, baseType, values)
		if err != nil {
			return nil, errs.WithStack(err)
		}
		return workitem.EnumType{workitem.SimpleType{*kind}, baseType, converted}, nil
	default:
		return workitem.SimpleType{*kind}, nil
	}
}

func ConvertFieldDefinitionsToModel(fields map[string]app.FieldDefinition) (map[string]workitem.FieldDefinition, error) {
	modelFields := map[string]workitem.FieldDefinition{}
	// now process new fields, checking whether they are ok to add.
	for field, definition := range fields {
		ct, err := ConvertFieldTypeToModel(*definition.Type)
		if err != nil {
			return nil, errs.WithStack(err)
		}
		converted := workitem.FieldDefinition{
			Label:       definition.Label,
			Description: definition.Description,
			Required:    definition.Required,
			Type:        ct,
		}
		modelFields[field] = converted
	}
	return modelFields, nil
}
