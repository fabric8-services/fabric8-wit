package gormsupport

import (
	"time"

	"github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/lib/pq"
	errs "github.com/pkg/errors"
)

const (
	// ChanSpaceTemplateUpdates is the name for the postgres notification
	// channel on which subscribers are informed about updates to the space
	// templates (e.g. when a migration has happened).
	ChanSpaceTemplateUpdates = "f8_space_template_updates"
)

// A SubscriberFunc describes the function signature that a subscriber needs to
// have. The channel parameter is just an arbitrary identifier string the
// identities a channel. The extra parameter is can contain optional data that
// was sent along with the notification.
type SubscriberFunc func(channel, extra string)

// SetupDatabaseListener sets up a Postgres LISTEN/NOTIFY connection and listens
// on events that we have subscribers for. You can have more than one subscriber
// for a single event channel.
func SetupDatabaseListener(config configuration.Registry, subscribers map[string][]SubscriberFunc) error {
	if len(subscribers) == 0 {
		return nil
	}

	dbConnectCallback := func(ev pq.ListenerEventType, err error) {
		switch ev {
		case pq.ListenerEventConnected:
			log.Logger().Infof("database connection for LISTEN/NOTIFY established successfully")
		case pq.ListenerEventDisconnected:
			log.Logger().Errorf("lost LISTEN/NOTIFY database connection: %+v", err)
		case pq.ListenerEventReconnected:
			log.Logger().Infof("database connection for LISTEN/NOTIFY re-established successfully")
		case pq.ListenerEventConnectionAttemptFailed:
			log.Logger().Errorf("failed to connect to database for LISTEN/NOTIFY: %+v", err)
		}
	}

	listener := pq.NewListener(config.GetPostgresConfigString(), config.GetPostgresListenNotifyMinReconnectInterval(), config.GetPostgresListenNotifyMaxReconnectInterval(), dbConnectCallback)

	// listen on every subscribed channel
	for channel := range subscribers {
		err := listener.Listen(channel)
		if err != nil {
			log.Logger().Errorf("unable to open connection to database for LISTEN/NOTIFY %v", err)
			return errs.Wrapf(err, "failed listen to postgres channel \"%s\"", channel)
		}
	}

	// asynchronously handle notifications
	go func() {
		for {
			select {
			case n := <-listener.Notify:
				subs, ok := subscribers[n.Channel]
				if ok {
					log.Logger().Debugf("received notification from postgres channel \"%s\": %s", n.Channel, n.Extra)
					for _, sub := range subs {
						sub(n.Channel, n.Extra)
					}
				}
			case <-time.After(90 * time.Second):
				log.Logger().Infof("received no events for 90 seconds, checking connection")
				go func() {
					err := listener.Ping()
					if err != nil {
						log.Panic(nil, map[string]interface{}{
							"err": err,
						}, "failed to ping for LISTEN/NOTIFY database connection")
					}
				}()
			}
		}
	}()
	return nil
}
