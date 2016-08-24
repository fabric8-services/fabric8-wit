package remotetracker

import (
	"fmt"

	"github.com/jinzhu/gorm"
	"github.com/robfig/cron"
)

type trackerSchedule struct {
	URL         string
	TrackerType string
	Query       string
	Schedule    string
}

// Schedule fetch and import of remote tracker items
func Schedule(db *gorm.DB) {
	c := cron.New()
	tq := fetchTrackerQueries(db)
	for _, v := range tq {
		scheduleFetchAndImport(v, c)
	}
	c.Start()
}

func fetchTrackerQueries(db *gorm.DB) []trackerSchedule {
	tsList := []trackerSchedule{}
	err := db.Table("trackers").Select("trackers.url, trackers.type as tracker_type, tracker_queries.query, tracker_queries.schedule").Joins("left join tracker_queries on tracker_queries.tracker_refer = trackers.id").Scan(&tsList).Error
	if err != nil {
		fmt.Println("schedule")
	}
	return tsList
}

func scheduleFetchAndImport(ts trackerSchedule, c *cron.Cron) {
	switch ts.TrackerType {
	case "github":
		g := Github{}
		c.AddFunc(ts.Schedule, func() {
			g.Fetch(ts.URL, ts.Query)
			g.Import()
		})
	case "jira":
		fmt.Println("jira")
	}
}
