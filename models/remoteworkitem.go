package models

import "strconv"

// RemoteWorkItem represents a work item as it is stored in the database
type RemoteWorkItem struct {
	Fields map[string]interface{}
}

// The intregration code for a specific remote provider must populate this.
type WorkItemMap map[string]interface{}

// Map maps RemoteWorkItem to WorkItem
func (rwi RemoteWorkItem) MapRemote(wiMap WorkItemMap, wiType *WorkItemType) WorkItem {
	workItem := WorkItem{
		Type:   strconv.FormatUint(wiType.ID, 10),
		Fields: make(map[string]interface{}),
	}

	for from_key, to_key := range wiMap {
		workItem.Fields[to_key.(string)] = rwi.Fields[from_key]
	}
	return workItem
}
