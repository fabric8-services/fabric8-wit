package remoteworkitem

import (
	"github.com/almighty/almighty-core/log"
	"github.com/almighty/almighty-core/models"

	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/robfig/cron"
	uuid "github.com/satori/go.uuid"
)

// TrackerSchedule capture all configuration
type trackerSchedule struct {
	TrackerID   int
	URL         string
	TrackerType string
	Query       string
	Schedule    string
}

// Scheduler represents scheduler
type Scheduler struct {
	db *gorm.DB
}

var cr *cron.Cron

// NewScheduler creates a new Scheduler
func NewScheduler(db *gorm.DB) *Scheduler {
	s := Scheduler{db: db}
	return &s
}

// Stop scheduler
// This should be called only from main
func (s *Scheduler) Stop() {
	cr.Stop()
}

func batchID() string {
	u1 := uuid.NewV4().String()
	return u1
}

// ScheduleAllQueries fetch and import of remote tracker items
func (s *Scheduler) ScheduleAllQueries() {
	cr.Stop()

	trackerQueries := fetchTrackerQueries(s.db)
	for _, tq := range trackerQueries {
		cr.AddFunc(tq.Schedule, func() {
			tr := lookupProvider(tq)
			for i := range tr.Fetch() {
				models.Transactional(s.db, func(tx *gorm.DB) error {
					// Save the remote items in a 'temporary' table.
					err := upload(tx, tq.TrackerID, i)
					if err != nil {
						return errors.WithStack(err)
					}
					// Convert the remote item into a local work item and persist in the DB.
					_, err = convert(tx, tq.TrackerID, i, tq.TrackerType)
					return errors.WithStack(err)
				})
			}
		})
	}
	cr.Start()
}

func fetchTrackerQueries(db *gorm.DB) []trackerSchedule {
	tsList := []trackerSchedule{}
	err := db.Table("tracker_queries").Select("trackers.id as tracker_id, trackers.url, trackers.type as tracker_type, tracker_queries.query, tracker_queries.schedule").Joins("left join trackers on tracker_queries.tracker_id = trackers.id").Where("trackers.deleted_at is NULL AND tracker_queries.deleted_at is NULL").Scan(&tsList).Error
	if err != nil {
		log.LogError(nil, map[string]interface{}{
			"err": err,
		}, "Fetch failed for tracker queries")
	}
	return tsList
}

// lookupProvider provides the respective tracker based on the type
func lookupProvider(ts trackerSchedule) TrackerProvider {
	switch ts.TrackerType {
	case ProviderGithub:
		return &GithubTracker{URL: ts.URL, Query: ts.Query}
	case ProviderJira:
		return &JiraTracker{URL: ts.URL, Query: ts.Query}
	}
	return nil
}

// TrackerItemContent represents a remote tracker item with it's content and unique ID
type TrackerItemContent struct {
	ID      string
	Content []byte
}

// TrackerProvider represents a remote tracker
type TrackerProvider interface {
	Fetch() chan TrackerItemContent // TODO: Change to an interface to enforce the contract
}

func init() {
	cr = cron.New()
	cr.Start()
}
