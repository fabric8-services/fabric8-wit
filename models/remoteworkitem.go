package models

// RemoteWorkItem represents a work item as it is stored in the database
type RemoteWorkItem struct {
	Fields map[string]interface{}	
}

// The intregration code for a specific remote provider must populate this.
type WorkItemMap map[string]interface{}

// Map maps RemoteWorkItem to WorkItem
func (rwi RemoteWorkItem) MapRemote (wiMap WorkItemMap, wiType WorkItemType) WorkItem {
	workItem := WorkItem{
		Type: wiType.ID
	}	
	
	for from_key, to_key := range wiMap {
		workItem.Fields[to_key] = remoteWorkItem.Fields[from_key]
	}
	return workItem
}
