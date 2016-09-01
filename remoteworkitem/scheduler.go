package remoteworkitem

import (
	"log"

	"github.com/almighty/almighty-core/models"
	"github.com/jinzhu/gorm"
	"github.com/robfig/cron"
)

// TrackerSchedule capture all configuration
type trackerSchedule struct {
	TrackerQueryID int
	URL            string
	TrackerType    string
	Query          string
	Schedule       string
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

// ScheduleAllQueries fetch and import of remote tracker items
func (s *Scheduler) ScheduleAllQueries() {
	cr.Stop()
	trackerQueries := fetchTrackerQueries(s.db)
	for _, tq := range trackerQueries {
		cr.AddFunc(tq.Schedule, func() {
			tr := LookupProvider(tq)
			go tr.Fetch()

			for i := range tr.NextItem() {
				upload(s.db, tq.TrackerQueryID, i)
			}
		})
	}
	cr.Start()
}

func fetchTrackerQueries(db *gorm.DB) []trackerSchedule {
	tsList := []trackerSchedule{}
	err := db.Table("trackers").Select("tracker_queries.id as tracker_query_id, trackers.url, trackers.type as tracker_type, tracker_queries.query, tracker_queries.schedule").Joins("left join tracker_queries on tracker_queries.tracker = trackers.id").Scan(&tsList).Error
	if err != nil {
		log.Printf("Fetch failed %v\n", err)
	}
	return tsList
}

// Github represents the Github tracker provider
type Github struct {
	URL   string
	Query string
	Item  chan map[string]string
}

// Jira represents the Jira tracker provider
type Jira struct {
	URL   string
	Query string
	Item  chan map[string]string
}

// LookupProvider provides the respective tracker based on the type
func LookupProvider(ts trackerSchedule) TrackerProvider {
	switch ts.TrackerType {
	case "github":
		item := make(chan map[string]string)
		return &Github{URL: ts.URL, Query: ts.Query, Item: item}
	case "jira":
		item := make(chan map[string]string)
		return &Jira{URL: ts.URL, Query: ts.Query, Item: item}
	}
	return nil
}

// TrackerProvider represents a remote tracker
type TrackerProvider interface {
	Fetch()
	NextItem() chan map[string]string
}

// upload imports the items into database
func upload(db *gorm.DB, tqID int, item map[string]string) error {
	ti := models.TrackerItem{Item: item["content"], RemoteItemID: item["id"], BatchID: item["batch_id"], TrackerQuery: uint64(tqID)}
	err := db.Create(&ti).Error
	return err
}

func init() {
	cr = cron.New()
	cr.Start()
}
