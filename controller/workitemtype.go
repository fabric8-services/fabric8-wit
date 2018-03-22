package controller

import (
	"fmt"
	"net/http"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/ptr"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/fabric8-services/fabric8-wit/space"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/goadesign/goa"
	errs "github.com/pkg/errors"
)

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
	return application.Transactional(c.db, func(appl application.Application) error {
		witModel, err := appl.WorkItemTypes().Load(ctx.Context, ctx.WitID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		return ctx.ConditionalRequest(*witModel, c.config.GetCacheControlWorkItemType, func() error {
			witData := ConvertWorkItemTypeFromModel(ctx.Request, witModel)
			wit := &app.WorkItemTypeSingle{Data: &witData}
			return ctx.OK(wit)
		})
	})
}

// ConvertWorkItemTypeFromModel converts from models to app representation
func ConvertWorkItemTypeFromModel(request *http.Request, t *workitem.WorkItemType) app.WorkItemTypeData {
	spaceTemplateRelatedURL := rest.AbsoluteURL(request, app.SpaceTemplateHref(t.SpaceTemplateID.String()))
	spaceRelatedURL := rest.AbsoluteURL(request, app.SpaceHref(space.SystemSpace.String()))
	var converted = app.WorkItemTypeData{
		Type: APIStringTypeWorkItemType,
		ID:   ptr.UUID(t.ID),
		Attributes: &app.WorkItemTypeAttributes{
			CreatedAt:    ptr.Time(t.CreatedAt.UTC()),
			UpdatedAt:    ptr.Time(t.UpdatedAt.UTC()),
			Version:      &t.Version,
			Description:  t.Description,
			Icon:         t.Icon,
			Name:         t.Name,
			Fields:       map[string]*app.FieldDefinition{},
			CanConstruct: ptr.Bool(t.CanConstruct),
		},
		Relationships: &app.WorkItemTypeRelationships{
			// TODO(kwk): The Space relationship should be deprecated after clients adopted
			Space:         app.NewSpaceRelation(space.SystemSpace, spaceRelatedURL),
			SpaceTemplate: app.NewSpaceTemplateRelation(t.SpaceTemplateID, spaceTemplateRelatedURL),
		},
	}
	for name, def := range t.Fields {
		ct := convertFieldTypeFromModel(def.Type)
		converted.Attributes.Fields[name] = &app.FieldDefinition{
			Required:    def.Required,
			Label:       def.Label,
			Description: def.Description,
			Type:        &ct,
		}
	}
	if len(t.ChildTypeIDs) > 0 {
		converted.Relationships.GuidedChildTypes = &app.RelationGenericList{
			Data: make([]*app.GenericData, len(t.ChildTypeIDs)),
		}
		for i, id := range t.ChildTypeIDs {
			converted.Relationships.GuidedChildTypes.Data[i] = &app.GenericData{
				ID:   ptr.String(id.String()),
				Type: ptr.String(APIStringTypeWorkItemType),
			}
		}
	}
	return converted
}

// converts the field type from modesl to app representation
func convertFieldTypeFromModel(t workitem.FieldType) app.FieldType {
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

func convertFieldTypeToModel(t app.FieldType) (workitem.FieldType, error) {
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
		return workitem.EnumType{
			SimpleType: workitem.SimpleType{*kind},
			BaseType:   baseType,
			Values:     converted,
		}, nil
	default:
		return workitem.SimpleType{*kind}, nil
	}
}

func ConvertFieldDefinitionsToModel(fields map[string]app.FieldDefinition) (map[string]workitem.FieldDefinition, error) {
	modelFields := map[string]workitem.FieldDefinition{}
	// now process new fields, checking whether they are ok to add.
	for field, definition := range fields {
		ct, err := convertFieldTypeToModel(*definition.Type)
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
