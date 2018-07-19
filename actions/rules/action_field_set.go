package rules

import (
	"context"
	"github.com/fabric8-services/fabric8-wit/application"
	"encoding/json"
	"reflect"
	"errors"

	uuid "github.com/satori/go.uuid"

	"github.com/fabric8-services/fabric8-wit/convert"
	"github.com/fabric8-services/fabric8-wit/workitem"
)

// ActionFieldSet takes a configuration JSON object that has field 
// names as the keys and a value as the argument. It updates the 
// given ChangeDetector and sets the Field[key] value to the 
// values given. Note that this only works on WorkItems.
type ActionFieldSet struct {
	Db application.DB
	Ctx context.Context
	UserID *uuid.UUID
}

// make sure the rule is implementing the interface.
var _ Action = ActionFieldSet{}

// OnChange executes the action rule.
func (act ActionFieldSet) OnChange(newContext convert.ChangeDetector, contextChanges []convert.Change, configuration string, actionChanges *[]convert.Change) (convert.ChangeDetector, []convert.Change, error) {
	// check if the newContext is a WorkItem, fail otherwise.
	wiContext, ok := newContext.(workitem.WorkItem)
	if !ok {
		return nil, nil, errors.New("Given context is not a WorkItem: " + reflect.TypeOf(newContext).String())
	}
	// deserialize the config JSON
	rawType := map[string]interface{}{}
	err := json.Unmarshal([]byte(configuration), &rawType)
	if err != nil {
		return nil, nil, errors.New("Failed to unmarshall from action configuration to a map: " + configuration)
	}
	var convertChanges []convert.Change
	for k, v := range rawType { 
		if wiContext.Fields[k] != v {
			convertChanges = append(convertChanges, convert.Change{
				AttributeName: k,
				NewValue: v,
				OldValue: wiContext.Fields[k],
			})
			wiContext.Fields[k] = v
		}
	}
	return newContext, convertChanges, nil
}
