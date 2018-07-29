package gormsupport_test

import (
	"sync"
	"testing"

	"github.com/fabric8-services/fabric8-wit/gormsupport"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/migration"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestListenerSuite struct {
	gormtestsupport.DBTestSuite
}

func TestListener(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &TestListenerSuite{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

func (s *TestListenerSuite) TestSetupDatabaseListener() {
	s.T().Run("setup listener", func(t *testing.T) {
		// given
		channelName := "f8_custom_event_channel"
		payload := "some additional info about the event"
		wg := sync.WaitGroup{}
		wg.Add(2)
		var executedMigration bool

		err := gormsupport.SetupDatabaseListener(*s.Configuration, map[string]gormsupport.SubscriberFunc{
			// This is the channel we send to from this test
			channelName: func(channel, extra string) {
				t.Logf("received notification on channel %s: %s", channel, extra)
				require.Equal(t, channelName, channel)
				require.Equal(t, payload, extra)
				wg.Done()
			},
			// This is the channel that we send to from
			// migration.PopulateCommonTypes() which is called by
			// gormtestsupport.DBTestSuite internally.
			gormsupport.ChanSpaceTemplateUpdates: func(channel, extra string) {
				// potentially the migration is executed twice but we're only
				// interested in one event.
				if !executedMigration {
					executedMigration = true
					t.Logf("received notification on channel %s: %s", channel, extra)
					require.Equal(t, gormsupport.ChanSpaceTemplateUpdates, channel)
					require.Equal(t, "", extra)
					wg.Done()
				}
			},
		})
		require.NoError(t, err)

		// Send a notification from a completely different connection than the
		// one we established to listen to channels.
		db := s.DB.Debug().Exec("SELECT pg_notify($1, $2)", channelName, payload)
		require.NoError(t, db.Error)

		// This will send a notification on the
		// gormsupport.ChanSpaceTemplateUpdates channel
		err = migration.PopulateCommonTypes(nil, s.DB)
		require.NoError(t, err)

		// wait until notification was received
		wg.Wait()
	})
}
