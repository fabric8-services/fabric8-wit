package actions

import (
	"github.com/fabric8-services/fabric8-wit/convert"
)

type ActionStateToMetaState struct {
}

func (action *ActionStateToMetaState) onChange(newContext convert.ChangeDetector, contextChanges []convert.Change, configuration string, actionChanges *[]convert.Change) (convert.ChangeDetector, []convert.Change, error) {
	return nil, nil, nil
}
