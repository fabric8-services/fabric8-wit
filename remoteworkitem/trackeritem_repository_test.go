package remoteworkitem

import (
	"golang.org/x/net/context"

	"testing"

	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/test"
	"github.com/almighty/almighty-core/transaction"
	"github.com/stretchr/testify/assert"
)

func TestConvertNewWorkItem(t *testing.T) {
	resource.Require(t, resource.Database)

	// Setting up the dependent tracker query and tracker data in the Database
	tr := Tracker{URL: "https://api.github.com/", Type: ProviderGithub}
	db.Create(&tr)
	defer db.Delete(&tr)

	tq := TrackerQuery{Query: "some random query", Schedule: "0 0 0 * * *", TrackerID: tr.ID}
	db.Create(&tq)
	defer db.Delete(&tq)

	t.Log("Created Tracker Query and Tracker")

	ts := models.NewGormTransactionSupport(db)

	transaction.Do(ts, func() error {
		t.Log("Scenario 1 : Scenario 1: Adding a work item which wasn't present.")

		remoteItemData := map[string]string{
			"content":  `{"title":"linking","url":"http://github.com/sbose/api/testonly/1","state":"closed","body":"body of issue","user.login":"sbose78","assignee.login":"pranav"}`,
			"id":       "http://github.com/sbose/api/testonly/1",
			"batch_id": "1",
		}

		workItem, err := convert(ts, int(tq.ID), remoteItemData, ProviderGithub)

		assert.Nil(t, err)
		assert.Equal(t, "linking", workItem.Fields[SystemTitle])
		assert.Equal(t, "sbose78", workItem.Fields[SystemCreator])
		assert.Equal(t, "pranav", workItem.Fields[SystemAssignee])
		assert.Equal(t, "closed", workItem.Fields[SystemState])

		witr := models.NewWorkItemTypeRepository(ts)
		wir := models.NewWorkItemRepository(ts, witr)
		wir.Delete(context.Background(), workItem.ID)

		return err
	})
}

func TestConvertExistingWorkItem(t *testing.T) {
	resource.Require(t, resource.Database)

	// Setting up the dependent tracker query and tracker data in the Database
	tr := Tracker{URL: "https://api.github.com/", Type: ProviderGithub}
	db.Create(&tr)
	defer db.Delete(&tr)

	tq := TrackerQuery{Query: "some random query", Schedule: "0 0 0 * * *", TrackerID: tr.ID}
	db.Create(&tq)
	defer db.Delete(&tq)

	t.Log("Created Tracker Query and Tracker")

	ts := models.NewGormTransactionSupport(db)

	transaction.Do(ts, func() error {
		t.Log("Adding a work item which wasn't present.")

		remoteItemData := map[string]string{
			"content":  `{"title":"linking","url":"http://github.com/sbose/api/testonly/1","state":"closed","body":"body of issue","user.login":"sbose78","assignee.login":"pranav"}`,
			"id":       "http://github.com/sbose/api/testonly/1",
			"batch_id": "1",
		}

		workItem, err := convert(ts, int(tq.ID), remoteItemData, ProviderGithub)

		assert.Nil(t, err)
		assert.Equal(t, "linking", workItem.Fields[SystemTitle])
		assert.Equal(t, "sbose78", workItem.Fields[SystemCreator])
		assert.Equal(t, "pranav", workItem.Fields[SystemAssignee])
		assert.Equal(t, "closed", workItem.Fields[SystemState])
		return err
	})

	t.Log("Updating the existing work item when it's reimported.")

	transaction.Do(ts, func() error {
		remoteItemDataUpdated := map[string]string{
			"content":  `{"title":"linking-updated","url":"http://github.com/api/testonly/1","state":"closed","body":"body of issue","user.login":"sbose78","assignee.login":"pranav"}`,
			"id":       "http://github.com/sbose/api/testonly/1",
			"batch_id": "2",
		}
		workItemUpdated, err := convert(ts, int(tq.ID), remoteItemDataUpdated, ProviderGithub)

		assert.Nil(t, err)
		assert.Equal(t, "linking-updated", workItemUpdated.Fields[SystemTitle])
		assert.Equal(t, "sbose78", workItemUpdated.Fields[SystemCreator])
		assert.Equal(t, "pranav", workItemUpdated.Fields[SystemAssignee])
		assert.Equal(t, "closed", workItemUpdated.Fields[SystemState])

		witr := models.NewWorkItemTypeRepository(ts)
		wir := models.NewWorkItemRepository(ts, witr)
		wir.Delete(context.Background(), workItemUpdated.ID)

		return err
	})

}

func TestConvertGithubIssue(t *testing.T) {
	resource.Require(t, resource.Database)

	t.Log("Scenario 3 : Mapping and persisting a Github issue")

	ts := models.NewGormTransactionSupport(db)

	tr := Tracker{URL: "https://api.github.com/", Type: ProviderGithub}
	db.Create(&tr)
	defer db.Delete(&tr)

	tq := TrackerQuery{Query: "some random query", Schedule: "0 0 0 * * *", TrackerID: tr.ID}
	db.Create(&tq)
	defer db.Delete(&tq)

	transaction.Do(ts, func() error {
		content, err := test.LoadTestData("github_issue_mapping.json", provideRemoteGithubDataWithAssignee)
		if err != nil {
			t.Fatal(err)
		}

		remoteItemDataGithub := map[string]string{
			"content":  string(content[:]),
			"id":       GithubIssueWithAssignee, // GH issue url
			"batch_id": "2",
		}

		workItemGithub, err := convert(ts, int(tq.ID), remoteItemDataGithub, ProviderGithub)

		assert.Nil(t, err)
		assert.Equal(t, "map flatten : test case : with assignee", workItemGithub.Fields[SystemTitle])
		assert.Equal(t, "sbose78", workItemGithub.Fields[SystemCreator])
		assert.Equal(t, "sbose78", workItemGithub.Fields[SystemAssignee])
		assert.Equal(t, "open", workItemGithub.Fields[SystemState])

		return err
	})

}
