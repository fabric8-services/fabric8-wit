package remoteworkitem

import (
	"golang.org/x/net/context"

	"testing"

	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/rendering"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/test"
	"github.com/almighty/almighty-core/workitem"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertNewWorkItem(t *testing.T) {
	resource.Require(t, resource.Database)

	// Setting up the dependent tracker query and tracker data in the Database
	tr := Tracker{URL: "https://api.github.com/", Type: ProviderGithub}
	db = db.Create(&tr)
	require.Nil(t, db.Error)
	defer db.Delete(&tr)

	tq := TrackerQuery{Query: "some random query", Schedule: "0 0 0 * * *", TrackerID: tr.ID}
	db = db.Create(&tq)
	require.Nil(t, db.Error)
	defer db.Delete(&tq)

	t.Log("Created Tracker Query and Tracker")

	models.Transactional(db, func(tx *gorm.DB) error {
		t.Log("Scenario 1 : Scenario 1: Adding a work item which wasn't present.")

		remoteItemData := TrackerItemContent{
			Content: []byte(`{"title":"linking","url":"http://github.com/sbose/api/testonly/1","state":"closed","body":"body of issue","user.login":"sbose78","assignee.login":"pranav"}`),
			ID:      "http://github.com/sbose/api/testonly/1",
		}

		workItem, err := convert(db, int(tq.ID), remoteItemData, ProviderGithub)

		require.Nil(t, err)
		require.NotNil(t, workItem.Fields)
		assert.Equal(t, "linking", workItem.Fields[workitem.SystemTitle])
		assert.Equal(t, "sbose78", workItem.Fields[workitem.SystemCreator])
		assert.Equal(t, "pranav", workItem.Fields[workitem.SystemAssignees].([]interface{})[0])
		assert.Equal(t, "closed", workItem.Fields[workitem.SystemState])
		require.NotNil(t, workItem.Fields[workitem.SystemDescription])
		description := workItem.Fields[workitem.SystemDescription].(rendering.MarkupContent)
		assert.Equal(t, "body of issue", description.Content)
		assert.Equal(t, rendering.SystemMarkupMarkdown, description.Markup)

		wir := workitem.NewWorkItemRepository(db)
		wir.Delete(context.Background(), workItem.ID)

		return errors.WithStack(err)
	})
}

func TestConvertExistingWorkItem(t *testing.T) {
	resource.Require(t, resource.Database)

	// Setting up the dependent tracker query and tracker data in the Database
	tr := Tracker{URL: "https://api.github.com/", Type: ProviderGithub}
	db = db.Create(&tr)
	require.Nil(t, db.Error)
	defer db.Delete(&tr)

	tq := TrackerQuery{Query: "some random query", Schedule: "0 0 0 * * *", TrackerID: tr.ID}
	db = db.Create(&tq)
	require.Nil(t, db.Error)
	defer db.Delete(&tq)

	t.Log("Created Tracker Query and Tracker")

	models.Transactional(db, func(tx *gorm.DB) error {
		t.Log("Adding a work item which wasn't present.")

		remoteItemData := TrackerItemContent{
			Content: []byte(`{"title":"linking","url":"http://github.com/sbose/api/testonly/1","state":"closed","body":"body of issue","user.login":"sbose78","assignee.login":"pranav"}`),
			ID:      "http://github.com/sbose/api/testonly/1",
		}

		workItem, err := convert(tx, int(tq.ID), remoteItemData, ProviderGithub)

		assert.Nil(t, err)
		assert.Equal(t, "linking", workItem.Fields[workitem.SystemTitle])
		assert.Equal(t, "sbose78", workItem.Fields[workitem.SystemCreator])
		assert.Equal(t, "pranav", workItem.Fields[workitem.SystemAssignees].([]interface{})[0])
		assert.Equal(t, "closed", workItem.Fields[workitem.SystemState])
		return errors.WithStack(err)
	})

	t.Log("Updating the existing work item when it's reimported.")

	models.Transactional(db, func(tx *gorm.DB) error {
		remoteItemDataUpdated := TrackerItemContent{
			Content: []byte(`{"title":"linking-updated","url":"http://github.com/api/testonly/1","state":"closed","body":"body of issue","user.login":"sbose78","assignee.login":"pranav"}`),
			ID:      "http://github.com/sbose/api/testonly/1",
		}
		workItemUpdated, err := convert(tx, int(tq.ID), remoteItemDataUpdated, ProviderGithub)

		assert.Nil(t, err)
		assert.Equal(t, "linking-updated", workItemUpdated.Fields[workitem.SystemTitle])
		assert.Equal(t, "sbose78", workItemUpdated.Fields[workitem.SystemCreator])
		assert.Equal(t, "pranav", workItemUpdated.Fields[workitem.SystemAssignees].([]interface{})[0])
		assert.Equal(t, "closed", workItemUpdated.Fields[workitem.SystemState])

		wir := workitem.NewWorkItemRepository(tx)
		wir.Delete(context.Background(), workItemUpdated.ID)

		return errors.WithStack(err)
	})

}

var GitIssueWithAssignee = "http://api.github.com/repos/almighty-test/almighty-test-unit/issues/2"

func TestConvertGithubIssue(t *testing.T) {
	resource.Require(t, resource.Database)

	t.Log("Scenario 3 : Mapping and persisting a Github issue")

	tr := Tracker{URL: "https://api.github.com/", Type: ProviderGithub}
	db = db.Create(&tr)
	require.Nil(t, db.Error)
	defer db.Delete(&tr)

	tq := TrackerQuery{Query: "some random query", Schedule: "0 0 0 * * *", TrackerID: tr.ID}
	db.Create(&tq)
	defer db.Delete(&tq)

	models.Transactional(db, func(tx *gorm.DB) error {
		content, err := test.LoadTestData("github_issue_mapping.json", func() ([]byte, error) {
			return provideRemoteData(GitIssueWithAssignee)
		})
		if err != nil {
			t.Fatal(err)
		}

		remoteItemDataGithub := TrackerItemContent{
			Content: content[:],
			ID:      GitIssueWithAssignee, // GH issue url
		}

		workItemGithub, err := convert(tx, int(tq.ID), remoteItemDataGithub, ProviderGithub)

		assert.Nil(t, err)
		assert.Equal(t, "map flatten : test case : with assignee", workItemGithub.Fields[workitem.SystemTitle])
		assert.Equal(t, "sbose78", workItemGithub.Fields[workitem.SystemCreator])
		assert.Equal(t, "sbose78", workItemGithub.Fields[workitem.SystemAssignees].([]interface{})[0])
		assert.Equal(t, "open", workItemGithub.Fields[workitem.SystemState])

		return errors.WithStack(err)
	})

}
