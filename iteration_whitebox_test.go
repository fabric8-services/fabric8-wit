package main

import (
	"testing"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/iteration"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/workitem"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateIterationsWithCounts(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	spaceID := uuid.NewV4()
	//create 2 iteration
	i1 := createMinimumAppIteration("Spting 1234", spaceID.String())
	i2 := createMinimumAppIteration("Spting 1234", spaceID.String())
	i3 := createMinimumAppIteration("Spting 1234", spaceID.String())
	var iterationSlice []*app.Iteration
	iterationSlice = append(iterationSlice, i1, i2, i3)
	counts := make(map[string]workitem.WICountsPerIteration)
	counts[i1.ID.String()] = workitem.WICountsPerIteration{
		IterationId: i1.ID.String(),
		Total:       10,
		Closed:      8,
	}
	counts[i2.ID.String()] = workitem.WICountsPerIteration{
		IterationId: i2.ID.String(),
		Total:       3,
		Closed:      1,
	}
	iterationSliceWithCounts := UpdateIterationsWithCounts(iterationSlice, counts)
	require.Len(t, iterationSliceWithCounts, 3)
	for _, itr := range iterationSliceWithCounts {
		require.NotNil(t, itr.Relationships)
		require.NotNil(t, itr.Relationships.Workitems)
		require.NotNil(t, itr.Relationships.Workitems.Meta)
		if itr.ID.String() == i1.ID.String() {
			assert.Equal(t, 10, itr.Relationships.Workitems.Meta["total"])
			assert.Equal(t, 8, itr.Relationships.Workitems.Meta["closed"])
		}
		if itr.ID.String() == i2.ID.String() {
			assert.Equal(t, 3, itr.Relationships.Workitems.Meta["total"])
			assert.Equal(t, 1, itr.Relationships.Workitems.Meta["closed"])
		}
		if itr.ID.String() == i3.ID.String() {
			assert.Equal(t, 0, itr.Relationships.Workitems.Meta["total"])
			assert.Equal(t, 0, itr.Relationships.Workitems.Meta["closed"])
		}
	}
}

// helper function to get random app.Iteration
func createMinimumAppIteration(name string, spaceID string) *app.Iteration {
	iterationID := uuid.NewV4()
	spaceType := "spaces"
	iterationState := iteration.IterationStateNew
	i := &app.Iteration{
		Type: iteration.APIStringTypeIteration,
		ID:   &iterationID,
		Attributes: &app.IterationAttributes{
			Name:  &name,
			State: &iterationState,
		},
		Relationships: &app.IterationRelations{
			Space: &app.RelationGeneric{
				Data: &app.GenericData{
					Type: &spaceType,
					ID:   &spaceID,
				},
			},
		},
	}
	return i
}
