package main

// PermissionDefinition defines the Permissions available
type PermissionDefinition struct {
	CreateWorkItem string
	ReadWorkItem   string
	UpdateWorkItem string
	DeleteWorkItem string
}

// CRUDWotkItem returns all CRUD permissions for a WorkItem
func (p *PermissionDefinition) CRUDWotkItem() []string {
	return []string{p.CreateWorkItem, p.ReadWorkItem, p.UpdateWorkItem, p.DeleteWorkItem}
}

var (
	// Permissions defines the value of each Permission
	Permissions = PermissionDefinition{
		CreateWorkItem: "create.workitem",
		ReadWorkItem:   "read.workitem",
		UpdateWorkItem: "update.workitem",
		DeleteWorkItem: "delete.workitem",
	}
)
