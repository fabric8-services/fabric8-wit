package typegroup

import (
	uuid "github.com/satori/go.uuid"

	"github.com/fabric8-services/fabric8-wit/workitem"
)

// WorkItemTypeGroup represents the node in the group of work item types
type WorkItemTypeGroup struct {
	Level                  int
	Sublevel               int
	Group                  string
	Name                   string
	WorkItemTypeCollection []uuid.UUID
}

var Portfolio0 = WorkItemTypeGroup{
	Group:    "portfolio",
	Level:    0,
	Name:     "Portfolio 0",
	Sublevel: 0,
	WorkItemTypeCollection: []uuid.UUID{
		workitem.SystemScenario,
		workitem.SystemFundamental,
		workitem.SystemPapercuts,
	},
}

var Portfolio1 = WorkItemTypeGroup{
	Group:    "portfolio",
	Level:    0,
	Name:     "Portfolio 1",
	Sublevel: 1,
	WorkItemTypeCollection: []uuid.UUID{
		workitem.SystemExperience,
		workitem.SystemValueProposition,
	},
}

var Requirements0 = WorkItemTypeGroup{
	Group:    "requirements",
	Level:    1,
	Name:     "Requirements 0",
	Sublevel: 0,
	WorkItemTypeCollection: []uuid.UUID{
		workitem.SystemFeature,
		workitem.SystemBug,
	},
}
