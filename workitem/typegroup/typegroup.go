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

// Portfolio0 defines first level of typegroup Portfolio
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

// Portfolio1 defines second level of typegroup Portfolio
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

// Requirements0 defines first level of typegroup Requirements
// This group has less priority than Portfolio
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
