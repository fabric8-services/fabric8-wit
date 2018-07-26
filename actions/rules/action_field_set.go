package rules

import (
	"context"
	"encoding/json"
	"reflect"

	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"

	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/convert"
	"github.com/fabric8-services/fabric8-wit/workitem"
)

// ActionFieldSet takes a configuration JSON object that has field
// names as the keys and a value as the argument. It updates the
// given ChangeDetector and sets the Field[key] value to the
// values given. Note that this only works on WorkItems.
type ActionFieldSet struct {
	Db     application.DB
	Ctx    context.Context
	UserID *uuid.UUID
}

// make sure the rule is implementing the interface.
var _ Action = ActionFieldSet{}

func (act ActionFieldSet) storeWorkItem(workitem *workitem.WorkItem) (*workitem.WorkItem, error) {
	if act.Ctx == nil {
		return nil, errors.New("Context is nil")
	}
	if act.Db == nil {
		return nil, errors.New("Database is nil")
	}
	if act.UserID == nil {
		return nil, errors.New("UserID is nil")
	}
	err := application.Transactional(act.Db, func(appl application.Application) error {
		var err error
		workitem, err = appl.WorkItems().Save(act.Ctx, workitem.SpaceID, *workitem, *act.UserID)
		if err != nil {
			return errors.Wrap(err, "Error updating work item")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return workitem, nil
}

// OnChange executes the action rule.
func (act ActionFieldSet) OnChange(newContext convert.ChangeDetector, contextChanges []convert.Change, configuration string, actionChanges *[]convert.Change) (convert.ChangeDetector, []convert.Change, error) {
	// check if the newContext is a WorkItem, fail otherwise.
	wiContext, ok := newContext.(workitem.WorkItem)
	if !ok {
		return nil, nil, errors.New("Given context is not a WorkItem: " + reflect.TypeOf(newContext).String())
	}
	// deserialize the config JSON
	var rawType map[string]interface{}
	err := json.Unmarshal([]byte(configuration), &rawType)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Failed to unmarshall from action configuration to a map: " + configuration)
	}
	var convertChanges []convert.Change
	for k, v := range rawType {
		if wiContext.Fields[k] != v {
			convertChanges = append(convertChanges, convert.Change{
				AttributeName: k,
				NewValue:      v,
				OldValue:      wiContext.Fields[k],
			})
			wiContext.Fields[k] = v
		}
	}
	// store the WorkItem.
	newContext, err = act.storeWorkItem(&wiContext)
	if err != nil {
		return nil, nil, err
	}
	return newContext, convertChanges, nil
}
